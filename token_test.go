package notification

import (
	"context"
	"testing"
	"time"

	"github.com/namhq1989/go-utilities/appcontext"
)

// mockTokenStore is a test mock for TokenStore
type mockTokenStore struct {
	tokens          map[string]*FCMToken
	createErr       error
	findByHashErr   error
	deactivateErr   error
	deleteErr       error
	deactivateCount int
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]*FCMToken)}
}

func (m *mockTokenStore) Create(_ *appcontext.AppContext, token FCMToken) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.tokens[token.TokenHash] = &token
	return nil
}

func (m *mockTokenStore) FindByHash(_ *appcontext.AppContext, tokenHash string) (*FCMToken, error) {
	if m.findByHashErr != nil {
		return nil, m.findByHashErr
	}
	return m.tokens[tokenHash], nil
}

func (m *mockTokenStore) FindActiveByUserID(_ *appcontext.AppContext, userID string) ([]FCMToken, error) {
	var result []FCMToken
	for _, t := range m.tokens {
		if t.AppUserID == userID && t.IsActive {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTokenStore) FindActiveByUserIDs(_ *appcontext.AppContext, userIDs []string) ([]FCMToken, error) {
	userSet := make(map[string]bool)
	for _, uid := range userIDs {
		userSet[uid] = true
	}
	var result []FCMToken
	for _, t := range m.tokens {
		if userSet[t.AppUserID] && t.IsActive {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTokenStore) UpdateLastUsed(_ *appcontext.AppContext, tokenID string) error {
	for _, t := range m.tokens {
		if t.ID == tokenID {
			t.LastUsedAt = time.Now().UTC()
			return nil
		}
	}
	return nil
}

func (m *mockTokenStore) Reactivate(_ *appcontext.AppContext, tokenID string) error {
	for _, t := range m.tokens {
		if t.ID == tokenID {
			t.IsActive = true
			t.LastUsedAt = time.Now().UTC()
			return nil
		}
	}
	return nil
}

func (m *mockTokenStore) DeactivateByHash(_ *appcontext.AppContext, tokenHash string) error {
	if m.deactivateErr != nil {
		return m.deactivateErr
	}
	if t, ok := m.tokens[tokenHash]; ok {
		t.IsActive = false
	}
	m.deactivateCount++
	return nil
}

func (m *mockTokenStore) DeactivateByUserAndPlatform(_ *appcontext.AppContext, userID string, platform Platform) error {
	for _, t := range m.tokens {
		if t.AppUserID == userID && t.Platform == platform && t.IsActive {
			t.IsActive = false
		}
	}
	return nil
}

func (m *mockTokenStore) DeactivateOlderThan(_ *appcontext.AppContext, age time.Duration) (int, error) {
	count := 0
	cutoff := time.Now().UTC().Add(-age)
	for _, t := range m.tokens {
		if t.IsActive && t.LastUsedAt.Before(cutoff) {
			t.IsActive = false
			count++
		}
	}
	return count, nil
}

func (m *mockTokenStore) DeleteByUserID(_ *appcontext.AppContext, userID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for hash, t := range m.tokens {
		if t.AppUserID == userID {
			delete(m.tokens, hash)
		}
	}
	return nil
}

func newTestCtx() *appcontext.AppContext {
	return appcontext.NewWorker(context.Background())
}

// fakeToken generates a token string of valid length (152 chars)
func fakeToken(suffix string) string {
	base := "dRK3bG7vT9mN1pQ5sW8xZ2cF4hJ6kL0wE9rY1uI3oA5"
	token := base + base + base + suffix
	if len(token) > 200 {
		token = token[:200]
	}
	return token
}

func TestSaveToken_NewToken(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := newTestCtx()

	rawToken := fakeToken("new-token-001")
	err := client.SaveToken(ctx, store, SaveTokenParams{
		UserID:   "user-1",
		RawToken: rawToken,
		Platform: PlatformIOS,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hash := HashToken(rawToken)
	if _, ok := store.tokens[hash]; !ok {
		t.Error("token not created in store")
	}
}

func TestSaveToken_ExistingActiveToken(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := newTestCtx()

	rawToken := fakeToken("existing-active")
	hash := HashToken(rawToken)
	store.tokens[hash] = &FCMToken{
		ID:         "token-1",
		AppUserID:  "user-1",
		TokenHash:  hash,
		RawToken:   rawToken,
		Platform:   PlatformIOS,
		IsActive:   true,
		LastUsedAt: time.Now().Add(-24 * time.Hour),
	}

	err := client.SaveToken(ctx, store, SaveTokenParams{
		UserID:   "user-1",
		RawToken: rawToken,
		Platform: PlatformIOS,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should update last_used_at (via UpdateLastUsed)
	token := store.tokens[hash]
	if token == nil {
		t.Fatal("token should still exist")
	}
}

func TestSaveToken_ExistingInactiveToken(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := newTestCtx()

	rawToken := fakeToken("existing-inactive")
	hash := HashToken(rawToken)
	store.tokens[hash] = &FCMToken{
		ID:        "token-2",
		AppUserID: "user-1",
		TokenHash: hash,
		RawToken:  rawToken,
		Platform:  PlatformAndroid,
		IsActive:  false,
	}

	err := client.SaveToken(ctx, store, SaveTokenParams{
		UserID:   "user-1",
		RawToken: rawToken,
		Platform: PlatformAndroid,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	token := store.tokens[hash]
	if !token.IsActive {
		t.Error("token should be reactivated")
	}
}

func TestSaveToken_InvalidFormat(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := newTestCtx()

	// Too short
	err := client.SaveToken(ctx, store, SaveTokenParams{
		UserID:   "user-1",
		RawToken: "short",
		Platform: PlatformIOS,
	})

	e, ok := IsGoNotificationError(err)
	if !ok || e.Key != KeyNotificationInvalidToken {
		t.Errorf("expected KeyNotificationInvalidToken, got %v", err)
	}
}

func TestSaveToken_InvalidPlatform(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := newTestCtx()

	err := client.SaveToken(ctx, store, SaveTokenParams{
		UserID:   "user-1",
		RawToken: fakeToken("valid-token"),
		Platform: Platform("web"),
	})

	e, ok := IsGoNotificationError(err)
	if !ok || e.Key != KeyNotificationInvalidPlatform {
		t.Errorf("expected KeyNotificationInvalidPlatform, got %v", err)
	}
}

func TestRemoveToken(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := newTestCtx()

	rawToken := fakeToken("to-remove")
	hash := HashToken(rawToken)
	store.tokens[hash] = &FCMToken{
		ID:       "token-3",
		IsActive: true,
	}

	err := client.RemoveToken(ctx, store, "user-1", rawToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.tokens[hash].IsActive {
		t.Error("token should be deactivated")
	}
}

func TestRemoveAllTokens(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := newTestCtx()

	store.tokens["hash1"] = &FCMToken{AppUserID: "user-1", IsActive: true}
	store.tokens["hash2"] = &FCMToken{AppUserID: "user-1", IsActive: true}
	store.tokens["hash3"] = &FCMToken{AppUserID: "user-2", IsActive: true}

	err := client.RemoveAllTokens(ctx, store, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// user-1 tokens deleted
	for _, token := range store.tokens {
		if token.AppUserID == "user-1" {
			t.Error("user-1 tokens should be deleted")
		}
	}

	// user-2 token still exists
	if store.tokens["hash3"] == nil {
		t.Error("user-2 token should still exist")
	}
}

func TestCleanupExpiredTokens(t *testing.T) {
	client := &Client{enabled: false}
	store := newMockTokenStore()
	ctx := newTestCtx()

	// Old token (100 days ago)
	store.tokens["old"] = &FCMToken{
		ID:         "old-token",
		AppUserID:  "user-1",
		IsActive:   true,
		LastUsedAt: time.Now().UTC().Add(-100 * 24 * time.Hour),
	}

	// Recent token (1 day ago)
	store.tokens["recent"] = &FCMToken{
		ID:         "recent-token",
		AppUserID:  "user-2",
		IsActive:   true,
		LastUsedAt: time.Now().UTC().Add(-24 * time.Hour),
	}

	result, err := client.CleanupExpiredTokens(ctx, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TokensDeactivated != 1 {
		t.Errorf("expected 1 deactivated, got %d", result.TokensDeactivated)
	}

	if store.tokens["old"].IsActive {
		t.Error("old token should be deactivated")
	}

	if !store.tokens["recent"].IsActive {
		t.Error("recent token should still be active")
	}
}
