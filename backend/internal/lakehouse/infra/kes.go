package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/minio/kes-go"
)

type kesAPI interface {
	CreateKey(ctx context.Context, name string) error
}

// KESKeys creates per-tenant keys in KES.
type KESKeys struct{ api kesAPI }

// NewKESKeys connects to a KES server with an API key.
func NewKESKeys(endpoint, apiKey string) (*KESKeys, error) {
	if endpoint == "" || apiKey == "" {
		return nil, notConfigured("KES", "KES_ENDPOINT", "KES_API_KEY")
	}
	key, err := kes.ParseAPIKey(apiKey)
	if err != nil {
		return nil, fmt.Errorf("parse KES API key: %w", err)
	}
	c, err := kes.NewClient(endpoint, key)
	if err != nil {
		return nil, fmt.Errorf("create KES client: %w", err)
	}
	return &KESKeys{api: c}, nil
}

// EnsureKey creates the key if it does not exist and does nothing if it does, so a
// retry or a re-run is safe. It never replaces or rotates an existing key.
func (k *KESKeys) EnsureKey(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("a key name is required")
	}
	if err := k.api.CreateKey(ctx, name); err != nil && !errors.Is(err, kes.ErrKeyExists) {
		return fmt.Errorf("create KES key %q: %w", name, err)
	}
	return nil
}

type unconfiguredKeys struct{ err error }

func (u unconfiguredKeys) EnsureKey(context.Context, string) error { return u.err }
