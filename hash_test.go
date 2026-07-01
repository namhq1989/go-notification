package notification

import (
	"testing"
)

func TestHashToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "consistent hash",
			token:    "test-token-12345",
			expected: HashToken("test-token-12345"),
		},
		{
			name:  "different tokens produce different hashes",
			token: "another-token-67890",
		},
		{
			name:  "empty token",
			token: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashToken(tt.token)

			// SHA-256 hex is always 64 chars
			if len(result) != 64 {
				t.Errorf("expected hash length 64, got %d", len(result))
			}

			// Consistent
			if HashToken(tt.token) != result {
				t.Error("hash is not consistent")
			}

			// If expected is set, verify
			if tt.expected != "" && result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}

	// Different tokens → different hashes
	h1 := HashToken("token-a")
	h2 := HashToken("token-b")
	if h1 == h2 {
		t.Error("different tokens should produce different hashes")
	}
}
