package notification

import (
	"time"

	"github.com/namhq1989/go-utilities/appcontext"
)

// Notification defines the interface for push notification operations.
// Callers use this interface, never *Client directly.
type Notification interface {
	// Token management — store passed per-call (stateless client)
	SaveToken(ctx *appcontext.AppContext, store TokenStore, params SaveTokenParams) error
	RemoveToken(ctx *appcontext.AppContext, store TokenStore, userID, rawToken string) error
	RemoveAllTokens(ctx *appcontext.AppContext, store TokenStore, userID string) error

	// Send
	SendToUser(ctx *appcontext.AppContext, store TokenStore, userID string, msg Message) (*SendResult, error)
	SendToUsers(ctx *appcontext.AppContext, store TokenStore, userIDs []string, msg Message) (*SendResult, error)
	SendToTopic(ctx *appcontext.AppContext, topic string, msg Message) error

	// Cleanup (cron daily)
	CleanupExpiredTokens(ctx *appcontext.AppContext, store TokenStore) (*CleanupResult, error)
}

// TokenStore defines the interface callers must implement for token persistence.
// Injected per method call (stateless client pattern, like go-auth).
type TokenStore interface {
	Create(ctx *appcontext.AppContext, token FCMToken) error
	FindByHash(ctx *appcontext.AppContext, tokenHash string) (*FCMToken, error)
	FindActiveByUserID(ctx *appcontext.AppContext, userID string) ([]FCMToken, error)
	FindActiveByUserIDs(ctx *appcontext.AppContext, userIDs []string) ([]FCMToken, error)
	UpdateLastUsed(ctx *appcontext.AppContext, tokenID string) error
	Reactivate(ctx *appcontext.AppContext, tokenID string) error
	DeactivateByHash(ctx *appcontext.AppContext, tokenHash string) error
	DeactivateByUserAndPlatform(ctx *appcontext.AppContext, userID string, platform Platform) error
	DeactivateOlderThan(ctx *appcontext.AppContext, age time.Duration) (int, error)
	DeleteByUserID(ctx *appcontext.AppContext, userID string) error
}

// Compile-time check
var _ Notification = (*Client)(nil)
