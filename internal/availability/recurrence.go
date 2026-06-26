package availability

import (
	"time"

	"github.com/teambition/rrule-go"
)

// RecurringBlackout represents a blackout with recurrence rules
type RecurringBlackout struct {
	ID                 string
	TenantID           string
	StartTime          time.Time
	EndTime            time.Time
	RecurrenceRule     string
	RecurrenceTimezone string
	RecurrenceEnd      *time.Time
	IsRecurring        bool
}

// Occurrence represents a single instance of a recurring blackout
type Occurrence struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ExpandOccurrences generates all occurrences within a time range
func (rb *RecurringBlackout) ExpandOccurrences(rangeStart, rangeEnd time.Time) ([]Occurrence, error) {
	// One-time blackout: simple overlap check
	if !rb.IsRecurring || rb.RecurrenceRule == "" {
		if rangeStart.Before(rb.EndTime) && rangeEnd.After(rb.StartTime) {
			return []Occurrence{{Start: rb.StartTime, End: rb.EndTime}}, nil
		}
		return nil, nil
	}

	// Parse RRULE
	rule, err := rrule.StrToRRule(rb.RecurrenceRule)
	if err != nil {
		return nil, err
	}

	// Set timezone for rule
	loc, err := time.LoadLocation(rb.RecurrenceTimezone)
	if err != nil {
		loc = time.UTC
	}
	rule.DTStart(rb.StartTime.In(loc))

	// Determine query range (add buffer for safety)
	queryStart := rangeStart.Add(-24 * time.Hour)
	queryEnd := rangeEnd.Add(24 * time.Hour)

	// If recurrence has an end, respect it
	if rb.RecurrenceEnd != nil && queryEnd.After(*rb.RecurrenceEnd) {
		queryEnd = *rb.RecurrenceEnd
	}

	// Generate occurrences
	occurrences := rule.Between(queryStart, queryEnd, true)

	// Calculate duration from original blackout
	duration := rb.EndTime.Sub(rb.StartTime)

	// Convert to Occurrence structs
	result := make([]Occurrence, 0, len(occurrences))
	for _, occStart := range occurrences {
		occEnd := occStart.Add(duration)

		// Only include if it overlaps our query range
		if occEnd.After(rangeStart) && occStart.Before(rangeEnd) {
			result = append(result, Occurrence{
				Start: occStart.UTC(),
				End:   occEnd.UTC(),
			})
		}
	}

	return result, nil
}

// GetNextOccurrence returns the next occurrence after a given time
func (rb *RecurringBlackout) GetNextOccurrence(after time.Time) (*Occurrence, error) {
	if !rb.IsRecurring || rb.RecurrenceRule == "" {
		if rb.StartTime.After(after) {
			return &Occurrence{Start: rb.StartTime, End: rb.EndTime}, nil
		}
		return nil, nil
	}

	rule, err := rrule.StrToRRule(rb.RecurrenceRule)
	if err != nil {
		return nil, err
	}

	loc, _ := time.LoadLocation(rb.RecurrenceTimezone)
	rule.DTStart(rb.StartTime.In(loc))

	// Get next occurrence
	next := rule.After(after, false)
	if next.IsZero() {
		return nil, nil
	}

	// Check recurrence end
	if rb.RecurrenceEnd != nil && next.After(*rb.RecurrenceEnd) {
		return nil, nil
	}

	duration := rb.EndTime.Sub(rb.StartTime)
	return &Occurrence{
		Start: next.UTC(),
		End:   next.Add(duration).UTC(),
	}, nil
}
