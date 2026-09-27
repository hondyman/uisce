package mastering

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/schedule"
	"github.com/hondyman/uisce/backend/internal/stagingbind"
)

// BindingLister lists the staging bindings a tenant sees
// (stagingbind.Store implements it).
type BindingLister interface {
	List(ctx context.Context, tenantID string) ([]stagingbind.Binding, error)
}

// Runner masters on the one scheduler (kind "mastering"), so a mastering
// job has a timetable, business calendars, run history and can be driven by
// an enterprise scheduler (Tidal, Control-M) through the trigger API.
//
// The target is an entity and the staging table its source lands in
// ("product:staging.ff_product"). Each firing masters every completed load
// of that table not yet mastered, oldest first. Each load keeps its own
// idempotency key (load:<id>), so a retried or doubled firing never masters
// a load twice; with nothing new, the run succeeds and says so.
type Runner struct {
	Engine   *Engine
	Bindings BindingLister
}

const runnerKind = "mastering"

func (r *Runner) Kind() string  { return runnerKind }
func (r *Runner) Label() string { return "Mastering" }

// ParseTarget splits "entity:staging.table".
func ParseTarget(ref string) (entity, table string, ok bool) {
	entity, table, ok = strings.Cut(ref, ":")
	if !ok || entity == "" || !qualified.MatchString(table) || !strings.HasPrefix(table, "staging.") {
		return "", "", false
	}
	return strings.ToLower(entity), table, true
}

// targets pairs each mastered entity with the staging tables bound to its
// business object.
func (r *Runner) targets(ctx context.Context, tenantID string) ([]schedule.TargetInfo, error) {
	if r.Engine == nil || r.Engine.Data == nil {
		return nil, msgNoDataPlane()
	}
	profiles, err := r.Engine.Profiles(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	bindings, err := r.Bindings.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var out []schedule.TargetInfo
	for _, p := range profiles {
		if !p.IsActive {
			continue
		}
		for _, b := range bindings {
			if b.BOKey != p.BOKey {
				continue
			}
			out = append(out, schedule.TargetInfo{
				Ref:         strings.ToLower(p.EntityCd) + ":" + b.StagingTable,
				Name:        fmt.Sprintf("%s from %s", p.DisplayName, b.StagingTable),
				Description: fmt.Sprintf("Master new %s loads into %s golden records", b.StagingTable, p.DisplayName),
			})
		}
	}
	return out, nil
}

func (r *Runner) Targets(ctx context.Context, tenantID, _ string) ([]schedule.TargetInfo, error) {
	return r.targets(ctx, tenantID)
}

// Check: the entity is mastered and the table is bound to its object.
// Mastering belongs to the tenant, like pipelines, so any tenant user who
// may schedule may schedule it.
func (r *Runner) Check(ctx context.Context, tenantID, _ string, ref string, _ map[string]any) error {
	if _, _, ok := ParseTarget(ref); !ok {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	ts, err := r.targets(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, t := range ts {
		if t.Ref == ref {
			return nil
		}
	}
	return schedule.MsgTargetNotFound(r.Label(), ref)
}

func (r *Runner) Run(ctx context.Context, rc schedule.RunContext) (*schedule.Outcome, error) {
	entity, table, ok := ParseTarget(rc.Ref)
	if !ok {
		return nil, schedule.MsgTargetNotFound(r.Label(), rc.Ref)
	}
	if r.Engine == nil || r.Engine.Data == nil {
		return nil, msgNoDataPlane()
	}
	loads, err := r.Engine.PendingLoads(ctx, rc.TenantID, entity, table)
	if err != nil {
		return nil, err
	}
	out := &schedule.Outcome{Refs: map[string]any{}}
	if len(loads) == 0 {
		out.Summary = fmt.Sprintf("%s: no new loads to master", table)
		return out, nil
	}

	var total Counts
	var runIDs []string
	var failed []error
	for i, load := range loads {
		if rc.Heartbeat != nil {
			rc.Heartbeat(fmt.Sprintf("mastering load %d of %d", i+1, len(loads)))
		}
		req := RunRequest{Entity: entity, StagingTable: table, LoadRunID: load, Trigger: "schedule",
			ScheduleRunID: rc.RunID, StartedBy: "schedule:" + rc.ScheduleID, StartedByID: rc.OwnerID}
		run, fresh, err := r.Engine.Start(ctx, rc.TenantID, req)
		if err != nil {
			failed = append(failed, fmt.Errorf("load %s: %w", load, err))
			continue
		}
		if fresh {
			run, err = r.Engine.Execute(ctx, rc.TenantID, run, req)
			if err != nil {
				failed = append(failed, fmt.Errorf("load %s: %w", load, err))
			}
		}
		if run != nil {
			runIDs = append(runIDs, run.ID)
			var c Counts
			if decodeCounts(run, &c) == nil {
				total.add(c)
			}
		}
	}
	out.Rows = total.Published
	out.Refs["mastering_run_ids"] = runIDs
	out.Summary = fmt.Sprintf("%s: %d load(s), %d records, %d published, %d unchanged, %d new, %d rejected, %d exceptions",
		table, len(loads), total.Records, total.Published, total.Unchanged, total.New, total.Invalid, total.Exceptions)
	return out, errors.Join(failed...)
}

// PendingLoads lists the completed loads in a staging table that have no
// finished mastering run under their load key, oldest first.
func (e *Engine) PendingLoads(ctx context.Context, tenantID, entity, table string) ([]string, error) {
	if !qualified.MatchString(table) {
		return nil, msgNoBinding(table, entity)
	}
	var out []string
	err := e.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &out, fmt.Sprintf(`SELECT l.id::text FROM staging._load_run l
			WHERE upper(l.status) = 'COMPLETED'
			  AND EXISTS (SELECT 1 FROM %s s WHERE s._load_run_id = l.id)
			  AND NOT EXISTS (SELECT 1 FROM mdm.mastering_run m WHERE m.entity_cd = $1
			        AND m.idempotency_key = 'load:' || l.id::text AND m.status IN ('COMPLETED', 'PARTIAL', 'RUNNING'))
			ORDER BY l.started_at NULLS FIRST, l.id`, qi(table)), strings.ToUpper(entity))
	})
	return out, err
}

func (c *Counts) add(o Counts) {
	c.Records += o.Records
	c.Valid += o.Valid
	c.Invalid += o.Invalid
	c.Xref += o.Xref
	c.Deterministic += o.Deterministic
	c.Fuzzy += o.Fuzzy
	c.Review += o.Review
	c.New += o.New
	c.Conflicts += o.Conflicts
	c.Published += o.Published
	c.HeldForReview += o.HeldForReview
	c.Unchanged += o.Unchanged
	c.Exceptions += o.Exceptions
	c.Restated += o.Restated
	c.Rechecked += o.Rechecked
}
