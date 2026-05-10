package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
)

func keyFromSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func SealJSON(secret string, value interface{}) (nonceB64 string, ciphertextB64 string, err error) {
	plain, err := json.Marshal(value)
	if err != nil {
		return "", "", err
	}
	block, err := aes.NewCipher(keyFromSecret(secret))
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}
	ciphertext := gcm.Seal(nil, nonce, plain, nil)
	return base64.StdEncoding.EncodeToString(nonce), base64.StdEncoding.EncodeToString(ciphertext), nil
}

func OpenJSON(secret, nonceB64, ciphertextB64 string, out interface{}) error {
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(keyFromSecret(secret))
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, out)
}
