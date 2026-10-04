package workflows_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
)

// These run the REAL copy activities against an in-memory Iceberg destination. The case that matters
// is an append whose commit succeeds and whose reply is lost: the copy must come out exactly right
// because it resumes from the destination, not from a counter.

var genesis = strings.Repeat("0", 64)

// chain builds n entries (ids 1..n) linked the way the audit trigger links them.
func chain(n int) []registry.AuditEntry {
	out := make([]registry.AuditEntry, n)
	prev := genesis
	for i := range out {
		id := int64(i + 1)
		out[i] = registry.AuditEntry{ID: id, At: time.Date(2026, 10, 3, 0, 0, int(id%60), 0, time.UTC), ActorID: "alice", ActorRole: "global_admin",
			Action: "state_changed", After: []byte(fmt.Sprintf(`{"n":%d}`, id)), PrevHash: prev, Hash: fmt.Sprintf("h%d", id)}
		prev = out[i].Hash
	}
	return out
}

// memDest is one tenant's Iceberg audit table.
type memDest struct {
	mu            sync.Mutex
	rows          []infra.AuditRow
	ensured       int
	appends       []int // size of each append
	hashCalls     int
	storeThenFail int // the next N appends STORE the rows and then return an error (an unknown outcome)
	appendErr     error
	ensureErr     error
	hideHashes    bool // AuditHash reports every entry as absent, though MaxAuditID reports them
	dropSilently  bool // AppendAudit reports success but stores nothing
}

func (d *memDest) EnsureAuditDestination(context.Context, infra.AuditDestinationSpec) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensured++
	return d.ensureErr
}
func (d *memDest) MaxAuditID(context.Context, uuid.UUID) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.rows) == 0 {
		return 0, nil
	}
	return d.rows[len(d.rows)-1].ID, nil
}
func (d *memDest) AuditHash(_ context.Context, _ uuid.UUID, id int64) (string, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hashCalls++
	if d.hideHashes {
		return "", false, nil
	}
	for _, r := range d.rows {
		if r.ID == id {
			return r.Hash, true, nil
		}
	}
	return "", false, nil
}
func (d *memDest) AppendAudit(_ context.Context, _ uuid.UUID, rows []infra.AuditRow) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.appends = append(d.appends, len(rows))
	if d.appendErr != nil {
		return d.appendErr
	}
	if d.dropSilently {
		return nil
	}
	d.rows = append(d.rows, rows...)
	if d.storeThenFail > 0 {
		d.storeThenFail--
		return errors.New("connection reset after commit")
	}
	return nil
}
func (d *memDest) AuditRange(_ context.Context, _ uuid.UUID, after int64, limit int) ([]infra.AuditRow, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []infra.AuditRow
	for _, r := range d.rows {
		if r.ID > after && len(out) < limit {
			out = append(out, r)
		}
	}
	return out, nil
}
func (d *memDest) ids() []int64 {
	var out []int64
	for _, r := range d.rows {
		out = append(out, r.ID)
	}
	return out
}

// copyReg serves audit entries per tenant on top of the shared lakehouse fakes.
type copyReg struct {
	*lhFakes
	mu      sync.Mutex
	entries map[uuid.UUID][]registry.AuditEntry
	cfgs    map[uuid.UUID]*registry.Config
	copied  map[uuid.UUID]int64
	tenants []uuid.UUID
	// brokenAlpha makes alpha's own chain fail to recompute at the given id.
	brokenAlpha map[uuid.UUID]int64
}

func (r *copyReg) VerifyAudit(_ context.Context, id uuid.UUID) (*int64, error) {
	if n, ok := r.brokenAlpha[id]; ok {
		return &n, nil
	}
	return nil, nil
}

func (r *copyReg) Get(_ context.Context, id uuid.UUID) (*registry.Config, error) {
	if c, ok := r.cfgs[id]; ok {
		cp := *c
		return &cp, nil
	}
	cp := *r.lhFakes.cfg
	cp.TenantID = id.String()
	return &cp, nil
}
func (r *copyReg) AuditAfter(_ context.Context, id uuid.UUID, after int64, limit int) ([]registry.AuditEntry, error) {
	var out []registry.AuditEntry
	for _, e := range r.entries[id] {
		if e.ID > after && len(out) < limit {
			out = append(out, e)
		}
	}
	return out, nil
}
func (r *copyReg) MarkAuditCopied(_ context.Context, id uuid.UUID, through int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if through > r.copied[id] {
		r.copied[id] = through
	}
	return nil
}
func (r *copyReg) ProvisionedTenants(context.Context) ([]uuid.UUID, error) { return r.tenants, nil }

// router sends each call to the tenant's own destination.
type router struct {
	mu    sync.Mutex
	dests map[uuid.UUID]*memDest
}

func (r *router) d(id uuid.UUID) *memDest {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dests[id] == nil {
		r.dests[id] = &memDest{}
	}
	return r.dests[id]
}
func (r *router) EnsureAuditDestination(ctx context.Context, s infra.AuditDestinationSpec) error {
	return r.d(s.TenantID).EnsureAuditDestination(ctx, s)
}
func (r *router) MaxAuditID(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.d(id).MaxAuditID(ctx, id)
}
func (r *router) AuditHash(ctx context.Context, id uuid.UUID, n int64) (string, bool, error) {
	return r.d(id).AuditHash(ctx, id, n)
}
func (r *router) AppendAudit(ctx context.Context, id uuid.UUID, rows []infra.AuditRow) error {
	return r.d(id).AppendAudit(ctx, id, rows)
}

func (r *router) AuditRange(ctx context.Context, id uuid.UUID, after int64, limit int) ([]infra.AuditRow, error) {
	return r.d(id).AuditRange(ctx, id, after, limit)
}

type copyRig struct {
	reg  *copyReg
	rt   *router
	rc   *recordingConverter
	acts *activities.TenantLakehouseActivities
}

func newCopyRig() *copyRig {
	f, _ := newLH()
	f.cfg.Provisioned, f.cfg.Configured, f.cfg.LifecycleState, f.cfg.WarehouseName = true, true, "active", "ivy-t-x"
	reg := &copyReg{lhFakes: f, entries: map[uuid.UUID][]registry.AuditEntry{}, cfgs: map[uuid.UUID]*registry.Config{}, copied: map[uuid.UUID]int64{}}
	rt := &router{dests: map[uuid.UUID]*memDest{}}
	return &copyRig{reg: reg, rt: rt, rc: &recordingConverter{DataConverter: converter.GetDefaultDataConverter()},
		acts: &activities.TenantLakehouseActivities{Registry: reg, Credentials: f, Destination: rt, Keys: f, Buckets: f, Warehouses: f,
			S3Endpoint: "http://minio:9000", S3Region: "us-east-1"}}
}

func (r *copyRig) env() *testsuite.TestWorkflowEnvironment {
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	env.SetDataConverter(r.rc)
	env.RegisterActivity(r.acts)
	env.RegisterWorkflow(workflows.TenantLakehouseAuditCopyWorkflow)
	return env
}

func (r *copyRig) copyOne(t *testing.T, id uuid.UUID) (workflows.AuditCopyRunResult, error) {
	t.Helper()
	env := r.env()
	env.ExecuteWorkflow(workflows.TenantLakehouseAuditCopyWorkflow, activities.LakehouseProvisionInput{TenantID: id.String(), ActorID: "alice", ActorRole: "global_admin"})
	require.True(t, env.IsWorkflowCompleted())
	var res workflows.AuditCopyRunResult
	if err := env.GetWorkflowError(); err != nil {
		return res, err
	}
	require.NoError(t, env.GetWorkflowResult(&res))
	return res, nil
}

func TestAuditCopy_CopiesTheChainAndKeepsTheCredentialOutOfHistory(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.reg.entries[id] = chain(3)
	res, err := r.copyOne(t, id)
	require.NoError(t, err)

	d := r.rt.d(id)
	require.Equal(t, []int64{1, 2, 3}, d.ids())
	require.Equal(t, 3, res.Copied)
	require.EqualValues(t, 3, res.ThroughID)
	require.Equal(t, 1, d.ensured, "the destination is prepared once")
	require.Equal(t, []int{3}, d.appends, "one append: one Iceberg commit")
	require.EqualValues(t, 3, r.reg.copied[id], "the operator marker follows the destination")
	require.Equal(t, "alice", d.rows[0].ActorID)
	require.Equal(t, `{"n":1}`, d.rows[0].AfterJSON)

	require.NotEmpty(t, r.rc.seen)
	for _, p := range r.rc.seen {
		require.False(t, strings.Contains(p, secretValue), "a credential reached workflow history: %s", p)
	}
}

func TestAuditCopy_ResumesFromTheDestinationAndNeverDuplicates(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	all := chain(5)
	r.reg.entries[id] = all
	// The destination already holds 1 and 2, and the registry marker was never advanced (stale).
	d := r.rt.d(id)
	for _, e := range all[:2] {
		d.rows = append(d.rows, infra.AuditRow{ID: e.ID, PrevHash: e.PrevHash, Hash: e.Hash})
	}
	res, err := r.copyOne(t, id)
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2, 3, 4, 5}, d.ids(), "exactly once each")
	require.Equal(t, 3, res.Copied)
	require.Equal(t, []int{3}, d.appends, "only what was missing")
}

// THE case the design exists for. The first append commits and then reports an error, so the caller
// cannot know whether it landed. A counter-based copy would append again and duplicate; this one asks the
// destination, finds the rows already there, and ships nothing.
func TestAuditCopy_AnAppendWithAnUnknownOutcomeIsRetriedWithoutDuplicating(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.reg.entries[id] = chain(4)
	r.rt.d(id).storeThenFail = 1

	_, err := r.copyOne(t, id)
	require.NoError(t, err, "the retry recovers")
	d := r.rt.d(id)
	require.Equal(t, []int64{1, 2, 3, 4}, d.ids(), "each row exactly once, none duplicated, none lost")
	require.Equal(t, []int{4}, d.appends, "the retry found the rows already in the destination and appended nothing")
	require.EqualValues(t, 4, r.reg.copied[id])
}

func TestAuditCopy_RefusesToCopyOverABreakInTheChain(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	all := chain(4)
	r.reg.entries[id] = all
	d := r.rt.d(id)
	for _, e := range all[:2] {
		d.rows = append(d.rows, infra.AuditRow{ID: e.ID, PrevHash: e.PrevHash, Hash: e.Hash})
	}
	d.rows[1].Hash = "a-different-hash" // the copy's last entry no longer matches what alpha says entry 3 follows

	_, err := r.copyOne(t, id)
	require.Error(t, err)
	require.Empty(t, d.appends, "nothing is copied over a break")
	require.Equal(t, 1, d.hashCalls, "and it is not retried: a break needs a person")
	require.Equal(t, []int64{1, 2}, d.ids())
}

// The destination says its highest id is 2, but cannot produce entry 2. That is a damaged copy, not a gap
// to fill, so nothing is appended and it is not retried.
func TestAuditCopy_AHighestIdThatCannotBeReadBackIsABreak(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	all := chain(4)
	r.reg.entries[id] = all
	d := r.rt.d(id)
	for _, e := range all[:2] {
		d.rows = append(d.rows, infra.AuditRow{ID: e.ID, PrevHash: e.PrevHash, Hash: e.Hash})
	}
	d.hideHashes = true
	_, err := r.copyOne(t, id)
	require.Error(t, err)
	require.Empty(t, d.appends)
	require.Equal(t, 1, d.hashCalls, "not retried")
}

// An empty copy only accepts a chain that starts at the genesis hash: otherwise earlier entries are missing.
func TestAuditCopy_AnEmptyCopyOnlyAcceptsAChainThatStartsAtGenesis(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	bad := chain(2)
	bad[0].PrevHash = "not-genesis"
	r.reg.entries[id] = bad
	_, err := r.copyOne(t, id)
	require.Error(t, err)
	require.Empty(t, r.rt.d(id).appends)
}

// A chain that breaks INSIDE one batch is refused as a whole.
func TestAuditCopy_AChainThatBreaksInsideABatchIsRefusedWhole(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	bad := chain(4)
	bad[2].PrevHash = "tampered"
	r.reg.entries[id] = bad
	_, err := r.copyOne(t, id)
	require.Error(t, err)
	require.Empty(t, r.rt.d(id).appends, "not even the entries before the break are copied")
}

func TestAuditCopy_NothingToShipCommitsNothing(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	all := chain(3)
	r.reg.entries[id] = all
	d := r.rt.d(id)
	for _, e := range all {
		d.rows = append(d.rows, infra.AuditRow{ID: e.ID, PrevHash: e.PrevHash, Hash: e.Hash})
	}
	res, err := r.copyOne(t, id)
	require.NoError(t, err)
	require.Zero(t, res.Copied)
	require.Empty(t, d.appends, "no append, so no Iceberg commit and no files kept for years under Object Lock")
	require.EqualValues(t, 3, r.reg.copied[id], "but the operator marker is brought up to date")
}

func TestAuditCopy_BatchesAreBoundedAndEachIsOneCommit(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.reg.entries[id] = chain(1200)
	res, err := r.copyOne(t, id)
	require.NoError(t, err)
	d := r.rt.d(id)
	require.Equal(t, []int{500, 500, 200}, d.appends)
	require.Len(t, d.rows, 1200)
	require.Equal(t, 1200, res.Copied)
	require.False(t, res.Truncated)
	for i := 1; i < len(d.rows); i++ {
		require.Equal(t, d.rows[i-1].ID+1, d.rows[i].ID, "in order, no gaps")
	}
}

func TestAuditCopy_FailsFastWhenThereIsNothingToCopyInto(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.reg.cfgs[id] = &registry.Config{Configured: true, Provisioned: false, LifecycleState: "provisioning"}
	r.reg.entries[id] = chain(2)
	_, err := r.copyOne(t, id)
	require.Error(t, err)
	d := r.rt.d(id)
	require.Zero(t, d.ensured, "no catalog is created for a tenant with no warehouse")
	require.Empty(t, d.appends)
}

func TestAuditCopy_MissingStarRocksConfigurationIsNotRetried(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.reg.entries[id] = chain(2)
	r.rt.d(id).ensureErr = errors.Join(errors.New("StarRocks needs [LAKEHOUSE_STARROCKS_DSN]"), infra.ErrNotConfigured)
	_, err := r.copyOne(t, id)
	require.Error(t, err)
	require.Equal(t, 1, r.rt.d(id).ensured, "missing configuration fails fast instead of retrying five times")
}

func TestAuditCopyAll_OneBrokenTenantDoesNotStopTheOthers(t *testing.T) {
	r := newCopyRig()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	r.reg.tenants = []uuid.UUID{a, b, c}
	for _, id := range r.reg.tenants {
		r.reg.entries[id] = chain(2)
	}
	r.rt.d(b).appendErr = errors.Join(errors.New("iceberg commit refused"), infra.ErrNotConfigured) // fails for B only

	env := r.env()
	env.RegisterWorkflow(workflows.TenantLakehouseAuditCopyAllWorkflow)
	env.ExecuteWorkflow(workflows.TenantLakehouseAuditCopyAllWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError(), "the scheduled run itself must not fail, or the cron would stop being useful")
	var res workflows.AuditCopyAllResult
	require.NoError(t, env.GetWorkflowResult(&res))

	require.Equal(t, 3, res.Tenants)
	require.Equal(t, 4, res.Copied, "A and C copied their 2 entries each")
	require.Contains(t, res.Failed, b.String())
	require.Len(t, res.Failed, 1)
	require.Equal(t, []int64{1, 2}, r.rt.d(a).ids())
	require.Equal(t, []int64{1, 2}, r.rt.d(c).ids())
	require.Empty(t, r.rt.d(b).ids())
}

func TestAuditCopyAll_NoTenantsIsFine(t *testing.T) {
	r := newCopyRig()
	env := r.env()
	env.RegisterWorkflow(workflows.TenantLakehouseAuditCopyAllWorkflow)
	env.ExecuteWorkflow(workflows.TenantLakehouseAuditCopyAllWorkflow)
	require.NoError(t, env.GetWorkflowError())
	var res workflows.AuditCopyAllResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Zero(t, res.Tenants)
}

// A destination that acknowledges a write and keeps nothing must never be recorded as copied. The batch is read
// back after the append; if the rows are not there the run fails and the operator marker does not move.
func TestAuditCopy_AnAcknowledgedWriteThatIsNotThereIsNeverRecordedAsCopied(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.reg.entries[id] = chain(3)
	r.rt.d(id).dropSilently = true

	_, err := r.copyOne(t, id)
	require.Error(t, err)
	require.Empty(t, r.rt.d(id).ids())
	require.Zero(t, r.reg.copied[id], "the marker must not claim rows the copy does not hold")
	require.Greater(t, len(r.rt.d(id).appends), 1, "and it is retried, since the destination may recover")
}
