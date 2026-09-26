package datapipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Schedule runs a saved pipeline on a cron schedule in a time zone.
type Schedule struct {
	Cron     string `json:"cron"`     // standard 5 fields: minute hour day-of-month month day-of-week
	TimeZone string `json:"timezone"` // IANA name, e.g. Europe/Dublin; empty = UTC
	Enabled  bool   `json:"enabled"`
}

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func (s Schedule) location() (*time.Location, error) {
	if s.TimeZone == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(s.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("unknown time zone %q", s.TimeZone)
	}
	return loc, nil
}

// Validate checks the cron expression and time zone, and refuses schedules
// that fire more often than every 5 minutes.
func (s Schedule) Validate() error {
	if _, err := s.location(); err != nil {
		return err
	}
	sched, err := cronParser.Parse(strings.TrimSpace(s.Cron))
	if err != nil {
		return fmt.Errorf("schedule %q is not a valid 5-field cron expression: %v", s.Cron, err)
	}
	t := sched.Next(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))
	for i := 0; i < 50; i++ {
		n := sched.Next(t)
		if n.Sub(t) < 5*time.Minute {
			return fmt.Errorf("a pipeline can run at most every 5 minutes")
		}
		t = n
	}
	return nil
}

// Next returns the next n run times after from, in the schedule's zone.
func (s Schedule) Next(n int, from time.Time) ([]time.Time, error) {
	loc, err := s.location()
	if err != nil {
		return nil, err
	}
	sched, err := cronParser.Parse(strings.TrimSpace(s.Cron))
	if err != nil {
		return nil, err
	}
	out := make([]time.Time, 0, n)
	t := from.In(loc)
	for i := 0; i < n; i++ {
		t = sched.Next(t)
		out = append(out, t)
	}
	return out, nil
}

// --- persistence --------------------------------------------------------------

func (s *Store) GetSchedule(ctx context.Context, tenantID, pipelineID string) (*Schedule, error) {
	var raw []byte
	err := s.DB.GetContext(ctx, &raw, `SELECT schedule FROM data_pipeline_definitions
		WHERE id = $1 AND tenant_id = $2 AND mode = $3 AND is_active`, pipelineID, tenantID, ModeLoader)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil || raw == nil {
		return nil, err
	}
	var sc Schedule
	return &sc, json.Unmarshal(raw, &sc)
}

// SetSchedule stores (nil: clears) a pipeline's schedule.
func (s *Store) SetSchedule(ctx context.Context, tenantID, pipelineID string, sc *Schedule) error {
	var raw any
	if sc != nil {
		b, err := json.Marshal(sc)
		if err != nil {
			return err
		}
		raw = string(b)
	}
	res, err := s.DB.ExecContext(ctx, `UPDATE data_pipeline_definitions SET schedule = $3::jsonb, last_modified_at = now()
		WHERE id = $1 AND tenant_id = $2 AND mode = $4 AND is_active`, pipelineID, tenantID, raw, ModeLoader)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Temporal --------------------------------------------------------------

// Scheduler keeps the live schedule in step with a pipeline's Schedule.
type Scheduler interface {
	Apply(ctx context.Context, tenantID, pipelineID string, sc *Schedule) error // nil or disabled: remove
}

// ScheduleID is the Temporal schedule id for a tenant's pipeline.
func ScheduleID(tenantID, pipelineID string) string {
	return "data-pipeline-" + tenantID + "-" + pipelineID
}

// TemporalScheduler backs schedules with Temporal Schedules. A firing while
// the previous scheduled run is still going is skipped, never overlapped.
type TemporalScheduler struct{ Client client.Client }

func (t *TemporalScheduler) Apply(ctx context.Context, tenantID, pipelineID string, sc *Schedule) error {
	id := ScheduleID(tenantID, pipelineID)
	h := t.Client.ScheduleClient().GetHandle(ctx, id)
	if sc == nil || !sc.Enabled {
		err := h.Delete(ctx)
		var nf *serviceerror.NotFound
		if err != nil && !errors.As(err, &nf) {
			return err
		}
		return nil
	}
	spec := client.ScheduleSpec{CronExpressions: []string{sc.Cron}, TimeZoneName: sc.TimeZone}
	action := &client.ScheduleWorkflowAction{
		ID:        "data-pipeline-scheduled-" + pipelineID,
		Workflow:  ScheduledWorkflow,
		Args:      []interface{}{ScheduledInput{TenantID: tenantID, PipelineID: pipelineID}},
		TaskQueue: TaskQueue,
	}
	_, err := t.Client.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: id, Spec: spec, Action: action, Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
	})
	var exists *serviceerror.AlreadyExists
	if err == nil {
		return nil
	}
	if !errors.As(err, &exists) && !errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return err
	}
	return h.Update(ctx, client.ScheduleUpdateOptions{DoUpdate: func(in client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
		s := in.Description.Schedule
		s.Spec = &spec
		s.Action = action
		if s.Policy == nil {
			s.Policy = &client.SchedulePolicies{}
		}
		s.Policy.Overlap = enums.SCHEDULE_OVERLAP_POLICY_SKIP
		if s.State != nil {
			s.State.Paused = false
		}
		return &client.ScheduleUpdate{Schedule: &s}, nil
	}})
}

// ScheduledInput identifies a scheduled pipeline.
type ScheduledInput struct {
	TenantID   string
	PipelineID string
}

// ScheduledActivityName is the registered name of Activities.ScheduledRun.
const ScheduledActivityName = "DataPipelineScheduledRun"

// ScheduledWorkflow runs the saved pipeline once as a new tracked run.
func ScheduledWorkflow(ctx workflow.Context, in ScheduledInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 12 * time.Hour,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	return workflow.ExecuteActivity(ctx, ScheduledActivityName, in).Get(ctx, nil)
}

// ScheduledRun snapshots the pipeline as it is now and executes it.
func (a *Activities) ScheduledRun(ctx context.Context, in ScheduledInput) error {
	d, err := a.Store.Get(ctx, in.TenantID, in.PipelineID)
	if err != nil {
		return fmt.Errorf("scheduled pipeline %s: %w", in.PipelineID, err)
	}
	runID, err := a.Store.CreateRun(ctx, in.TenantID, d)
	if err != nil {
		return err
	}
	if errs := d.Spec.Validate(); len(errs) > 0 {
		verr := fmt.Errorf("the pipeline has problems: %w", errors.Join(errs...))
		_ = a.Store.FinishRun(ctx, in.TenantID, runID, nil, verr)
		return verr
	}
	return a.Run(ctx, RunInput{TenantID: in.TenantID, RunID: runID})
}
