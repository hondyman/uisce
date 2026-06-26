package availability

import (
	"time"

	"github.com/teambition/rrule-go"
)

// RecurringBlackout represents a blackout period that can recur
type RecurringBlackout struct {
	ID                 string
	StartTime          time.Time
	EndTime            time.Time
	RecurrenceRule     string
	RecurrenceTimezone string
	RecurrenceEnd      *time.Time
	IsRecurring        bool
}

// Occurrence represents a single occurrence of a blackout
type Occurrence struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// ExpandOccurrences expands a recurring blackout into individual occurrences within a date range
func (rb *RecurringBlackout) ExpandOccurrences(rangeStart, rangeEnd time.Time) ([]Occurrence, error) {
	var occurrences []Occurrence

	if !rb.IsRecurring || rb.RecurrenceRule == "" {
		// Non-recurring blackout - return single occurrence if it overlaps with range
		if rb.StartTime.Before(rangeEnd) && rb.EndTime.After(rangeStart) {
			occurrences = append(occurrences, Occurrence{
				StartTime: rb.StartTime,
				EndTime:   rb.EndTime,
			})
		}
		return occurrences, nil
	}

	// Parse the recurrence rule
	rruleSet, err := rrule.StrToRRuleSet(rb.RecurrenceRule)
	if err != nil {
		return nil, err
	}

	// Generate occurrences within the range
	// The duration of each occurrence matches the original blackout duration
	duration := rb.EndTime.Sub(rb.StartTime)

	// Get dates when the recurrence occurs
	occurrenceDates := rruleSet.Between(rangeStart, rangeEnd, true)

	for _, dt := range occurrenceDates {
		// Apply timezone conversion if needed
		occStartTime := dt
		occEndTime := dt.Add(duration)

		// Check if occurrence overlaps with the requested range
		if occStartTime.Before(rangeEnd) && occEndTime.After(rangeStart) {
			occurrences = append(occurrences, Occurrence{
				StartTime: occStartTime,
				EndTime:   occEndTime,
			})
		}
	}

	return occurrences, nil
}
