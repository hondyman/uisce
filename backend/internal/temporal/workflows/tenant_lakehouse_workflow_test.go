package workflows_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
)

// These run the REAL lakehouse activities against fake dependencies, so they cover the
// workflow, the activities and their retry classification together. A recording data
// converter captures every payload Temporal would write to workflow history, to prove
// a credential never enters it.

const secretValue = "s3cr3t-value-must-never-be-in-history"

type recordingConverter struct {
	converter.DataConverter
	mu   sync.Mutex
	seen []string
}

func (c *recordingConverter) note(p ...*commonpb.Payload) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, x := range p {
		if x != nil {
			c.seen = append(c.seen, string(x.GetData()))
		}
	}
}
func (c *recordingConverter) ToPayload(v interface{}) (*commonpb.Payload, error) {
	p, err := c.DataConverter.ToPayload(v)
	c.note(p)
	return p, err
}
func (c *recordingConverter) ToPayloads(v ...interface{}) (*commonpb.Payloads, error) {
	p, err := c.DataConverter.ToPayloads(v...)
	if p != nil {
		c.note(p.GetPayloads()...)
	}
	return p, err
}

type lhFakes struct {
	mu                 sync.Mutex
	order              []string
	failAt             map[string]int // step -> number of times it fails before succeeding (-1 = always)
	calls              map[string]int
	cfg                *registry.Config
	marked             []uuid.UUID
	fails              []string // "step: reason"
	appliedAtProvision int
	appliedDays        []int
	extendedTo         []uint
	syncFails          []string
	// errors injected per step
	errFor        map[string]error
	failRecordErr error
}

func (f *lhFakes) hit(step string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.order = append(f.order, step)
	f.calls[step]++
	if e, ok := f.errFor[step]; ok {
		n := f.failAt[step]
		if n == -1 || f.calls[step] <= n {
			return e
		}
	}
	return nil
}

func (f *lhFakes) Get(context.Context, uuid.UUID) (*registry.Config, error) {
	if err := f.hit("Get"); err != nil {
		return nil, err
	}
	return f.cfg, nil
}
func (f *lhFakes) MarkProvisioned(_ context.Context, t, _ uuid.UUID, _ string, applied int, _ registry.Actor) error {
	if err := f.hit("MarkProvisioned"); err != nil {
		return err
	}
	f.marked = append(f.marked, t)
	f.appliedAtProvision = applied
	return nil
}
func (f *lhFakes) MarkRetentionApplied(_ context.Context, _ uuid.UUID, days int, _ registry.Actor) error {
	if err := f.hit("MarkRetentionApplied"); err != nil {
		return err
	}
	f.appliedDays = append(f.appliedDays, days)
	return nil
}
func (f *lhFakes) RecordRetentionSyncFailure(_ context.Context, _ uuid.UUID, _ registry.Actor, step, reason string) error {
	f.mu.Lock()
	f.syncFails = append(f.syncFails, step+": "+reason)
	f.mu.Unlock()
	return f.failRecordErr
}
func (f *lhFakes) ExtendTenantRetention(_ context.Context, _ uuid.UUID, days uint) (uint, error) {
	if err := f.hit("ExtendTenantRetention"); err != nil {
		return 0, err
	}
	f.extendedTo = append(f.extendedTo, days)
	return days, nil
}
func (f *lhFakes) RecordProvisionFailure(_ context.Context, _ uuid.UUID, _ registry.Actor, step, reason string) error {
	f.mu.Lock()
	f.fails = append(f.fails, step+": "+reason)
	f.mu.Unlock()
	return f.failRecordErr
}
func (f *lhFakes) EnsureKey(context.Context, string) error { return f.hit("EnsureKey") }
func (f *lhFakes) EnsureTenantBucket(context.Context, iceberg.TenantBucketSpec) (*iceberg.TenantBucket, error) {
	return &iceberg.TenantBucket{}, f.hit("EnsureTenantBucket")
}
func (f *lhFakes) EnsureBucketCredential(context.Context, uuid.UUID, string, bool) error {
	return f.hit("EnsureBucketCredential")
}
func (f *lhFakes) MarkCredentialIssued(context.Context, uuid.UUID) error {
	return f.hit("MarkCredentialIssued")
}
func (f *lhFakes) Read(context.Context, uuid.UUID) (string, string, error) {
	return "AKIA-TEST", secretValue, f.hit("ReadCredential")
}
func (f *lhFakes) EnsureTenantWarehouse(_ context.Context, s iceberg.TenantWarehouseSpec) (*iceberg.TenantWarehouse, error) {
	if s.SecretAccessKey != secretValue {
		return nil, errors.New("the warehouse step did not receive the stored credential")
	}
	return &iceberg.TenantWarehouse{ID: warehouseID}, f.hit("EnsureTenantWarehouse")
}

var warehouseID = uuid.NewString()

func newLH() (*lhFakes, uuid.UUID) {
	id := uuid.New()
	days := 2555
	return &lhFakes{
		failAt: map[string]int{}, calls: map[string]int{}, errFor: map[string]error{},
		cfg: &registry.Config{TenantID: id.String(), Configured: true, AuditRetentionDays: &days, LifecycleState: "provisioning"},
	}, id
}

type lhResult struct {
	err    error
	result workflows.LakehouseProvisionResult
	rc     *recordingConverter
}

func runLakehouse(t *testing.T, f *lhFakes, id uuid.UUID) lhResult {
	t.Helper()
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	rc := &recordingConverter{DataConverter: converter.GetDefaultDataConverter()}
	env.SetDataConverter(rc)
	acts := &activities.TenantLakehouseActivities{
		Registry: f, Keys: f, Buckets: f, Credentials: f, Warehouses: f, S3Endpoint: "http://minio:9000", S3Region: "us-east-1",
	}
	env.RegisterActivity(acts)
	env.ExecuteWorkflow(workflows.TenantLakehouseProvisioningWorkflow, activities.LakehouseProvisionInput{
		TenantID: id.String(), ActorID: "alice", ActorRole: "global_admin",
	})
	require.True(t, env.IsWorkflowCompleted())
	out := lhResult{err: env.GetWorkflowError(), rc: rc}
	if out.err == nil {
		require.NoError(t, env.GetWorkflowResult(&out.result))
	}
	return out
}

func TestLakehouse_HappyPath(t *testing.T) {
	f, id := newLH()
	r := runLakehouse(t, f, id)
	require.NoError(t, r.err)

	require.Equal(t, []string{"Get", "EnsureKey", "EnsureTenantBucket", "Get", "EnsureBucketCredential", "MarkCredentialIssued", "ReadCredential", "EnsureTenantWarehouse", "MarkProvisioned"}, f.order,
		"key, then bucket, then credential, then warehouse, then the registry")
	require.Equal(t, []uuid.UUID{id}, f.marked)
	require.Equal(t, 2555, f.appliedAtProvision, "the registry records the retention the bucket was created with")
	require.Equal(t, warehouseID, r.result.WarehouseID)
	require.False(t, r.result.AlreadyProvisioned)
	require.Empty(t, f.fails)

	require.NotEmpty(t, r.rc.seen, "the recorder must actually be capturing payloads")
	for _, p := range r.rc.seen {
		require.False(t, strings.Contains(p, secretValue), "a credential reached workflow history: %s", p)
	}
}

func TestLakehouse_AlreadyProvisionedTouchesNothing(t *testing.T) {
	f, id := newLH()
	f.cfg.Provisioned = true
	r := runLakehouse(t, f, id)
	require.NoError(t, r.err)
	require.True(t, r.result.AlreadyProvisioned)
	require.Equal(t, []string{"Get"}, f.order, "no infrastructure call for a tenant that is done")
}

func TestLakehouse_NotConfiguredFailsFastAndIsRecorded(t *testing.T) {
	f, id := newLH()
	f.cfg.AuditRetentionDays = nil
	r := runLakehouse(t, f, id)
	require.Error(t, r.err)
	require.Equal(t, []string{"Get"}, f.order, "retention missing is not retried and nothing is created")
	require.Len(t, f.fails, 1)
	require.True(t, strings.HasPrefix(f.fails[0], "LoadLakehouseSpec: "), f.fails[0])
}

// A failing step stops the run, nothing after it runs, nothing is undone, and the
// failure is written to the audit trail with the step name.
func TestLakehouse_FailureStopsThereAndIsRecorded(t *testing.T) {
	steps := []struct{ fake, step string }{
		{"EnsureKey", "EnsureLakehouseKey"},
		{"EnsureTenantBucket", "EnsureLakehouseBucket"},
		{"EnsureBucketCredential", "EnsureLakehouseCredential"},
		{"EnsureTenantWarehouse", "EnsureLakehouseWarehouse"},
		{"MarkProvisioned", "MarkLakehouseProvisioned"},
	}
	full := []string{"Get", "EnsureKey", "EnsureTenantBucket", "EnsureBucketCredential", "MarkCredentialIssued", "ReadCredential", "EnsureTenantWarehouse", "MarkProvisioned"}
	for _, s := range steps {
		t.Run(s.step, func(t *testing.T) {
			f, id := newLH()
			f.errFor[s.fake], f.failAt[s.fake] = errors.New("infrastructure down"), -1
			r := runLakehouse(t, f, id)
			require.Error(t, r.err)

			// Everything up to and including the failing call ran; nothing after it.
			upTo := 0
			for i, name := range full {
				if name == s.fake {
					upTo = i
				}
			}
			seen := map[string]bool{}
			for _, o := range f.order {
				seen[o] = true
			}
			for _, name := range full[upTo+1:] {
				require.False(t, seen[name], "%s must not run after %s failed", name, s.fake)
			}
			require.Empty(t, f.marked, "a failed run never marks the tenant provisioned")
			require.Len(t, f.fails, 1)
			require.True(t, strings.HasPrefix(f.fails[0], s.step+": "), "the failure names its step: %s", f.fails[0])
			require.Equal(t, 5, f.calls[s.fake], "a transient failure is retried up to the policy limit")
		})
	}
}

func TestLakehouse_BucketConflictIsNotRetried(t *testing.T) {
	f, id := newLH()
	f.errFor["EnsureTenantBucket"], f.failAt["EnsureTenantBucket"] = errors.Join(errors.New("no object lock"), iceberg.ErrBucketConflict), -1
	r := runLakehouse(t, f, id)
	require.Error(t, r.err)
	require.Equal(t, 1, f.calls["EnsureTenantBucket"], "a conflict needs a person; retrying cannot help")
	require.NotContains(t, f.order, "EnsureBucketCredential")
	require.Len(t, f.fails, 1)
}

func TestLakehouse_TransientFailureThenSuccessCompletes(t *testing.T) {
	f, id := newLH()
	f.errFor["EnsureKey"], f.failAt["EnsureKey"] = errors.New("kes timeout"), 2 // fails twice, then works
	r := runLakehouse(t, f, id)
	require.NoError(t, r.err)
	require.Equal(t, 3, f.calls["EnsureKey"])
	require.Equal(t, []uuid.UUID{id}, f.marked)
	require.Empty(t, f.fails, "a run that recovers records no failure")
}

func TestLakehouse_FailingToRecordDoesNotHideTheRealError(t *testing.T) {
	f, id := newLH()
	f.errFor["EnsureKey"], f.failAt["EnsureKey"] = errors.New("kes is down"), -1
	f.failRecordErr = errors.New("audit write failed")
	r := runLakehouse(t, f, id)
	require.Error(t, r.err)
	require.Contains(t, r.err.Error(), "kes is down", "the original failure is what surfaces")
	require.NotContains(t, r.err.Error(), "audit write failed")
}

// The no-secrets assertion is only worth something if the recorder sees what Temporal
// would store, so prove it does: a secret put through the converter is captured.
func TestLakehouse_RecorderWouldCatchALeak(t *testing.T) {
	rc := &recordingConverter{DataConverter: converter.GetDefaultDataConverter()}
	_, err := rc.ToPayloads("prefix-" + secretValue)
	require.NoError(t, err)
	leaked := false
	for _, p := range rc.seen {
		leaked = leaked || strings.Contains(p, secretValue)
	}
	require.True(t, leaked, "the recorder failed to capture a payload that carries a secret")
}

// The case the registry marker exists for: the credential was issued, the secrets store
// cannot produce it. The run must stop at once, not retry, not mint, and never reach the
// warehouse step with a different credential.
func TestLakehouse_LostCredentialStopsTheRunAndNeverReachesTheWarehouse(t *testing.T) {
	f, id := newLH()
	f.cfg.CredentialIssued = true
	f.errFor["EnsureBucketCredential"], f.failAt["EnsureBucketCredential"] = infra.ErrCredentialLost, -1
	r := runLakehouse(t, f, id)

	require.Error(t, r.err)
	require.Equal(t, 1, f.calls["EnsureBucketCredential"], "a lost credential needs a person; retrying cannot help")
	require.Zero(t, f.calls["MarkCredentialIssued"], "and it is not re-recorded")
	require.Zero(t, f.calls["EnsureTenantWarehouse"], "the warehouse must not be touched")
	require.Empty(t, f.marked)
	require.Len(t, f.fails, 1)
	require.True(t, strings.HasPrefix(f.fails[0], "EnsureLakehouseCredential: "), f.fails[0])
}

// ---- retention reconcile ----

func runRetention(t *testing.T, f *lhFakes, id uuid.UUID) (error, workflows.LakehouseRetentionResult) {
	t.Helper()
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	acts := &activities.TenantLakehouseActivities{
		Registry: f, Keys: f, Buckets: f, Credentials: f, Warehouses: f, S3Endpoint: "http://minio:9000", S3Region: "us-east-1",
	}
	env.RegisterActivity(acts)
	env.ExecuteWorkflow(workflows.TenantLakehouseRetentionWorkflow, activities.LakehouseProvisionInput{TenantID: id.String(), ActorID: "alice", ActorRole: "global_admin"})
	require.True(t, env.IsWorkflowCompleted())
	var res workflows.LakehouseRetentionResult
	if err := env.GetWorkflowError(); err != nil {
		return err, res
	}
	require.NoError(t, env.GetWorkflowResult(&res))
	return nil, res
}

func pendingTenant(desired int) (*lhFakes, uuid.UUID) {
	f, id := newLH()
	f.cfg.Provisioned, f.cfg.RetentionPending = true, true
	f.cfg.AuditRetentionDays = &desired
	return f, id
}

func TestRetention_RaisesTheBucketThenRecordsIt(t *testing.T) {
	f, id := pendingTenant(3650)
	err, res := runRetention(t, f, id)
	require.NoError(t, err)
	require.Equal(t, []string{"Get", "ExtendTenantRetention", "MarkRetentionApplied"}, f.order, "raise the bucket first, record it only after it is raised")
	require.Equal(t, []uint{3650}, f.extendedTo)
	require.Equal(t, []int{3650}, f.appliedDays)
	require.Equal(t, 3650, res.AppliedDays)
	require.False(t, res.AlreadyCurrent)
	require.Empty(t, f.syncFails)
}

func TestRetention_AlreadyCurrentTouchesNothing(t *testing.T) {
	f, id := pendingTenant(3650)
	f.cfg.RetentionPending = false
	err, res := runRetention(t, f, id)
	require.NoError(t, err)
	require.True(t, res.AlreadyCurrent)
	require.Equal(t, []string{"Get"}, f.order)
}

func TestRetention_NotProvisionedFailsFastAndIsRecorded(t *testing.T) {
	f, id := pendingTenant(365)
	f.cfg.Provisioned = false
	err, _ := runRetention(t, f, id)
	require.Error(t, err)
	require.Equal(t, []string{"Get"}, f.order, "no bucket call for a tenant with no bucket")
	require.Len(t, f.syncFails, 1)
	require.True(t, strings.HasPrefix(f.syncFails[0], "LoadRetentionTarget: "), f.syncFails[0])
	require.Empty(t, f.fails, "a retention failure is not recorded as a provisioning failure")
}

func TestRetention_BucketConflictIsNotRetriedAndTheRegistryIsNotTold(t *testing.T) {
	f, id := pendingTenant(3650)
	f.errFor["ExtendTenantRetention"], f.failAt["ExtendTenantRetention"] = errors.Join(errors.New("governance mode"), iceberg.ErrBucketConflict), -1
	err, _ := runRetention(t, f, id)
	require.Error(t, err)
	require.Equal(t, 1, f.calls["ExtendTenantRetention"])
	require.Zero(t, f.calls["MarkRetentionApplied"], "the registry must never record what the bucket does not enforce")
	require.Empty(t, f.appliedDays)
	require.Len(t, f.syncFails, 1)
	require.True(t, strings.HasPrefix(f.syncFails[0], "ExtendBucketRetention: "), f.syncFails[0])
}

func TestRetention_TransientFailureThenSuccessCompletes(t *testing.T) {
	f, id := pendingTenant(3650)
	f.errFor["ExtendTenantRetention"], f.failAt["ExtendTenantRetention"] = errors.New("minio timeout"), 2
	err, res := runRetention(t, f, id)
	require.NoError(t, err)
	require.Equal(t, 3, f.calls["ExtendTenantRetention"])
	require.Equal(t, 3650, res.AppliedDays)
	require.Empty(t, f.syncFails)
}

func TestRetention_FailingToRecordTheRegistryIsRetriedAndSafe(t *testing.T) {
	f, id := pendingTenant(3650)
	f.errFor["MarkRetentionApplied"], f.failAt["MarkRetentionApplied"] = errors.New("db blip"), 1
	err, res := runRetention(t, f, id)
	require.NoError(t, err)
	require.Equal(t, 2, f.calls["MarkRetentionApplied"])
	require.Equal(t, []uint{3650}, f.extendedTo, "the bucket was raised once; the retry only re-recorded it")
	require.Equal(t, 3650, res.AppliedDays)
}

func (f *lhFakes) AuditAfter(context.Context, uuid.UUID, int64, int) ([]registry.AuditEntry, error) {
	return nil, nil
}
func (f *lhFakes) MarkAuditCopied(context.Context, uuid.UUID, int64) error { return nil }
func (f *lhFakes) RecordAuditVerification(context.Context, uuid.UUID, registry.AuditVerification) error {
	return nil
}
func (f *lhFakes) VerifyAudit(context.Context, uuid.UUID) (*int64, error)  { return nil, nil }
func (f *lhFakes) ProvisionedTenants(context.Context) ([]uuid.UUID, error) { return nil, nil }
