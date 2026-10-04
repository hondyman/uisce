package activities_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

type fakeReg struct {
	cfg         *registry.Config
	getErr      error
	markErr     error
	marked      []markCall
	failures    []failCall
	credMarks   int
	credMarkErr error
	applied     []int
	appliedErr  error
	syncFails   []failCall
}
type markCall struct {
	tenant, warehouse uuid.UUID
	key               string
	applied           int
	actor             registry.Actor
}
type failCall struct {
	tenant       uuid.UUID
	step, reason string
	actor        registry.Actor
}

func (f *fakeReg) Get(context.Context, uuid.UUID) (*registry.Config, error) { return f.cfg, f.getErr }
func (f *fakeReg) MarkProvisioned(_ context.Context, t, w uuid.UUID, k string, applied int, a registry.Actor) error {
	f.marked = append(f.marked, markCall{t, w, k, applied, a})
	return f.markErr
}
func (f *fakeReg) MarkRetentionApplied(_ context.Context, _ uuid.UUID, days int, _ registry.Actor) error {
	f.applied = append(f.applied, days)
	return f.appliedErr
}
func (f *fakeReg) RecordRetentionSyncFailure(_ context.Context, t uuid.UUID, a registry.Actor, step, reason string) error {
	f.syncFails = append(f.syncFails, failCall{t, step, reason, a})
	return nil
}
func (f *fakeReg) MarkCredentialIssued(context.Context, uuid.UUID) error {
	f.credMarks++
	return f.credMarkErr
}
func (f *fakeReg) RecordProvisionFailure(_ context.Context, t uuid.UUID, a registry.Actor, step, reason string) error {
	f.failures = append(f.failures, failCall{t, step, reason, a})
	return nil
}

type fakeKeys struct {
	names []string
	err   error
}

func (f *fakeKeys) EnsureKey(_ context.Context, n string) error {
	f.names = append(f.names, n)
	return f.err
}

type fakeBuckets struct {
	specs    []iceberg.TenantBucketSpec
	err      error
	extended []uint
	extErr   error
	extGot   uint
}

func (f *fakeBuckets) EnsureTenantBucket(_ context.Context, s iceberg.TenantBucketSpec) (*iceberg.TenantBucket, error) {
	f.specs = append(f.specs, s)
	return &iceberg.TenantBucket{Name: "b"}, f.err
}

func (f *fakeBuckets) ExtendTenantRetention(_ context.Context, _ uuid.UUID, days uint) (uint, error) {
	f.extended = append(f.extended, days)
	if f.extErr != nil {
		return 0, f.extErr
	}
	if f.extGot != 0 {
		return f.extGot, nil
	}
	return days, nil
}

type fakeCreds struct {
	ensured            []string
	mayMint            []bool
	ensureErr, readErr error
	key, secret        string
}

func (f *fakeCreds) EnsureBucketCredential(_ context.Context, _ uuid.UUID, bucket string, mayMint bool) error {
	f.ensured = append(f.ensured, bucket)
	f.mayMint = append(f.mayMint, mayMint)
	return f.ensureErr
}
func (f *fakeCreds) Read(context.Context, uuid.UUID) (string, string, error) {
	return f.key, f.secret, f.readErr
}

type fakeWarehouses struct {
	specs []iceberg.TenantWarehouseSpec
	id    string
	err   error
}

func (f *fakeWarehouses) EnsureTenantWarehouse(_ context.Context, s iceberg.TenantWarehouseSpec) (*iceberg.TenantWarehouse, error) {
	f.specs = append(f.specs, s)
	if f.err != nil {
		return nil, f.err
	}
	return &iceberg.TenantWarehouse{ID: f.id}, nil
}

type rig struct {
	acts *activities.TenantLakehouseActivities
	reg  *fakeReg
	keys *fakeKeys
	bkt  *fakeBuckets
	cr   *fakeCreds
	wh   *fakeWarehouses
	in   activities.LakehouseProvisionInput
	id   uuid.UUID
}

func newRig() *rig {
	id := uuid.New()
	days := 2555
	r := &rig{
		reg:  &fakeReg{cfg: &registry.Config{TenantID: id.String(), Configured: true, AuditRetentionDays: &days, LifecycleState: "provisioning"}},
		keys: &fakeKeys{}, bkt: &fakeBuckets{}, cr: &fakeCreds{key: "AKIA-TEST", secret: "s3cr3t-value"},
		wh: &fakeWarehouses{id: uuid.NewString()}, id: id,
		in: activities.LakehouseProvisionInput{TenantID: id.String(), ActorID: "alice", ActorRole: "global_admin"},
	}
	r.acts = &activities.TenantLakehouseActivities{
		Registry: r.reg, Keys: r.keys, Buckets: r.bkt, Credentials: r.cr, Warehouses: r.wh,
		S3Endpoint: "http://minio:9000", S3Region: "us-east-1",
	}
	return r
}

func isNonRetryable(err error) bool {
	var ae *temporal.ApplicationError
	return errors.As(err, &ae) && ae.NonRetryable()
}

func wantName(id uuid.UUID) string {
	n, _ := iceberg.TenantWarehouseName(id)
	return n
}

func TestLoadLakehouseSpec(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the retention for a configured tenant", func(t *testing.T) {
		r := newRig()
		spec, err := r.acts.LoadLakehouseSpec(ctx, r.in)
		require.NoError(t, err)
		require.Equal(t, 2555, spec.RetentionDays)
		require.False(t, spec.AlreadyProvisioned)
	})

	t.Run("an already provisioned tenant is a no-op, not an error", func(t *testing.T) {
		r := newRig()
		r.reg.cfg.Provisioned = true
		spec, err := r.acts.LoadLakehouseSpec(ctx, r.in)
		require.NoError(t, err)
		require.True(t, spec.AlreadyProvisioned)
	})

	for name, mutate := range map[string]func(*rig){
		"no registry row":     func(r *rig) { r.reg.cfg = &registry.Config{LifecycleState: "unconfigured"} },
		"retention not set":   func(r *rig) { r.reg.cfg.AuditRetentionDays = nil },
		"suspended":           func(r *rig) { r.reg.cfg.LifecycleState = "suspended" },
		"offboarding":         func(r *rig) { r.reg.cfg.LifecycleState = "offboarding" },
		"unknown tenant":      func(r *rig) { r.reg.getErr = registry.ErrTenantNotFound },
		"malformed tenant id": func(r *rig) { r.in.TenantID = "nope" },
		"nil tenant id":       func(r *rig) { r.in.TenantID = uuid.Nil.String() },
	} {
		t.Run("is not retried when "+name, func(t *testing.T) {
			r := newRig()
			mutate(r)
			_, err := r.acts.LoadLakehouseSpec(ctx, r.in)
			require.Error(t, err)
			require.True(t, isNonRetryable(err), "a condition a retry cannot fix must not be retried: %v", err)
		})
	}

	t.Run("a registry outage is retried", func(t *testing.T) {
		r := newRig()
		r.reg.getErr = errors.New("connection refused")
		_, err := r.acts.LoadLakehouseSpec(ctx, r.in)
		require.Error(t, err)
		require.False(t, isNonRetryable(err))
	})
}

func TestEnsureLakehouseKey(t *testing.T) {
	r := newRig()
	key, err := r.acts.EnsureLakehouseKey(context.Background(), r.in)
	require.NoError(t, err)
	require.Equal(t, wantName(r.id), key, "the key is named for the tenant's warehouse; the name is derived, never supplied")
	require.Equal(t, []string{key}, r.keys.names)

	r.keys.err = errors.New("kes unavailable")
	_, err = r.acts.EnsureLakehouseKey(context.Background(), r.in)
	require.Error(t, err)
	require.False(t, isNonRetryable(err), "an outage is retried")
}

func TestEnsureLakehouseBucket(t *testing.T) {
	ctx := context.Background()

	t.Run("passes the tenant, key, retention and region through", func(t *testing.T) {
		r := newRig()
		require.NoError(t, r.acts.EnsureLakehouseBucket(ctx, r.in, "the-key", 2555))
		require.Len(t, r.bkt.specs, 1)
		require.Equal(t, iceberg.TenantBucketSpec{TenantID: r.id, KMSKeyID: "the-key", RetentionDays: 2555, Region: "us-east-1"}, r.bkt.specs[0])
	})

	t.Run("a bucket that conflicts with the required settings is not retried", func(t *testing.T) {
		r := newRig()
		r.bkt.err = errors.Join(errors.New("detail"), iceberg.ErrBucketConflict)
		err := r.acts.EnsureLakehouseBucket(ctx, r.in, "k", 365)
		require.Error(t, err)
		require.True(t, isNonRetryable(err))
	})

	t.Run("an outage is retried", func(t *testing.T) {
		r := newRig()
		r.bkt.err = errors.New("minio unreachable")
		err := r.acts.EnsureLakehouseBucket(ctx, r.in, "k", 365)
		require.Error(t, err)
		require.False(t, isNonRetryable(err))
	})

	t.Run("a non-positive retention never reaches the provisioner", func(t *testing.T) {
		r := newRig()
		err := r.acts.EnsureLakehouseBucket(ctx, r.in, "k", 0)
		require.True(t, isNonRetryable(err))
		require.Empty(t, r.bkt.specs)
	})
}

func TestEnsureLakehouseCredential(t *testing.T) {
	r := newRig()
	require.NoError(t, r.acts.EnsureLakehouseCredential(context.Background(), r.in))
	require.Equal(t, []string{wantName(r.id)}, r.cr.ensured, "scoped to the tenant's own derived bucket")

	r.cr.ensureErr = errors.New("admin api down")
	err := r.acts.EnsureLakehouseCredential(context.Background(), r.in)
	require.Error(t, err)
	require.False(t, isNonRetryable(err))
}

func TestEnsureLakehouseWarehouse(t *testing.T) {
	ctx := context.Background()

	t.Run("reads the credential itself and returns only the warehouse id", func(t *testing.T) {
		r := newRig()
		id, err := r.acts.EnsureLakehouseWarehouse(ctx, r.in)
		require.NoError(t, err)
		require.Equal(t, r.wh.id, id)
		require.NotContains(t, id, "s3cr3t", "a credential must never come back from an activity")
		require.Equal(t, []iceberg.TenantWarehouseSpec{{
			TenantID: r.id, Region: "us-east-1", Endpoint: "http://minio:9000", AccessKeyID: "AKIA-TEST", SecretAccessKey: "s3cr3t-value",
		}}, r.wh.specs)
	})

	t.Run("no credential, no warehouse", func(t *testing.T) {
		r := newRig()
		r.cr.readErr = errors.New("secret not found")
		_, err := r.acts.EnsureLakehouseWarehouse(ctx, r.in)
		require.Error(t, err)
		require.Empty(t, r.wh.specs, "the warehouse step must not run without the tenant's own credential")
	})

	t.Run("a Lakekeeper outage is retried", func(t *testing.T) {
		r := newRig()
		r.wh.err = errors.New("lakekeeper 503")
		_, err := r.acts.EnsureLakehouseWarehouse(ctx, r.in)
		require.Error(t, err)
		require.False(t, isNonRetryable(err))
	})
}

func TestMarkLakehouseProvisioned(t *testing.T) {
	ctx := context.Background()
	wh := uuid.NewString()

	t.Run("records the warehouse, key and actor", func(t *testing.T) {
		r := newRig()
		require.NoError(t, r.acts.MarkLakehouseProvisioned(ctx, r.in, wh, "key", 2555))
		require.Len(t, r.reg.marked, 1)
		require.Equal(t, r.id, r.reg.marked[0].tenant)
		require.Equal(t, wh, r.reg.marked[0].warehouse.String())
		require.Equal(t, "key", r.reg.marked[0].key)
		require.Equal(t, 2555, r.reg.marked[0].applied, "records the retention the bucket was created with")
		require.Equal(t, registry.Actor{ID: "alice", Role: "global_admin"}, r.reg.marked[0].actor)
	})

	t.Run("a bad warehouse id is not retried and writes nothing", func(t *testing.T) {
		r := newRig()
		require.True(t, isNonRetryable(r.acts.MarkLakehouseProvisioned(ctx, r.in, "not-a-uuid", "key", 2555)))
		require.Empty(t, r.reg.marked)
	})

	for name, e := range map[string]error{
		"a different warehouse already bound": registry.ErrWarehouseMismatch,
		"tenant not configured":               registry.ErrNotConfigured,
		"wrong lifecycle state":               registry.ErrInvalidState,
	} {
		t.Run("is not retried for "+name, func(t *testing.T) {
			r := newRig()
			r.reg.markErr = e
			require.True(t, isNonRetryable(r.acts.MarkLakehouseProvisioned(ctx, r.in, wh, "key", 2555)))
		})
	}

	t.Run("a database outage is retried", func(t *testing.T) {
		r := newRig()
		r.reg.markErr = errors.New("deadlock")
		err := r.acts.MarkLakehouseProvisioned(ctx, r.in, wh, "key", 2555)
		require.Error(t, err)
		require.False(t, isNonRetryable(err))
	})
}

func TestRecordLakehouseFailure(t *testing.T) {
	r := newRig()
	require.NoError(t, r.acts.RecordLakehouseFailure(context.Background(), r.in, "EnsureLakehouseBucket", "boom"))
	require.Equal(t, []failCall{{r.id, "EnsureLakehouseBucket", "boom", registry.Actor{ID: "alice", Role: "global_admin"}}}, r.reg.failures)
}

// Missing infrastructure configuration cannot be fixed by retrying, so every step that
// reaches infrastructure must fail fast on it, and must still retry a real outage.
func TestInfrastructureNotConfiguredIsNeverRetried(t *testing.T) {
	ctx := context.Background()
	notConfigured := errors.Join(errors.New("KES needs [KES_ENDPOINT KES_API_KEY]"), infra.ErrNotConfigured)

	steps := map[string]func(r *rig) error{
		"key": func(r *rig) error { r.keys.err = notConfigured; _, e := r.acts.EnsureLakehouseKey(ctx, r.in); return e },
		"bucket": func(r *rig) error {
			r.bkt.err = notConfigured
			return r.acts.EnsureLakehouseBucket(ctx, r.in, "k", 365)
		},
		"credential": func(r *rig) error { r.cr.ensureErr = notConfigured; return r.acts.EnsureLakehouseCredential(ctx, r.in) },
		"credential read": func(r *rig) error {
			r.cr.readErr = notConfigured
			_, e := r.acts.EnsureLakehouseWarehouse(ctx, r.in)
			return e
		},
	}
	for name, run := range steps {
		t.Run(name, func(t *testing.T) {
			err := run(newRig())
			require.Error(t, err)
			require.True(t, isNonRetryable(err), "missing configuration must fail fast: %v", err)
			require.ErrorContains(t, err, "KES_ENDPOINT", "and say what is missing")
		})
	}
}

func TestEnsureLakehouseWarehouse_NoEndpointFailsFastWithoutTouchingCredentials(t *testing.T) {
	r := newRig()
	r.acts.S3Endpoint = ""
	_, err := r.acts.EnsureLakehouseWarehouse(context.Background(), r.in)
	require.Error(t, err)
	require.True(t, isNonRetryable(err))
	require.ErrorContains(t, err, "S3_ENDPOINT")
	require.Empty(t, r.wh.specs)
}

// A credential may be minted only while the registry says none was ever issued. After
// that, "not found" is a loss that needs a person, never a prompt to issue a new one.
func TestEnsureLakehouseCredential_MintingFollowsTheRegistryNotTheSecretsStore(t *testing.T) {
	ctx := context.Background()

	t.Run("first issue may mint, and is then recorded", func(t *testing.T) {
		r := newRig()
		require.NoError(t, r.acts.EnsureLakehouseCredential(ctx, r.in))
		require.Equal(t, []bool{true}, r.cr.mayMint)
		require.Equal(t, 1, r.reg.credMarks)
	})

	t.Run("once issued, minting is not allowed and nothing is re-recorded", func(t *testing.T) {
		r := newRig()
		r.reg.cfg.CredentialIssued = true
		require.NoError(t, r.acts.EnsureLakehouseCredential(ctx, r.in))
		require.Equal(t, []bool{false}, r.cr.mayMint)
		require.Zero(t, r.reg.credMarks)
	})

	t.Run("a lost credential is not retried and is not recorded as issued", func(t *testing.T) {
		r := newRig()
		r.reg.cfg.CredentialIssued = true
		r.cr.ensureErr = infra.ErrCredentialLost
		err := r.acts.EnsureLakehouseCredential(ctx, r.in)
		require.Error(t, err)
		require.True(t, isNonRetryable(err), "a person must restore or rotate it: %v", err)
		require.Zero(t, r.reg.credMarks)
	})

	t.Run("a failed issue does not mark the credential issued", func(t *testing.T) {
		r := newRig()
		r.cr.ensureErr = errors.New("minio down")
		require.Error(t, r.acts.EnsureLakehouseCredential(ctx, r.in))
		require.Zero(t, r.reg.credMarks, "marking before the credential exists would block the retry from minting it")
	})

	t.Run("failing to record the issue is retried, and the retry is safe", func(t *testing.T) {
		r := newRig()
		r.reg.credMarkErr = errors.New("db blip")
		err := r.acts.EnsureLakehouseCredential(ctx, r.in)
		require.Error(t, err)
		require.False(t, isNonRetryable(err))
	})

	t.Run("a registry outage is retried before anything is created", func(t *testing.T) {
		r := newRig()
		r.reg.getErr = errors.New("db down")
		err := r.acts.EnsureLakehouseCredential(ctx, r.in)
		require.Error(t, err)
		require.False(t, isNonRetryable(err))
		require.Empty(t, r.cr.ensured)
	})
}

func TestLoadRetentionTarget(t *testing.T) {
	ctx := context.Background()
	provisionedCfg := func(r *rig, desired int, pending bool) {
		r.reg.cfg.Provisioned = true
		r.reg.cfg.AuditRetentionDays = &desired
		r.reg.cfg.RetentionPending = pending
	}

	t.Run("reports what the registry wants and whether the bucket is behind", func(t *testing.T) {
		r := newRig()
		provisionedCfg(r, 3650, true)
		got, err := r.acts.LoadRetentionTarget(ctx, r.in)
		require.NoError(t, err)
		require.Equal(t, activities.RetentionTarget{DesiredDays: 3650, Pending: true}, got)

		provisionedCfg(r, 3650, false)
		got, err = r.acts.LoadRetentionTarget(ctx, r.in)
		require.NoError(t, err)
		require.False(t, got.Pending)
	})

	for name, mutate := range map[string]func(*rig){
		"not provisioned":  func(r *rig) { provisionedCfg(r, 365, true); r.reg.cfg.Provisioned = false },
		"no retention":     func(r *rig) { provisionedCfg(r, 365, true); r.reg.cfg.AuditRetentionDays = nil },
		"unknown tenant":   func(r *rig) { r.reg.getErr = registry.ErrTenantNotFound },
		"malformed tenant": func(r *rig) { r.in.TenantID = "nope" },
	} {
		t.Run("is not retried when "+name, func(t *testing.T) {
			r := newRig()
			mutate(r)
			_, err := r.acts.LoadRetentionTarget(ctx, r.in)
			require.Error(t, err)
			require.True(t, isNonRetryable(err), "%v", err)
		})
	}

	t.Run("a registry outage is retried", func(t *testing.T) {
		r := newRig()
		r.reg.getErr = errors.New("db down")
		_, err := r.acts.LoadRetentionTarget(ctx, r.in)
		require.Error(t, err)
		require.False(t, isNonRetryable(err))
	})
}

func TestExtendBucketRetention(t *testing.T) {
	ctx := context.Background()

	t.Run("passes the tenant and days through and returns what the bucket now enforces", func(t *testing.T) {
		r := newRig()
		got, err := r.acts.ExtendBucketRetention(ctx, r.in, 2555)
		require.NoError(t, err)
		require.Equal(t, 2555, got)
		require.Equal(t, []uint{2555}, r.bkt.extended)
	})

	t.Run("a bucket that is not compliance-locked is not retried", func(t *testing.T) {
		r := newRig()
		r.bkt.extErr = errors.Join(errors.New("governance"), iceberg.ErrBucketConflict)
		_, err := r.acts.ExtendBucketRetention(ctx, r.in, 2555)
		require.True(t, isNonRetryable(err))
	})

	t.Run("an outage is retried and missing configuration is not", func(t *testing.T) {
		r := newRig()
		r.bkt.extErr = errors.New("minio unreachable")
		_, err := r.acts.ExtendBucketRetention(ctx, r.in, 2555)
		require.Error(t, err)
		require.False(t, isNonRetryable(err))

		r.bkt.extErr = errors.Join(errors.New("MinIO needs [LAKEHOUSE_MINIO_ENDPOINT]"), infra.ErrNotConfigured)
		_, err = r.acts.ExtendBucketRetention(ctx, r.in, 2555)
		require.True(t, isNonRetryable(err))
	})

	t.Run("a non-positive retention never reaches the bucket", func(t *testing.T) {
		r := newRig()
		_, err := r.acts.ExtendBucketRetention(ctx, r.in, 0)
		require.True(t, isNonRetryable(err))
		require.Empty(t, r.bkt.extended)
	})
}

func TestMarkRetentionApplied(t *testing.T) {
	ctx := context.Background()
	r := newRig()
	require.NoError(t, r.acts.MarkRetentionApplied(ctx, r.in, 2555))
	require.Equal(t, []int{2555}, r.reg.applied)

	for name, e := range map[string]error{
		"not provisioned":    registry.ErrInvalidState,
		"not configured":     registry.ErrNotConfigured,
		"exceeds the wanted": registry.ErrInvalidRetention,
	} {
		r := newRig()
		r.reg.appliedErr = e
		require.True(t, isNonRetryable(r.acts.MarkRetentionApplied(ctx, r.in, 2555)), name)
	}
	r2 := newRig()
	r2.reg.appliedErr = errors.New("deadlock")
	err := r2.acts.MarkRetentionApplied(ctx, r2.in, 2555)
	require.Error(t, err)
	require.False(t, isNonRetryable(err))

	r3 := newRig()
	require.NoError(t, r3.acts.RecordRetentionSyncFailure(ctx, r3.in, "ExtendBucketRetention", "boom"))
	require.Equal(t, []failCall{{r3.id, "ExtendBucketRetention", "boom", registry.Actor{ID: "alice", Role: "global_admin"}}}, r3.reg.syncFails)
}

func (f *fakeReg) AuditAfter(context.Context, uuid.UUID, int64, int) ([]registry.AuditEntry, error) {
	return nil, nil
}
func (f *fakeReg) MarkAuditCopied(context.Context, uuid.UUID, int64) error { return nil }
func (f *fakeReg) RecordAuditVerification(context.Context, uuid.UUID, registry.AuditVerification) error {
	return nil
}
func (f *fakeReg) VerifyAudit(context.Context, uuid.UUID) (*int64, error)  { return nil, nil }
func (f *fakeReg) ProvisionedTenants(context.Context) ([]uuid.UUID, error) { return nil, nil }
