package notification

import (
	"fmt"
	"strings"
	"time"

	"github.com/namhq1989/go-utilities/appcontext"
	"github.com/namhq1989/go-utilities/logger"
	"github.com/namhq1989/go-utilities/uuid"
)

const (
	tokenMinLength = 100
	tokenMaxLength = 300
	tokenMaxAge    = 90 * 24 * time.Hour
)

// SaveToken registers or updates an FCM token for a user.
// Idempotent: existing active token → update last_used_at; inactive → reactivate.
// New token → deactivate old tokens for same user+platform → create.
//
// Race safety: concurrent SaveToken with same rawToken is handled via isDuplicateError.
// Concurrent SaveToken for same user+platform with DIFFERENT rawToken relies on the
// partial unique index (app_user_id, platform) WHERE is_active=true in the caller's DB.
// This index is REQUIRED in the migration — without it, the "1 active/user/platform" invariant breaks.
func (c *Client) SaveToken(ctx *appcontext.AppContext, store TokenStore, params SaveTokenParams) error {
	ctx.Logger().Text("save FCM token")

	if err := validateTokenFormat(params.RawToken); err != nil {
		ctx.Logger().SilentError("invalid token format", err, logger.Fields{
			"userID": params.UserID,
		})
		return err
	}

	if !params.Platform.IsValid() {
		ctx.Logger().SilentError("invalid platform", nil, logger.Fields{
			"userID":   params.UserID,
			"platform": params.Platform.String(),
		})
		return newError(KeyNotificationInvalidPlatform)
	}

	tokenHash := HashToken(params.RawToken)

	// Check if token already exists
	existing, err := store.FindByHash(ctx, tokenHash)
	if err != nil {
		ctx.Logger().SilentError("failed to find token by hash", err, logger.Fields{})
		return newError(KeyNotificationSendFailed)
	}

	if existing != nil {
		if existing.IsActive {
			// Active token exists — update last_used_at
			if err = store.UpdateLastUsed(ctx, existing.ID); err != nil {
				ctx.Logger().SilentError("failed to update last used", err, logger.Fields{
					"tokenID": existing.ID,
				})
				return newError(KeyNotificationSendFailed)
			}
			return nil
		}
		// Inactive token — reactivate
		if err = store.Reactivate(ctx, existing.ID); err != nil {
			ctx.Logger().SilentError("failed to reactivate token", err, logger.Fields{
				"tokenID": existing.ID,
			})
			return newError(KeyNotificationSendFailed)
		}
		return nil
	}

	// New token — deactivate old tokens for same user+platform
	if err = store.DeactivateByUserAndPlatform(ctx, params.UserID, params.Platform); err != nil {
		ctx.Logger().SilentError("failed to deactivate old tokens", err, logger.Fields{
			"userID":   params.UserID,
			"platform": params.Platform.String(),
		})
		return newError(KeyNotificationSendFailed)
	}

	// Create new token
	now := time.Now().UTC()
	token := FCMToken{
		ID:         uuid.New(),
		AppUserID:  params.UserID,
		TokenHash:  tokenHash,
		RawToken:   params.RawToken,
		Platform:   params.Platform,
		IsActive:   true,
		LastUsedAt: now,
		UpdatedAt:  now,
		CreatedAt:  now,
	}

	if err = store.Create(ctx, token); err != nil {
		// Handle race condition: concurrent SaveToken with same rawToken
		// If duplicate on token_hash unique constraint, treat as success
		if isDuplicateError(err) {
			ctx.Logger().Info("token already created by concurrent request", logger.Fields{})
			return nil
		}
		ctx.Logger().SilentError("failed to create token", err, logger.Fields{
			"userID": params.UserID,
		})
		return newError(KeyNotificationSendFailed)
	}

	return nil
}

// RemoveToken deactivates a specific token for a user (e.g., on single device logout)
func (c *Client) RemoveToken(ctx *appcontext.AppContext, store TokenStore, userID, rawToken string) error {
	ctx.Logger().Text("remove FCM token")

	tokenHash := HashToken(rawToken)

	if err := store.DeactivateByHash(ctx, tokenHash); err != nil {
		ctx.Logger().SilentError("failed to deactivate token", err, logger.Fields{
			"userID": userID,
		})
		return newError(KeyNotificationSendFailed)
	}

	return nil
}

// RemoveAllTokens deletes all tokens for a user (e.g., on account deletion)
func (c *Client) RemoveAllTokens(ctx *appcontext.AppContext, store TokenStore, userID string) error {
	ctx.Logger().Text("remove all FCM tokens for user")

	if err := store.DeleteByUserID(ctx, userID); err != nil {
		ctx.Logger().SilentError("failed to delete all tokens", err, logger.Fields{
			"userID": userID,
		})
		return newError(KeyNotificationSendFailed)
	}

	return nil
}

// CleanupExpiredTokens deactivates tokens unused for 90+ days
func (c *Client) CleanupExpiredTokens(ctx *appcontext.AppContext, store TokenStore) (*CleanupResult, error) {
	ctx.Logger().Text("cleanup expired FCM tokens")

	start := time.Now()

	count, err := store.DeactivateOlderThan(ctx, tokenMaxAge)
	if err != nil {
		ctx.Logger().SilentError("failed to cleanup expired tokens", err, logger.Fields{})
		return nil, newError(KeyNotificationSendFailed)
	}

	result := &CleanupResult{
		TokensDeactivated: count,
		Duration:          time.Since(start),
	}

	ctx.Logger().Info("expired tokens cleaned up", logger.Fields{
		"deactivated": count,
		"durationMs":  result.Duration.Milliseconds(),
	})

	return result, nil
}

// validateTokenFormat validates FCM token format (100-300 chars)
func validateTokenFormat(token string) error {
	if token == "" {
		return newError(KeyNotificationInvalidToken)
	}
	if len(token) < tokenMinLength || len(token) > tokenMaxLength {
		return newError(KeyNotificationInvalidToken)
	}
	return nil
}

// isDuplicateError checks if the error is a unique constraint violation.
// This is a simple heuristic; callers using go-jet/pgx will have proper error checking.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := fmt.Sprintf("%v", err)
	return strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "unique constraint") || strings.Contains(errStr, "23505")
}
