package notification

import "time"

// FCMToken represents a Firebase Cloud Messaging token entity
type FCMToken struct {
	ID         string
	AppUserID  string
	TokenHash  string
	RawToken   string
	Platform   Platform
	IsActive   bool
	LastUsedAt time.Time
	UpdatedAt  time.Time
	CreatedAt  time.Time
}
