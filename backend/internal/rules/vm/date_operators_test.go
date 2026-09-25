package vm

import (
	"testing"
	"time"
)

func TestDateOperators(t *testing.T) {
	// Wednesday 2026-09-23 15:00 UTC (ISO week 39).
	fixed := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	now = func() time.Time { return fixed }
	defer func() { now = time.Now }()

	data := map[string]interface{}{
		"hired":     "2020-01-15",
		"ts":        "2026-09-23T09:30:00Z",
		"ts_offset": "2026-09-23T23:30:00-05:00", // 04:30 UTC on the 24th
		"monday":    "2026-09-21",
		"lastweek":  "2026-09-20 12:00:00",
		"soon":      "2026-09-25",
		"typed":     fixed.Add(-time.Hour),
		"junk":      "not a date",
	}
	for _, tc := range []struct {
		name  string
		node  RuleNode
		want  bool
		isErr bool
	}{
		{"before", cond("hired", "before", "2020-02-01"), true, false},
		{"after", cond("hired", "after", "2020-02-01"), false, false},
		{"on_or_before same instant", cond("hired", "on_or_before", "2020-01-15"), true, false},
		{"on_or_after same instant", cond("hired", "on_or_after", "2020-01-15T00:00:00Z"), true, false},
		{"after with offset", cond("ts_offset", "after", "2026-09-24"), true, false},
		{"time.Time value", cond("typed", "before", "2026-09-23T15:00:00Z"), true, false},
		{"non-date value fails", cond("junk", "before", "2030-01-01"), false, false},
		{"non-date bound is an error", cond("hired", "before", "someday"), false, true},
		{"in_last_n_days inside", cond("monday", "in_last_n_days", 7), true, false},
		{"in_last_n_days outside", cond("hired", "in_last_n_days", "30"), false, false},
		{"in_last_n_days excludes future", cond("soon", "in_last_n_days", 7), false, false},
		{"in_next_n_days", cond("soon", "in_next_n_days", 3), true, false},
		{"in_next_n_days excludes past", cond("monday", "in_next_n_days", 30), false, false},
		{"negative day count is an error", cond("soon", "in_next_n_days", -1), false, true},
		{"is_today", cond("ts", "is_today", nil), true, false},
		{"is_today uses UTC", cond("ts_offset", "is_today", nil), false, false},
		{"is_this_week from Monday", cond("monday", "is_this_week", nil), true, false},
		{"is_this_week excludes prior Sunday", cond("lastweek", "is_this_week", nil), false, false},
		{"is_this_month", cond("monday", "is_this_month", nil), true, false},
		{"is_this_year", cond("hired", "is_this_year", nil), false, false},
		{"absent field fails", cond("missing", "is_today", nil), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewAdvancedEvaluator().Evaluate(tc.node, data)
			if (err != nil) != tc.isErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.isErr)
			}
			if !tc.isErr && got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
