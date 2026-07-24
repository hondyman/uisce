// Package secrets provides a configurable secrets management interface
// that supports HashiCorp Vault (dev), AWS Secrets Manager, and Azure Key Vault (prod).
package secrets

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrSecretNotFound is returned when a secret does not exist
var ErrSecretNotFound = errors.New("secret not found")

// Provider defines the interface for secrets management backends
type Provider interface {
	// Get retrieves a secret value by key
	Get(ctx context.Context, key string) (string, error)

	// GetMap retrieves a secret as key-value pairs (for complex secrets)
	GetMap(ctx context.Context, key string) (map[string]string, error)

	// Put stores a secret value
	Put(ctx context.Context, key string, value string) error

	// PutMap stores a secret as key-value pairs
	PutMap(ctx context.Context, key string, values map[string]string) error

	// Delete removes a secret
	Delete(ctx context.Context, key string) error

	// List returns all secret keys under a prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Rotate generates a new version of a secret (for dynamic secrets)
	Rotate(ctx context.Context, key string) error

	// Health checks if the provider is healthy
	Health(ctx context.Context) error

	// Close releases any resources
	Close() error
}

// Config defines the configuration for secrets providers
type Config struct {
	// Type is the provider type: "vault", "aws", "azure", "memory" (for testing)
	Type string `yaml:"type" json:"type"`

	// Vault configuration
	VaultAddr  string `yaml:"vault_addr" json:"vault_addr"`
	VaultToken string `yaml:"vault_token" json:"vault_token"`
	VaultMount string `yaml:"vault_mount" json:"vault_mount"` // Default: "secret"

	// AWS Secrets Manager configuration
	AWSRegion string `yaml:"aws_region" json:"aws_region"`

	// Azure Key Vault configuration
	AzureVaultURL string `yaml:"azure_vault_url" json:"azure_vault_url"`
	AzureTenantID string `yaml:"azure_tenant_id" json:"azure_tenant_id"`
	AzureClientID string `yaml:"azure_client_id" json:"azure_client_id"`
}

// NewProvider creates a new secrets provider based on configuration
// This is the main factory function - switch providers via config, not code
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Type {
	case "vault":
		return NewVaultProvider(cfg)
	case "aws":
		return NewAWSProvider(cfg)
	case "azure":
		return NewAzureProvider(cfg)
	case "memory":
		return NewMemoryProvider(), nil
	case "":
		return nil, errors.New("secrets provider type is required")
	default:
		return nil, fmt.Errorf("unsupported secrets provider type: %s", cfg.Type)
	}
}

// MemoryProvider is an in-memory provider for testing
type MemoryProvider struct {
	mu      sync.RWMutex
	secrets map[string]map[string]string
}

// NewMemoryProvider creates a new in-memory provider for testing
func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		secrets: make(map[string]map[string]string),
	}
}

func (m *MemoryProvider) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if vals, ok := m.secrets[key]; ok {
		if v, ok := vals["value"]; ok {
			return v, nil
		}
	}
	return "", ErrSecretNotFound
}

func (m *MemoryProvider) GetMap(ctx context.Context, key string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if vals, ok := m.secrets[key]; ok {
		result := make(map[string]string)
		for k, v := range vals {
			result[k] = v
		}
		return result, nil
	}
	return nil, ErrSecretNotFound
}

func (m *MemoryProvider) Put(ctx context.Context, key string, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secrets[key] = map[string]string{"value": value}
	return nil
}

func (m *MemoryProvider) PutMap(ctx context.Context, key string, values map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secrets[key] = values
	return nil
}

func (m *MemoryProvider) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.secrets, key)
	return nil
}

func (m *MemoryProvider) List(ctx context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var keys []string
	for k := range m.secrets {
		if len(prefix) == 0 || len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (m *MemoryProvider) Rotate(ctx context.Context, key string) error {
	// No-op for memory provider
	return nil
}

func (m *MemoryProvider) Health(ctx context.Context) error {
	return nil
}

func (m *MemoryProvider) Close() error {
	return nil
}
