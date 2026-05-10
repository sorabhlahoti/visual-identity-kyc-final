package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

func SignJWT(secret, subject string, ttl time.Duration) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now()
	claims := Claims{Sub: subject, Iat: now.Unix(), Exp: now.Add(ttl).Unix()}

	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	unsigned := b64(hb) + "." + b64(cb)
	sig := sign(secret, unsigned)
	return unsigned + "." + sig, nil
}

func VerifyJWT(secret, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	unsigned := parts[0] + "." + parts[1]
	expected := sign(secret, unsigned)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}
	if claims.Exp < time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

func b64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func sign(secret, unsigned string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
