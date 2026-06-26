package secrets

import (
	"context"
	"fmt"
)

// AzureProvider implements Provider using Azure Key Vault
// This is a stub - implement with azidentity + azsecrets for production
type AzureProvider struct {
	vaultURL string
}

// NewAzureProvider creates a new Azure Key Vault provider
func NewAzureProvider(cfg Config) (*AzureProvider, error) {
	if cfg.AzureVaultURL == "" {
		return nil, fmt.Errorf("azure_vault_url is required for Azure provider")
	}
	return &AzureProvider{
		vaultURL: cfg.AzureVaultURL,
	}, nil
}

func (a *AzureProvider) Get(ctx context.Context, key string) (string, error) {
	// TODO: Implement with Azure SDK
	// cred, _ := azidentity.NewDefaultAzureCredential(nil)
	// client, _ := azsecrets.NewClient(a.vaultURL, cred, nil)
	// resp, _ := client.GetSecret(ctx, key, "", nil)
	return "", fmt.Errorf("Azure provider not yet implemented - add Azure SDK dependency")
}

func (a *AzureProvider) GetMap(ctx context.Context, key string) (map[string]string, error) {
	return nil, fmt.Errorf("Azure provider not yet implemented")
}

func (a *AzureProvider) Put(ctx context.Context, key string, value string) error {
	return fmt.Errorf("Azure provider not yet implemented")
}

func (a *AzureProvider) PutMap(ctx context.Context, key string, values map[string]string) error {
	return fmt.Errorf("Azure provider not yet implemented")
}

func (a *AzureProvider) Delete(ctx context.Context, key string) error {
	return fmt.Errorf("Azure provider not yet implemented")
}

func (a *AzureProvider) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, fmt.Errorf("Azure provider not yet implemented")
}

func (a *AzureProvider) Rotate(ctx context.Context, key string) error {
	return fmt.Errorf("Azure provider not yet implemented")
}

func (a *AzureProvider) Health(ctx context.Context) error {
	return fmt.Errorf("Azure provider not yet implemented")
}

func (a *AzureProvider) Close() error {
	return nil
}
