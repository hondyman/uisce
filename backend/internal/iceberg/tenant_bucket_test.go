package iceberg

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/sse"
	"github.com/stretchr/testify/require"
)

// fakeBuckets models just the bucket state this provisioner reads and writes.
type fakeBuckets struct {
	exists     bool
	lockOn     bool
	mode       *minio.RetentionMode
	validity   *uint
	unit       *minio.ValidityUnit
	kmsKey     string // "" = no default encryption
	made       []minio.MakeBucketOptions
	madeNames  []string
	setLock    int
	ignoreSet  bool
	setSSE     int
	lastSSEKey string
}

func (f *fakeBuckets) BucketExists(context.Context, string) (bool, error) { return f.exists, nil }

func (f *fakeBuckets) MakeBucket(_ context.Context, name string, o minio.MakeBucketOptions) error {
	f.made = append(f.made, o)
	f.madeNames = append(f.madeNames, name)
	f.exists, f.lockOn = true, o.ObjectLocking
	return nil
}

func (f *fakeBuckets) GetObjectLockConfig(context.Context, string) (string, *minio.RetentionMode, *uint, *minio.ValidityUnit, error) {
	if !f.lockOn {
		return "", nil, nil, nil, nil
	}
	return "Enabled", f.mode, f.validity, f.unit, nil
}

func (f *fakeBuckets) SetObjectLockConfig(_ context.Context, _ string, m *minio.RetentionMode, v *uint, u *minio.ValidityUnit) error {
	f.setLock++
	if f.ignoreSet {
		return nil
	}
	f.mode, f.validity, f.unit = m, v, u
	return nil
}

func (f *fakeBuckets) GetBucketEncryption(context.Context, string) (*sse.Configuration, error) {
	if f.kmsKey == "" {
		return nil, minio.ErrorResponse{Code: sseNotFoundCode}
	}
	return sse.NewConfigurationSSEKMS(f.kmsKey), nil
}

func (f *fakeBuckets) SetBucketEncryption(_ context.Context, _ string, c *sse.Configuration) error {
	f.setSSE++
	f.kmsKey = c.Rules[0].Apply.KmsMasterKeyID
	f.lastSSEKey = f.kmsKey
	return nil
}

func goodSpec() TenantBucketSpec {
	return TenantBucketSpec{TenantID: uuid.New(), KMSKeyID: "tenant-key", RetentionDays: 2555}
}

func prov(f *fakeBuckets) *TenantBucketProvisioner { return &TenantBucketProvisioner{api: f} }

func ptr[T any](v T) *T { return &v }

func TestEnsureTenantBucket_CreatesWormEncryptedBucket(t *testing.T) {
	f := &fakeBuckets{}
	spec := goodSpec()
	got, err := prov(f).EnsureTenantBucket(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := TenantWarehouseName(spec.TenantID)
	if got.Name != want || !got.Created {
		t.Fatalf("unexpected result %+v", got)
	}
	if len(f.made) != 1 || !f.made[0].ObjectLocking {
		t.Fatalf("bucket must be created with Object Lock, got %+v", f.made)
	}
	if f.madeNames[0] != want {
		t.Fatalf("bucket name %q, want the tenant's own %q", f.madeNames[0], want)
	}
	if f.mode == nil || *f.mode != minio.Compliance || *f.validity != 2555 || *f.unit != minio.Days {
		t.Fatalf("want COMPLIANCE 2555 DAYS, got mode=%v validity=%v unit=%v", f.mode, f.validity, f.unit)
	}
	if f.kmsKey != "tenant-key" {
		t.Fatalf("want default SSE-KMS under the tenant key, got %q", f.kmsKey)
	}
}

func TestEnsureTenantBucket_IdempotentAndUntouchedWhenCorrect(t *testing.T) {
	f := &fakeBuckets{}
	spec := goodSpec()
	if _, err := prov(f).EnsureTenantBucket(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	locks, sses := f.setLock, f.setSSE
	got, err := prov(f).EnsureTenantBucket(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if got.Created || len(f.made) != 1 || f.setLock != locks || f.setSSE != sses {
		t.Fatalf("a correct bucket must not be touched: created=%v makes=%d lock=%d->%d sse=%d->%d",
			got.Created, len(f.made), locks, f.setLock, sses, f.setSSE)
	}
}

func TestEnsureTenantBucket_CompletesAPartialEarlierRun(t *testing.T) {
	// Bucket made with lock, but the process died before retention and SSE.
	f := &fakeBuckets{exists: true, lockOn: true}
	if _, err := prov(f).EnsureTenantBucket(context.Background(), goodSpec()); err != nil {
		t.Fatal(err)
	}
	if len(f.made) != 0 || f.setLock != 1 || f.setSSE != 1 {
		t.Fatalf("want retention and encryption applied without re-creating: makes=%d lock=%d sse=%d", len(f.made), f.setLock, f.setSSE)
	}
}

func TestEnsureTenantBucket_RefusesBucketWithoutObjectLock(t *testing.T) {
	f := &fakeBuckets{exists: true, lockOn: false}
	_, err := prov(f).EnsureTenantBucket(context.Background(), goodSpec())
	if err == nil || !strings.Contains(err.Error(), "without Object Lock") {
		t.Fatalf("a non-WORM bucket must be refused, got %v", err)
	}
	require.ErrorIs(t, err, ErrBucketConflict, "a refusal is a conflict, which a caller must not retry")
	if f.setLock != 0 || f.setSSE != 0 {
		t.Fatal("nothing may be configured on a bucket that cannot hold audit")
	}
}

func TestEnsureTenantBucket_NeverChangesExistingRetention(t *testing.T) {
	spec := goodSpec() // wants 2555 days

	t.Run("governance mode is refused", func(t *testing.T) {
		f := &fakeBuckets{exists: true, lockOn: true, mode: ptr(minio.Governance), validity: ptr(uint(9999)), unit: ptr(minio.Days)}
		_, err := prov(f).EnsureTenantBucket(context.Background(), spec)
		require.ErrorIs(t, err, ErrBucketConflict)
		if f.setLock != 0 {
			t.Fatal("must not rewrite retention")
		}
	})
	t.Run("shorter compliance retention is refused", func(t *testing.T) {
		f := &fakeBuckets{exists: true, lockOn: true, mode: ptr(minio.Compliance), validity: ptr(uint(30)), unit: ptr(minio.Days)}
		_, err := prov(f).EnsureTenantBucket(context.Background(), spec)
		require.ErrorIs(t, err, ErrBucketConflict)
		if f.setLock != 0 {
			t.Fatal("must not rewrite retention")
		}
	})
	t.Run("longer compliance retention is accepted untouched", func(t *testing.T) {
		f := &fakeBuckets{exists: true, lockOn: true, mode: ptr(minio.Compliance), validity: ptr(uint(10)), unit: ptr(minio.Years), kmsKey: "tenant-key"}
		if _, err := prov(f).EnsureTenantBucket(context.Background(), spec); err != nil {
			t.Fatalf("10 years >= 2555 days must be accepted: %v", err)
		}
		if f.setLock != 0 {
			t.Fatal("must not rewrite retention")
		}
	})
}

func TestEnsureTenantBucket_RefusesDifferentKey(t *testing.T) {
	f := &fakeBuckets{exists: true, lockOn: true, mode: ptr(minio.Compliance), validity: ptr(uint(2555)), unit: ptr(minio.Days), kmsKey: "someone-elses-key"}
	_, err := prov(f).EnsureTenantBucket(context.Background(), goodSpec())
	require.ErrorIs(t, err, ErrBucketConflict, "a bucket under a different key must be refused")
	if f.setSSE != 0 {
		t.Fatal("must not change the default key")
	}
}

func TestEnsureTenantBucket_RequiresKeyAndRetention(t *testing.T) {
	f := &fakeBuckets{}
	cases := map[string]func(*TenantBucketSpec){
		"nil tenant":     func(s *TenantBucketSpec) { s.TenantID = uuid.Nil },
		"no kms key":     func(s *TenantBucketSpec) { s.KMSKeyID = "" },
		"zero retention": func(s *TenantBucketSpec) { s.RetentionDays = 0 },
	}
	for name, mutate := range cases {
		s := goodSpec()
		mutate(&s)
		_, err := prov(f).EnsureTenantBucket(context.Background(), s)
		if err == nil {
			t.Errorf("%s: want error", name)
		}
		require.NotErrorIs(t, err, ErrBucketConflict, "%s: bad input is not a conflict with an existing bucket", name)
	}
	if len(f.made) != 0 {
		t.Fatalf("an invalid spec must not create anything; made %d", len(f.made))
	}
}

func TestExtendTenantRetention(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	compliance := func(days uint) *fakeBuckets {
		return &fakeBuckets{exists: true, lockOn: true, mode: ptr(minio.Compliance), validity: ptr(days), unit: ptr(minio.Days), kmsKey: "k"}
	}

	t.Run("raises the default retention and verifies it", func(t *testing.T) {
		f := compliance(365)
		got, err := prov(f).ExtendTenantRetention(ctx, id, 2555)
		require.NoError(t, err)
		require.EqualValues(t, 2555, got)
		require.Equal(t, 1, f.setLock)
		require.Equal(t, minio.Compliance, *f.mode)
		require.EqualValues(t, 2555, *f.validity)
		require.Equal(t, minio.Days, *f.unit)
	})

	t.Run("a bucket already enforcing at least that much is untouched", func(t *testing.T) {
		for name, have := range map[string]uint{"equal": 2555, "stricter": 9999} {
			f := compliance(have)
			got, err := prov(f).ExtendTenantRetention(ctx, id, 2555)
			require.NoError(t, err, name)
			require.EqualValues(t, 2555, got, name+": reports that it enforces at least what was asked")
			require.Zero(t, f.setLock, name+": a retry or a stricter bucket is never rewritten, and never lowered")
			require.EqualValues(t, have, *f.validity, name)
		}
	})

	t.Run("counts years as 365 days when comparing", func(t *testing.T) {
		f := &fakeBuckets{exists: true, lockOn: true, mode: ptr(minio.Compliance), validity: ptr(uint(10)), unit: ptr(minio.Years)}
		_, err := prov(f).ExtendTenantRetention(ctx, id, 3000)
		require.NoError(t, err)
		require.Zero(t, f.setLock, "10 years is 3650 days, already more than 3000")
	})

	t.Run("refuses, and touches nothing, when the bucket is not a compliance-locked one", func(t *testing.T) {
		cases := map[string]*fakeBuckets{
			"no object lock":  {exists: true, lockOn: false},
			"governance mode": {exists: true, lockOn: true, mode: ptr(minio.Governance), validity: ptr(uint(9999)), unit: ptr(minio.Days)},
			"missing bucket":  {exists: false},
		}
		for name, f := range cases {
			_, err := prov(f).ExtendTenantRetention(ctx, id, 2555)
			require.ErrorIs(t, err, ErrBucketConflict, name)
			require.Zero(t, f.setLock, name)
		}
	})

	t.Run("does not trust a write the bucket did not accept", func(t *testing.T) {
		f := compliance(365)
		f.ignoreSet = true // the server acknowledges but keeps the old value
		_, err := prov(f).ExtendTenantRetention(ctx, id, 2555)
		require.ErrorContains(t, err, "did not accept")
	})

	t.Run("rejects bad input before touching the bucket", func(t *testing.T) {
		f := compliance(365)
		_, err := prov(f).ExtendTenantRetention(ctx, id, 0)
		require.Error(t, err)
		_, err = prov(f).ExtendTenantRetention(ctx, uuid.Nil, 10)
		require.Error(t, err)
		require.Zero(t, f.setLock)
	})
}
