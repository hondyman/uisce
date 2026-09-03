package semantic_bridge

import (
	"fmt"

	"github.com/hondyman/uisce/backend/internal/security"
)

// CredentialVaultKeyEnv / CredentialVaultDevFallbackEnv name the env vars
// used to seal target credentials (Snowflake/Databricks tokens) at rest.
// Reuses the same AES-256-GCM key material shape as the API dispatcher's
// per-tenant credential vault (see internal/api/helpers.go) rather than
// inventing a second secret to manage.
const (
	CredentialVaultKeyEnv         = "API_TOKEN_ENCRYPTION_KEY"
	CredentialVaultDevFallbackEnv = "API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK"
)

// CredentialVault encrypts/decrypts AI bridge target credentials
// (warehouse tokens, PATs) before they are persisted to
// catalog_ai.ai_bridge_targets.credentials_vaulted.
type CredentialVault struct {
	encryptor *security.TokenEncryptor
	hmacKey   []byte
}

// NewCredentialVault loads the shared server key and fails closed if it is
// missing and no dev fallback is enabled — we never want to silently store
// plaintext credentials.
func NewCredentialVault() (*CredentialVault, error) {
	key, err := security.LoadKeyFromEnv(CredentialVaultKeyEnv, CredentialVaultDevFallbackEnv)
	if err != nil {
		return nil, err
	}
	enc, err := security.NewTokenEncryptor(key)
	if err != nil {
		return nil, err
	}
	return &CredentialVault{encryptor: enc, hmacKey: key}, nil
}

// Seal encrypts every value in a plaintext credential map (e.g.
// {"token": "dapi...", "warehouse_id": "abc"}) so it is safe to store in the
// credentials_vaulted JSONB column.
func (v *CredentialVault) Seal(plain map[string]string) (map[string]interface{}, error) {
	sealed := make(map[string]interface{}, len(plain))
	for k, val := range plain {
		if val == "" {
			continue
		}
		ct, err := v.encryptor.Encrypt(val)
		if err != nil {
			return nil, fmt.Errorf("sealing credential %q: %w", k, err)
		}
		sealed[k] = ct
	}
	return sealed, nil
}

// Open decrypts a credentials_vaulted map back into plaintext for use in an
// outbound API call. Values that fail to decrypt (e.g. legacy plaintext rows
// written before this vault existed) are skipped rather than returned as-is,
// so a caller never accidentally sends garbage ciphertext as a bearer token.
func (v *CredentialVault) Open(sealed map[string]interface{}) map[string]string {
	plain := make(map[string]string, len(sealed))
	for k, raw := range sealed {
		ct, ok := raw.(string)
		if !ok {
			continue
		}
		pt, err := v.encryptor.Decrypt(ct)
		if err != nil {
			continue
		}
		plain[k] = pt
	}
	return plain
}
