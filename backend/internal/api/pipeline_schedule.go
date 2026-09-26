package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/datapipeline"
	"github.com/hondyman/uisce/backend/internal/schedule"
	"github.com/hondyman/uisce/backend/internal/security"
)

// --- data pipelines on the one scheduler ------------------------------------

// dataPipelineRunner runs a saved pipeline for the scheduler, as a new
// tracked pipeline run - exactly what the pipeline's own scheduled workflow
// did before pipelines moved onto internal/schedule.
type dataPipelineRunner struct {
	store *datapipeline.Store
	acts  *datapipeline.Activities
}

func (r *dataPipelineRunner) Kind() string  { return "data_pipeline" }
func (r *dataPipelineRunner) Label() string { return "Data pipeline" }

// Pipelines belong to the tenant (not to a user), so any user of the tenant
// may schedule them.
func (r *dataPipelineRunner) Check(ctx context.Context, tenantID, _ string, ref string, _ map[string]any) error {
	if _, err := r.store.Get(ctx, tenantID, ref); err != nil {
		if errors.Is(err, datapipeline.ErrNotFound) {
			return schedule.MsgTargetNotFound(r.Label(), ref)
		}
		return err
	}
	return nil
}

func (r *dataPipelineRunner) Targets(ctx context.Context, tenantID, _ string) ([]schedule.TargetInfo, error) {
	list, err := r.store.List(ctx, tenantID)
	out := make([]schedule.TargetInfo, 0, len(list))
	for _, d := range list {
		out = append(out, schedule.TargetInfo{Ref: d.ID, Name: d.Name, Description: d.Description})
	}
	return out, err
}

func (r *dataPipelineRunner) Run(ctx context.Context, rc schedule.RunContext) (*schedule.Outcome, error) {
	d, err := r.store.Get(ctx, rc.TenantID, rc.Ref)
	if errors.Is(err, datapipeline.ErrNotFound) {
		return nil, schedule.MsgTargetNotFound(r.Label(), rc.Ref)
	}
	if err != nil {
		return nil, err
	}
	runID, err := r.store.CreateRun(ctx, rc.TenantID, d)
	if err != nil {
		return nil, err
	}
	out := &schedule.Outcome{Refs: map[string]any{"pipeline_run_id": runID}}
	if errs := d.Spec.Validate(); len(errs) > 0 {
		verr := fmt.Errorf("pipeline %s has problems: %w", d.ID, errors.Join(errs...))
		_ = r.store.FinishRun(ctx, rc.TenantID, runID, nil, verr)
		return out, verr
	}
	runErr := r.acts.Run(ctx, datapipeline.RunInput{TenantID: rc.TenantID, RunID: runID})
	if rec, err := r.store.GetRun(ctx, rc.TenantID, runID); err == nil {
		out.Rows = int(rec.RecordsOut)
		out.Summary = fmt.Sprintf("%s: %d read, %d written, %d rejected", d.Name, rec.RecordsIn, rec.RecordsOut, rec.Errors)
		if runErr == nil && rec.Status == "failed" {
			runErr = fmt.Errorf("pipeline run %s failed", runID)
		}
	}
	return out, runErr
}

// pipelineCoreScheduler implements the pipeline handler's Scheduler on the
// one scheduler: the pipeline's schedule (HTTP and MCP) is a core schedule
// of kind data_pipeline, so it shows in the Schedules console and run
// history and can use business calendars. The service is resolved when
// used: pipelines are wired before the scheduler.
type pipelineCoreScheduler struct {
	srv   *Server
	store *datapipeline.Store
}

func (p *pipelineCoreScheduler) Apply(ctx context.Context, tenantID, pipelineID string, sc *datapipeline.Schedule) error {
	svc := p.srv.ScheduleService
	if svc == nil {
		return fmt.Errorf("the scheduler is not available")
	}
	actor := schedule.Actor{TenantID: tenantID, UserID: "system:data-pipelines"}
	if auth, ok := security.AuthInfoFromContext(ctx); ok && auth.UserID != "" {
		actor.UserID = auth.UserID
	}
	existing, err := svc.Store.List(ctx, tenantID, schedule.ListFilter{Kind: "data_pipeline", Ref: pipelineID})
	if err != nil {
		return err
	}
	if sc == nil { // pipeline deleted
		for _, s := range existing {
			if err := svc.Delete(ctx, actor, s.ID); err != nil {
				return err
			}
		}
		return nil
	}
	if len(existing) == 0 {
		if !sc.Enabled {
			return nil
		}
		d, err := p.store.Get(ctx, tenantID, pipelineID)
		if err != nil {
			return err
		}
		enabled := true
		_, err = svc.Create(ctx, actor, schedule.Input{
			Name:    d.Name,
			Target:  schedule.Target{Kind: "data_pipeline", Ref: pipelineID},
			Timing:  schedule.Timing{Cron: sc.Cron, TimeZone: sc.TimeZone},
			Enabled: &enabled,
		})
		return err
	}
	// Keep what the pipeline's dialog doesn't know about (name, calendar
	// rule set in the Schedules console); change the timetable and state.
	cur := existing[0]
	t := cur.Timing
	t.Cron, t.TimeZone = sc.Cron, sc.TimeZone
	enabled := sc.Enabled
	_, err = svc.Update(ctx, actor, cur.ID, schedule.Input{
		Name: cur.Name, Description: cur.Description, Target: cur.Target, Timing: t, Enabled: &enabled,
	})
	return err
}

// Current reads the pipeline's schedule from the one scheduler, which is the
// source of truth once it can be edited in the Schedules console. ok is
// false when the scheduler isn't available, so the caller falls back to the
// copy stored on the pipeline.
func (p *pipelineCoreScheduler) Current(ctx context.Context, tenantID, pipelineID string) (*datapipeline.Schedule, bool, error) {
	svc := p.srv.ScheduleService
	if svc == nil {
		return nil, false, nil
	}
	existing, err := svc.Store.List(ctx, tenantID, schedule.ListFilter{Kind: "data_pipeline", Ref: pipelineID})
	if err != nil || len(existing) == 0 {
		return nil, err == nil, err
	}
	cur := existing[0]
	return &datapipeline.Schedule{Cron: cur.Timing.Cron, TimeZone: cur.Timing.TimeZone, Enabled: cur.Enabled}, true, nil
}
