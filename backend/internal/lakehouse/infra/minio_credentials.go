package infra

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/hondyman/uisce/backend/internal/secrets"
	"github.com/minio/madmin-go/v3"
)

// Secret keys inside a tenant's folder in the secrets store.
const (
	KeyAccessKeyID     = "ACCESS_KEY_ID"
	KeySecretAccessKey = "SECRET_ACCESS_KEY"
)

// SecretPath is the one place a tenant's lakehouse credential lives. It is derived from
// the tenant id, never supplied, so one tenant's path cannot name another's.
func SecretPath(tenantID uuid.UUID) string { return "/lakehouse/" + tenantID.String() }

type adminAPI interface {
	AddServiceAccount(ctx context.Context, req madmin.AddServiceAccountReq) (madmin.Credentials, error)
	UpdateServiceAccount(ctx context.Context, accessKey string, req madmin.UpdateServiceAccountReq) error
}

// BucketCredentials issues a MinIO service account limited to one tenant's bucket and
// keeps its keys in the secrets store.
type BucketCredentials struct {
	admin   adminAPI
	secrets secrets.Provider
	rand    io.Reader
}

// NewBucketCredentials builds the issuer. admin is a MinIO admin client; the identity
// behind it is the parent of every credential it creates, and a service account can only
// ever narrow its parent's rights, so give that identity access to the tenant buckets
// and nothing else.
func NewBucketCredentials(admin *madmin.AdminClient, store secrets.Provider) (*BucketCredentials, error) {
	if admin == nil || store == nil {
		return nil, errors.New("an admin client and a secrets store are required")
	}
	return &BucketCredentials{admin: admin, secrets: store, rand: rand.Reader}, nil
}

// BucketPolicy is the entire authority a tenant's credential has: read, write and list in
// its own bucket. Nothing else is allowed, so it cannot change the bucket's lock,
// versioning or encryption, delete the bucket, or touch any other bucket.
func BucketPolicy(bucket string) json.RawMessage {
	p := map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{{
			"Effect": "Allow",
			"Action": []string{
				"s3:GetObject", "s3:PutObject", "s3:DeleteObject",
				"s3:ListBucket", "s3:GetBucketLocation",
				"s3:AbortMultipartUpload", "s3:ListMultipartUploadParts", "s3:ListBucketMultipartUploads",
			},
			"Resource": []string{"arn:aws:s3:::" + bucket, "arn:aws:s3:::" + bucket + "/*"},
		}},
	}
	b, _ := json.Marshal(p)
	return b
}

// EnsureBucketCredential makes sure a credential limited to this tenant's bucket exists,
// and is idempotent. mayMint is false once a credential has been issued; from then on a
// credential that cannot be read is ErrCredentialLost, never a reason to mint a new one.
//
// Order matters. The keys are stored FIRST and the MinIO account created second, so a
// crash between the two cannot lose a secret: the next run finds the stored keys and
// creates the account with them. Then it creates the account, and if that fails it
// updates it to the desired state, which succeeds only if the account already exists and
// also repairs a drifted policy or secret. It does not depend on the server's exact
// "already exists" error text.
func (c *BucketCredentials) EnsureBucketCredential(ctx context.Context, tenantID uuid.UUID, bucket string, mayMint bool) error {
	want, err := iceberg.TenantWarehouseName(tenantID)
	if err != nil {
		return err
	}
	if bucket != want {
		return fmt.Errorf("bucket %q is not tenant %s's bucket", bucket, tenantID)
	}
	return c.ensureCredential(ctx, SecretPath(tenantID), accountName(tenantID),
		"Ivy lakehouse storage credential for tenant "+tenantID.String(), bucket, mayMint)
}

// PlatformSecretPath is the one place the platform warehouse's credential lives. It is deliberately NOT a
// tenant path: tenant paths are "/lakehouse/<uuid>", so this can never be one, and no tenant id (reserved or
// otherwise) is used to stand in for the platform.
const PlatformSecretPath = "/lakehouse/platform/" + iceberg.ControlWarehouseName

// platformAccountName is the MinIO service account of the platform credential: at most 32 characters,
// starting with a letter, and distinct from every tenant's "ivy-lh-<12 hex>".
const platformAccountName = "ivy-lh-platform-control"

// EnsurePlatformCredential is EnsureBucketCredential for the ivy-control bucket, with the same ordering and
// the same mayMint rule: the platform's registry row, not the secrets store, says whether one was issued.
func (c *BucketCredentials) EnsurePlatformCredential(ctx context.Context, mayMint bool) error {
	return c.ensureCredential(ctx, PlatformSecretPath, platformAccountName,
		"Ivy lakehouse storage credential for the platform warehouse", iceberg.ControlWarehouseName, mayMint)
}

// ReadPlatform returns the stored platform credential, under the same rule as Read: only a caller that must
// hand it straight to a storage client may use it.
func (c *BucketCredentials) ReadPlatform(ctx context.Context) (string, string, error) {
	return c.readAt(ctx, PlatformSecretPath, "the platform warehouse")
}

// ensureCredential holds the ordering that matters, once, for tenants and the platform alike.
func (c *BucketCredentials) ensureCredential(ctx context.Context, path, account, description, bucket string, mayMint bool) error {
	access, secret, err := c.storedOrNew(ctx, path, bucket, mayMint)
	if err != nil {
		return err
	}
	policy := BucketPolicy(bucket)

	_, addErr := c.admin.AddServiceAccount(ctx, madmin.AddServiceAccountReq{
		Policy:      policy,
		AccessKey:   access,
		SecretKey:   secret,
		Name:        account,
		Description: description,
	})
	if addErr == nil {
		return nil
	}
	upErr := c.admin.UpdateServiceAccount(ctx, access, madmin.UpdateServiceAccountReq{
		NewPolicy:    policy,
		NewSecretKey: secret,
		NewStatus:    "on",
	})
	if upErr == nil {
		return nil
	}
	// Neither error carries the secret: the requests above are never formatted.
	return fmt.Errorf("create storage credential: %w (update also failed: %v)", addErr, upErr)
}

// Read returns the stored credential. Only a caller that must hand it straight to a
// storage client may use it; it must never be returned from an activity.
func (c *BucketCredentials) Read(ctx context.Context, tenantID uuid.UUID) (string, string, error) {
	return c.readAt(ctx, SecretPath(tenantID), "tenant "+tenantID.String())
}

func (c *BucketCredentials) readAt(ctx context.Context, path, who string) (string, string, error) {
	m, err := c.secrets.GetMap(ctx, path)
	if err != nil {
		return "", "", fmt.Errorf("read storage credential: %w", err)
	}
	if m[KeyAccessKeyID] == "" || m[KeySecretAccessKey] == "" {
		return "", "", fmt.Errorf("storage credential for %s is incomplete", who)
	}
	return m[KeyAccessKeyID], m[KeySecretAccessKey], nil
}

func (c *BucketCredentials) storedOrNew(ctx context.Context, path, bucket string, mayMint bool) (access, secret string, err error) {
	m, err := c.secrets.GetMap(ctx, path)
	switch {
	case err == nil && m[KeyAccessKeyID] != "" && m[KeySecretAccessKey] != "":
		return m[KeyAccessKeyID], m[KeySecretAccessKey], nil
	case err != nil && !errors.Is(err, secrets.ErrSecretNotFound):
		return "", "", fmt.Errorf("read storage credential: %w", err)
	}
	if !mayMint {
		// Not found, but the registry says one was issued. The secrets store reports every
		// failure, including an outage, as "not found", so this cannot be trusted as proof
		// of absence: minting here could overwrite a live credential.
		return "", "", ErrCredentialLost
	}
	if access, err = c.random(20, accessAlphabet, "IVY"); err != nil {
		return "", "", err
	}
	if secret, err = c.random(40, secretAlphabet, ""); err != nil {
		return "", "", err
	}
	if err = c.secrets.PutMap(ctx, path, map[string]string{
		KeyAccessKeyID: access, KeySecretAccessKey: secret, "BUCKET": bucket,
	}); err != nil {
		return "", "", fmt.Errorf("store storage credential: %w", err)
	}
	return access, secret, nil
}

const (
	accessAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	secretAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

// random returns total characters: prefix, then characters drawn uniformly from alphabet.
func (c *BucketCredentials) random(total int, alphabet, prefix string) (string, error) {
	out := []byte(prefix)
	max := big.NewInt(int64(len(alphabet)))
	for len(out) < total {
		n, err := rand.Int(c.rand, max)
		if err != nil {
			return "", fmt.Errorf("generate credential: %w", err)
		}
		out = append(out, alphabet[n.Int64()])
	}
	return string(out), nil
}

// accountName is "ivy-lh-" plus the first 12 hex digits of the tenant id: at most 32
// characters, starting with a letter, as MinIO requires.
func accountName(tenantID uuid.UUID) string {
	hex := make([]byte, 0, 12)
	for _, r := range tenantID.String() {
		if r != '-' {
			hex = append(hex, byte(r))
		}
		if len(hex) == 12 {
			break
		}
	}
	return "ivy-lh-" + string(hex)
}

type unconfiguredCredentials struct{ err error }

func (u unconfiguredCredentials) EnsureBucketCredential(context.Context, uuid.UUID, string, bool) error {
	return u.err
}
func (u unconfiguredCredentials) Read(context.Context, uuid.UUID) (string, string, error) {
	return "", "", u.err
}
