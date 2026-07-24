package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
)

// VaultProvider implements Provider using HashiCorp Vault
type VaultProvider struct {
	client *api.Client
	mount  string
}

// NewVaultProvider creates a new HashiCorp Vault provider
func NewVaultProvider(cfg Config) (*VaultProvider, error) {
	config := api.DefaultConfig()
	if cfg.VaultAddr != "" {
		config.Address = cfg.VaultAddr
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	if cfg.VaultToken != "" {
		client.SetToken(cfg.VaultToken)
	}

	mount := cfg.VaultMount
	if mount == "" {
		mount = "secret" // Default KV v2 mount
	}

	return &VaultProvider{
		client: client,
		mount:  mount,
	}, nil
}

func (v *VaultProvider) Get(ctx context.Context, key string) (string, error) {
	vals, err := v.GetMap(ctx, key)
	if err != nil {
		return "", err
	}
	if val, ok := vals["value"]; ok {
		return val, nil
	}
	// Return first value if no "value" key
	for _, val := range vals {
		return val, nil
	}
	return "", ErrSecretNotFound
}

func (v *VaultProvider) GetMap(ctx context.Context, key string) (map[string]string, error) {
	path := fmt.Sprintf("%s/data/%s", v.mount, key)
	secret, err := v.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, ErrSecretNotFound
	}

	// KV v2 stores data under "data" key
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid secret format")
	}

	result := make(map[string]string)
	for k, v := range data {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result, nil
}

func (v *VaultProvider) Put(ctx context.Context, key string, value string) error {
	return v.PutMap(ctx, key, map[string]string{"value": value})
}

func (v *VaultProvider) PutMap(ctx context.Context, key string, values map[string]string) error {
	path := fmt.Sprintf("%s/data/%s", v.mount, key)
	data := make(map[string]interface{})
	for k, v := range values {
		data[k] = v
	}

	_, err := v.client.Logical().WriteWithContext(ctx, path, map[string]interface{}{
		"data": data,
	})
	if err != nil {
		return fmt.Errorf("failed to write secret: %w", err)
	}
	return nil
}

func (v *VaultProvider) Delete(ctx context.Context, key string) error {
	path := fmt.Sprintf("%s/metadata/%s", v.mount, key)
	_, err := v.client.Logical().DeleteWithContext(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

func (v *VaultProvider) List(ctx context.Context, prefix string) ([]string, error) {
	path := fmt.Sprintf("%s/metadata/%s", v.mount, prefix)
	secret, err := v.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	result := make([]string, 0, len(keys))
	for _, k := range keys {
		if s, ok := k.(string); ok {
			result = append(result, strings.TrimSuffix(s, "/"))
		}
	}
	return result, nil
}

func (v *VaultProvider) Rotate(ctx context.Context, key string) error {
	// For KV v2, rotation means creating a new version
	// Get current value, modify timestamp, and put back
	vals, err := v.GetMap(ctx, key)
	if err != nil {
		return err
	}
	vals["rotated_at"] = fmt.Sprintf("%d", ctx.Value("timestamp"))
	return v.PutMap(ctx, key, vals)
}

func (v *VaultProvider) Health(ctx context.Context) error {
	health, err := v.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}
	if !health.Initialized || health.Sealed {
		return fmt.Errorf("vault is not ready: initialized=%v, sealed=%v", health.Initialized, health.Sealed)
	}
	return nil
}

func (v *VaultProvider) Close() error {
	// Vault client doesn't require explicit close
	return nil
}
