package schedule

import (
	"context"
	"errors"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// TaskQueue is where the scheduler's workflows and activities run.
const TaskQueue = "uisce-schedules"

// Engine keeps live timetables in step with stored schedules.
type Engine interface {
	Apply(ctx context.Context, s *Schedule) error // enabled: create/update; disabled: pause
	Remove(ctx context.Context, tenantID, id string) error
	RunNow(ctx context.Context, s *Schedule, actor string) (string, error)
}

// EngineID is the Temporal schedule id of a schedule.
func EngineID(tenantID, id string) string { return "schedule-" + tenantID + "-" + id }

// TemporalEngine backs schedules with Temporal Schedules: exactly-once
// firing across any number of API replicas, a firing that would overlap a
// still-running one is skipped.
type TemporalEngine struct{ Client client.Client }

func (e *TemporalEngine) spec(s *Schedule) client.ScheduleSpec {
	spec := client.ScheduleSpec{CronExpressions: []string{s.Timing.Cron}, TimeZoneName: s.Timing.TimeZone}
	if s.Timing.StartAt != nil {
		spec.StartAt = *s.Timing.StartAt
	}
	if s.Timing.EndAt != nil {
		spec.EndAt = *s.Timing.EndAt
	}
	return spec
}

func (e *TemporalEngine) action(s *Schedule) *client.ScheduleWorkflowAction {
	return &client.ScheduleWorkflowAction{
		ID:        "schedule-run-" + s.ID,
		Workflow:  FireWorkflow,
		Args:      []any{FireInput{TenantID: s.TenantID, ScheduleID: s.ID}},
		TaskQueue: TaskQueue,
	}
}

func (e *TemporalEngine) Apply(ctx context.Context, s *Schedule) error {
	if e == nil || e.Client == nil {
		return msgEngineUnavailable()
	}
	id := EngineID(s.TenantID, s.ID)
	spec, action := e.spec(s), e.action(s)
	_, err := e.Client.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: id, Spec: spec, Action: action, Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP, Paused: !s.Enabled,
		Note: s.Name,
	})
	if err == nil {
		return nil
	}
	var exists *serviceerror.AlreadyExists
	if !errors.As(err, &exists) && !errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return msgEngineUnavailable().Wrap(err)
	}
	err = e.Client.ScheduleClient().GetHandle(ctx, id).Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(in client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			sc := in.Description.Schedule
			sc.Spec, sc.Action = &spec, action
			if sc.Policy == nil {
				sc.Policy = &client.SchedulePolicies{}
			}
			sc.Policy.Overlap = enums.SCHEDULE_OVERLAP_POLICY_SKIP
			if sc.State == nil {
				sc.State = &client.ScheduleState{}
			}
			sc.State.Paused = !s.Enabled
			sc.State.Note = s.Name
			return &client.ScheduleUpdate{Schedule: &sc}, nil
		},
	})
	if err != nil {
		return msgEngineUnavailable().Wrap(err)
	}
	return nil
}

func (e *TemporalEngine) Remove(ctx context.Context, tenantID, id string) error {
	if e == nil || e.Client == nil {
		return msgEngineUnavailable()
	}
	err := e.Client.ScheduleClient().GetHandle(ctx, EngineID(tenantID, id)).Delete(ctx)
	var nf *serviceerror.NotFound
	if err != nil && !errors.As(err, &nf) {
		return msgEngineUnavailable().Wrap(err)
	}
	return nil
}

// RunNow starts one run immediately, outside the timetable and its calendar
// rule (someone asked for it).
func (e *TemporalEngine) RunNow(ctx context.Context, s *Schedule, actor string) (string, error) {
	if e == nil || e.Client == nil {
		return "", msgEngineUnavailable()
	}
	wid := "schedule-run-" + s.ID + "-manual-" + time.Now().UTC().Format("20060102T150405.000")
	run, err := e.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: wid, TaskQueue: TaskQueue},
		FireWorkflow, FireInput{TenantID: s.TenantID, ScheduleID: s.ID, Manual: true, TriggeredBy: actor})
	if err != nil {
		return "", msgEngineUnavailable().Wrap(err)
	}
	return run.GetID(), nil
}
