# go-notification

Internal reusable push notification toolkit for Go services. Wraps Firebase Cloud Messaging with token lifecycle management, batch sending, and automatic error classification.

## Install

```bash
go get github.com/namhq1989/go-notification
```

## Architecture

```
types.go         — Platform, Priority, Message, Config, SendResult, CleanupResult
entity.go        — FCMToken struct
interface.go     — Notification interface + TokenStore interface
notification.go  — Client constructor, hasCredentials
token.go         — SaveToken, RemoveToken, RemoveAllTokens, CleanupExpiredTokens
send.go          — SendToUser, SendToUsers, SendToTopic
firebase.go      — FirebaseClient (FCM wrapper, batch + single send)
hash.go          — HashToken (SHA-256)
errors.go        — Error type, ClassifyFCMError, FCMErrorType helpers
```

## Quick Start

```go
import notification "github.com/namhq1989/go-notification"

client, err := notification.New(notification.Config{
    ProjectID:         "my-gcp-project",
    CredentialsBase64: os.Getenv("FCM_CREDENTIALS_BASE64"), // or CredentialsFile
    Enabled:           true,
})
if err != nil {
    log.Fatal(err)
}

// Send to a single user (all active tokens)
result, err := client.SendToUser(ctx, store, "user-123", notification.Message{
    Title:    "New lesson available",
    Body:     "Your daily practice is ready",
    Data:     map[string]string{"screen": "practice"},
    Priority: notification.PriorityHigh,
})
// result.Sent = number of successful deliveries
// result.Failed = []string of userIDs with zero successful tokens
```

## Key Types

```go
type Platform string // PlatformIOS ("ios") | PlatformAndroid ("android")

type Priority string // PriorityNormal ("normal") | PriorityHigh ("high")

type Message struct {
    Title    string
    Body     string
    Data     map[string]string
    Priority Priority      // default: normal
    TTL      time.Duration // 0 = FCM default (4 weeks)
}

type FCMToken struct {
    ID         string
    AppUserID  string
    TokenHash  string   // SHA-256 hex
    RawToken   string   // original FCM token
    Platform   Platform
    IsActive   bool
    LastUsedAt time.Time
    UpdatedAt  time.Time
    CreatedAt  time.Time
}

type SendResult struct {
    Sent    int
    Failed  []string // userIDs that failed ALL tokens
    Partial error    // non-nil if some (not all) users failed
}

type Config struct {
    ProjectID         string
    CredentialsFile   string // path to service account JSON
    CredentialsBase64 string // base64 wins if both set
    Enabled           bool   // false = dev mode (log only)
}
```

## TokenStore Interface

Each app implements storage with its own DB layer:

```go
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
```

## Error Handling

### ClassifyFCMError

Classifies raw FCM errors into actionable types:

```go
errType := notification.ClassifyFCMError(fcmErr)

if errType.ShouldDeactivateToken() {
    // invalid_token, sender_mismatch → remove token from store
}

if errType.ShouldRetry() {
    // server_error, quota_exceeded, device_blocked → retry with backoff
}
```

### FCMErrorType values

| Type | ShouldRetry | ShouldDeactivateToken |
|------|:-----------:|:---------------------:|
| `invalid_token` | no | yes |
| `sender_mismatch` | no | yes |
| `quota_exceeded` | yes | no |
| `server_error` | yes | no |
| `device_blocked` | yes | no |
| `invalid_payload` | no | no |
| `authentication` | no | no |
| `message_too_large` | no | no |
| `unknown` | no | no |

### Package errors (i18n keys)

All errors are returned as `*notification.Error{Key: "..."}`. The key is an i18n message ID — caller maps it to a localized message + error code.

```go
notifErr, ok := notification.IsGoNotificationError(err)
if ok {
    // notifErr.Key is one of the keys below
}
```

| Key | When |
|-----|------|
| `NotificationInvalidToken` | Token empty or outside 100-300 char range |
| `NotificationInvalidPlatform` | Platform is not "ios" or "android" |
| `NotificationInvalidPayload` | Message validation failed |
| `NotificationSendFailed` | FCM send or store operation failed |
| `NotificationMissingConfig` | ProjectID or credentials not provided |
| `NotificationTopicEmpty` | SendToTopic called with empty topic |

## Token Management Pattern

**Hash + raw**: tokens are stored with both `TokenHash` (SHA-256, used for lookups/dedup) and `RawToken` (sent to FCM).

**One active per user/platform**: `SaveToken` enforces at most one active token per `(user_id, platform)` pair. Old tokens are deactivated before creating a new one.

Required DB constraint:
```sql
CREATE UNIQUE INDEX idx_fcm_tokens_active_user_platform
ON fcm_tokens (app_user_id, platform)
WHERE is_active = true;
```

**Idempotent SaveToken**:
- Existing active token with same hash → update `last_used_at`
- Existing inactive token with same hash → reactivate
- New token → deactivate old tokens for same user+platform → create

**Race safety**: concurrent `SaveToken` with the same `rawToken` is handled via duplicate key detection. Concurrent calls with different tokens for the same user+platform rely on the partial unique index above.

## Dev Mode

```go
client, _ := notification.New(notification.Config{
    Enabled: false,
})
```

When `Enabled: false`, no Firebase client is created. All send methods log the notification details and return success without making FCM calls. Useful for local development and testing.
