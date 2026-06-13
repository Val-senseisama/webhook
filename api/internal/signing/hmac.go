package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Sign produces the X-Webhook-Signature-256 value.
// Format: sha256=HMAC(secret, "<deliveryID>.<timestamp>.<body>")
func Sign(secret, deliveryID, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s.%s.", deliveryID, timestamp)
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify checks a subscriber-provided signature against the expected one.
// Uses constant-time comparison to prevent timing attacks.
func Verify(secret, deliveryID, timestamp, signature string, body []byte) bool {
	expected := Sign(secret, deliveryID, timestamp, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// HashAPIKey returns a hex-encoded SHA-256 hash of a raw API key.
// Store this in the database; never store the raw key.
func HashAPIKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
