package api

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

// TestDecodeEncryptionKeyRaw32 verifies the helper accepts a 32-byte ASCII key
// verbatim (rare in practice but supported).
func TestDecodeEncryptionKeyRaw32(t *testing.T) {
	key := bytes.Repeat([]byte("a"), 32)
	got, err := decodeEncryptionKey(string(key))
	if err != nil {
		t.Fatalf("decodeEncryptionKey raw32: %v", err)
	}
	if !bytes.Equal(got, key) {
		t.Fatalf("decoded key mismatch: got %x want %x", got, key)
	}
}

// TestDecodeEncryptionKeyHex64 verifies a 64-char hex string decodes to 32 bytes.
func TestDecodeEncryptionKeyHex64(t *testing.T) {
	raw := bytes.Repeat([]byte{0xAB}, 32)
	hexStr := hex.EncodeToString(raw)
	got, err := decodeEncryptionKey(hexStr)
	if err != nil {
		t.Fatalf("decodeEncryptionKey hex: %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("decoded key mismatch: got %x want %x", got, raw)
	}
}

// TestDecodeEncryptionKeyBase64 verifies a base64 (standard) encoding of 32 bytes
// decodes correctly.
func TestDecodeEncryptionKeyBase64(t *testing.T) {
	raw := bytes.Repeat([]byte{0xCD}, 32)
	enc := base64.StdEncoding.EncodeToString(raw)
	got, err := decodeEncryptionKey(enc)
	if err != nil {
		t.Fatalf("decodeEncryptionKey base64: %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("decoded key mismatch: got %x want %x", got, raw)
	}
}

// TestDecodeEncryptionKeyRawURL verifies the URL-safe raw base64 encoding (the
// one security.TokenEncryptor emits) decodes correctly. This is the form
// produced by `openssl rand -base64 32` when wrapped with `-A`.
func TestDecodeEncryptionKeyRawURL(t *testing.T) {
	raw := bytes.Repeat([]byte{0xEF}, 32)
	enc := base64.RawURLEncoding.EncodeToString(raw)
	got, err := decodeEncryptionKey(enc)
	if err != nil {
		t.Fatalf("decodeEncryptionKey rawURL: %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("decoded key mismatch: got %x want %x", got, raw)
	}
}

// TestDecodeEncryptionKeyRejectsInvalid verifies that strings that cannot
// represent 32 bytes are rejected.
func TestDecodeEncryptionKeyRejectsInvalid(t *testing.T) {
	cases := []string{
		"",
		"too short",
		strings.Repeat("x", 31),
		strings.Repeat("x", 33),
		"!!!not-valid-base64-or-hex!!!",
	}
	for _, s := range cases {
		if _, err := decodeEncryptionKey(s); err == nil {
			t.Fatalf("expected error for %q, got nil", s)
		}
	}
}
