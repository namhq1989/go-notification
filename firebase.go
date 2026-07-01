package notification

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/namhq1989/go-utilities/appcontext"
	"github.com/namhq1989/go-utilities/logger"
	"google.golang.org/api/option"
)

const maxFCMBatchSize = 500

// firebaseSender defines the internal interface for sending FCM notifications.
// *FirebaseClient implements this. Tests inject a mock.
type firebaseSender interface {
	SendBatch(ctx *appcontext.AppContext, messages []*messaging.Message) (*messaging.BatchResponse, error)
	SendToDevice(ctx *appcontext.AppContext, token string, msg Message) error
	SendToTopic(ctx *appcontext.AppContext, topic string, msg Message) error
	buildMessage(token, topic string, msg Message) *messaging.Message
}

// Compile-time check
var _ firebaseSender = (*FirebaseClient)(nil)

// FirebaseClient wraps the Firebase messaging client
type FirebaseClient struct {
	client *messaging.Client
}

// NewFirebaseClient creates a new Firebase messaging client
func NewFirebaseClient(cfg Config) (*FirebaseClient, error) {
	ctx := context.Background()

	var opt option.ClientOption
	if cfg.CredentialsBase64 != "" {
		jsonBytes, err := base64.StdEncoding.DecodeString(cfg.CredentialsBase64)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to decode base64 credentials: %v", KeyNotificationMissingConfig, err)
		}
		opt = option.WithCredentialsJSON(jsonBytes)
	} else {
		opt = option.WithCredentialsFile(cfg.CredentialsFile)
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: cfg.ProjectID,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", KeyNotificationMissingConfig, err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", KeyNotificationMissingConfig, err)
	}

	return &FirebaseClient{client: client}, nil
}

// SendToDevice sends a notification to a specific device token
func (f *FirebaseClient) SendToDevice(ctx *appcontext.AppContext, token string, msg Message) error {
	message := f.buildMessage(token, "", msg)

	_, err := f.client.Send(ctx.Context(), message)
	if err != nil {
		ctx.Logger().SilentError("failed to send notification to device", err, logger.Fields{})
		return err
	}

	return nil
}

// SendToTopic sends a notification to a topic
func (f *FirebaseClient) SendToTopic(ctx *appcontext.AppContext, topic string, msg Message) error {
	message := f.buildMessage("", topic, msg)

	_, err := f.client.Send(ctx.Context(), message)
	if err != nil {
		ctx.Logger().SilentError("failed to send notification to topic", err, logger.Fields{
			"topic": topic,
		})
		return err
	}

	return nil
}

// SendBatch sends multiple messages via Firebase SendEach (up to 500 messages per call).
// Caller MUST chunk to maxFCMBatchSize before calling.
func (f *FirebaseClient) SendBatch(ctx *appcontext.AppContext, messages []*messaging.Message) (*messaging.BatchResponse, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	if len(messages) > maxFCMBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds limit %d", len(messages), maxFCMBatchSize)
	}

	resp, err := f.client.SendEach(ctx.Context(), messages)
	if err != nil {
		ctx.Logger().SilentError("failed to send batch notification", err, logger.Fields{
			"count": len(messages),
		})
		return nil, err
	}

	ctx.Logger().Info("batch notification sent", logger.Fields{
		"success": resp.SuccessCount,
		"failure": resp.FailureCount,
	})

	return resp, nil
}

// buildMessage constructs a Firebase messaging.Message
func (f *FirebaseClient) buildMessage(token, topic string, msg Message) *messaging.Message {
	m := &messaging.Message{
		Notification: &messaging.Notification{
			Title: msg.Title,
			Body:  msg.Body,
		},
		Data: msg.Data,
	}

	if token != "" {
		m.Token = token
	}
	if topic != "" {
		m.Topic = topic
	}

	// Set Android config with priority and TTL
	priority := "normal"
	if msg.Priority == PriorityHigh {
		priority = "high"
	}

	var ttl *time.Duration
	if msg.TTL > 0 {
		ttl = &msg.TTL
	}

	m.Android = &messaging.AndroidConfig{
		Priority: priority,
		TTL:      ttl,
	}

	// Set APNS config
	if msg.Priority == PriorityHigh {
		m.APNS = &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": "10",
			},
		}
	}

	return m
}
