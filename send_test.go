package notification

import (
	"context"
	"errors"
	"testing"

	"firebase.google.com/go/v4/messaging"
	"github.com/namhq1989/go-utilities/appcontext"
)

// mockFirebase implements firebaseSender for testing
type mockFirebase struct {
	// batchResponses is returned by SendBatch (one per call, consumed in order)
	batchResponses []*messaging.BatchResponse
	batchErrors    []error
	batchCallCount int

	sendToDeviceErr error
	sendToTopicErr  error
}

func (m *mockFirebase) SendBatch(ctx *appcontext.AppContext, messages []*messaging.Message) (*messaging.BatchResponse, error) {
	idx := m.batchCallCount
	m.batchCallCount++

	if idx < len(m.batchErrors) && m.batchErrors[idx] != nil {
		return nil, m.batchErrors[idx]
	}
	if idx < len(m.batchResponses) {
		return m.batchResponses[idx], nil
	}
	// Default: all success
	responses := make([]*messaging.SendResponse, len(messages))
	for i := range messages {
		responses[i] = &messaging.SendResponse{Success: true}
	}
	return &messaging.BatchResponse{
		Responses:    responses,
		SuccessCount: len(messages),
	}, nil
}

func (m *mockFirebase) SendToDevice(_ *appcontext.AppContext, _ string, _ Message) error {
	return m.sendToDeviceErr
}

func (m *mockFirebase) SendToTopic(_ *appcontext.AppContext, _ string, _ Message) error {
	return m.sendToTopicErr
}

func (m *mockFirebase) buildMessage(token, topic string, msg Message) *messaging.Message {
	m2 := &messaging.Message{
		Notification: &messaging.Notification{Title: msg.Title, Body: msg.Body},
		Data:         msg.Data,
	}
	if token != "" {
		m2.Token = token
	}
	if topic != "" {
		m2.Topic = topic
	}
	return m2
}

// --- Tests exercising core send logic ---

func TestSendToUser_AllSuccess(t *testing.T) {
	fb := &mockFirebase{}
	client := &Client{firebase: fb, enabled: true}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	// Add 2 active tokens for user
	store.tokens["h1"] = &FCMToken{ID: "t1", AppUserID: "u1", TokenHash: "h1", RawToken: "raw1", Platform: PlatformIOS, IsActive: true}
	store.tokens["h2"] = &FCMToken{ID: "t2", AppUserID: "u1", TokenHash: "h2", RawToken: "raw2", Platform: PlatformAndroid, IsActive: true}

	result, err := client.SendToUser(ctx, store, "u1", Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 2 {
		t.Errorf("expected Sent=2, got %d", result.Sent)
	}
	if len(result.Failed) != 0 {
		t.Errorf("expected no failures, got %v", result.Failed)
	}
}

func TestSendToUser_PartialFailure_DeactivatesToken(t *testing.T) {
	// 2 tokens: first fails with invalid token error, second succeeds
	fb := &mockFirebase{
		batchResponses: []*messaging.BatchResponse{
			{
				Responses: []*messaging.SendResponse{
					{Success: false, Error: errors.New("registration-token-not-registered")},
					{Success: true},
				},
				SuccessCount: 1,
				FailureCount: 1,
			},
		},
	}
	client := &Client{firebase: fb, enabled: true}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	store.tokens["h1"] = &FCMToken{ID: "t1", AppUserID: "u1", TokenHash: "h1", RawToken: "raw1", Platform: PlatformIOS, IsActive: true}
	store.tokens["h2"] = &FCMToken{ID: "t2", AppUserID: "u1", TokenHash: "h2", RawToken: "raw2", Platform: PlatformAndroid, IsActive: true}

	result, err := client.SendToUser(ctx, store, "u1", Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 1 {
		t.Errorf("expected Sent=1, got %d", result.Sent)
	}
	// First token should be deactivated (invalid token error)
	if store.tokens["h1"].IsActive {
		t.Error("expected token h1 to be deactivated")
	}
	// Second token still active
	if !store.tokens["h2"].IsActive {
		t.Error("expected token h2 to remain active")
	}
}

func TestSendToUser_AllFail(t *testing.T) {
	fb := &mockFirebase{
		batchResponses: []*messaging.BatchResponse{
			{
				Responses: []*messaging.SendResponse{
					{Success: false, Error: errors.New("internal-error")},
					{Success: false, Error: errors.New("registration-token-not-registered")},
				},
				FailureCount: 2,
			},
		},
	}
	client := &Client{firebase: fb, enabled: true}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	store.tokens["h1"] = &FCMToken{ID: "t1", AppUserID: "u1", TokenHash: "h1", RawToken: "raw1", Platform: PlatformIOS, IsActive: true}
	store.tokens["h2"] = &FCMToken{ID: "t2", AppUserID: "u1", TokenHash: "h2", RawToken: "raw2", Platform: PlatformAndroid, IsActive: true}

	result, err := client.SendToUser(ctx, store, "u1", Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 0 {
		t.Errorf("expected Sent=0, got %d", result.Sent)
	}
	if len(result.Failed) != 1 || result.Failed[0] != "u1" {
		t.Errorf("expected Failed=[u1], got %v", result.Failed)
	}
}

func TestSendToUsers_MultiUser_PartialSuccess(t *testing.T) {
	// u1 has 1 token (success), u2 has 1 token (fail with invalid), u3 has 1 token (success)
	fb := &mockFirebase{
		batchResponses: []*messaging.BatchResponse{
			{
				Responses: []*messaging.SendResponse{
					{Success: true},
					{Success: false, Error: errors.New("NotRegistered")},
					{Success: true},
				},
				SuccessCount: 2,
				FailureCount: 1,
			},
		},
	}
	client := &Client{firebase: fb, enabled: true}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	store.tokens["h1"] = &FCMToken{ID: "t1", AppUserID: "u1", TokenHash: "h1", RawToken: "raw1", Platform: PlatformIOS, IsActive: true}
	store.tokens["h2"] = &FCMToken{ID: "t2", AppUserID: "u2", TokenHash: "h2", RawToken: "raw2", Platform: PlatformIOS, IsActive: true}
	store.tokens["h3"] = &FCMToken{ID: "t3", AppUserID: "u3", TokenHash: "h3", RawToken: "raw3", Platform: PlatformAndroid, IsActive: true}

	result, err := client.SendToUsers(ctx, store, []string{"u1", "u2", "u3"}, Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 2 {
		t.Errorf("expected Sent=2, got %d", result.Sent)
	}
	if len(result.Failed) != 1 || result.Failed[0] != "u2" {
		t.Errorf("expected Failed=[u2], got %v", result.Failed)
	}
	if result.Partial == nil {
		t.Error("expected Partial error to be set")
	}
	// u2's token should be deactivated
	if store.tokens["h2"].IsActive {
		t.Error("expected u2's token to be deactivated")
	}
}

func TestSendToUser_BatchError_ReturnsFailed(t *testing.T) {
	fb := &mockFirebase{
		batchErrors: []error{errors.New("firebase unavailable")},
	}
	client := &Client{firebase: fb, enabled: true}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	store.tokens["h1"] = &FCMToken{ID: "t1", AppUserID: "u1", TokenHash: "h1", RawToken: "raw1", Platform: PlatformIOS, IsActive: true}

	result, err := client.SendToUser(ctx, store, "u1", Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 0 {
		t.Errorf("expected Sent=0, got %d", result.Sent)
	}
	if len(result.Failed) != 1 {
		t.Errorf("expected Failed=[u1], got %v", result.Failed)
	}
}

func TestSendToTopic_ValidTopic(t *testing.T) {
	fb := &mockFirebase{}
	client := &Client{firebase: fb, enabled: true}
	ctx := appcontext.NewWorker(context.Background())

	err := client.SendToTopic(ctx, "news", Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendToTopic_FirebaseError(t *testing.T) {
	fb := &mockFirebase{sendToTopicErr: errors.New("auth failed")}
	client := &Client{firebase: fb, enabled: true}
	ctx := appcontext.NewWorker(context.Background())

	err := client.SendToTopic(ctx, "news", Message{Title: "T", Body: "B"})
	if err == nil {
		t.Error("expected error from firebase")
	}
}

// --- Edge case tests (kept from original) ---

func TestSendToUser_DevMode(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	result, err := client.SendToUser(ctx, store, "user-1", Message{Title: "Test", Body: "Hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 1 {
		t.Errorf("expected Sent=1 in dev mode, got %d", result.Sent)
	}
}

func TestSendToUser_NoTokens(t *testing.T) {
	fb := &mockFirebase{}
	client := &Client{enabled: true, firebase: fb}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	result, err := client.SendToUser(ctx, store, "user-1", Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 0 {
		t.Errorf("expected Sent=0, got %d", result.Sent)
	}
	if len(result.Failed) != 1 || result.Failed[0] != "user-1" {
		t.Errorf("expected Failed=[user-1], got %v", result.Failed)
	}
}

func TestSendToUsers_DevMode(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	result, err := client.SendToUsers(ctx, store, []string{"u1", "u2", "u3"}, Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 3 {
		t.Errorf("expected Sent=3 in dev mode, got %d", result.Sent)
	}
}

func TestSendToUsers_EmptyList(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	result, err := client.SendToUsers(ctx, store, []string{}, Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 0 {
		t.Errorf("expected Sent=0, got %d", result.Sent)
	}
}

func TestSendToUsers_NoTokens(t *testing.T) {
	fb := &mockFirebase{}
	client := &Client{enabled: true, firebase: fb}
	store := newMockTokenStore()
	ctx := appcontext.NewWorker(context.Background())

	result, err := client.SendToUsers(ctx, store, []string{"u1", "u2"}, Message{Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sent != 0 {
		t.Errorf("expected Sent=0, got %d", result.Sent)
	}
	if len(result.Failed) != 2 {
		t.Errorf("expected 2 failed, got %d", len(result.Failed))
	}
}

func TestSendToTopic_EmptyTopic(t *testing.T) {
	client := &Client{enabled: false}
	ctx := appcontext.NewWorker(context.Background())

	err := client.SendToTopic(ctx, "", Message{Title: "T", Body: "B"})
	e, ok := IsGoNotificationError(err)
	if !ok || e.Key != KeyNotificationTopicEmpty {
		t.Errorf("expected KeyNotificationTopicEmpty, got %v", err)
	}
}

func TestSendToUsers_StoreError(t *testing.T) {
	fb := &mockFirebase{}
	client := &Client{enabled: true, firebase: fb}
	ctx := appcontext.NewWorker(context.Background())

	_, err := client.SendToUsers(ctx, &errTokenStore{err: errors.New("db down")}, []string{"u1"}, Message{Title: "T", Body: "B"})
	if err == nil {
		t.Error("expected error from store")
	}
}

// errTokenStore is a store that always returns an error on Find methods
type errTokenStore struct {
	mockTokenStore
	err error
}

func (e *errTokenStore) FindActiveByUserIDs(_ *appcontext.AppContext, _ []string) ([]FCMToken, error) {
	return nil, e.err
}

func (e *errTokenStore) FindActiveByUserID(_ *appcontext.AppContext, _ string) ([]FCMToken, error) {
	return nil, e.err
}
