package iceberg

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

func controlSpec() ControlBucketSpec {
	return ControlBucketSpec{KMSKeyID: "ivy-control-key", RetentionDays: 2555}
}

// The platform bucket must never be weaker than a tenant's: Object Lock at creation, COMPLIANCE for the
// whole retention, default SSE-KMS under its own key.
func TestEnsureControlBucket_IsWormAndEncryptedLikeATenants(t *testing.T) {
	f := &fakeBuckets{}
	got, err := prov(f).EnsureControlBucket(context.Background(), controlSpec())
	require.NoError(t, err)
	require.Equal(t, ControlWarehouseName, got.Name)
	require.True(t, got.Created)
	require.Equal(t, []string{ControlWarehouseName}, f.madeNames)
	require.True(t, f.made[0].ObjectLocking, "Object Lock can only be requested at creation")
	require.Equal(t, minio.Compliance, *f.mode)
	require.EqualValues(t, 2555, *f.validity)
	require.Equal(t, minio.Days, *f.unit)
	require.Equal(t, "ivy-control-key", f.kmsKey)
}

func TestEnsureControlBucket_RequiresKeyAndRetentionWithNoDefault(t *testing.T) {
	for name, s := range map[string]ControlBucketSpec{
		"no key":       {RetentionDays: 365},
		"no retention": {KMSKeyID: "k"},
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeBuckets{}
			_, err := prov(f).EnsureControlBucket(context.Background(), s)
			require.Error(t, err)
			require.Empty(t, f.made, "nothing is created from an incomplete spec")
		})
	}
}

func TestEnsureControlBucket_IdempotentAndNeverChangesAnExistingBucket(t *testing.T) {
	f := &fakeBuckets{}
	_, err := prov(f).EnsureControlBucket(context.Background(), controlSpec())
	require.NoError(t, err)
	locks, ssed := f.setLock, f.setSSE
	got, err := prov(f).EnsureControlBucket(context.Background(), controlSpec())
	require.NoError(t, err)
	require.False(t, got.Created)
	require.Equal(t, locks, f.setLock, "an already-correct bucket is left untouched")
	require.Equal(t, ssed, f.setSSE)
	require.Len(t, f.made, 1)
}

func TestEnsureControlBucket_RefusesWhatItMustNotReconcile(t *testing.T) {
	// Exists with no Object Lock: it can never be made WORM now.
	f := &fakeBuckets{exists: true}
	_, err := prov(f).EnsureControlBucket(context.Background(), controlSpec())
	require.True(t, errors.Is(err, ErrBucketConflict))

	// Exists, locked, under a different key: never re-keyed.
	f = &fakeBuckets{}
	_, err = prov(f).EnsureControlBucket(context.Background(), controlSpec())
	require.NoError(t, err)
	other := controlSpec()
	other.KMSKeyID = "some-other-key"
	_, err = prov(f).EnsureControlBucket(context.Background(), other)
	require.True(t, errors.Is(err, ErrBucketConflict))
	require.Equal(t, "ivy-control-key", f.kmsKey)
}

func TestControlWarehouseName_CanNeverBeATenantsName(t *testing.T) {
	require.False(t, strings.HasPrefix(ControlWarehouseName, tenantWarehousePrefix))
	for i := 0; i < 200; i++ {
		n, err := TenantWarehouseName(uuid.New())
		require.NoError(t, err)
		require.NotEqual(t, ControlWarehouseName, n)
	}
	require.LessOrEqual(t, len(ControlWarehouseName), 63, "S3 bucket name limit")
}

func TestEnsureControlWarehouse_CreatesOverItsOwnBucketOnly(t *testing.T) {
	f := &fakeLakekeeper{warehouses: map[string]string{}}
	p := newProvisioner(t, f)
	got, err := p.EnsureControlWarehouse(context.Background(), ControlWarehouseSpec{Endpoint: "http://minio:9000", AccessKeyID: "ctl-key", SecretAccessKey: "ctl-secret"})
	require.NoError(t, err)
	require.Equal(t, ControlWarehouseName, got.Name)
	require.Equal(t, ControlWarehouseName, got.Bucket)
	require.True(t, got.Created)
	require.Len(t, f.posts, 1)
	profile := f.posts[0]["storage-profile"].(map[string]interface{})
	require.Equal(t, ControlWarehouseName, profile["bucket"])
	require.NotContains(t, profile["bucket"], "shared-bucket", "the provisioner's shared bucket is never used")
	cred := f.posts[0]["storage-credential"].(map[string]interface{})
	require.Equal(t, "ctl-key", cred["access-key-id"])
}

func TestEnsureControlWarehouse_IdempotentAndRejectsAnIncompleteSpec(t *testing.T) {
	f := &fakeLakekeeper{warehouses: map[string]string{}}
	p := newProvisioner(t, f)
	ok := ControlWarehouseSpec{Endpoint: "http://minio:9000", AccessKeyID: "k", SecretAccessKey: "s"}
	_, err := p.EnsureControlWarehouse(context.Background(), ok)
	require.NoError(t, err)
	again, err := p.EnsureControlWarehouse(context.Background(), ok)
	require.NoError(t, err)
	require.False(t, again.Created)
	require.Len(t, f.posts, 1, "no second create")

	for name, s := range map[string]ControlWarehouseSpec{
		"no key":      {Endpoint: "e", SecretAccessKey: "s"},
		"no secret":   {Endpoint: "e", AccessKeyID: "k"},
		"no endpoint": {AccessKeyID: "k", SecretAccessKey: "s"},
	} {
		t.Run(name, func(t *testing.T) {
			g := &fakeLakekeeper{warehouses: map[string]string{}}
			_, err := newProvisioner(t, g).EnsureControlWarehouse(context.Background(), s)
			require.Error(t, err)
			require.Empty(t, g.posts)
		})
	}
}
