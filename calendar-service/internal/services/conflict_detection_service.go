package services

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Conflict represents a scheduling conflict
type Conflict struct {
	Type        string    `json:"type"`     // "overlap", "back_to_back", "insufficient_buffer"
	Severity    string    `json:"severity"` // "high", "medium", "low"
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Description string    `json:"description"`
	AffectedIDs []string  `json:"affected_ids"` // IDs of conflicting events
}

// BlackoutPeriod represents a time when scheduling is not allowed
type BlackoutPeriod struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	ProfileID  string    `json:"profile_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Reason     string    `json:"reason"` // e.g., "Maintenance", "Team Closure"
	TimezoneID string    `json:"timezone_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ConflictDetectionServiceTenantAware defines conflict detection operations
type ConflictDetectionServiceTenantAware interface {
	// Check for conflicts between a new event and existing events
	DetectConflicts(ctx context.Context, profileID, tenantID string, newEvent *RecurringEventOccurrence) ([]*Conflict, error)

	// Check if a time slot is available
	IsTimeSlotAvailable(ctx context.Context, profileID, tenantID string, startTime, endTime time.Time) (bool, error)

	// Find all conflicts within a date range
	FindConflictsInRange(ctx context.Context, profileID, tenantID string, from, to time.Time) ([]*Conflict, error)

	// Create a blackout period
	CreateBlackoutPeriod(ctx context.Context, period *BlackoutPeriod) error

	// Get blackout periods for a profile
	GetBlackoutPeriods(ctx context.Context, profileID, tenantID string, from, to time.Time) ([]*BlackoutPeriod, error)

	// Delete a blackout period
	DeleteBlackoutPeriod(ctx context.Context, id, tenantID string) error

	// Check if time falls within any blackout period
	IsInBlackout(ctx context.Context, profileID, tenantID string, checkTime time.Time) (bool, *BlackoutPeriod, error)

	// Get conflict statistics for a profile
	GetConflictStats(ctx context.Context, profileID, tenantID string, from, to time.Time) (map[string]interface{}, error)
}

// ConflictDetectionService implements ConflictDetectionServiceTenantAware
type ConflictDetectionService struct {
	repo ServiceRepository
}

// NewConflictDetectionService creates a new conflict detection service
func NewConflictDetectionService(repo ServiceRepository) *ConflictDetectionService {
	return &ConflictDetectionService{
		repo: repo,
	}
}

// DetectConflicts checks for scheduling conflicts with a new event
func (s *ConflictDetectionService) DetectConflicts(ctx context.Context, profileID, tenantID string, newEvent *RecurringEventOccurrence) ([]*Conflict, error) {
	conflicts := []*Conflict{}

	// 1. Check against existing calendars
	calendars, _, err := s.repo.ListCalendars(ctx, profileID, tenantID, 1000, 0)
	if err != nil {
		return nil, err
	}

	for _, calendar := range calendars {
		// Get events for this calendar
		events, err := s.repo.GetCalendarEvents(ctx, calendar.ID, tenantID)
		if err != nil {
			continue
		}

		for _, event := range events {
			if s.hasConflict(newEvent, &RecurringEventOccurrence{
				StartTime:  event.StartTime,
				EndTime:    event.EndTime,
				TimezoneID: event.TimezoneID,
			}) {
				conflicts = append(conflicts, &Conflict{
					Type:        "overlap",
					Severity:    "high",
					StartTime:   newEvent.StartTime,
					EndTime:     newEvent.EndTime,
					Description: fmt.Sprintf("Overlaps with event %s on calendar %s", event.ID, calendar.Name),
					AffectedIDs: []string{event.ID, calendar.ID},
				})
			}
		}
	}

	// 2. Check against recurring events
	recurringRules, _, err := s.repo.ListRecurrenceRules(ctx, profileID, tenantID, 1000, 0)
	if err != nil {
		return nil, err
	}

	recurringService := NewRecurringEventService(s.repo)
	for _, rule := range recurringRules {
		occurrences, err := recurringService.GenerateOccurrences(ctx, rule.ID, tenantID, newEvent.StartTime.AddDate(0, -1, 0), newEvent.StartTime.AddDate(0, 1, 0))
		if err != nil {
			continue
		}

		for _, occ := range occurrences {
			if s.hasConflict(newEvent, occ) {
				conflicts = append(conflicts, &Conflict{
					Type:        "overlap",
					Severity:    "high",
					StartTime:   newEvent.StartTime,
					EndTime:     newEvent.EndTime,
					Description: fmt.Sprintf("Overlaps with recurring event %s", rule.ID),
					AffectedIDs: []string{rule.ID},
				})
				break // Only report one conflict per rule
			}
		}
	}

	// 3. Check against blackout periods
	blackoutPeriods, err := s.repo.GetBlackoutPeriods(ctx, profileID, tenantID, newEvent.StartTime, newEvent.EndTime)
	if err == nil {
		for _, blackout := range blackoutPeriods {
			if s.hasConflict(newEvent, &RecurringEventOccurrence{
				StartTime:  blackout.StartTime,
				EndTime:    blackout.EndTime,
				TimezoneID: blackout.TimezoneID,
			}) {
				conflicts = append(conflicts, &Conflict{
					Type:        "blackout",
					Severity:    "critical",
					StartTime:   newEvent.StartTime,
					EndTime:     newEvent.EndTime,
					Description: fmt.Sprintf("Falls within blackout period: %s", blackout.Reason),
					AffectedIDs: []string{blackout.ID},
				})
			}
		}
	}

	return conflicts, nil
}

// IsTimeSlotAvailable checks if a time slot is free
func (s *ConflictDetectionService) IsTimeSlotAvailable(ctx context.Context, profileID, tenantID string, startTime, endTime time.Time) (bool, error) {
	testEvent := &RecurringEventOccurrence{
		StartTime: startTime,
		EndTime:   endTime,
	}

	conflicts, err := s.DetectConflicts(ctx, profileID, tenantID, testEvent)
	if err != nil {
		return false, err
	}

	return len(conflicts) == 0, nil
}

// FindConflictsInRange finds all conflicts within a date range
func (s *ConflictDetectionService) FindConflictsInRange(ctx context.Context, profileID, tenantID string, from, to time.Time) ([]*Conflict, error) {
	allConflicts := []*Conflict{}

	// Get all events in range
	calendars, _, err := s.repo.ListCalendars(ctx, profileID, tenantID, 1000, 0)
	if err != nil {
		return nil, err
	}

	eventList := []*RecurringEventOccurrence{}

	for _, calendar := range calendars {
		events, err := s.repo.GetCalendarEvents(ctx, calendar.ID, tenantID)
		if err != nil {
			continue
		}

		for _, event := range events {
			if !event.EndTime.Before(from) && !event.StartTime.After(to) {
				eventList = append(eventList, &RecurringEventOccurrence{
					StartTime:  event.StartTime,
					EndTime:    event.EndTime,
					TimezoneID: event.TimezoneID,
				})
			}
		}
	}

	// Sort events by start time
	sort.Slice(eventList, func(i, j int) bool {
		return eventList[i].StartTime.Before(eventList[j].StartTime)
	})

	// Find overlaps
	for i := 0; i < len(eventList); i++ {
		for j := i + 1; j < len(eventList); j++ {
			if s.hasConflict(eventList[i], eventList[j]) {
				conflict := &Conflict{
					Type:      "overlap",
					Severity:  "high",
					StartTime: eventList[i].StartTime,
					EndTime:   eventList[j].StartTime,
				}
				allConflicts = append(allConflicts, conflict)
			}
		}
	}

	// Check against blackout periods
	blackoutPeriods, err := s.repo.GetBlackoutPeriods(ctx, profileID, tenantID, from, to)
	if err == nil {
		for _, blackout := range blackoutPeriods {
			for _, event := range eventList {
				if s.hasConflict(event, &RecurringEventOccurrence{
					StartTime:  blackout.StartTime,
					EndTime:    blackout.EndTime,
					TimezoneID: blackout.TimezoneID,
				}) {
					conflict := &Conflict{
						Type:        "blackout_violation",
						Severity:    "critical",
						StartTime:   event.StartTime,
						EndTime:     event.EndTime,
						Description: fmt.Sprintf("Scheduled during blackout: %s", blackout.Reason),
					}
					allConflicts = append(allConflicts, conflict)
				}
			}
		}
	}

	return allConflicts, nil
}

// CreateBlackoutPeriod creates a new blackout period
func (s *ConflictDetectionService) CreateBlackoutPeriod(ctx context.Context, period *BlackoutPeriod) error {
	if period.StartTime.After(period.EndTime) {
		return fmt.Errorf("start time must be before end time")
	}

	period.CreatedAt = time.Now().UTC()
	period.UpdatedAt = time.Now().UTC()

	return s.repo.StoreBlackoutPeriod(ctx, period)
}

// GetBlackoutPeriods retrieves blackout periods within a date range
func (s *ConflictDetectionService) GetBlackoutPeriods(ctx context.Context, profileID, tenantID string, from, to time.Time) ([]*BlackoutPeriod, error) {
	return s.repo.GetBlackoutPeriods(ctx, profileID, tenantID, from, to)
}

// DeleteBlackoutPeriod removes a blackout period
func (s *ConflictDetectionService) DeleteBlackoutPeriod(ctx context.Context, id, tenantID string) error {
	return s.repo.DeleteBlackoutPeriod(ctx, id, tenantID)
}

// IsInBlackout checks if a specific time is within any blackout period
func (s *ConflictDetectionService) IsInBlackout(ctx context.Context, profileID, tenantID string, checkTime time.Time) (bool, *BlackoutPeriod, error) {
	periods, err := s.repo.GetBlackoutPeriods(ctx, profileID, tenantID, checkTime.AddDate(0, 0, -1), checkTime.AddDate(0, 0, 1))
	if err != nil {
		return false, nil, err
	}

	for _, period := range periods {
		if !checkTime.Before(period.StartTime) && !checkTime.After(period.EndTime) {
			return true, period, nil
		}
	}

	return false, nil, nil
}

// GetConflictStats returns statistics about conflicts
func (s *ConflictDetectionService) GetConflictStats(ctx context.Context, profileID, tenantID string, from, to time.Time) (map[string]interface{}, error) {
	conflicts, err := s.FindConflictsInRange(ctx, profileID, tenantID, from, to)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	stats["total_conflicts"] = len(conflicts)
	stats["high_severity"] = 0
	stats["medium_severity"] = 0
	stats["low_severity"] = 0

	for _, conflict := range conflicts {
		switch conflict.Severity {
		case "high":
			stats["high_severity"] = stats["high_severity"].(int) + 1
		case "medium":
			stats["medium_severity"] = stats["medium_severity"].(int) + 1
		case "low":
			stats["low_severity"] = stats["low_severity"].(int) + 1
		}
	}

	stats["date_range_start"] = from
	stats["date_range_end"] = to
	stats["utilization_rate"] = calculateUtilizationRate(conflicts, from, to)

	return stats, nil
}

// hasConflict checks if two time slots overlap
func (s *ConflictDetectionService) hasConflict(slot1, slot2 *RecurringEventOccurrence) bool {
	// Two events don't conflict if one ends when or before the other starts
	return !(slot1.EndTime.Before(slot2.StartTime) || slot1.EndTime.Equal(slot2.StartTime) ||
		slot2.EndTime.Before(slot1.StartTime) || slot2.EndTime.Equal(slot1.StartTime))
}

// calculateUtilizationRate estimates how much of the time period is occupied
func calculateUtilizationRate(conflicts []*Conflict, from, to time.Time) float64 {
	if len(conflicts) == 0 {
		return 0.0
	}

	totalDuration := to.Sub(from)
	conflictDuration := time.Duration(0)

	for _, conflict := range conflicts {
		conflictDuration += conflict.EndTime.Sub(conflict.StartTime)
	}

	if totalDuration == 0 {
		return 0.0
	}

	rate := float64(conflictDuration) / float64(totalDuration)
	if rate > 1.0 {
		rate = 1.0
	}

	return rate
}
