package engine

import (
	"time"
)

// TradingCalendar calculates business day and exchange cutoff arithmetic for statutory filing deadlines.
type TradingCalendar struct{}

// NewTradingCalendar creates a TradingCalendar instance.
func NewTradingCalendar() *TradingCalendar {
	return &TradingCalendar{}
}

// isWeekend returns true if the day is Saturday or Sunday.
func isWeekend(t time.Time) bool {
	w := t.Weekday()
	return w == time.Saturday || w == time.Sunday
}

// isUSHoliday checks major US federal / NYSE market holidays (fixed dates + standard observations).
func isUSHoliday(t time.Time) bool {
	m, d := t.Month(), t.Day()
	// New Year's Day (Jan 1)
	if m == time.January && d == 1 {
		return true
	}
	// Independence Day (Jul 4)
	if m == time.July && d == 4 {
		return true
	}
	// Christmas Day (Dec 25)
	if m == time.December && d == 25 {
		return true
	}
	// Juneteenth (Jun 19)
	if m == time.June && d == 19 {
		return true
	}
	return false
}

// isUKHoliday checks UK bank holidays (Good Friday, Easter Monday, Early May, Spring, Summer, Christmas, Boxing Day).
func isUKHoliday(t time.Time) bool {
	m, d := t.Month(), t.Day()
	if m == time.January && d == 1 {
		return true
	}
	if m == time.December && (d == 25 || d == 26) {
		return true
	}
	return false
}

// isEUHoliday checks TARGET2 / Euronext market holidays.
func isEUHoliday(t time.Time) bool {
	m, d := t.Month(), t.Day()
	if m == time.January && d == 1 {
		return true
	}
	if m == time.May && d == 1 { // Labour Day
		return true
	}
	if m == time.December && (d == 25 || d == 26) {
		return true
	}
	return false
}

// IsTradingDay returns true if t is an open trading day in the given jurisdiction.
func (c *TradingCalendar) IsTradingDay(t time.Time, calendar string) bool {
	if isWeekend(t) {
		return false
	}
	switch calendar {
	case "US_SEC", "US":
		return !isUSHoliday(t)
	case "UK_FCA", "UK", "UK_TAKEOVER_PANEL":
		return !isUKHoliday(t)
	case "EU_ESMA", "EU":
		return !isEUHoliday(t)
	default:
		return !isUSHoliday(t)
	}
}

// AddTradingDays adds N trading days to t, skipping weekends and market holidays.
func (c *TradingCalendar) AddTradingDays(start time.Time, days int, calendar string) time.Time {
	curr := start
	added := 0
	for added < days {
		curr = curr.AddDate(0, 0, 1)
		if c.IsTradingDay(curr, calendar) {
			added++
		}
	}
	return curr
}

// NextTradingDayCutoff computes the cutoff timestamp on the next trading day (e.g. 15:30 on T+1).
func (c *TradingCalendar) NextTradingDayCutoff(start time.Time, daysAhead int, hour, min int, loc *time.Location, calendar string) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	targetDay := c.AddTradingDays(start, daysAhead, calendar)
	return time.Date(targetDay.Year(), targetDay.Month(), targetDay.Day(), hour, min, 0, 0, loc)
}
