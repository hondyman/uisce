package schedule

import (
	"context"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/logging"
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
