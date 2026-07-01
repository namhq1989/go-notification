package notification

import "strings"

// Error is the standard error returned by go-notification methods.
// Key is an i18n message ID — the caller maps it to a localized message + error code.
type Error struct {
	Key string
}

func (e *Error) Error() string { return e.Key }

func newError(key string) *Error {
	return &Error{Key: key}
}

// IsGoNotificationError checks if an error is a go-notification Error and extracts the key.
func IsGoNotificationError(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	if e, ok := err.(*Error); ok {
		return e, true
	}
	return nil, false
}

// i18n keys — caller must register translations for these keys.
const (
	KeyNotificationInvalidToken   = "NotificationInvalidToken"
	KeyNotificationInvalidPayload = "NotificationInvalidPayload"
	KeyNotificationSendFailed     = "NotificationSendFailed"
	KeyNotificationMissingConfig  = "NotificationMissingConfig"
	KeyNotificationTopicEmpty     = "NotificationTopicEmpty"
	KeyNotificationInvalidPlatform = "NotificationInvalidPlatform"
)

// FCMErrorType represents the type of FCM error
type FCMErrorType string

const (
	FCMErrorTypeUnknown         FCMErrorType = "unknown"
	FCMErrorTypeInvalidToken    FCMErrorType = "invalid_token"
	FCMErrorTypeQuotaExceeded   FCMErrorType = "quota_exceeded"
	FCMErrorTypeServerError     FCMErrorType = "server_error"
	FCMErrorTypeInvalidPayload  FCMErrorType = "invalid_payload"
	FCMErrorTypeAuthentication  FCMErrorType = "authentication"
	FCMErrorTypeSenderMismatch  FCMErrorType = "sender_mismatch"
	FCMErrorTypeMessageTooLarge FCMErrorType = "message_too_large"
	FCMErrorTypeDeviceBlocked   FCMErrorType = "device_blocked"
)

// ClassifyFCMError classifies an FCM error into a specific type
func ClassifyFCMError(err error) FCMErrorType {
	if err == nil {
		return FCMErrorTypeUnknown
	}

	errStr := err.Error()

	if IsInvalidTokenError(err) {
		return FCMErrorTypeInvalidToken
	}

	if strings.Contains(errStr, "MismatchSenderId") {
		return FCMErrorTypeSenderMismatch
	}

	if strings.Contains(errStr, "quota-exceeded") || strings.Contains(errStr, "QuotaExceeded") {
		return FCMErrorTypeQuotaExceeded
	}

	if strings.Contains(errStr, "internal-error") || strings.Contains(errStr, "unavailable") {
		return FCMErrorTypeServerError
	}

	if strings.Contains(errStr, "invalid-argument") || strings.Contains(errStr, "InvalidParameters") {
		return FCMErrorTypeInvalidPayload
	}

	if strings.Contains(errStr, "authentication-error") || strings.Contains(errStr, "InvalidServerKey") {
		return FCMErrorTypeAuthentication
	}

	if strings.Contains(errStr, "message-too-large") || strings.Contains(errStr, "MessageTooBig") {
		return FCMErrorTypeMessageTooLarge
	}

	if strings.Contains(errStr, "device-message-rate-exceeded") {
		return FCMErrorTypeDeviceBlocked
	}

	return FCMErrorTypeUnknown
}

// ShouldRetry returns whether the error type should be retried
func (t FCMErrorType) ShouldRetry() bool {
	switch t {
	case FCMErrorTypeServerError, FCMErrorTypeQuotaExceeded, FCMErrorTypeDeviceBlocked:
		return true
	default:
		return false
	}
}

// ShouldDeactivateToken returns whether the token should be deactivated for this error
func (t FCMErrorType) ShouldDeactivateToken() bool {
	switch t {
	case FCMErrorTypeInvalidToken, FCMErrorTypeSenderMismatch:
		return true
	default:
		return false
	}
}

// IsInvalidTokenError checks if an error indicates an invalid FCM token
func IsInvalidTokenError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	invalidTokenErrors := []string{
		"registration-token-not-registered",
		"invalid-registration-token",
		"not-registered",
		"InvalidRegistration",
		"NotRegistered",
		"InvalidApnsCredential",
	}

	for _, invalidErr := range invalidTokenErrors {
		if strings.Contains(errStr, invalidErr) {
			return true
		}
	}

	return false
}
