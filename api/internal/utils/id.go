package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewID returns a readable internal ID for non-Qdrant records such as
// transaction IDs. Do not use this as a Qdrant point ID because Qdrant only
// accepts unsigned integers or UUID strings as point IDs.
func NewID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

// NewUUID returns a RFC-4122 UUID v4 string. Qdrant accepts UUID strings as
// point IDs, so use this for identity IDs / vector point IDs.
func NewUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	// UUID version 4 and variant bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
