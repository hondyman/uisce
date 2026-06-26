package security

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestTokenEncryptorRoundTrip(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 32))
	encryptor, err := NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("unexpected error creating encryptor: %v", err)
	}

	plaintext := "sensitive-token"
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	got, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if got != plaintext {
		t.Fatalf("round-trip mismatch: got %q, want %q", got, plaintext)
	}
}
