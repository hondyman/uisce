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
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// These run the REAL platform activities and workflow against in-memory infrastructure. What matters: the
// credential is never minted twice, a bucket that cannot be reconciled stops the run, nothing is recorded that
// the bucket does not enforce, and no secret reaches workflow history.

type platformFakes struct {
	mu    sync.Mutex
	order []string

	cfg *registry.PlatformConfig // nil = not configured

	failOnce   map[string]int   // step -> times it fails (retryably) before succeeding
	failAlways map[string]error // step -> permanent error
	credLost   bool
	minted     int
	whID       uuid.UUID
}

func newPlatformFakes() *platformFakes {
	return &platformFakes{failOnce: map[string]int{}, failAlways: map[string]error{}, whID: uuid.New()}
}

func (f *platformFakes) step(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.order = append(f.order, name)
	if err := f.failAlways[name]; err != nil {
		return err
	}
	if f.failOnce[name] > 0 {
		f.failOnce[name]--
		return errors.New(name + ": transient outage")
	}
	return nil
}

// registry
func (f *platformFakes) Get(context.Context) (*registry.PlatformConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cfg == nil {
		return nil, registry.ErrPlatformNotConfigured
	}
	c := *f.cfg
	return &c, nil
}
func (f *platformFakes) ConfigureRetention(_ context.Context, days int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.order = append(f.order, "ConfigureRetention")
	if f.cfg != nil && f.cfg.AuditRetentionDays > days {
		return registry.ErrRetentionLowered
	}
	if f.cfg == nil {
		f.cfg = &registry.PlatformConfig{}
	}
	f.cfg.AuditRetentionDays = days
	return nil
}
func (f *platformFakes) MarkCredentialIssued(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.order = append(f.order, "MarkCredentialIssued")
	f.cfg.CredentialIssued = true
	return nil
}
func (f *platformFakes) MarkProvisioned(_ context.Context, wh uuid.UUID, kms string, applied int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.order = append(f.order, "MarkProvisioned")
	f.cfg.Provisioned, f.cfg.WarehouseID, f.cfg.KMSKeyID, f.cfg.RetentionAppliedDays = true, &wh, &kms, &applied
	return nil
}

// infrastructure
func (f *platformFakes) EnsureKey(_ context.Context, name string) error {
	if name != iceberg.ControlWarehouseName {
		return errors.New("the platform key must be named for ivy-control")
	}
	return f.step("EnsureKey")
}
func (f *platformFakes) EnsureControlBucket(_ context.Context, s iceberg.ControlBucketSpec) (*iceberg.TenantBucket, error) {
	if err := f.step("EnsureControlBucket"); err != nil {
		return nil, err
	}
	return &iceberg.TenantBucket{Name: iceberg.ControlWarehouseName, RetentionDays: s.RetentionDays}, nil
}
func (f *platformFakes) EnsurePlatformCredential(_ context.Context, mayMint bool) error {
	if err := f.step("EnsurePlatformCredential"); err != nil {
		return err
	}
	if f.credLost && !mayMint {
		return infra.ErrCredentialLost
	}
	if mayMint {
		f.mu.Lock()
		f.minted++
		f.mu.Unlock()
	}
	return nil
}
func (f *platformFakes) ReadPlatform(context.Context) (string, string, error) {
	return "AKIA-PLATFORM", secretValue, f.step("ReadPlatform")
}
func (f *platformFakes) EnsureControlWarehouse(_ context.Context, s iceberg.ControlWarehouseSpec) (*iceberg.TenantWarehouse, error) {
	if err := f.step("EnsureControlWarehouse"); err != nil {
		return nil, err
	}
	if s.AccessKeyID != "AKIA-PLATFORM" || s.SecretAccessKey != secretValue {
		return nil, errors.New("the warehouse was not given the platform credential")
	}
	return &iceberg.TenantWarehouse{ID: f.whID.String(), Name: iceberg.ControlWarehouseName, Bucket: iceberg.ControlWarehouseName}, nil
}

func (f *platformFakes) run(t *testing.T, in activities.PlatformLakehouseInput) (*workflows.PlatformLakehouseProvisionResult, error, *recordingConverter) {
	t.Helper()
	rc := &recordingConverter{DataConverter: converter.GetDefaultDataConverter()}
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	env.SetDataConverter(rc)
	env.RegisterActivity(&activities.PlatformLakehouseActivities{
		Registry: f, Keys: f, Buckets: f, Credentials: f, Warehouses: f, S3Endpoint: "http://minio:9000", S3Region: "us-east-1",
	})
	env.RegisterWorkflow(workflows.PlatformLakehouseProvisioningWorkflow)
	env.ExecuteWorkflow(workflows.PlatformLakehouseProvisioningWorkflow, in)
	require.True(t, env.IsWorkflowCompleted())
	if err := env.GetWorkflowError(); err != nil {
		return nil, err, rc
	}
	var res workflows.PlatformLakehouseProvisionResult
	require.NoError(t, env.GetWorkflowResult(&res))
	return &res, nil, rc
}

func TestPlatformLakehouse_ProvisionsInOrderAndKeepsTheSecretOutOfHistory(t *testing.T) {
	f := newPlatformFakes()
	res, err, rc := f.run(t, activities.PlatformLakehouseInput{RetentionDays: 2555, ActorID: "admin"})
	require.NoError(t, err)
	require.Equal(t, f.whID.String(), res.WarehouseID)
	require.Equal(t, []string{"ConfigureRetention", "EnsureKey", "EnsureControlBucket", "EnsurePlatformCredential", "MarkCredentialIssued",
		"ReadPlatform", "EnsureControlWarehouse", "MarkProvisioned"}, f.order)
	require.True(t, f.cfg.Provisioned)
	require.Equal(t, 2555, *f.cfg.RetentionAppliedDays)
	require.Equal(t, 1, f.minted)
	for _, p := range rc.seen {
		require.False(t, strings.Contains(p, secretValue), "a credential reached workflow history: %s", p)
	}
}

func TestPlatformLakehouse_ARerunDoesNothing(t *testing.T) {
	f := newPlatformFakes()
	_, err, _ := f.run(t, activities.PlatformLakehouseInput{RetentionDays: 365})
	require.NoError(t, err)
	f.order = nil
	res, err, _ := f.run(t, activities.PlatformLakehouseInput{RetentionDays: 365})
	require.NoError(t, err)
	require.True(t, res.AlreadyProvisioned)
	require.Empty(t, f.order, "no infrastructure is touched, and nothing is written")
	require.Equal(t, 1, f.minted)
}

// A transient outage mid-run resumes without minting a second credential.
func TestPlatformLakehouse_ResumesAfterAnOutageWithoutMintingAgain(t *testing.T) {
	f := newPlatformFakes()
	f.failOnce["EnsureControlWarehouse"] = 2
	res, err, _ := f.run(t, activities.PlatformLakehouseInput{RetentionDays: 365})
	require.NoError(t, err)
	require.Equal(t, f.whID.String(), res.WarehouseID)
	require.Equal(t, 1, f.minted)
	require.Equal(t, 1, strings.Count(strings.Join(f.order, ","), "MarkCredentialIssued"))
}

// The registry says a credential was issued, and the secrets store cannot produce it: never mint another.
func TestPlatformLakehouse_ALostCredentialIsNeverReplaced(t *testing.T) {
	f := newPlatformFakes()
	f.cfg = &registry.PlatformConfig{AuditRetentionDays: 365, CredentialIssued: true}
	f.credLost = true
	_, err, _ := f.run(t, activities.PlatformLakehouseInput{RetentionDays: 365})
	require.Error(t, err)
	var app *temporal.ApplicationError
	require.ErrorAs(t, err, &app)
	require.True(t, app.NonRetryable(), "it needs a person")
	require.Zero(t, f.minted)
	require.NotContains(t, f.order, "EnsureControlWarehouse")
	require.False(t, f.cfg.Provisioned)
}

func TestPlatformLakehouse_ABucketThatConflictsStopsTheRunNonRetryably(t *testing.T) {
	f := newPlatformFakes()
	f.failAlways["EnsureControlBucket"] = iceberg.ErrBucketConflict
	_, err, _ := f.run(t, activities.PlatformLakehouseInput{RetentionDays: 365})
	require.Error(t, err)
	require.Equal(t, 1, strings.Count(strings.Join(f.order, ","), "EnsureControlBucket"), "not retried")
	require.NotContains(t, f.order, "EnsurePlatformCredential", "nothing proceeds past a bucket that cannot be reconciled")
	require.False(t, f.cfg.Provisioned)
}

func TestPlatformLakehouse_RetentionIsRequiredAndNeverLowered(t *testing.T) {
	f := newPlatformFakes()
	for _, days := range []int{0, -5, registry.MaxRetentionDays + 1} {
		_, err, _ := f.run(t, activities.PlatformLakehouseInput{RetentionDays: days})
		require.Error(t, err)
	}
	require.Empty(t, f.order, "nothing is touched without a valid explicit retention")
	require.Nil(t, f.cfg)

	f.cfg = &registry.PlatformConfig{AuditRetentionDays: 2555}
	_, err, _ := f.run(t, activities.PlatformLakehouseInput{RetentionDays: 365})
	require.Error(t, err, "a lower number than the one recorded is refused, never ignored")
	require.NotContains(t, f.order, "EnsureControlBucket")
	require.Equal(t, 2555, f.cfg.AuditRetentionDays)
}

// A record must not claim retention the bucket does not enforce.
func TestPlatformLakehouse_RaisingAProvisionedBucketsRetentionIsRefusedAndNothingIsWritten(t *testing.T) {
	f := newPlatformFakes()
	_, err, _ := f.run(t, activities.PlatformLakehouseInput{RetentionDays: 365})
	require.NoError(t, err)
	f.order = nil
	_, err, _ = f.run(t, activities.PlatformLakehouseInput{RetentionDays: 730})
	require.Error(t, err)
	require.Empty(t, f.order, "not even the registry is written")
	require.Equal(t, 365, f.cfg.AuditRetentionDays)
}

func TestPlatformLakehouse_MissingInfrastructureFailsFastAndReadably(t *testing.T) {
	f := newPlatformFakes()
	f.failAlways["EnsureKey"] = infra.ErrNotConfigured
	_, err, _ := f.run(t, activities.PlatformLakehouseInput{RetentionDays: 365})
	require.ErrorContains(t, err, "not configured")
	require.Equal(t, 1, strings.Count(strings.Join(f.order, ","), "EnsureKey"), "a missing configuration is not retried")
}
