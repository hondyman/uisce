package datapipeline

import (
	"strings"
	"testing"
	"time"
)

func TestScheduleValidateAndNext(t *testing.T) {
	for _, c := range []struct {
		s       Schedule
		wantErr string
	}{
		{Schedule{Cron: "0 6 * * 1-5", TimeZone: "Europe/Dublin"}, ""},
		{Schedule{Cron: "*/15 * * * *"}, ""},
		{Schedule{Cron: "* * * * *"}, "at most every 5 minutes"},
		{Schedule{Cron: "0 6 * *"}, "not a valid 5-field cron"},
		{Schedule{Cron: "0 6 * * *", TimeZone: "Mars/Olympus"}, "unknown time zone"},
	} {
		err := c.s.Validate()
		if (c.wantErr == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), c.wantErr)) {
			t.Errorf("%+v: err = %v, want %q", c.s, err, c.wantErr)
		}
	}
	// Weekdays 06:00 Dublin, from Friday 2026-09-25 12:00 UTC: Mon 28, Tue 29 at 06:00 local (05:00 UTC, IST).
	next, err := Schedule{Cron: "0 6 * * 1-5", TimeZone: "Europe/Dublin"}.Next(2, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if next[0].UTC() != time.Date(2026, 9, 28, 5, 0, 0, 0, time.UTC) || next[1].UTC() != time.Date(2026, 9, 29, 5, 0, 0, 0, time.UTC) {
		t.Errorf("next = %v", next)
	}
}
