package notification

import (
	"strings"
	"time"
)

// Platform represents the notification platform
type Platform string

const (
	PlatformUnknown Platform = ""
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
)

// String returns the string representation
func (p Platform) String() string {
	return string(p)
}

// IsValid checks if the platform is valid
func (p Platform) IsValid() bool {
	return p == PlatformIOS || p == PlatformAndroid
}

// ToPlatform converts a string to Platform
func ToPlatform(s string) Platform {
	switch strings.ToLower(s) {
	case "ios":
		return PlatformIOS
	case "android":
		return PlatformAndroid
	default:
		return PlatformUnknown
	}
}

// Priority represents FCM message priority
type Priority string

const (
	PriorityUnknown Priority = ""
	PriorityNormal  Priority = "normal"
	PriorityHigh    Priority = "high"
)

// String returns the string representation
func (p Priority) String() string {
	return string(p)
}

// IsValid checks if the priority is valid
func (p Priority) IsValid() bool {
	return p == PriorityNormal || p == PriorityHigh
}

// ToPriority converts a string to Priority (case-insensitive)
func ToPriority(s string) Priority {
	switch strings.ToLower(s) {
	case "high":
		return PriorityHigh
	case "normal":
		return PriorityNormal
	default:
		return PriorityUnknown
	}
}

// Message represents a push notification message
type Message struct {
	Title    string
	Body     string
	Data     map[string]string
	Priority Priority      // high | normal (default: normal)
	TTL      time.Duration // 0 = FCM default (4 weeks)
}

// SaveTokenParams holds parameters for saving an FCM token
type SaveTokenParams struct {
	UserID   string
	RawToken string
	Platform Platform
}

// SendResult contains the result of a send operation
type SendResult struct {
	Sent    int
	Failed  []string // userIDs that failed ALL tokens
	Partial error    // non-nil if some (not all) failed
}

// CleanupResult contains the result of a cleanup operation
type CleanupResult struct {
	TokensDeactivated int
	Duration          time.Duration
}

// Config holds FCM configuration
type Config struct {
	ProjectID         string
	CredentialsFile   string // path to service account JSON
	CredentialsBase64 string // base64 wins if both set; file ignored with warning log
	Enabled           bool   // false = dev mode, log only
}
