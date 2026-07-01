package notification

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken hashes an FCM token using SHA-256, returning 64-char hex string
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
