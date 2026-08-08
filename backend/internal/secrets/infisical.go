package secrets

import (
	"context"
	"fmt"
	"strings"

	infisicalsdk "github.com/hondyman/uisce/libs/infisical"
)

// InfisicalProvider implements secrets.Provider using Infisical
type InfisicalProvider struct {
	client *infisicalsdk.Client
}

// NewInfisicalProvider initializes an Infisical secret provider
func NewInfisicalProvider(cfg Config) (Provider, error) {
	infClient, err := infisicalsdk.NewClient(context.Background(), infisicalsdk.Config{
		SiteURL:      cfg.InfisicalURL,
		ClientID:     cfg.InfisicalClientID,
		ClientSecret: cfg.InfisicalClientSecret,
		ProjectID:    cfg.InfisicalProjectID,
		Environment:  cfg.InfisicalEnvironment,
		ServiceToken: cfg.InfisicalServiceToken,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create infisical client: %w", err)
	}

	return &InfisicalProvider{
		client: infClient,
	}, nil
}

func parseKeyPath(key string) (secretKey string, secretPath string) {
	parts := strings.Split(key, "/")
	if len(parts) > 1 {
		secretKey = parts[len(parts)-1]
		secretPath = "/" + strings.Join(parts[:len(parts)-1], "/")
	} else {
		secretKey = key
		secretPath = "/"
	}
	return secretKey, secretPath
}

func (p *InfisicalProvider) Get(ctx context.Context, key string) (string, error) {
	secretKey, secretPath := parseKeyPath(key)
	val, err := p.client.GetSecret(ctx, secretKey, secretPath)
	if err != nil {
		return "", ErrSecretNotFound
	}
	return val, nil
}

func (p *InfisicalProvider) GetMap(ctx context.Context, key string) (map[string]string, error) {
	path := key
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	secrets, err := p.client.ListSecrets(ctx, path)
	if err != nil {
		return nil, ErrSecretNotFound
	}

	result := make(map[string]string)
	for _, s := range secrets {
		result[s.SecretKey] = s.SecretValue
	}
	return result, nil
}

func (p *InfisicalProvider) Put(ctx context.Context, key string, value string) error {
	secretKey, secretPath := parseKeyPath(key)
	return p.client.SetSecret(ctx, secretKey, value, secretPath)
}

func (p *InfisicalProvider) PutMap(ctx context.Context, key string, values map[string]string) error {
	path := key
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	for k, v := range values {
		if err := p.client.SetSecret(ctx, k, v, path); err != nil {
			return err
		}
	}
	return nil
}

func (p *InfisicalProvider) Delete(ctx context.Context, key string) error {
	secretKey, secretPath := parseKeyPath(key)
	return p.client.DeleteSecret(ctx, secretKey, secretPath)
}

func (p *InfisicalProvider) List(ctx context.Context, prefix string) ([]string, error) {
	path := prefix
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	secrets, err := p.client.ListSecrets(ctx, path)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, s := range secrets {
		keys = append(keys, s.SecretKey)
	}
	return keys, nil
}

func (p *InfisicalProvider) Rotate(ctx context.Context, key string) error {
	return nil
}

func (p *InfisicalProvider) Health(ctx context.Context) error {
	_, err := p.client.ListSecrets(ctx, "/")
	return err
}

func (p *InfisicalProvider) Close() error {
	return nil
}
