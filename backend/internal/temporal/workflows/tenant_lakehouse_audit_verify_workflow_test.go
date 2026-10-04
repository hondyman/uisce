package workflows_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"
)

// These run the REAL verify activities and workflow against an in-memory alpha and an in-memory Iceberg
// copy. The point of each case is what a verifier must NOT say: "verified" for a copy that has lost,
// invented or altered an entry, or that was compared with a record that no longer recomputes.

func (r *copyRig) verify(t *testing.T, id uuid.UUID) (*activities.AuditVerifyReport, error) {
	t.Helper()
	env := r.env()
	env.RegisterWorkflow(workflows.TenantLakehouseAuditVerifyWorkflow)
	env.ExecuteWorkflow(workflows.TenantLakehouseAuditVerifyWorkflow, workflows.AuditVerifyInput{
		Lakehouse: activities.LakehouseProvisionInput{TenantID: id.String(), ActorID: "alice", ActorRole: "global_admin"}})
	require.True(t, env.IsWorkflowCompleted())
	if err := env.GetWorkflowError(); err != nil {
		return nil, err
	}
	var rep activities.AuditVerifyReport
	require.NoError(t, env.GetWorkflowResult(&rep))
	return &rep, nil
}

// copied runs the real copy first, so the verifier is judging what the copy workflow really wrote.
func (r *copyRig) copied(t *testing.T, id uuid.UUID, n int) {
	t.Helper()
	r.reg.entries[id] = chain(n)
	_, err := r.copyOne(t, id)
	require.NoError(t, err)
}

func TestAuditVerify_ACopyTheCopyWorkflowWroteVerifies(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 25)
	rep, err := r.verify(t, id)
	require.NoError(t, err)
	require.True(t, rep.Verified)
	require.Nil(t, rep.Finding)
	require.EqualValues(t, 25, rep.Rows)
	require.EqualValues(t, 25, rep.ThroughID)
}

func TestAuditVerify_ALaggingCopyVerifiesWhatItHoldsAndReportsHowFar(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 10)
	r.reg.entries[id] = chain(14) // alpha has since recorded four more
	rep, err := r.verify(t, id)
	require.NoError(t, err)
	require.True(t, rep.Verified, "pending entries are not a finding")
	require.EqualValues(t, 10, rep.ThroughID, "the report says how far the proof reaches")
}

func TestAuditVerify_AnEmptyCopyVerifiesNothing(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.reg.entries[id] = chain(4)
	rep, err := r.verify(t, id)
	require.NoError(t, err)
	require.Zero(t, rep.Rows)
	require.Zero(t, rep.ThroughID)
}

func TestAuditVerify_ALostEntryIsFound(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 8)
	d := r.rt.d(id)
	d.rows = append(d.rows[:3], d.rows[4:]...) // the copy loses id 4
	rep, err := r.verify(t, id)
	require.NoError(t, err, "a finding is a result, not a failure to retry")
	require.False(t, rep.Verified)
	require.Equal(t, activities.FindingMissingInCopy, rep.Finding.Kind)
	require.EqualValues(t, 4, rep.Finding.ID)
}

func TestAuditVerify_AnAlteredEntryIsFound(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 8)
	r.rt.d(id).rows[5].ActorID = "mallory"
	rep, err := r.verify(t, id)
	require.NoError(t, err)
	require.False(t, rep.Verified)
	require.Equal(t, activities.FindingDiffers, rep.Finding.Kind)
	require.EqualValues(t, 6, rep.Finding.ID)
}

func TestAuditVerify_AnInventedEntryIsFound(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 5)
	d := r.rt.d(id)
	last := d.rows[len(d.rows)-1]
	d.rows = append(d.rows, infra.AuditRow{ID: 6, At: last.At, ActorID: "mallory", PrevHash: last.Hash, Hash: "forged"})
	rep, err := r.verify(t, id)
	require.NoError(t, err)
	require.False(t, rep.Verified)
	require.Equal(t, activities.FindingExtraInCopy, rep.Finding.Kind)
}

func TestAuditVerify_ABrokenAlphaIsReportedBeforeAnythingIsCompared(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 5)
	r.reg.brokenAlpha = map[uuid.UUID]int64{id: 3}
	rep, err := r.verify(t, id)
	require.NoError(t, err)
	require.False(t, rep.Verified, "a copy that matches a broken record proves nothing")
	require.Equal(t, activities.FindingAlphaChainBroken, rep.Finding.Kind)
	require.EqualValues(t, 3, rep.Finding.ID)
	require.Zero(t, rep.Rows, "the comparison never started")
}

func TestAuditVerify_TheCopyIsNotTouched(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 5)
	d := r.rt.d(id)
	before, appends, ensured := len(d.rows), len(d.appends), d.ensured
	r.rt.d(id).rows[2].Hash = "tampered"
	_, err := r.verify(t, id)
	require.NoError(t, err)
	require.Equal(t, before, len(d.rows))
	require.Equal(t, appends, len(d.appends), "a verifier never writes")
	require.Equal(t, ensured, d.ensured, "and never prepares a destination")
	require.Equal(t, "tampered", d.rows[2].Hash, "and never repairs")
}

func TestAuditVerify_ATenantsOutageIsRetriedNotReportedAsAFinding(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.reg.entries[id] = chain(2)
	r.acts.Destination = &downDest{}
	_, err := r.verify(t, id)
	require.Error(t, err)
}

// A chain longer than one run's page budget continues as a new run from its cursor. Nothing is lost at
// the seam and the continuation does not re-check alpha from the start.
func TestAuditVerify_ALongChainContinuesAsANewRunFromItsCursor(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	n := activities.AuditVerifyPageSize*50 + 7
	all := chain(n)
	r.reg.entries[id] = all
	d := r.rt.d(id)
	for _, e := range all {
		d.rows = append(d.rows, infra.AuditRow{ID: e.ID, At: e.At, ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action,
			AfterJSON: string(e.After), PrevHash: e.PrevHash, Hash: e.Hash})
	}

	env := r.env()
	env.RegisterWorkflow(workflows.TenantLakehouseAuditVerifyWorkflow)
	in := activities.LakehouseProvisionInput{TenantID: id.String(), ActorID: "alice", ActorRole: "global_admin"}
	env.ExecuteWorkflow(workflows.TenantLakehouseAuditVerifyWorkflow, workflows.AuditVerifyInput{Lakehouse: in})
	require.True(t, env.IsWorkflowCompleted())
	var can *workflow.ContinueAsNewError
	require.ErrorAs(t, env.GetWorkflowError(), &can, "the first run hands over rather than growing its history")

	// The next run is what the continuation starts. Verify it from the cursor the first one reached.
	cur := activities.AuditVerifyCursor{AfterID: int64(activities.AuditVerifyPageSize * 50), PrevHash: fmt.Sprintf("h%d", activities.AuditVerifyPageSize*50), Rows: int64(activities.AuditVerifyPageSize * 50)}
	env2 := r.env()
	env2.RegisterWorkflow(workflows.TenantLakehouseAuditVerifyWorkflow)
	env2.ExecuteWorkflow(workflows.TenantLakehouseAuditVerifyWorkflow, workflows.AuditVerifyInput{Lakehouse: in, Cursor: cur})
	require.NoError(t, env2.GetWorkflowError())
	var rep activities.AuditVerifyReport
	require.NoError(t, env2.GetWorkflowResult(&rep))
	require.True(t, rep.Verified)
	require.EqualValues(t, n, rep.Rows)
}

func TestAuditVerify_TheWorkflowIDIsPerTenant(t *testing.T) {
	require.Equal(t, "lakehouse-audit-verify-abc", workflows.AuditVerifyWorkflowID("abc"))
}

// downDest is a destination whose reads fail, as in an outage.
type downDest struct{ memDest }

func (d *downDest) AuditRange(_ context.Context, _ uuid.UUID, _ int64, _ int) ([]infra.AuditRow, error) {
	return nil, fmt.Errorf("starrocks: connection refused")
}

func TestAuditVerify_APassIsRecordedWithHowFarItReaches(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 6)
	r.reg.entries[id] = chain(9) // three pending
	_, err := r.verify(t, id)
	require.NoError(t, err)
	require.Equal(t, []registry.AuditVerification{{ThroughID: 6}}, r.reg.recorded[id])
}

// The point of recording: a later bad run must take back an earlier pass.
func TestAuditVerify_AFindingReplacesAnEarlierPass(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 6)
	_, err := r.verify(t, id)
	require.NoError(t, err)
	r.rt.d(id).rows[3].Hash = "tampered"
	_, err = r.verify(t, id)
	require.NoError(t, err)
	got := r.reg.recorded[id]
	require.Len(t, got, 2)
	require.Empty(t, got[0].FindingKind)
	require.Equal(t, registry.AuditVerification{ThroughID: 3, FindingKind: activities.FindingDiffers, FindingID: 4}, got[1],
		"the finding names the kind and entry, and the proof stops at the last good one")
}

func TestAuditVerify_ABrokenAlphaIsRecordedToo(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 4)
	r.reg.brokenAlpha = map[uuid.UUID]int64{id: 2}
	_, err := r.verify(t, id)
	require.NoError(t, err)
	require.Equal(t, []registry.AuditVerification{{FindingKind: activities.FindingAlphaChainBroken, FindingID: 2}}, r.reg.recorded[id])
}

// A result that could not be stored is not a result: the run fails, so nobody reads silence as a pass.
func TestAuditVerify_AnOutcomeThatCannotBeRecordedFailsTheRun(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	r.copied(t, id, 3)
	r.reg.recordErr = errors.New("alpha unavailable")
	_, err := r.verify(t, id)
	require.Error(t, err)
}

func TestAuditVerify_ARunThatHandsOverRecordsNothing(t *testing.T) {
	r := newCopyRig()
	id := uuid.New()
	all := chain(activities.AuditVerifyPageSize*50 + 3)
	r.reg.entries[id] = all
	d := r.rt.d(id)
	for _, e := range all {
		d.rows = append(d.rows, infra.AuditRow{ID: e.ID, At: e.At, ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action,
			AfterJSON: string(e.After), PrevHash: e.PrevHash, Hash: e.Hash})
	}
	env := r.env()
	env.RegisterWorkflow(workflows.TenantLakehouseAuditVerifyWorkflow)
	env.ExecuteWorkflow(workflows.TenantLakehouseAuditVerifyWorkflow, workflows.AuditVerifyInput{
		Lakehouse: activities.LakehouseProvisionInput{TenantID: id.String(), ActorID: "a", ActorRole: "global_admin"}})
	var can *workflow.ContinueAsNewError
	require.ErrorAs(t, env.GetWorkflowError(), &can)
	require.Empty(t, r.reg.recorded[id], "only the last run knows the outcome")
}

func TestAuditVerifyAll_OneTenantsFindingOrFailureDoesNotStopTheOthers(t *testing.T) {
	r := newCopyRig()
	good, bad, down := uuid.New(), uuid.New(), uuid.New()
	r.reg.tenants = []uuid.UUID{good, bad, down}
	for _, id := range r.reg.tenants {
		r.copied(t, id, 3)
	}
	r.rt.d(bad).rows[1].ActorID = "mallory"
	r.rt.fail = map[uuid.UUID]bool{down: true}

	env := r.env()
	env.RegisterWorkflow(workflows.TenantLakehouseAuditVerifyWorkflow)
	env.RegisterWorkflow(workflows.TenantLakehouseAuditVerifyAllWorkflow)
	env.ExecuteWorkflow(workflows.TenantLakehouseAuditVerifyAllWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError(), "the schedule must keep running")
	var res workflows.AuditVerifyAllResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, 3, res.Tenants)
	require.Equal(t, 1, res.Verified)
	require.Equal(t, map[string]string{bad.String(): activities.FindingDiffers}, res.Findings)
	require.Contains(t, res.Failed, down.String())
	require.Len(t, res.Failed, 1)
	require.Len(t, r.reg.recorded[good], 1)
	require.Len(t, r.reg.recorded[bad], 1)
	require.Empty(t, r.reg.recorded[down], "an outage records nothing: the earlier outcome stands and its age shows")
}
