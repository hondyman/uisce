package infra

import (
	"context"
	"os"
	"sync"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Buckets creates a tenant's WORM, encrypted bucket.
type Buckets interface {
	EnsureTenantBucket(ctx context.Context, spec iceberg.TenantBucketSpec) (*iceberg.TenantBucket, error)
}

// Environment, all required, none defaulted:
//
//	KES_ENDPOINT, KES_API_KEY                      the KMS that holds per-tenant keys
//	LAKEHOUSE_MINIO_ENDPOINT                       host:port of MinIO, no scheme
//	LAKEHOUSE_MINIO_ACCESS_KEY / _SECRET_KEY       the identity that creates tenant buckets
//	                                               and their credentials (never the root user
//	                                               in production)
//	LAKEHOUSE_MINIO_TLS                            "true" if MinIO is served over TLS
//	SECRETS_PROVIDER (+ its settings)              where tenant credentials are stored
//
// Anything missing makes the adapter return ErrNotConfigured when used, which the
// workflow treats as non-retryable, so provisioning fails fast with a readable reason
// instead of falling back to a shared key or a default password.

// KeysFromEnv returns the KES adapter, or one that reports ErrNotConfigured.
func KeysFromEnv() Keys {
	k, err := NewKESKeys(os.Getenv("KES_ENDPOINT"), os.Getenv("KES_API_KEY"))
	if err != nil {
		return unconfiguredKeys{err}
	}
	return k
}

type minioEnv struct {
	endpoint, access, secret string
	secure                   bool
}

func readMinioEnv() (minioEnv, error) {
	e := minioEnv{
		endpoint: os.Getenv("LAKEHOUSE_MINIO_ENDPOINT"),
		access:   os.Getenv("LAKEHOUSE_MINIO_ACCESS_KEY"),
		secret:   os.Getenv("LAKEHOUSE_MINIO_SECRET_KEY"),
		secure:   os.Getenv("LAKEHOUSE_MINIO_TLS") == "true",
	}
	if e.endpoint == "" || e.access == "" || e.secret == "" {
		return e, notConfigured("MinIO", "LAKEHOUSE_MINIO_ENDPOINT", "LAKEHOUSE_MINIO_ACCESS_KEY", "LAKEHOUSE_MINIO_SECRET_KEY")
	}
	return e, nil
}

// credentialsBuilder is a variable so a test can count constructions.
var credentialsBuilder = buildCredentialsFromEnv

func buildCredentialsFromEnv() (Credentials, error) {
	e, err := readMinioEnv()
	if err != nil {
		return nil, err
	}
	store, err := dscreds.ProviderFromEnv()
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, notConfigured("the secrets store", "SECRETS_PROVIDER")
	}
	admin, err := madmin.NewWithOptions(e.endpoint, &madmin.Options{
		Creds: credentials.NewStaticV4(e.access, e.secret, ""), Secure: e.secure,
	})
	if err != nil {
		return nil, err
	}
	return NewBucketCredentials(admin, store)
}

// CredentialsFromEnv returns the credential issuer. It is built on first use, not now:
// constructing the secrets provider authenticates over the network, and the worker must
// never start slowly, or fail to start, because of lakehouse configuration it may not even
// use. A failed build is not cached, so a later retry can succeed; a successful one is.
func CredentialsFromEnv() Credentials {
	return &lazyCredentials{build: func() (Credentials, error) { return credentialsBuilder() }}
}

type lazyCredentials struct {
	mu    sync.Mutex
	build func() (Credentials, error)
	inner Credentials
}

func (l *lazyCredentials) get() (Credentials, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inner != nil {
		return l.inner, nil
	}
	c, err := l.build()
	if err != nil {
		return nil, err
	}
	l.inner = c
	return c, nil
}

func (l *lazyCredentials) EnsureBucketCredential(ctx context.Context, tenantID uuid.UUID, bucket string, mayMint bool) error {
	c, err := l.get()
	if err != nil {
		return err
	}
	return c.EnsureBucketCredential(ctx, tenantID, bucket, mayMint)
}

func (l *lazyCredentials) Read(ctx context.Context, tenantID uuid.UUID) (string, string, error) {
	c, err := l.get()
	if err != nil {
		return "", "", err
	}
	return c.Read(ctx, tenantID)
}

// BucketsFromEnv returns the bucket provisioner, or one that reports ErrNotConfigured.
func BucketsFromEnv() Buckets {
	e, err := readMinioEnv()
	if err != nil {
		return unconfiguredBuckets{err}
	}
	cl, err := minio.New(e.endpoint, &minio.Options{Creds: credentials.NewStaticV4(e.access, e.secret, ""), Secure: e.secure})
	if err != nil {
		return unconfiguredBuckets{err}
	}
	return iceberg.NewTenantBucketProvisioner(cl)
}

type unconfiguredBuckets struct{ err error }

func (u unconfiguredBuckets) EnsureTenantBucket(context.Context, iceberg.TenantBucketSpec) (*iceberg.TenantBucket, error) {
	return nil, u.err
}
