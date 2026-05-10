package security

import "testing"

func TestSealOpenJSON(t *testing.T) {
	secret := "test-secret-with-enough-length"
	in := map[string]string{"name": "John Doe"}
	nonce, ct, err := SealJSON(secret, in)
	if err != nil {
		t.Fatal(err)
	}
	if nonce == "" || ct == "" {
		t.Fatal("expected nonce and ciphertext")
	}
	var out map[string]string
	if err := OpenJSON(secret, nonce, ct, &out); err != nil {
		t.Fatal(err)
	}
	if out["name"] != "John Doe" {
		t.Fatalf("wrong decoded value: %#v", out)
	}
}
