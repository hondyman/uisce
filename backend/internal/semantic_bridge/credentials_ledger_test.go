package semantic_bridge_test

import (
	"crypto/rand"
	"testing"

	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/semantic_bridge"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generating test key: %v", err)
	}
	return key
}

func TestCredentialVault_SealOpenRoundTrip(t *testing.T) {
	key := testKey(t)
	enc, err := security.NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	// Seal/Open go through the vault's own encryptor; build one via the
	// exported constructor path is env-dependent, so exercise the same
	// Encrypt/Decrypt contract the vault relies on directly here and prove
	// round-tripping works and ciphertext isn't the plaintext.
	ct, err := enc.Encrypt("dapi-super-secret-token")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ct == "dapi-super-secret-token" {
		t.Fatalf("ciphertext must not equal plaintext")
	}
	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if pt != "dapi-super-secret-token" {
		t.Fatalf("expected round-tripped plaintext, got %q", pt)
	}
}

func TestCredentialVault_OpenSkipsUndecryptableValues(t *testing.T) {
	t.Setenv(semantic_bridge.CredentialVaultKeyEnv, "")
	t.Setenv(semantic_bridge.CredentialVaultDevFallbackEnv, "true")

	vault, err := semantic_bridge.NewCredentialVault()
	if err != nil {
		t.Fatalf("NewCredentialVault: %v", err)
	}

	sealed, err := vault.Seal(map[string]string{"token": "abc123"})
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if sealed["token"] == "abc123" {
		t.Fatalf("sealed value must not be plaintext")
	}

	// Legacy/garbage value that isn't valid ciphertext must be dropped, not
	// passed through as if it were a real credential.
	sealed["legacy_plaintext"] = "not-encrypted-at-all"

	opened := vault.Open(sealed)
	if opened["token"] != "abc123" {
		t.Fatalf("expected decrypted token 'abc123', got %q", opened["token"])
	}
	if _, ok := opened["legacy_plaintext"]; ok {
		t.Fatalf("undecryptable legacy value should have been dropped, not passed through")
	}
}

func TestLedgerKeyRequired(t *testing.T) {
	t.Setenv(semantic_bridge.CredentialVaultKeyEnv, "")
	t.Setenv(semantic_bridge.CredentialVaultDevFallbackEnv, "false")

	if _, err := semantic_bridge.NewCredentialVault(); err == nil {
		t.Fatalf("expected NewCredentialVault to fail closed with no key and no dev fallback")
	}
}
