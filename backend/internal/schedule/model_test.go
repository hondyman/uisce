package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// A London-style April 2026: Good Friday 3rd and Easter Monday 6th closed,
// weekends closed.
func april2026() Days {
	days := Days{}
	for d := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC); d.Before(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)); d = d.AddDate(0, 0, 1) {
		day := Day{Date: d, Business: d.Weekday() != time.Saturday && d.Weekday() != time.Sunday}
		switch dateKey(d) {
		case "2026-04-03":
			day.Business, day.Holiday = false, "Good Friday"
		case "2026-04-06":
			day.Business, day.Holiday = false, "Easter Monday"
		}
		days[dateKey(d)] = day
	}
	return days
}

func code(err error) string {
	var me *msgcat.Error
	if errors.As(err, &me) {
		return me.Code()
	}
	return ""
}

func TestValidate(t *testing.T) {
	ok := Timing{Cron: "0 18 * * 1-5", TimeZone: "Europe/London"}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		t    Timing
		code string
	}{
		"bad cron":          {Timing{Cron: "every day"}, "9200-2"},
		"every minute":      {Timing{Cron: "* * * * *"}, "9200-3"},
		"every 2 minutes":   {Timing{Cron: "*/2 9-17 * * 1-5"}, "9200-3"},
		"zone":              {Timing{Cron: "0 9 * * *", TimeZone: "Mars/Olympus"}, "9200-4"},
		"rule, no calendar": {Timing{Cron: "0 9 * * *", CalendarRule: RuleSkip}, "9200-6"},
		"bd zero":           {Timing{Cron: "0 9 * * *", Calendar: "XLON", CalendarRule: RuleBusinessDayOfMonth}, "9200-7"},
		"bd too far":        {Timing{Cron: "0 9 * * *", Calendar: "XLON", CalendarRule: RuleBusinessDayOfMonth, BusinessDay: 30}, "9200-7"},
		"unknown rule":      {Timing{Cron: "0 9 * * *", Calendar: "XLON", CalendarRule: "sometimes"}, "9200-16"},
	}
	for name, c := range cases {
		if got := code(c.t.Validate()); got != c.code {
			t.Errorf("%s: got %q, want %q", name, got, c.code)
		}
	}
	every5 := Timing{Cron: "*/5 * * * *"}
	if err := every5.Validate(); err != nil {
		t.Errorf("every 5 minutes is the floor, not below it: %v", err)
	}
}

func TestDecide_SkipAndWait(t *testing.T) {
	days := april2026()
	london, _ := time.LoadLocation("Europe/London")
	goodFriday := time.Date(2026, 4, 3, 18, 0, 0, 0, london)

	skip := Timing{Cron: "0 18 * * 1-5", TimeZone: "Europe/London", Calendar: "XLON", CalendarRule: RuleSkip}
	d, err := skip.Decide(goodFriday, days)
	if err != nil || d.Action != ActionSkip || d.Reason != "XLON is closed: Good Friday" {
		t.Fatalf("skip: %+v %v", d, err)
	}

	next := skip
	next.CalendarRule = RuleNextBusinessDay
	d, err = next.Decide(goodFriday, days)
	if err != nil || d.Action != ActionWait || dateKey(d.WaitUntil) != "2026-04-07" {
		t.Fatalf("next business day over Easter: %+v %v", d, err)
	}

	thursday := time.Date(2026, 4, 2, 18, 0, 0, 0, london)
	if d, _ := next.Decide(thursday, days); d.Action != ActionRun {
		t.Fatalf("a business day runs: %+v", d)
	}
}

// The date judged is the date in the schedule's zone, not UTC.
func TestDecide_UsesScheduleZone(t *testing.T) {
	days := april2026()
	ny := Timing{Cron: "0 21 * * *", TimeZone: "America/New_York", Calendar: "XNYS", CalendarRule: RuleSkip}
	// 21:00 New York on Thursday 2 April is already Friday 3 April in UTC.
	at := time.Date(2026, 4, 3, 1, 0, 0, 0, time.UTC)
	d, err := ny.Decide(at, days)
	if err != nil || d.Date != "2026-04-02" || d.Action != ActionRun {
		t.Fatalf("%+v %v", d, err)
	}
}

func TestDecide_BusinessDayOfMonth(t *testing.T) {
	days := april2026()
	third := Timing{Cron: "0 7 * * *", Calendar: "XLON", CalendarRule: RuleBusinessDayOfMonth, BusinessDay: 3}
	// April 2026 business days: 1 (Wed), 2 (Thu), 7 (Tue) - Good Friday,
	// the weekend and Easter Monday are closed.
	runs := []string{}
	for d := 1; d <= 30; d++ {
		dec, err := third.Decide(time.Date(2026, 4, d, 7, 0, 0, 0, time.UTC), days)
		if err != nil {
			t.Fatal(err)
		}
		if dec.Action == ActionRun {
			runs = append(runs, dec.Date)
		}
	}
	if len(runs) != 1 || runs[0] != "2026-04-07" {
		t.Fatalf("BD3 of April 2026 = %v, want [2026-04-07]", runs)
	}

	last := third
	last.BusinessDay = -1
	runs = runs[:0]
	for d := 1; d <= 30; d++ {
		dec, _ := last.Decide(time.Date(2026, 4, d, 7, 0, 0, 0, time.UTC), days)
		if dec.Action == ActionRun {
			runs = append(runs, dec.Date)
		}
	}
	if len(runs) != 1 || runs[0] != "2026-04-30" {
		t.Fatalf("last BD of April 2026 = %v, want [2026-04-30]", runs)
	}
}

// A date the calendar does not cover fails loudly: never a silent run.
func TestDecide_CalendarGapIsAnError(t *testing.T) {
	skip := Timing{Cron: "0 9 * * *", Calendar: "XLON", CalendarRule: RuleSkip}
	_, err := skip.Decide(time.Date(2031, 1, 2, 9, 0, 0, 0, time.UTC), april2026())
	if code(err) != "9200-11" {
		t.Fatalf("err = %v", err)
	}
	none := Timing{Cron: "0 9 * * *"}
	if d, err := none.Decide(time.Date(2031, 1, 2, 9, 0, 0, 0, time.UTC), nil); err != nil || d.Action != ActionRun {
		t.Fatalf("no calendar rule needs no calendar: %+v %v", d, err)
	}
}

type fakeCalendars struct{ days Days }

func (f fakeCalendars) Days(context.Context, string, string, time.Time, time.Time) (Days, error) {
	return f.days, nil
}
func (f fakeCalendars) List(context.Context, string) ([]CalendarInfo, error) { return nil, nil }

func TestPreview_ShowsWhatWillHappen(t *testing.T) {
	s := &Service{Calendars: fakeCalendars{april2026()}, Now: func() time.Time { return time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC) }}
	up, err := s.Preview(context.Background(), "t", Timing{Cron: "0 18 * * 1-5", TimeZone: "UTC", Calendar: "XLON", CalendarRule: RuleNextBusinessDay}, 5)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, u := range up {
		s := dateKey(u.At) + ":" + u.Action
		if u.Action == ActionWait {
			s += "->" + dateKey(u.RunsAt)
		}
		got = append(got, s)
	}
	want := []string{"2026-04-01:run", "2026-04-02:run", "2026-04-03:wait->2026-04-07", "2026-04-06:wait->2026-04-07", "2026-04-07:run"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
