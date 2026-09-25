package vm

import (
	"fmt"
	"strings"
	"time"
)

// now is the clock for relative date operators; tests replace it.
var now = time.Now

// compareDate implements the rule editor's date operators. Calendar
// operators (is_today, is_this_week, ...) use UTC, so the server and the
// browser (rule_engine.wasm) agree regardless of where they run; weeks are
// ISO weeks (Monday first). A value that is not a date fails the condition;
// an unparseable bound or day count is a rule error.
func compareDate(actual interface{}, operator string, expected interface{}) (ok bool, handled bool, err error) {
	switch operator {
	case "before", "after", "on_or_before", "on_or_after":
		bound, bok := toTime(expected)
		if !bok {
			return false, true, fmt.Errorf("%s: %v is not a date", operator, expected)
		}
		t, tok := toTime(actual)
		if !tok {
			return false, true, nil
		}
		switch operator {
		case "before":
			return t.Before(bound), true, nil
		case "after":
			return t.After(bound), true, nil
		case "on_or_before":
			return !t.After(bound), true, nil
		default:
			return !t.Before(bound), true, nil
		}

	case "in_last_n_days", "in_next_n_days":
		n, nok := toFloatAny(expected)
		if !nok || n < 0 {
			return false, true, fmt.Errorf("%s: day count must be a non-negative number, got %v", operator, expected)
		}
		t, tok := toTime(actual)
		if !tok {
			return false, true, nil
		}
		cur := now().UTC()
		span := time.Duration(n * float64(24*time.Hour))
		if operator == "in_last_n_days" {
			return !t.Before(cur.Add(-span)) && !t.After(cur), true, nil
		}
		return !t.Before(cur) && !t.After(cur.Add(span)), true, nil

	case "is_today", "is_this_week", "is_this_month", "is_this_year":
		t, tok := toTime(actual)
		if !tok {
			return false, true, nil
		}
		t, cur := t.UTC(), now().UTC()
		switch operator {
		case "is_today":
			return t.Year() == cur.Year() && t.YearDay() == cur.YearDay(), true, nil
		case "is_this_week":
			ty, tw := t.ISOWeek()
			cy, cw := cur.ISOWeek()
			return ty == cy && tw == cw, true, nil
		case "is_this_month":
			return t.Year() == cur.Year() && t.Month() == cur.Month(), true, nil
		default:
			return t.Year() == cur.Year(), true, nil
		}
	}
	return false, false, nil
}

// dateLayouts are the accepted string forms; a form without a zone is UTC
// (as a date-only string is in JavaScript's Date).
var dateLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func toTime(v interface{}) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x, true
	case string:
		s := strings.TrimSpace(x)
		for _, layout := range dateLayouts {
			if t, err := time.Parse(layout, s); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}
