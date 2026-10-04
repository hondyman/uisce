package infra

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/hondyman/uisce/backend/internal/secrets"
	"github.com/minio/kes-go"
	"github.com/minio/madmin-go/v3"
	"github.com/stretchr/testify/require"
)

type fakeKES struct {
	names []string
	err   error
}

func (f *fakeKES) CreateKey(_ context.Context, n string) error {
	f.names = append(f.names, n)
	return f.err
}

func TestKESKeys(t *testing.T) {
	ctx := context.Background()
	f := &fakeKES{}
	k := &KESKeys{api: f}

	require.NoError(t, k.EnsureKey(ctx, "ivy-t-abc"))
	require.Equal(t, []string{"ivy-t-abc"}, f.names)

	f.err = kes.ErrKeyExists
	require.NoError(t, k.EnsureKey(ctx, "ivy-t-abc"), "an existing key is success: a re-run is safe and never rotates it")

	f.err = errors.New("kes unreachable")
	require.ErrorContains(t, k.EnsureKey(ctx, "ivy-t-abc"), "kes unreachable")

	require.Error(t, k.EnsureKey(ctx, ""))

	_, err := NewKESKeys("", "")
	require.ErrorIs(t, err, ErrNotConfigured)
	_, err = NewKESKeys("https://kes:7373", "")
	require.ErrorIs(t, err, ErrNotConfigured, "an endpoint without a key is not usable")
}

// fakeAdmin records calls and can assert what the secrets store held when it was called.
type fakeAdmin struct {
	addErr, updErr error
	adds           []madmin.AddServiceAccountReq
	updates        []struct {
		access string
		req    madmin.UpdateServiceAccountReq
	}
	onAdd func(madmin.AddServiceAccountReq)
}

func (f *fakeAdmin) AddServiceAccount(_ context.Context, r madmin.AddServiceAccountReq) (madmin.Credentials, error) {
	if f.onAdd != nil {
		f.onAdd(r)
	}
	f.adds = append(f.adds, r)
	return madmin.Credentials{}, f.addErr
}
func (f *fakeAdmin) UpdateServiceAccount(_ context.Context, a string, r madmin.UpdateServiceAccountReq) error {
	f.updates = append(f.updates, struct {
		access string
		req    madmin.UpdateServiceAccountReq
	}{a, r})
	return f.updErr
}

func newCreds() (*BucketCredentials, *fakeAdmin, *secrets.MemoryProvider) {
	admin, store := &fakeAdmin{}, secrets.NewMemoryProvider()
	return &BucketCredentials{admin: admin, secrets: store, rand: cryptoRand()}, admin, store
}

func bucketOf(id uuid.UUID) string { n, _ := iceberg.TenantWarehouseName(id); return n }

func TestEnsureBucketCredential_StoresBeforeCreatingAndScopesToTheBucket(t *testing.T) {
	ctx := context.Background()
	c, admin, store := newCreds()
	id := uuid.New()

	admin.onAdd = func(r madmin.AddServiceAccountReq) {
		m, err := store.GetMap(ctx, SecretPath(id))
		require.NoError(t, err, "the keys must already be stored when the account is created, or a crash could lose the secret")
		require.Equal(t, r.AccessKey, m[KeyAccessKeyID])
		require.Equal(t, r.SecretKey, m[KeySecretAccessKey])
	}
	require.NoError(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true))
	require.Len(t, admin.adds, 1)

	r := admin.adds[0]
	require.Len(t, r.AccessKey, 20)
	require.True(t, strings.HasPrefix(r.AccessKey, "IVY"))
	require.Regexp(t, `^[A-Z0-9]{20}$`, r.AccessKey)
	require.Regexp(t, `^[A-Za-z0-9]{40}$`, r.SecretKey)
	require.LessOrEqual(t, len(r.Name), 32)
	require.Regexp(t, `^[a-zA-Z][a-zA-Z0-9_-]*$`, r.Name)
	require.Equal(t, string(BucketPolicy(bucketOf(id))), string(r.Policy))
	require.Empty(t, admin.updates)

	key, secret, err := c.Read(ctx, id)
	require.NoError(t, err)
	require.Equal(t, r.AccessKey, key)
	require.Equal(t, r.SecretKey, secret)
}

func TestBucketPolicy_GrantsOnlyThisBucket(t *testing.T) {
	bucket := "ivy-t-" + strings.Repeat("a", 32)
	var p struct {
		Statement []struct {
			Effect   string
			Action   []string
			Resource []string
		}
	}
	require.NoError(t, json.Unmarshal(BucketPolicy(bucket), &p))
	require.Len(t, p.Statement, 1)
	s := p.Statement[0]
	require.Equal(t, "Allow", s.Effect)
	require.ElementsMatch(t, []string{"arn:aws:s3:::" + bucket, "arn:aws:s3:::" + bucket + "/*"}, s.Resource)
	for _, a := range s.Action {
		require.NotEqual(t, "s3:*", a)
		require.False(t, strings.Contains(a, "*"), "no wildcard action: %s", a)
	}
	for _, forbidden := range []string{
		"s3:PutBucketObjectLockConfiguration", "s3:PutObjectRetention", "s3:BypassGovernanceRetention", "s3:PutObjectLegalHold",
		"s3:DeleteBucket", "s3:PutEncryptionConfiguration", "s3:PutBucketVersioning", "s3:CreateBucket", "s3:ListAllMyBuckets",
	} {
		require.NotContains(t, s.Action, forbidden, "a tenant credential must not be able to do this")
	}
}

func TestEnsureBucketCredential_ReusesStoredKeysAndHealsAnExistingAccount(t *testing.T) {
	ctx := context.Background()
	c, admin, store := newCreds()
	id := uuid.New()
	require.NoError(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true))
	first, _, _ := c.Read(ctx, id)

	// Second run: the account already exists, so create fails and update succeeds.
	admin.addErr = errors.New("service account already exists")
	require.NoError(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true))
	require.Len(t, admin.updates, 1)
	up := admin.updates[0]
	require.Equal(t, first, up.access, "the same stored keys are reused, never regenerated")
	require.Equal(t, "on", up.req.NewStatus)
	require.Equal(t, string(BucketPolicy(bucketOf(id))), string(up.req.NewPolicy), "update restores the bucket-scoped policy")
	m, _ := store.GetMap(ctx, SecretPath(id))
	require.Equal(t, m[KeySecretAccessKey], up.req.NewSecretKey, "and the stored secret")
}

// A crash after the secret was stored and before the account was created must resume
// with the same keys, not mint new ones and orphan the stored secret.
func TestEnsureBucketCredential_ResumesAfterACrashBetweenStoreAndCreate(t *testing.T) {
	ctx := context.Background()
	c, admin, _ := newCreds()
	id := uuid.New()

	admin.addErr, admin.updErr = errors.New("minio down"), errors.New("minio down")
	require.Error(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true))
	stored, _, err := c.Read(ctx, id)
	require.NoError(t, err, "the secret survives the failed run")

	admin.addErr, admin.updErr = nil, nil
	require.NoError(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true))
	require.Equal(t, stored, admin.adds[len(admin.adds)-1].AccessKey, "the retry used the stored keys")
}

func TestEnsureBucketCredential_ErrorsNeverCarryTheSecret(t *testing.T) {
	ctx := context.Background()
	c, admin, _ := newCreds()
	id := uuid.New()
	admin.addErr, admin.updErr = errors.New("add refused"), errors.New("update refused")
	err := c.EnsureBucketCredential(ctx, id, bucketOf(id), true)
	require.Error(t, err)
	_, secret, rerr := c.Read(ctx, id)
	require.NoError(t, rerr)
	require.NotContains(t, err.Error(), secret)
	require.ErrorContains(t, err, "add refused")
	require.ErrorContains(t, err, "update refused")
}

func TestEnsureBucketCredential_RefusesAnotherTenantsBucket(t *testing.T) {
	ctx := context.Background()
	c, admin, store := newCreds()
	a, b := uuid.New(), uuid.New()
	require.Error(t, c.EnsureBucketCredential(ctx, a, bucketOf(b), true))
	require.Error(t, c.EnsureBucketCredential(ctx, a, "shared-bucket", true))
	require.Error(t, c.EnsureBucketCredential(ctx, uuid.Nil, "ivy-t-", true))
	require.Empty(t, admin.adds)
	require.Empty(t, admin.updates)
	_, err := store.GetMap(ctx, SecretPath(a))
	require.ErrorIs(t, err, secrets.ErrSecretNotFound, "nothing is stored for a refused request")
}

func TestEnsureBucketCredential_TenantsGetDifferentCredentialsAndPaths(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newCreds()
	a, b := uuid.New(), uuid.New()
	require.NoError(t, c.EnsureBucketCredential(ctx, a, bucketOf(a), true))
	require.NoError(t, c.EnsureBucketCredential(ctx, b, bucketOf(b), true))
	ka, sa, _ := c.Read(ctx, a)
	kb, sb, _ := c.Read(ctx, b)
	require.NotEqual(t, ka, kb)
	require.NotEqual(t, sa, sb)
	require.NotEqual(t, SecretPath(a), SecretPath(b))
	require.Equal(t, "/lakehouse/"+a.String(), SecretPath(a))
}

type failingStore struct {
	*secrets.MemoryProvider
	getErr, putErr error
}

func (f failingStore) GetMap(ctx context.Context, k string) (map[string]string, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.MemoryProvider.GetMap(ctx, k)
}
func (f failingStore) PutMap(ctx context.Context, k string, v map[string]string) error {
	if f.putErr != nil {
		return f.putErr
	}
	return f.MemoryProvider.PutMap(ctx, k, v)
}

func TestEnsureBucketCredential_SecretsStoreFailuresStopBeforeMinIO(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	for name, st := range map[string]failingStore{
		"read fails (not a not-found)": {MemoryProvider: secrets.NewMemoryProvider(), getErr: errors.New("vault sealed")},
		"write fails":                  {MemoryProvider: secrets.NewMemoryProvider(), putErr: errors.New("vault read-only")},
	} {
		t.Run(name, func(t *testing.T) {
			admin := &fakeAdmin{}
			c := &BucketCredentials{admin: admin, secrets: st, rand: cryptoRand()}
			require.Error(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true))
			require.Empty(t, admin.adds, "no MinIO account may be created unless its secret is safely stored")
		})
	}
}

func TestRead_RefusesMissingOrIncompleteCredentials(t *testing.T) {
	ctx := context.Background()
	c, _, store := newCreds()
	id := uuid.New()
	_, _, err := c.Read(ctx, id)
	require.Error(t, err)
	require.NoError(t, store.PutMap(ctx, SecretPath(id), map[string]string{KeyAccessKeyID: "only-half"}))
	_, _, err = c.Read(ctx, id)
	require.ErrorContains(t, err, "incomplete")
}

func TestAccountName(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	n := accountName(id)
	require.Equal(t, "ivy-lh-111111112222", n)
	require.LessOrEqual(t, len(n), 32)
	require.True(t, regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`).MatchString(n))
}

func TestFromEnv_UnconfiguredFailsClearlyAndNeverDefaults(t *testing.T) {
	for _, v := range []string{"KES_ENDPOINT", "KES_API_KEY", "LAKEHOUSE_MINIO_ENDPOINT", "LAKEHOUSE_MINIO_ACCESS_KEY", "LAKEHOUSE_MINIO_SECRET_KEY", "SECRETS_PROVIDER"} {
		t.Setenv(v, "")
	}
	ctx := context.Background()
	id := uuid.New()

	require.ErrorIs(t, KeysFromEnv().EnsureKey(ctx, "k"), ErrNotConfigured)
	require.ErrorIs(t, CredentialsFromEnv().EnsureBucketCredential(ctx, id, bucketOf(id), true), ErrNotConfigured)
	_, _, err := CredentialsFromEnv().Read(ctx, id)
	require.ErrorIs(t, err, ErrNotConfigured)
	_, err = BucketsFromEnv().EnsureTenantBucket(ctx, iceberg.TenantBucketSpec{TenantID: id, KMSKeyID: "k", RetentionDays: 1})
	require.ErrorIs(t, err, ErrNotConfigured)

	// MinIO configured but no secrets store is still unusable for credentials.
	t.Setenv("LAKEHOUSE_MINIO_ENDPOINT", "minio:9000")
	t.Setenv("LAKEHOUSE_MINIO_ACCESS_KEY", "a")
	t.Setenv("LAKEHOUSE_MINIO_SECRET_KEY", "b")
	require.ErrorIs(t, CredentialsFromEnv().EnsureBucketCredential(ctx, id, bucketOf(id), true), ErrNotConfigured)
}

func TestNewBucketCredentials_RequiresItsDependencies(t *testing.T) {
	_, err := NewBucketCredentials(nil, secrets.NewMemoryProvider())
	require.Error(t, err)
	_, err = NewBucketCredentials(&madmin.AdminClient{}, nil)
	require.Error(t, err)
}

// Once a credential has been issued, "not found" is not proof it is gone (the secrets
// store reports every failure that way), so it must never be replaced silently.
func TestEnsureBucketCredential_NeverMintsOnceIssued(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	t.Run("a credential that cannot be read is ErrCredentialLost, and nothing is created or stored", func(t *testing.T) {
		c, admin, store := newCreds()
		err := c.EnsureBucketCredential(ctx, id, bucketOf(id), false)
		require.ErrorIs(t, err, ErrCredentialLost)
		require.Empty(t, admin.adds)
		require.Empty(t, admin.updates)
		_, gerr := store.GetMap(ctx, SecretPath(id))
		require.ErrorIs(t, gerr, secrets.ErrSecretNotFound, "no replacement was written")
	})

	t.Run("a half-stored credential is also lost, not overwritten", func(t *testing.T) {
		c, admin, store := newCreds()
		require.NoError(t, store.PutMap(ctx, SecretPath(id), map[string]string{KeyAccessKeyID: "only-the-key"}))
		require.ErrorIs(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), false), ErrCredentialLost)
		require.Empty(t, admin.adds)
		m, _ := store.GetMap(ctx, SecretPath(id))
		require.Equal(t, "only-the-key", m[KeyAccessKeyID], "the existing value was left alone")
	})

	t.Run("a readable credential is still converged when minting is not allowed", func(t *testing.T) {
		c, admin, _ := newCreds()
		require.NoError(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true))
		stored, _, _ := c.Read(ctx, id)
		admin.addErr = errors.New("already exists")
		require.NoError(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), false))
		require.Equal(t, stored, admin.updates[len(admin.updates)-1].access, "it reuses the stored keys")
	})

	t.Run("a secrets-store outage is returned as is, never as a license to mint", func(t *testing.T) {
		admin := &fakeAdmin{}
		st := failingStore{MemoryProvider: secrets.NewMemoryProvider(), getErr: errors.New("vault sealed")}
		c := &BucketCredentials{admin: admin, secrets: st, rand: cryptoRand()}
		require.Error(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), false))
		require.Empty(t, admin.adds)
	})
}

// The worker must never start slowly, or fail to start, because of lakehouse
// configuration: constructing the secrets provider can authenticate over the network.
func TestCredentialsFromEnv_IsBuiltOnFirstUseNotAtBoot(t *testing.T) {
	var builds int
	var failNext bool
	orig := credentialsBuilder
	t.Cleanup(func() { credentialsBuilder = orig })
	good, _, _ := newCreds()
	credentialsBuilder = func() (Credentials, error) {
		builds++
		if failNext {
			return nil, errors.New("secrets provider unreachable")
		}
		return good, nil
	}

	c := CredentialsFromEnv()
	require.Zero(t, builds, "creating the adapter at worker start must do nothing")

	ctx := context.Background()
	id := uuid.New()
	failNext = true
	require.Error(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true))
	require.Equal(t, 1, builds)

	failNext = false
	require.NoError(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true), "a failed build is not cached, so a retry can succeed")
	require.Equal(t, 2, builds)

	_, _, err := c.Read(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 2, builds, "a successful build is cached")
}

// ADR-049: the platform credential has its own path and account, is scoped to ivy-control alone, and keeps
// every property of a tenant's: stored before created, never minted once issued.
func TestPlatformCredential_OwnPathOwnAccountScopedToIvyControl(t *testing.T) {
	ctx := context.Background()
	c, admin, store := newCreds()
	admin.onAdd = func(r madmin.AddServiceAccountReq) {
		m, err := store.GetMap(ctx, PlatformSecretPath)
		require.NoError(t, err, "stored before the account is created")
		require.Equal(t, r.AccessKey, m[KeyAccessKeyID])
	}
	require.NoError(t, c.EnsurePlatformCredential(ctx, true))
	require.Len(t, admin.adds, 1)
	r := admin.adds[0]
	require.Equal(t, string(BucketPolicy(iceberg.ControlWarehouseName)), string(r.Policy), "authority over ivy-control and nothing else")
	require.NotContains(t, string(r.Policy), "ivy-t-")
	require.Equal(t, platformAccountName, r.Name)
	require.LessOrEqual(t, len(r.Name), 32)
	require.Regexp(t, `^[a-zA-Z][a-zA-Z0-9_-]*$`, r.Name)

	k, s, err := c.ReadPlatform(ctx)
	require.NoError(t, err)
	require.Equal(t, r.AccessKey, k)
	require.Equal(t, r.SecretKey, s)
}

func TestPlatformCredential_CanNeverBeATenantsPathOrAccount(t *testing.T) {
	require.Equal(t, "/lakehouse/platform/ivy-control", PlatformSecretPath)
	for i := 0; i < 200; i++ {
		id := uuid.New()
		require.NotEqual(t, PlatformSecretPath, SecretPath(id))
		require.NotEqual(t, platformAccountName, accountName(id))
	}
}

func TestPlatformCredential_DoesNotDisturbTenantsAndIsNotDisturbedByThem(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newCreds()
	id := uuid.New()
	require.NoError(t, c.EnsureBucketCredential(ctx, id, bucketOf(id), true))
	require.NoError(t, c.EnsurePlatformCredential(ctx, true))
	tk, _, _ := c.Read(ctx, id)
	pk, _, _ := c.ReadPlatform(ctx)
	require.NotEqual(t, tk, pk)
}

func TestPlatformCredential_NeverMintsOnceIssued(t *testing.T) {
	ctx := context.Background()
	c, admin, store := newCreds()
	require.ErrorIs(t, c.EnsurePlatformCredential(ctx, false), ErrCredentialLost)
	require.Empty(t, admin.adds)
	_, err := store.GetMap(ctx, PlatformSecretPath)
	require.ErrorIs(t, err, secrets.ErrSecretNotFound, "nothing replaced it")

	// A secrets-store outage is returned as it is, never as a licence to mint.
	st := failingStore{MemoryProvider: secrets.NewMemoryProvider(), getErr: errors.New("vault sealed")}
	c2 := &BucketCredentials{admin: &fakeAdmin{}, secrets: st, rand: cryptoRand()}
	require.Error(t, c2.EnsurePlatformCredential(ctx, false))
}

func TestPlatformCredential_ReadRefusesMissingOrIncomplete(t *testing.T) {
	ctx := context.Background()
	c, _, store := newCreds()
	_, _, err := c.ReadPlatform(ctx)
	require.Error(t, err)
	require.NoError(t, store.PutMap(ctx, PlatformSecretPath, map[string]string{KeyAccessKeyID: "only"}))
	_, _, err = c.ReadPlatform(ctx)
	require.ErrorContains(t, err, "incomplete")
}

func TestPlatformCredentialsFromEnv_UnconfiguredFailsClearlyAndNeverDefaults(t *testing.T) {
	for _, k := range []string{"LAKEHOUSE_MINIO_ENDPOINT", "LAKEHOUSE_MINIO_ACCESS_KEY", "LAKEHOUSE_MINIO_SECRET_KEY"} {
		t.Setenv(k, "")
	}
	pc := PlatformCredentialsFromEnv() // building it must not fail or connect
	require.ErrorIs(t, pc.EnsurePlatformCredential(context.Background(), true), ErrNotConfigured)
	_, _, err := pc.ReadPlatform(context.Background())
	require.ErrorIs(t, err, ErrNotConfigured)
}
