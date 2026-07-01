package notification

import (
	"errors"
	"testing"
)

func TestClassifyFCMError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected FCMErrorType
	}{
		{name: "nil error", err: nil, expected: FCMErrorTypeUnknown},
		{name: "invalid token - not registered", err: errors.New("registration-token-not-registered"), expected: FCMErrorTypeInvalidToken},
		{name: "invalid token - InvalidRegistration", err: errors.New("InvalidRegistration"), expected: FCMErrorTypeInvalidToken},
		{name: "invalid token - NotRegistered", err: errors.New("NotRegistered"), expected: FCMErrorTypeInvalidToken},
		{name: "sender mismatch", err: errors.New("MismatchSenderId"), expected: FCMErrorTypeSenderMismatch},
		{name: "quota exceeded", err: errors.New("quota-exceeded"), expected: FCMErrorTypeQuotaExceeded},
		{name: "server error - internal", err: errors.New("internal-error"), expected: FCMErrorTypeServerError},
		{name: "server error - unavailable", err: errors.New("unavailable"), expected: FCMErrorTypeServerError},
		{name: "invalid payload", err: errors.New("invalid-argument"), expected: FCMErrorTypeInvalidPayload},
		{name: "authentication error", err: errors.New("authentication-error"), expected: FCMErrorTypeAuthentication},
		{name: "message too large", err: errors.New("message-too-large"), expected: FCMErrorTypeMessageTooLarge},
		{name: "device blocked", err: errors.New("device-message-rate-exceeded"), expected: FCMErrorTypeDeviceBlocked},
		{name: "unknown error", err: errors.New("some random error"), expected: FCMErrorTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyFCMError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFCMErrorType_ShouldRetry(t *testing.T) {
	tests := []struct {
		errType  FCMErrorType
		expected bool
	}{
		{FCMErrorTypeServerError, true},
		{FCMErrorTypeQuotaExceeded, true},
		{FCMErrorTypeDeviceBlocked, true},
		{FCMErrorTypeInvalidToken, false},
		{FCMErrorTypeSenderMismatch, false},
		{FCMErrorTypeInvalidPayload, false},
		{FCMErrorTypeAuthentication, false},
		{FCMErrorTypeMessageTooLarge, false},
		{FCMErrorTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.errType), func(t *testing.T) {
			if tt.errType.ShouldRetry() != tt.expected {
				t.Errorf("ShouldRetry() for %s: expected %v", tt.errType, tt.expected)
			}
		})
	}
}

func TestFCMErrorType_ShouldDeactivateToken(t *testing.T) {
	tests := []struct {
		errType  FCMErrorType
		expected bool
	}{
		{FCMErrorTypeInvalidToken, true},
		{FCMErrorTypeSenderMismatch, true},
		{FCMErrorTypeServerError, false},
		{FCMErrorTypeQuotaExceeded, false},
		{FCMErrorTypeDeviceBlocked, false},
		{FCMErrorTypeInvalidPayload, false},
		{FCMErrorTypeAuthentication, false},
		{FCMErrorTypeMessageTooLarge, false},
		{FCMErrorTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.errType), func(t *testing.T) {
			if tt.errType.ShouldDeactivateToken() != tt.expected {
				t.Errorf("ShouldDeactivateToken() for %s: expected %v", tt.errType, tt.expected)
			}
		})
	}
}

func TestIsInvalidTokenError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "nil", err: nil, expected: false},
		{name: "not-registered", err: errors.New("not-registered"), expected: true},
		{name: "InvalidRegistration", err: errors.New("InvalidRegistration"), expected: true},
		{name: "random error", err: errors.New("timeout"), expected: false},
		{name: "wrapped message", err: errors.New("fcm error: registration-token-not-registered for token xyz"), expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsInvalidTokenError(tt.err) != tt.expected {
				t.Errorf("expected %v for error %v", tt.expected, tt.err)
			}
		})
	}
}
