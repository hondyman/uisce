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

// --- scheduling --------------------------------------------------------------

// Scheduler keeps the live schedule in step with a pipeline's Schedule. It
// is implemented on the one scheduler (internal/schedule) by the API layer;
// pipelines no longer own Temporal schedules of their own.
type Scheduler interface {
	Apply(ctx context.Context, tenantID, pipelineID string, sc *Schedule) error // nil: pipeline deleted; disabled: paused
}
