package security

import (
	"testing"
	"time"
)

func TestJWTSignVerify(t *testing.T) {
	token, err := SignJWT("secret", "client", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := VerifyJWT("secret", token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Sub != "client" {
		t.Fatalf("wrong subject: %s", claims.Sub)
	}
	if _, err := VerifyJWT("other", token); err == nil {
		t.Fatalf("expected wrong secret to fail")
	}
}
