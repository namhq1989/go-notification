package notification

import (
	"fmt"

	"firebase.google.com/go/v4/messaging"
	"github.com/namhq1989/go-utilities/appcontext"
	"github.com/namhq1989/go-utilities/logger"
)

// SendToUser sends a notification to all active tokens of a user.
// Returns SendResult with details about success/failure.
func (c *Client) SendToUser(ctx *appcontext.AppContext, store TokenStore, userID string, msg Message) (*SendResult, error) {
	ctx.Logger().Text("send notification to user")

	if !c.enabled {
		ctx.Logger().Info("dev mode: notification logged, not sent", logger.Fields{
			"userID": userID,
			"title":  msg.Title,
		})
		return &SendResult{Sent: 1}, nil
	}

	tokens, err := store.FindActiveByUserID(ctx, userID)
	if err != nil {
		ctx.Logger().SilentError("failed to find tokens for user", err, logger.Fields{
			"userID": userID,
		})
		return nil, newError(KeyNotificationSendFailed)
	}

	if len(tokens) == 0 {
		return &SendResult{Sent: 0, Failed: []string{userID}}, nil
	}

	// Build messages and send as batch
	var messages []*messaging.Message
	for _, token := range tokens {
		messages = append(messages, c.firebase.buildMessage(token.RawToken, "", msg))
	}

	resp, batchErr := c.firebase.SendBatch(ctx, messages)
	if batchErr != nil {
		ctx.Logger().SilentError("failed to send to user", batchErr, logger.Fields{
			"userID": userID,
		})
		return &SendResult{Sent: 0, Failed: []string{userID}}, nil
	}

	// Process responses
	successCount := 0
	for j, sendResp := range resp.Responses {
		if sendResp.Success {
			successCount++
			continue
		}
		if sendResp.Error != nil {
			errType := ClassifyFCMError(sendResp.Error)
			if errType.ShouldDeactivateToken() {
				_ = store.DeactivateByHash(ctx, tokens[j].TokenHash)
			}
		}
	}

	result := &SendResult{Sent: successCount}
	if successCount == 0 {
		result.Failed = []string{userID}
	}

	return result, nil
}

// SendToUsers sends a notification to multiple users.
// Batches FCM messages (max 500 per call). Returns SendResult with partial failure info.
func (c *Client) SendToUsers(ctx *appcontext.AppContext, store TokenStore, userIDs []string, msg Message) (*SendResult, error) {
	ctx.Logger().Text("send notification to users")

	if len(userIDs) == 0 {
		return &SendResult{}, nil
	}

	if !c.enabled {
		ctx.Logger().Info("dev mode: batch notification logged, not sent", logger.Fields{
			"count": len(userIDs),
			"title": msg.Title,
		})
		return &SendResult{Sent: len(userIDs)}, nil
	}

	tokens, err := store.FindActiveByUserIDs(ctx, userIDs)
	if err != nil {
		ctx.Logger().SilentError("failed to find tokens for users", err, logger.Fields{
			"count": len(userIDs),
		})
		return nil, newError(KeyNotificationSendFailed)
	}

	if len(tokens) == 0 {
		return &SendResult{Sent: 0, Failed: userIDs}, nil
	}

	// Build Firebase messages
	var messages []*messaging.Message
	for _, token := range tokens {
		messages = append(messages, c.firebase.buildMessage(token.RawToken, "", msg))
	}

	// Track which users had at least one success
	userSuccess := make(map[string]bool)

	// Send in chunks of 500
	for i := 0; i < len(messages); i += maxFCMBatchSize {
		end := i + maxFCMBatchSize
		if end > len(messages) {
			end = len(messages)
		}

		chunk := messages[i:end]
		chunkTokens := tokens[i:end]

		resp, batchErr := c.firebase.SendBatch(ctx, chunk)
		if batchErr != nil {
			ctx.Logger().SilentError("batch send failed", batchErr, logger.Fields{
				"chunkStart": i,
				"chunkSize":  len(chunk),
			})
			continue
		}

		// Process individual responses
		for j, sendResp := range resp.Responses {
			tokenObj := chunkTokens[j]
			if sendResp.Success {
				userSuccess[tokenObj.AppUserID] = true
				continue
			}

			if sendResp.Error != nil {
				errType := ClassifyFCMError(sendResp.Error)
				if errType.ShouldDeactivateToken() {
					_ = store.DeactivateByHash(ctx, tokenObj.TokenHash)
				}
			}
		}
	}

	// Determine failed users
	var failed []string
	for _, uid := range userIDs {
		if !userSuccess[uid] {
			failed = append(failed, uid)
		}
	}

	result := &SendResult{
		Sent:   len(userIDs) - len(failed),
		Failed: failed,
	}

	if len(failed) > 0 && len(failed) < len(userIDs) {
		result.Partial = fmt.Errorf("partial failure: %d/%d users failed", len(failed), len(userIDs))
	}

	ctx.Logger().Info("batch send completed", logger.Fields{
		"sent":   result.Sent,
		"failed": len(result.Failed),
	})

	return result, nil
}

// SendToTopic sends a notification to a topic
func (c *Client) SendToTopic(ctx *appcontext.AppContext, topic string, msg Message) error {
	ctx.Logger().Text("send notification to topic")

	if topic == "" {
		return newError(KeyNotificationTopicEmpty)
	}

	if !c.enabled {
		ctx.Logger().Info("dev mode: topic notification logged, not sent", logger.Fields{
			"topic": topic,
			"title": msg.Title,
		})
		return nil
	}

	if err := c.firebase.SendToTopic(ctx, topic, msg); err != nil {
		ctx.Logger().SilentError("failed to send to topic", err, logger.Fields{
			"topic": topic,
		})
		return newError(KeyNotificationSendFailed)
	}

	return nil
}
