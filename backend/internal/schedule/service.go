package schedule

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// Service is the scheduler's operations, used by the API and MCP tools.
type Service struct {
	Store     *Store
	Engine    Engine
	Calendars Calendars
	Runners   *Registry
	Now       func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Actor is who is acting, from the authenticated token.
type Actor struct {
	UserID       string
	TenantID     string
	DatasourceID string
	Region       string
	// Machine is an enterprise scheduler's service account: it may read and
	// trigger schedules, never change them.
	Machine bool
	// CanTrigger: people always; a service account only with the
	// schedule_trigger role.
	CanTrigger bool
}

// Input is a schedule as a caller writes it.
type Input struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Target      Target `json:"target"`
	Timing      Timing `json:"timing"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

func (s *Service) check(ctx context.Context, tenantID, userID string, in Input) error {
	if strings.TrimSpace(in.Name) == "" {
		return msgNameRequired()
	}
	if in.Timing.TimeZone == "" {
		in.Timing.TimeZone = "UTC"
	}
	if err := in.Timing.Validate(); err != nil {
		return err
	}
	runner, err := s.Runners.Get(in.Target.Kind)
	if err != nil {
		return err
	}
	if err := runner.Check(ctx, tenantID, userID, in.Target.Ref, in.Target.Params); err != nil {
		return err
	}
	if in.Timing.Calendar != "" {
		// The calendar must exist for the tenant (core or its own).
		today := s.now().UTC()
		if _, err := s.Calendars.Days(ctx, tenantID, in.Timing.Calendar, today, today); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Create(ctx context.Context, a Actor, in Input) (*Schedule, error) {
	if in.Timing.TimeZone == "" {
		in.Timing.TimeZone = "UTC"
	}
	if err := s.check(ctx, a.TenantID, a.UserID, in); err != nil {
		return nil, err
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	sc, err := s.Store.Create(ctx, &Schedule{
		TenantID: a.TenantID, Name: strings.TrimSpace(in.Name), Description: in.Description, Target: in.Target,
		Timing: in.Timing, Enabled: enabled, OwnerID: a.UserID, DatasourceID: a.DatasourceID, Region: a.Region,
	}, a.UserID)
	if err != nil {
		return nil, err
	}
	if err := s.Engine.Apply(ctx, sc); err != nil {
		// Keep store and engine in step: a schedule the engine refused is removed.
		if _, derr := s.Store.Delete(ctx, a.TenantID, sc.ID, "system:rollback"); derr != nil {
			logging.GetLogger().Sugar().Errorw("schedule rollback failed", "schedule_id", sc.ID, "error", derr)
		}
		return nil, err
	}
	return sc, nil
}

func (s *Service) Update(ctx context.Context, a Actor, id string, in Input) (*Schedule, error) {
	if in.Timing.TimeZone == "" {
		in.Timing.TimeZone = "UTC"
	}
	if err := s.check(ctx, a.TenantID, a.UserID, in); err != nil {
		return nil, err
	}
	before, after, err := s.Store.Update(ctx, a.TenantID, id, "update", a.UserID, func(sc *Schedule) error {
		sc.Name, sc.Description, sc.Target, sc.Timing = strings.TrimSpace(in.Name), in.Description, in.Target, in.Timing
		if in.Enabled != nil {
			sc.Enabled = *in.Enabled
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return after, s.applyOrRevert(ctx, a, before, after)
}

// SetEnabled pauses or resumes.
func (s *Service) SetEnabled(ctx context.Context, a Actor, id string, enabled bool) (*Schedule, error) {
	action := "pause"
	if enabled {
		action = "resume"
	}
	before, after, err := s.Store.Update(ctx, a.TenantID, id, action, a.UserID, func(sc *Schedule) error {
		sc.Enabled = enabled
		return nil
	})
	if err != nil {
		return nil, err
	}
	return after, s.applyOrRevert(ctx, a, before, after)
}

// applyOrRevert pushes a change to the engine, and if the engine refuses it
// restores the stored schedule, so what is stored is what fires.
func (s *Service) applyOrRevert(ctx context.Context, a Actor, before, after *Schedule) error {
	err := s.Engine.Apply(ctx, after)
	if err == nil {
		return nil
	}
	if _, _, rerr := s.Store.Update(ctx, a.TenantID, before.ID, "revert", "system:rollback", func(sc *Schedule) error {
		sc.Name, sc.Description, sc.Target, sc.Timing, sc.Enabled = before.Name, before.Description, before.Target, before.Timing, before.Enabled
		return nil
	}); rerr != nil {
		logging.GetLogger().Sugar().Errorw("schedule revert failed", "schedule_id", before.ID, "error", rerr)
	}
	return err
}

func (s *Service) Delete(ctx context.Context, a Actor, id string) error {
	if _, err := s.Store.Get(ctx, a.TenantID, id); err != nil {
		return err
	}
	if err := s.Engine.Remove(ctx, a.TenantID, id); err != nil {
		return err
	}
	_, err := s.Store.Delete(ctx, a.TenantID, id, a.UserID)
	return err
}

func (s *Service) RunNow(ctx context.Context, a Actor, id string) (string, error) {
	sc, err := s.Store.Get(ctx, a.TenantID, id)
	if err != nil {
		return "", err
	}
	return s.Engine.RunNow(ctx, sc, a.UserID)
}

// TriggerStatus is where an external trigger stands: queued until its run
// is recorded, then the run's own status.
type TriggerStatus struct {
	ScheduleID     string `json:"schedule_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Status         string `json:"status"` // queued | running | succeeded | failed | skipped
	Run            *Run   `json:"run,omitempty"`
}

// Done reports whether the run has finished (or was skipped).
func (t TriggerStatus) Done() bool {
	return t.Status == "succeeded" || t.Status == "failed" || t.Status == "skipped"
}

const maxKey = 200

// Trigger fires an externally triggered schedule on an enterprise
// scheduler's request. The key makes it idempotent: a repeat returns where
// the first trigger stands and starts nothing.
func (s *Service) Trigger(ctx context.Context, a Actor, id string, t ExternalTrigger) (*TriggerStatus, error) {
	t.IdempotencyKey = strings.TrimSpace(t.IdempotencyKey)
	if t.IdempotencyKey == "" || len(t.IdempotencyKey) > maxKey {
		return nil, msgNeedKey()
	}
	t.System, t.Ref = clip(strings.TrimSpace(t.System), 100), clip(strings.TrimSpace(t.Ref), 200)
	sc, err := s.Store.Get(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	if st, err := s.triggerStatus(ctx, a, sc.ID, t.IdempotencyKey); err == nil {
		return st, nil // already triggered with this key
	} else if !isNoTrigger(err) {
		return nil, err
	}
	if !sc.Timing.External() {
		return nil, msgNotExternal(sc.ID)
	}
	if !sc.Enabled {
		return nil, msgPaused(sc.ID)
	}
	if err := s.Engine.Trigger(ctx, sc, a.UserID, t); err != nil {
		return nil, err
	}
	return &TriggerStatus{ScheduleID: sc.ID, IdempotencyKey: t.IdempotencyKey, Status: "queued"}, nil
}

// TriggerStatusOf is where the trigger with key stands. With wait > 0 it
// waits (up to wait) for the run to finish - a long poll for callers that
// block on the result, like the uisce-job CLI.
func (s *Service) TriggerStatusOf(ctx context.Context, a Actor, id, key string, wait time.Duration) (*TriggerStatus, error) {
	deadline := time.Now().Add(wait)
	for {
		st, err := s.triggerStatus(ctx, a, id, key)
		if err != nil || st.Done() || !time.Now().Before(deadline) {
			return st, err
		}
		select {
		case <-ctx.Done():
			return st, nil
		case <-time.After(time.Second):
		}
	}
}

func (s *Service) triggerStatus(ctx context.Context, a Actor, id, key string) (*TriggerStatus, error) {
	st := &TriggerStatus{ScheduleID: id, IdempotencyKey: key, Status: "queued"}
	run, err := s.Store.RunByKey(ctx, a.TenantID, id, key)
	if err != nil {
		return nil, err
	}
	if run != nil {
		st.Status, st.Run = run.Status, run
		return st, nil
	}
	// No run recorded: queued if the trigger's workflow is still going;
	// never started means the key is unknown; ended without recording a run
	// (it could not start) is a failure the caller must see, not a wait.
	state, err := s.Engine.TriggerState(ctx, id, key)
	if err != nil {
		return nil, err
	}
	switch state {
	case TriggerNone:
		return nil, msgNoTrigger(id, key)
	case TriggerClosed:
		st.Status = "failed"
	}
	return st, nil
}

func isNoTrigger(err error) bool {
	var me *msgcat.Error
	return errors.As(err, &me) && me.Set == SetSchedule && me.Nbr == 23
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Upcoming is one planned firing and what the calendar will make of it.
type Upcoming struct {
	At      time.Time `json:"at"`
	Action  string    `json:"action"` // run | skip | wait
	RunsAt  time.Time `json:"runs_at,omitempty"`
	Reason  string    `json:"reason,omitempty"`
	HalfDay bool      `json:"half_day,omitempty"`
}

// Preview lists the next n firings of a timing with the calendar applied,
// so an editor can show exactly what will happen.
func (s *Service) Preview(ctx context.Context, tenantID string, t Timing, n int) ([]Upcoming, error) {
	if t.TimeZone == "" {
		t.TimeZone = "UTC"
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	if n <= 0 || n > 50 {
		n = 10
	}
	// business_day_of_month keeps only a few firings a year from a daily
	// cron: look further ahead for it.
	scan := n
	if t.rule() == RuleBusinessDayOfMonth || t.rule() == RuleSkip {
		scan = n * 40
	}
	fires, err := t.Fires(scan, s.now())
	if err != nil {
		return nil, err
	}
	var days Days
	if t.rule() != RuleNone && len(fires) > 0 {
		first, last := fires[0], fires[len(fires)-1]
		from := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(last.Year(), last.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 2, 0)
		if days, err = s.Calendars.Days(ctx, tenantID, t.Calendar, from, to); err != nil {
			return nil, err
		}
	}
	loc, _ := t.location()
	out := []Upcoming{}
	for _, f := range fires {
		d, err := t.Decide(f, days)
		if err != nil {
			return out, err
		}
		if t.rule() == RuleBusinessDayOfMonth && d.Action == ActionSkip {
			continue // not a planned run; the list shows the runs
		}
		u := Upcoming{At: f, Action: d.Action, Reason: d.Reason}
		if d.Action == ActionWait {
			w := d.WaitUntil
			u.RunsAt = time.Date(w.Year(), w.Month(), w.Day(), f.Hour(), f.Minute(), 0, 0, loc)
		}
		if day, ok := days[d.Date]; ok {
			u.HalfDay = day.HalfDay
		}
		out = append(out, u)
		if len(out) == n {
			break
		}
	}
	return out, nil
}
