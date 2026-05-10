package did

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func ForIdentity(issuer, identityID string) string {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		issuer = "did:web:localhost"
	}
	sum := sha256.Sum256([]byte(identityID))
	return strings.TrimRight(issuer, ":") + ":kyc:" + hex.EncodeToString(sum[:8])
}
