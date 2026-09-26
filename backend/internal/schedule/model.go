// Package schedule is the one scheduler: one model for everything that
// runs on a timetable (reports, saved queries, data pipelines, ...), one
// engine (Temporal Schedules), business-calendar rules evaluated when a
// schedule fires, and one run history.
package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// Calendar rules: what a firing does on a day that is not a business day
// of the schedule's calendar.
const (
	RuleNone               = "none"                  // ignore calendars
	RuleSkip               = "skip"                  // don't run on non-business days
	RuleNextBusinessDay    = "next_business_day"     // run at the same time on the next business day
	RuleBusinessDayOfMonth = "business_day_of_month" // run only on business day N of the month (-1 = last)
)

// MinInterval is the shortest time between two firings a schedule may ask for.
const MinInterval = 5 * time.Minute

// Timing is when a schedule fires.
type Timing struct {
	Cron         string     `json:"cron"`      // minute hour day-of-month month day-of-week
	TimeZone     string     `json:"time_zone"` // IANA, e.g. America/New_York
	Calendar     string     `json:"calendar,omitempty"`
	CalendarRule string     `json:"calendar_rule,omitempty"`
	BusinessDay  int        `json:"business_day,omitempty"`
	StartAt      *time.Time `json:"start_at,omitempty"`
	EndAt        *time.Time `json:"end_at,omitempty"`
}

// Target is what a schedule runs: a kind with a runner, and the thing's id.
type Target struct {
	Kind   string         `json:"kind"`
	Ref    string         `json:"ref"`
	Params map[string]any `json:"params,omitempty"`
}

// Schedule is one scheduled target.
type Schedule struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Target       Target    `json:"target"`
	Timing       Timing    `json:"timing"`
	Enabled      bool      `json:"enabled"`
	OwnerID      string    `json:"owner_id"`
	DatasourceID string    `json:"datasource_id,omitempty"`
	Region       string    `json:"region,omitempty"`
	Version      int       `json:"version"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedBy    string    `json:"updated_by,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func (t Timing) location() (*time.Location, error) {
	if t.TimeZone == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(t.TimeZone)
	if err != nil {
		return nil, msgUnknownTimeZone(t.TimeZone)
	}
	return loc, nil
}

func (t Timing) rule() string {
	if t.CalendarRule == "" {
		return RuleNone
	}
	return t.CalendarRule
}

// Validate checks everything that can be checked without a database.
func (t Timing) Validate() error {
	if _, err := t.location(); err != nil {
		return err
	}
	sched, err := cronParser.Parse(strings.TrimSpace(t.Cron))
	if err != nil {
		return msgBadCron(t.Cron).Wrap(err)
	}
	// The shortest gap over a varied stretch of firings.
	at := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 200; i++ {
		n := sched.Next(at)
		if i > 0 && n.Sub(at) < MinInterval {
			return msgTooFrequent()
		}
		at = n
	}
	switch t.rule() {
	case RuleNone:
	case RuleSkip, RuleNextBusinessDay:
		if t.Calendar == "" {
			return msgRuleNeedsCalendar()
		}
	case RuleBusinessDayOfMonth:
		if t.Calendar == "" {
			return msgRuleNeedsCalendar()
		}
		if t.BusinessDay == 0 || t.BusinessDay < -23 || t.BusinessDay > 23 {
			return msgBadBusinessDay()
		}
	default:
		return msgBadRule(t.CalendarRule)
	}
	if t.StartAt != nil && t.EndAt != nil && !t.EndAt.After(*t.StartAt) {
		return msgEndBeforeStart()
	}
	return nil
}

// Fires returns the next n cron firings after from, in the schedule's zone
// (calendar rules not applied).
func (t Timing) Fires(n int, from time.Time) ([]time.Time, error) {
	loc, err := t.location()
	if err != nil {
		return nil, err
	}
	sched, err := cronParser.Parse(strings.TrimSpace(t.Cron))
	if err != nil {
		return nil, msgBadCron(t.Cron).Wrap(err)
	}
	out := make([]time.Time, 0, n)
	at := from.In(loc)
	for len(out) < n {
		at = sched.Next(at)
		if t.EndAt != nil && at.After(*t.EndAt) {
			break
		}
		if t.StartAt != nil && at.Before(*t.StartAt) {
			continue
		}
		out = append(out, at)
	}
	return out, nil
}

// --- calendar rules -------------------------------------------------------

// Day is one calendar day as the schedule sees it: the core calendar with
// the tenant's own layer applied.
type Day struct {
	Date     time.Time // midnight UTC of the date
	Business bool
	HalfDay  bool
	Holiday  string // why it is not a business day, when known
}

// Days is a calendar's days by ISO date ("2026-12-28").
type Days map[string]Day

func dateKey(t time.Time) string { return t.Format("2006-01-02") }

// Decision is what a firing does.
type Decision struct {
	Action    string    `json:"action"` // run | skip | wait
	Date      string    `json:"date"`   // the date judged, in the schedule's zone
	WaitUntil time.Time `json:"wait_until,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	Calendar  string    `json:"calendar,omitempty"`
}

const (
	ActionRun  = "run"
	ActionSkip = "skip"
	ActionWait = "wait"
)

// Decide judges a firing at `at` (any zone) against the calendar days.
// Days must cover at's month and the following weeks; a date the calendar
// does not cover is an error, never a silent run.
func (t Timing) Decide(at time.Time, days Days) (Decision, error) {
	loc, err := t.location()
	if err != nil {
		return Decision{}, err
	}
	local := at.In(loc)
	key := dateKey(local)
	d := Decision{Action: ActionRun, Date: key, Calendar: t.Calendar}
	if t.rule() == RuleNone {
		return d, nil
	}
	day, ok := days[key]
	if !ok {
		return d, msgCalendarGap(t.Calendar, key)
	}
	switch t.rule() {
	case RuleSkip:
		if !day.Business {
			d.Action, d.Reason = ActionSkip, notBusinessReason(t.Calendar, day)
		}
	case RuleNextBusinessDay:
		if !day.Business {
			next := local
			for i := 0; i < 31; i++ {
				next = next.AddDate(0, 0, 1)
				nd, ok := days[dateKey(next)]
				if !ok {
					return d, msgCalendarGap(t.Calendar, dateKey(next))
				}
				if nd.Business {
					d.Action, d.WaitUntil = ActionWait, next
					d.Reason = notBusinessReason(t.Calendar, day) + "; runs on the next business day, " + dateKey(next)
					return d, nil
				}
			}
			return d, msgCalendarGap(t.Calendar, dateKey(next))
		}
	case RuleBusinessDayOfMonth:
		n, err := businessDayOfMonth(local, days, t.BusinessDay < 0)
		if err != nil {
			return d, msgCalendarGap(t.Calendar, err.Error())
		}
		if !day.Business || n != abs(t.BusinessDay) {
			d.Action = ActionSkip
			d.Reason = fmt.Sprintf("not business day %d of the month on %s", t.BusinessDay, t.Calendar)
		}
	}
	return d, nil
}

// businessDayOfMonth is the day's business-day number within its month,
// counted from the start, or (fromEnd) from the end: the last is 1.
// It returns 0 for a non-business day.
func businessDayOfMonth(local time.Time, days Days, fromEnd bool) (int, error) {
	// Compare dates, not clock times: the firing's own day always counts.
	local = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	first := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, local.Location())
	last := first.AddDate(0, 1, -1)
	n, found := 0, 0
	if !fromEnd {
		for d := first; !d.After(local); d = d.AddDate(0, 0, 1) {
			day, ok := days[dateKey(d)]
			if !ok {
				return 0, fmt.Errorf("%s", dateKey(d))
			}
			if day.Business {
				n++
			}
		}
		found = n
	} else {
		for d := last; !d.Before(local); d = d.AddDate(0, 0, -1) {
			day, ok := days[dateKey(d)]
			if !ok {
				return 0, fmt.Errorf("%s", dateKey(d))
			}
			if day.Business {
				n++
			}
		}
		found = n
	}
	if day := days[dateKey(local)]; !day.Business {
		return 0, nil
	}
	return found, nil
}

func notBusinessReason(cal string, d Day) string {
	if d.Holiday != "" {
		return fmt.Sprintf("%s is closed: %s", cal, d.Holiday)
	}
	return fmt.Sprintf("not a business day on %s", cal)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
