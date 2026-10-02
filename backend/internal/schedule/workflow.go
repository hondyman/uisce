package schedule

import (
	"context"
	"errors"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// FireInput identifies one firing.
type FireInput struct {
	TenantID    string
	ScheduleID  string
	Manual      bool   // run now: no calendar rule
	TriggeredBy string // who asked, for a manual or external run
	// External is set when an enterprise scheduler fired the run; the
	// calendar rule still applies (it can only skip).
	External *ExternalTrigger `json:",omitempty"`
}

// trigger is how the run was started, as recorded in its history.
func (in FireInput) trigger() string {
	switch {
	case in.External != nil:
		return "external"
	case in.Manual:
		return "manual"
	}
	return "schedule"
}

// GateResult is what a firing should do.
type GateResult struct {
	Run      bool
	Decision *Decision
	Kind     string
	Ref      string
	// Stop: the schedule is gone or disabled - nothing to record.
	Stop bool
}

// Activity names.
const (
	ActGate   = "ScheduleGate"
	ActRecord = "ScheduleRecordSkip"
	ActRun    = "ScheduleRun"
)

// FireWorkflow is one firing: judge the calendar, then skip, wait for the
// next business day, or run. The run itself is never retried by the engine
// (a partly-done report or load must not be repeated behind the user's
// back); the gate and bookkeeping are retried briefly.
func FireWorkflow(ctx workflow.Context, in FireInput) error {
	short := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})
	firedAt := workflow.Now(ctx)
	for attempt := 0; attempt < 2; attempt++ {
		var g GateResult
		if err := workflow.ExecuteActivity(short, ActGate, in, firedAt).Get(ctx, &g); err != nil {
			return err
		}
		if g.Stop {
			return nil
		}
		if !g.Run && g.Decision != nil && g.Decision.Action == ActionWait {
			// Same local clock time on the next business day, then judge again.
			if d := g.Decision.WaitUntil.Sub(workflow.Now(ctx)); d > 0 {
				if err := workflow.Sleep(ctx, d); err != nil {
					return err
				}
			}
			firedAt = workflow.Now(ctx)
			continue
		}
		if !g.Run {
			return workflow.ExecuteActivity(short, ActRecord, in, firedAt, g).Get(ctx, nil)
		}
		long := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 12 * time.Hour,
			HeartbeatTimeout:    5 * time.Minute,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
		})
		return workflow.ExecuteActivity(long, ActRun, in, firedAt, g).Get(ctx, nil)
	}
	return nil
}

// Activities are the scheduler's Temporal activities.
type Activities struct {
	Store     *Store
	Calendars Calendars
	Runners   *Registry
}

// Gate loads the schedule and judges the firing against its calendar.
func (a *Activities) Gate(ctx context.Context, in FireInput, firedAt time.Time) (GateResult, error) {
	s, err := a.Store.Get(ctx, in.TenantID, in.ScheduleID)
	var me *msgcat.Error
	if errors.As(err, &me) && me.Set == SetSchedule && me.Nbr == 1 {
		return GateResult{Stop: true}, nil // deleted since the firing was planned
	}
	if err != nil {
		return GateResult{}, err
	}
	// An external trigger was accepted while the schedule was enabled (the
	// API refuses a paused one); the caller is waiting on its key, so it is
	// always recorded, never dropped.
	if !s.Enabled && !in.Manual && in.External == nil {
		return GateResult{Stop: true}, nil
	}
	g := GateResult{Run: true, Kind: s.Target.Kind, Ref: s.Target.Ref}
	if in.Manual || s.Timing.rule() == RuleNone {
		return g, nil
	}
	loc, _ := s.Timing.location()
	local := firedAt.In(loc)
	from := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 2, 0)
	days, err := a.Calendars.Days(ctx, s.TenantID, s.Timing.Calendar, from, to)
	if err != nil {
		return a.failedGate(ctx, s, in, firedAt, err)
	}
	d, err := s.Timing.Decide(firedAt, days)
	if err != nil {
		return a.failedGate(ctx, s, in, firedAt, err)
	}
	if d.Action == ActionWait {
		// The same local clock time on the business day.
		w := d.WaitUntil
		d.WaitUntil = time.Date(w.Year(), w.Month(), w.Day(), local.Hour(), local.Minute(), local.Second(), 0, loc)
	}
	g.Decision = &d
	g.Run = d.Action == ActionRun
	return g, nil
}

// failedGate records a firing the calendar could not judge (no data for the
// date, calendar missing) as a failed run: never a silent run or skip.
func (a *Activities) failedGate(ctx context.Context, s *Schedule, in FireInput, firedAt time.Time, cause error) (GateResult, error) {
	info := activity.GetInfo(ctx)
	runID, err := a.Store.StartRun(ctx, RunStart{TenantID: s.TenantID, ScheduleID: s.ID, Kind: s.Target.Kind, Ref: s.Target.Ref,
		Trigger: in.trigger(), TriggeredBy: in.TriggeredBy, External: in.External, ScheduledFor: firedAt, Status: "running",
		WorkflowID: info.WorkflowExecution.ID, WorkflowRunID: info.WorkflowExecution.RunID})
	if err != nil {
		return GateResult{}, err
	}
	code, params, detail := errorFields(cause, runID)
	if err := a.Store.FinishRun(ctx, s.TenantID, runID, nil, code, params, detail); err != nil {
		return GateResult{}, err
	}
	return GateResult{Stop: true}, nil
}

// RecordSkip records a firing the calendar skipped.
func (a *Activities) RecordSkip(ctx context.Context, in FireInput, firedAt time.Time, g GateResult) error {
	info := activity.GetInfo(ctx)
	reason := ""
	if g.Decision != nil {
		reason = g.Decision.Reason
	}
	_, err := a.Store.StartRun(ctx, RunStart{TenantID: in.TenantID, ScheduleID: in.ScheduleID, Kind: g.Kind, Ref: g.Ref,
		Trigger: in.trigger(), TriggeredBy: in.TriggeredBy, External: in.External, ScheduledFor: firedAt, Status: "skipped",
		SkipReason: reason, Decision: g.Decision,
		WorkflowID: info.WorkflowExecution.ID, WorkflowRunID: info.WorkflowExecution.RunID})
	return err
}

// Run records the run, dispatches it to the target's runner and records the
// result. A runner failure is recorded, not returned: the firing is done.
func (a *Activities) Run(ctx context.Context, in FireInput, firedAt time.Time, g GateResult) error {
	s, err := a.Store.Get(ctx, in.TenantID, in.ScheduleID)
	if err != nil {
		return err
	}
	info := activity.GetInfo(ctx)
	runID, err := a.Store.StartRun(ctx, RunStart{TenantID: s.TenantID, ScheduleID: s.ID, Kind: s.Target.Kind, Ref: s.Target.Ref,
		Trigger: in.trigger(), TriggeredBy: in.TriggeredBy, External: in.External, ScheduledFor: firedAt, Status: "running", Decision: g.Decision,
		WorkflowID: info.WorkflowExecution.ID, WorkflowRunID: info.WorkflowExecution.RunID})
	if err != nil {
		return err
	}
	out, runErr := a.dispatch(ctx, s, runID, firedAt)
	code, params, detail := "", []string(nil), ""
	if runErr != nil {
		code, params, detail = errorFields(runErr, runID)
		logging.GetLogger().Sugar().Warnw("scheduled run failed", "run_id", runID, "schedule_id", s.ID,
			"tenant", s.TenantID, "kind", s.Target.Kind, "code", code, "error", runErr)
	}
	return a.Store.FinishRun(ctx, s.TenantID, runID, out, code, params, detail)
}

func (a *Activities) dispatch(ctx context.Context, s *Schedule, runID string, firedAt time.Time) (out *Outcome, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = msgcat.Internal(runID)
			logging.GetLogger().Sugar().Errorw("scheduled run panicked", "run_id", runID, "panic", r)
		}
	}()
	runner, err := a.Runners.Get(s.Target.Kind)
	if err != nil {
		return nil, err
	}
	return runner.Run(ctx, RunContext{
		RunID: runID, ScheduleID: s.ID, TenantID: s.TenantID, OwnerID: s.OwnerID, DatasourceID: s.DatasourceID,
		Region: s.Region, Ref: s.Target.Ref, Params: s.Target.Params, ScheduledFor: firedAt,
		Heartbeat: func(d ...any) { activity.RecordHeartbeat(ctx, d...) },
	})
}

// errorFields splits an error into the catalog code and params users see
// and the internal detail operators see.
func errorFields(err error, runID string) (code string, params []string, detail string) {
	var me *msgcat.Error
	if !errors.As(err, &me) {
		me = msgcat.Internal(runID)
	}
	return me.Code(), me.Params, err.Error()
}
