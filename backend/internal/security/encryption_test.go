package security

import (
	"bytes"
	"encoding/base64"
	"testing"
)

// TestTokenEncryptorRoundTrip ensures a plaintext token can be encrypted and
// then decrypted back to its original value, and that the ciphertext differs
// on each call (nonce randomness).
func TestTokenEncryptorRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	enc, err := NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	plaintext := "sk-test-1234567890-abcdef"
	cipher1, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if cipher1 == plaintext {
		t.Fatalf("ciphertext equals plaintext; encryption did nothing")
	}
	if cipher1 == "" {
		t.Fatalf("ciphertext is empty")
	}

	got, err := enc.Decrypt(cipher1)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plaintext)
	}

	// Second encryption must produce a different ciphertext (nonce randomness).
	cipher2, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt (second call): %v", err)
	}
	if cipher1 == cipher2 {
		t.Fatalf("two encryptions of the same plaintext produced identical ciphertext; nonce is not random")
	}
	got2, err := enc.Decrypt(cipher2)
	if err != nil {
		t.Fatalf("Decrypt (second): %v", err)
	}
	if got2 != plaintext {
		t.Fatalf("round-trip mismatch on second call: got %q want %q", got2, plaintext)
	}
}

// TestTokenEncryptorRejectsWrongKeySize ensures NewTokenEncryptor enforces the
// AES-256 key length contract.
func TestTokenEncryptorRejectsWrongKeySize(t *testing.T) {
	cases := [][]byte{
		bytes.Repeat([]byte("a"), 16),
		bytes.Repeat([]byte("a"), 24),
		bytes.Repeat([]byte("a"), 31),
		bytes.Repeat([]byte("a"), 33),
		{},
	}
	for i, key := range cases {
		if _, err := NewTokenEncryptor(key); err == nil {
			t.Fatalf("case %d: expected error for %d-byte key, got nil", i, len(key))
		}
	}
}

// TestTokenEncryptorRejectsTamperedCiphertext ensures the AEAD tag is verified
// on decrypt and any modification fails.
func TestTokenEncryptorRejectsTamperedCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte("z"), 32)
	enc, err := NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	cipher, err := enc.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	raw, err := base64.RawURLEncoding.DecodeString(cipher)
	if err != nil {
		t.Fatalf("decode ciphertext: %v", err)
	}
	if len(raw) < 2 {
		t.Fatalf("ciphertext too short")
	}
	// Flip a byte near the end (in the AEAD tag region).
	raw[len(raw)-1] ^= 0xFF
	tampered := base64.RawURLEncoding.EncodeToString(raw)

	_, decErr := enc.Decrypt(tampered)
	if decErr == nil {
		t.Fatalf("expected decrypt of tampered ciphertext to fail; got nil")
	}
	if decErr.Error() == "" {
		t.Fatalf("decrypt error has empty message")
	}
}

// TestTokenEncryptorEmptyPlaintext verifies that an empty plaintext is still
// round-trippable; this matters for the api_dispatcher case where the
// auth_config JSON may marshal to "{}" or "".
func TestTokenEncryptorEmptyPlaintext(t *testing.T) {
	key := bytes.Repeat([]byte("m"), 32)
	enc, err := NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}
	cipher, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	got, err := enc.Decrypt(cipher)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty plaintext back, got %q", got)
	}
}
