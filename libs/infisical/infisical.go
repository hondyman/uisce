package infisical

import (
	"context"
	"fmt"

	infisical "github.com/infisical/go-sdk"
)

// Config contains connection parameters for Infisical
type Config struct {
	SiteURL          string `yaml:"site_url" json:"site_url"`                   // e.g., http://100.84.50.65:8085
	ClientID         string `yaml:"client_id" json:"client_id"`                 // Universal Auth Client ID
	ClientSecret     string `yaml:"client_secret" json:"client_secret"`         // Universal Auth Client Secret
	ProjectID        string `yaml:"project_id" json:"project_id"`               // Project ID / Workspace ID
	Environment      string `yaml:"environment" json:"environment"`             // dev, staging, prod
	ServiceToken     string `yaml:"service_token" json:"service_token"`         // Service Token alternative
}

// Client wraps the Infisical Go SDK client
type Client struct {
	client infisical.InfisicalClientInterface
	cfg    Config
}

// NewClient initializes a new Infisical client
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.SiteURL == "" {
		cfg.SiteURL = "http://100.84.50.65:8085"
	}
	if cfg.Environment == "" {
		cfg.Environment = "dev"
	}

	infisicalClient := infisical.NewInfisicalClient(ctx, infisical.Config{
		SiteUrl: cfg.SiteURL,
	})

	if cfg.ClientID != "" && cfg.ClientSecret != "" {
		_, err := infisicalClient.Auth().UniversalAuthLogin(cfg.ClientID, cfg.ClientSecret)
		if err != nil {
			return nil, fmt.Errorf("infisical universal auth login failed: %w", err)
		}
	}

	return &Client{
		client: infisicalClient,
		cfg:    cfg,
	}, nil
}

// GetSecret retrieves a secret value by key and path
func (c *Client) GetSecret(ctx context.Context, secretKey, path string) (string, error) {
	if path == "" {
		path = "/"
	}

	secret, err := c.client.Secrets().Retrieve(infisical.RetrieveSecretOptions{
		SecretKey:   secretKey,
		ProjectID:   c.cfg.ProjectID,
		Environment: c.cfg.Environment,
		SecretPath:  path,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get secret '%s' at path '%s': %w", secretKey, path, err)
	}

	return secret.SecretValue, nil
}

// SetSecret stores a secret value by key and path
func (c *Client) SetSecret(ctx context.Context, secretKey, secretValue, path string) error {
	if path == "" {
		path = "/"
	}

	_, err := c.client.Secrets().Create(infisical.CreateSecretOptions{
		SecretKey:   secretKey,
		SecretValue: secretValue,
		ProjectID:   c.cfg.ProjectID,
		Environment: c.cfg.Environment,
		SecretPath:  path,
	})
	if err != nil {
		// Try update if create fails
		_, errUpdate := c.client.Secrets().Update(infisical.UpdateSecretOptions{
			SecretKey:     secretKey,
			NewSecretValue: secretValue,
			ProjectID:     c.cfg.ProjectID,
			Environment:   c.cfg.Environment,
			SecretPath:    path,
		})
		if errUpdate != nil {
			return fmt.Errorf("failed to create or update secret '%s' at path '%s': %w (create err: %v)", secretKey, path, errUpdate, err)
		}
	}

	return nil
}

// DeleteSecret removes a secret
func (c *Client) DeleteSecret(ctx context.Context, secretKey, path string) error {
	if path == "" {
		path = "/"
	}

	_, err := c.client.Secrets().Delete(infisical.DeleteSecretOptions{
		SecretKey:   secretKey,
		ProjectID:   c.cfg.ProjectID,
		Environment: c.cfg.Environment,
		SecretPath:  path,
	})
	if err != nil {
		return fmt.Errorf("failed to delete secret '%s' at path '%s': %w", secretKey, path, err)
	}

	return nil
}

// ListSecrets retrieves all secrets in a given path
func (c *Client) ListSecrets(ctx context.Context, path string) ([]infisical.Secret, error) {
	if path == "" {
		path = "/"
	}

	secrets, err := c.client.Secrets().List(infisical.ListSecretsOptions{
		ProjectID:   c.cfg.ProjectID,
		Environment: c.cfg.Environment,
		SecretPath:  path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets at path '%s': %w", path, err)
	}

	return secrets, nil
}
