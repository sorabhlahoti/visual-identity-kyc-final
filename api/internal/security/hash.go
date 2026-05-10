package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// PepperHash is deterministic so re-KYC can perform exact matching, but non-reversible
// without the server-held pepper. Rotate pepper through a managed secret in production.
func PepperHash(pepper string, parts ...string) string {
	h := hmac.New(sha256.New, []byte(pepper))
	for i, p := range parts {
		if i > 0 {
			h.Write([]byte("|"))
		}
		h.Write([]byte(normalize(p)))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func DemographicHash(pepper, dob, gender string) string {
	return PepperHash(pepper, dob, gender)
}

func NameHash(pepper, name string) string {
	return PepperHash(pepper, name)
}
