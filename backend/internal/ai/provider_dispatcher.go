package ai

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// ProviderType represents supported LLM inference engines
type ProviderType string

const (
	ProviderOpenAI      ProviderType = "OPENAI"
	ProviderAzureOpenAI ProviderType = "AZURE_OPENAI"
	ProviderAnthropic   ProviderType = "ANTHROPIC"
	ProviderPrivateVLLM ProviderType = "PRIVATE_VLLM"
)

// TenantAIProviderConfig holds BYOK connection metadata (Rule 1 Alignment)
type TenantAIProviderConfig struct {
	ProviderID           string       `json:"provider_id" db:"provider_id"`
	TenantID             string       `json:"tenant_id" db:"tenant_id"`
	ProviderType         ProviderType `json:"provider_type" db:"provider_type"`
	APIEndpoint          string       `json:"api_endpoint" db:"api_endpoint"`
	EncryptedAPIKey      string       `json:"-" db:"encrypted_api_key"`
	DecryptedAPIKey      string       `json:"api_key,omitempty"`
	ModelDeploymentName  string       `json:"model_deployment_name" db:"model_deployment_name"`
	IsActive             bool         `json:"is_active" db:"is_active"`
	CreatedAt            time.Time    `json:"created_at" db:"created_at"`
}

// ProviderDispatcher dynamically routes AI prompts to platform or BYOK providers
type ProviderDispatcher struct {
	db        *sqlx.DB
	masterKey []byte // AES-256 key for secret vaulting
}

func NewProviderDispatcher(db *sqlx.DB) *ProviderDispatcher {
	// 32-byte secret key for AES-GCM encryption
	key := []byte("uisce_secure_byok_vault_key_32b")
	return &ProviderDispatcher{db: db, masterKey: key}
}

// EncryptSecret encrypts an API key using AES-GCM
func (d *ProviderDispatcher) EncryptSecret(plainText string) (string, error) {
	block, err := aes.NewCipher(d.masterKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return hex.EncodeToString(cipherText), nil
}

// DecryptSecret decrypts a vaulted API key
func (d *ProviderDispatcher) DecryptSecret(cipherTextHex string) (string, error) {
	data, err := hex.DecodeString(cipherTextHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(d.masterKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}

// ResolveProvider fetches active BYOK config or returns platform default (Rule 1 Alignment)
func (d *ProviderDispatcher) ResolveProvider(ctx context.Context, tenantID string) (*TenantAIProviderConfig, error) {
	if d.db != nil {
		query := `SELECT provider_id, tenant_id, provider_type, api_endpoint, encrypted_api_key, model_deployment_name, is_active, created_at FROM tenant_ai_providers WHERE tenant_id = $1 AND is_active = true LIMIT 1`
		var cfg TenantAIProviderConfig
		err := d.db.GetContext(ctx, &cfg, query, tenantID)
		if err == nil {
			decryptedKey, err := d.DecryptSecret(cfg.EncryptedAPIKey)
			if err == nil {
				cfg.DecryptedAPIKey = decryptedKey
				log.Printf("[BYOK Gateway] Dispatching AI execution to tenant custom provider (%s: %s)", cfg.ProviderType, cfg.APIEndpoint)
				return &cfg, nil
			}
		}
	}

	// Platform default fallback
	return &TenantAIProviderConfig{
		TenantID:            tenantID,
		ProviderType:        ProviderOpenAI,
		APIEndpoint:         "https://api.openai.com/v1",
		DecryptedAPIKey:     "platform_default_key",
		ModelDeploymentName: "gpt-4o",
		IsActive:            true,
	}, nil
}

// SaveProviderConfig vaults & persists tenant BYOK configuration
func (d *ProviderDispatcher) SaveProviderConfig(ctx context.Context, cfg TenantAIProviderConfig) error {
	encryptedKey, err := d.EncryptSecret(cfg.DecryptedAPIKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	if d.db != nil {
		query := `
			INSERT INTO tenant_ai_providers (
				tenant_id, provider_type, api_endpoint, encrypted_api_key, model_deployment_name, is_active, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (tenant_id, provider_type) DO UPDATE SET
				api_endpoint = EXCLUDED.api_endpoint,
				encrypted_api_key = EXCLUDED.encrypted_api_key,
				model_deployment_name = EXCLUDED.model_deployment_name,
				is_active = EXCLUDED.is_active,
				updated_at = NOW()`

		_, err = d.db.ExecContext(ctx, query, cfg.TenantID, cfg.ProviderType, cfg.APIEndpoint, encryptedKey, cfg.ModelDeploymentName, cfg.IsActive)
		return err
	}
	return nil
}

// HTTP Handlers

func (d *ProviderDispatcher) SaveBYOKHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}

	var cfg TenantAIProviderConfig
	if err := http.MaxBytesReader(w, r.Body, 1048576); err != nil {
		http.Error(w, "Request payload too large", http.StatusRequestEntityTooLarge)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	cfg.TenantID = tenantID
	cfg.ProviderType = ProviderType(r.FormValue("provider_type"))
	cfg.APIEndpoint = r.FormValue("api_endpoint")
	cfg.DecryptedAPIKey = r.FormValue("api_key")
	cfg.ModelDeploymentName = r.FormValue("model_deployment_name")
	cfg.IsActive = r.FormValue("is_active") == "true"

	if err := d.SaveProviderConfig(r.Context(), cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save BYOK provider: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true, "message": "BYOK AI Gateway configuration vaulted & saved"}`))
}
