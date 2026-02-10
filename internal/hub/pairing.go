package hub

import (
	"crypto/rand"
	"fmt"
	"time"
)

const tokenPrefix = "oc_pair_"

// GenerateToken creates a new pairing token.
func GenerateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based token if crypto/rand fails
		return fmt.Sprintf("%s%x", tokenPrefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s%x", tokenPrefix, b)
}
