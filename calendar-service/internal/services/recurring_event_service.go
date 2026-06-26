package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/teambition/rrule-go"
)

// RecurrenceRule represents an RRULE (RFC 5545 format)
type RecurrenceRule struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ProfileID     string    `json:"profile_id"`
	RRule         string    `json:"rrule"`          // RFC 5545 format
	StartTime     time.Time `json:"start_time"`     // Start of first occurrence
	EndTime       time.Time `json:"end_time"`       // End of first occurrence
	TimezoneID    string    `json:"timezone_id"`    // America/New_York, Europe/London, etc.
	MaxOccurrence int       `json:"max_occurrence"` // Max occurrences to generate (default 100)
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// RecurrenceException represents an exception to a recurring event
type RecurrenceException struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	RecurrenceID  string     `json:"recurrence_id"`
	ExceptionDate time.Time  `json:"exception_date"`
	IsDeleted     bool       `json:"is_deleted"` // true = deleted, false = modified
	NewStartTime  *time.Time `json:"new_start_time,omitempty"`
	NewEndTime    *time.Time `json:"new_end_time,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// RecurringEventOccurrence represents a single occurrence of a recurring event
type RecurringEventOccurrence struct {
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	TimezoneID    string    `json:"timezone_id"`
	OccurrenceNum int       `json:"occurrence_number"`
}

// RecurringEventServiceTenantAware defines operations for managing recurring events
type RecurringEventServiceTenantAware interface {
	// Create a new recurrence rule
	CreateRecurrenceRule(ctx context.Context, rule *RecurrenceRule) error

	// Get a recurrence rule by ID
	GetRecurrenceRule(ctx context.Context, id, tenantID string) (*RecurrenceRule, error)

	// List recurrence rules for a profile
	ListRecurrenceRules(ctx context.Context, profileID, tenantID string, limit, offset int) ([]*RecurrenceRule, int64, error)

	// Update a recurrence rule
	UpdateRecurrenceRule(ctx context.Context, rule *RecurrenceRule) error

	// Delete a recurrence rule
	DeleteRecurrenceRule(ctx context.Context, id, tenantID string) error

	// Generate occurrences within a date range
	GenerateOccurrences(ctx context.Context, id, tenantID string, from, to time.Time) ([]*RecurringEventOccurrence, error)

	// Create an exception for a specific occurrence
	CreateException(ctx context.Context, exception *RecurrenceException) error

	// Delete an exception
	DeleteException(ctx context.Context, excID, tenantID string) error

	// Get exceptions for a recurrence rule
	GetExceptions(ctx context.Context, recurrenceID, tenantID string) ([]*RecurrenceException, error)
}

// RecurringEventService implements RecurringEventServiceTenantAware
type RecurringEventService struct {
	repo ServiceRepository
}

// NewRecurringEventService creates a new recurring event service
func NewRecurringEventService(repo ServiceRepository) *RecurringEventService {
	return &RecurringEventService{
		repo: repo,
	}
}

// CreateRecurrenceRule validates and stores a recurrence rule
func (s *RecurringEventService) CreateRecurrenceRule(ctx context.Context, rule *RecurrenceRule) error {
	// Validate RRULE format
	if err := s.validateRRule(rule.RRule); err != nil {
		return fmt.Errorf("invalid rrule format: %w", err)
	}

	// Validate timezone
	if _, err := time.LoadLocation(rule.TimezoneID); err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}

	// Set defaults
	if rule.MaxOccurrence <= 0 {
		rule.MaxOccurrence = 100
	}

	rule.ID = uuid.New().String()
	rule.CreatedAt = time.Now().UTC()
	rule.UpdatedAt = time.Now().UTC()

	// Store in repository
	return s.repo.StoreRecurrenceRule(ctx, rule)
}

// GetRecurrenceRule retrieves a specific recurrence rule
func (s *RecurringEventService) GetRecurrenceRule(ctx context.Context, id, tenantID string) (*RecurrenceRule, error) {
	return s.repo.GetRecurrenceRule(ctx, id, tenantID)
}

// ListRecurrenceRules retrieves all recurrence rules for a profile
func (s *RecurringEventService) ListRecurrenceRules(ctx context.Context, profileID, tenantID string, limit, offset int) ([]*RecurrenceRule, int64, error) {
	return s.repo.ListRecurrenceRules(ctx, profileID, tenantID, limit, offset)
}

// UpdateRecurrenceRule updates an existing recurrence rule
func (s *RecurringEventService) UpdateRecurrenceRule(ctx context.Context, rule *RecurrenceRule) error {
	// Validate RRULE format
	if err := s.validateRRule(rule.RRule); err != nil {
		return fmt.Errorf("invalid rrule format: %w", err)
	}

	// Validate timezone
	if _, err := time.LoadLocation(rule.TimezoneID); err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}

	rule.UpdatedAt = time.Now().UTC()
	return s.repo.UpdateRecurrenceRule(ctx, rule)
}

// DeleteRecurrenceRule removes a recurrence rule
func (s *RecurringEventService) DeleteRecurrenceRule(ctx context.Context, id, tenantID string) error {
	return s.repo.DeleteRecurrenceRule(ctx, id, tenantID)
}

// GenerateOccurrences generates all occurrences of a recurring event within a date range
func (s *RecurringEventService) GenerateOccurrences(ctx context.Context, id, tenantID string, from, to time.Time) ([]*RecurringEventOccurrence, error) {
	// Get the recurrence rule
	rule, err := s.repo.GetRecurrenceRule(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Parse the RRULE
	rruleSet, err := rrule.StrToRRuleSet(buildFullRRule(rule))
	if err != nil {
		return nil, fmt.Errorf("failed to parse rrule: %w", err)
	}

	// Load timezone
	loc, err := time.LoadLocation(rule.TimezoneID)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	// Generate occurrences
	occurrences := []*RecurringEventOccurrence{}
	fromInTZ := from.In(loc)
	toInTZ := to.In(loc)

	// Get exceptions
	exceptions, _ := s.repo.GetExceptions(ctx, id, tenantID)
	exceptionMap := buildExceptionMap(exceptions)

	// Generate dates using rrule
	occNum := 0
	for _, dt := range rruleSet.Between(fromInTZ, toInTZ, true) {
		occNum++

		// Check if this occurrence has an exception
		if exceptionData, exists := exceptionMap[dt]; exists {
			if exceptionData.IsDeleted {
				continue // Skip deleted occurrence
			}
			// Use modified times if provided
			startTime := exceptionData.NewStartTime
			endTime := exceptionData.NewEndTime
			if startTime == nil {
				startTime = &dt
			}
			if endTime == nil {
				duration := rule.EndTime.Sub(rule.StartTime)
				adjustedEnd := startTime.Add(duration)
				endTime = &adjustedEnd
			}

			occurrences = append(occurrences, &RecurringEventOccurrence{
				StartTime:     *startTime,
				EndTime:       *endTime,
				TimezoneID:    rule.TimezoneID,
				OccurrenceNum: occNum,
			})
		} else {
			// Calculate duration from original event
			duration := rule.EndTime.Sub(rule.StartTime)
			endTime := dt.Add(duration)

			occurrences = append(occurrences, &RecurringEventOccurrence{
				StartTime:     dt,
				EndTime:       endTime,
				TimezoneID:    rule.TimezoneID,
				OccurrenceNum: occNum,
			})
		}

		// Cap at max occurrences
		if len(occurrences) >= rule.MaxOccurrence {
			break
		}
	}

	return occurrences, nil
}

// CreateException creates an exception for a specific occurrence
func (s *RecurringEventService) CreateException(ctx context.Context, exception *RecurrenceException) error {
	exception.ID = uuid.New().String()
	exception.CreatedAt = time.Now().UTC()
	return s.repo.StoreRecurrenceException(ctx, exception)
}

// DeleteException removes an exception
func (s *RecurringEventService) DeleteException(ctx context.Context, excID, tenantID string) error {
	return s.repo.DeleteRecurrenceException(ctx, excID)
}

// GetExceptions retrieves all exceptions for a recurrence rule
func (s *RecurringEventService) GetExceptions(ctx context.Context, recurrenceID, tenantID string) ([]*RecurrenceException, error) {
	return s.repo.GetExceptions(ctx, recurrenceID, tenantID)
}

// validateRRule validates RRULE format (basic validation)
func (s *RecurringEventService) validateRRule(ruleStr string) error {
	ruleStr = strings.TrimSpace(ruleStr)
	if ruleStr == "" {
		return fmt.Errorf("rrule cannot be empty")
	}

	// Basic check for RRULE format
	if !strings.HasPrefix(ruleStr, "FREQ=") {
		return fmt.Errorf("rrule must start with FREQ=")
	}

	validFreqs := map[string]bool{
		"DAILY":    true,
		"WEEKLY":   true,
		"MONTHLY":  true,
		"YEARLY":   true,
		"HOURLY":   true,
		"MINUTELY": true,
		"SECONDLY": true,
	}

	parts := strings.Split(ruleStr, ";")
	freqPart := parts[0]
	freqValue := strings.TrimPrefix(freqPart, "FREQ=")

	if !validFreqs[freqValue] {
		return fmt.Errorf("invalid frequency: %s", freqValue)
	}

	return nil
}

// buildFullRRule constructs a complete RRULE with DTSTART
func buildFullRRule(rule *RecurrenceRule) string {
	startTime := rule.StartTime.UTC().Format(time.RFC3339)
	// Remove colons for RRULE format (20260218T120000Z)
	startTime = strings.ReplaceAll(strings.ReplaceAll(startTime, "-", ""), ":", "")
	return fmt.Sprintf("DTSTART:%s\nRRULE:%s", startTime, rule.RRule)
}

// buildExceptionMap creates a map of exceptions by date
func buildExceptionMap(exceptions []*RecurrenceException) map[time.Time]*RecurrenceException {
	exMap := make(map[time.Time]*RecurrenceException)
	for _, exc := range exceptions {
		exMap[exc.ExceptionDate] = exc
	}
	return exMap
}

// SuggestAvailableSlots finds available time slots for a given duration
func (s *RecurringEventService) SuggestAvailableSlots(ctx context.Context, profileID, tenantID string, from, to time.Time, duration time.Duration, limit int) ([]*RecurringEventOccurrence, error) {
	// Get all profile events (using calendar repository)
	// Get all recurrence rules for the profile
	rules, _, err := s.repo.ListRecurrenceRules(ctx, profileID, tenantID, 1000, 0)
	if err != nil {
		return nil, err
	}

	// Generate all occurrences (aggregate from all rules)
	var allOccurrences []*RecurringEventOccurrence
	for _, rule := range rules {
		occs, err := s.GenerateOccurrences(ctx, rule.ID, tenantID, from, to)
		if err != nil {
			continue // Skip rules with errors
		}
		allOccurrences = append(allOccurrences, occs...)
	}

	// Find gaps (available slots)
	availableSlots := findAvailableSlots(from, to, duration, allOccurrences, limit)
	return availableSlots, nil
}

// findAvailableSlots finds gaps in a schedule where a certain duration fits
func findAvailableSlots(from, to time.Time, duration time.Duration, occupiedSlots []*RecurringEventOccurrence, limit int) []*RecurringEventOccurrence {
	availableSlots := []*RecurringEventOccurrence{}
	currentTime := from

	// Sort occupied slots by start time
	slotsByStart := make([]*RecurringEventOccurrence, len(occupiedSlots))
	copy(slotsByStart, occupiedSlots)

	// Simple bubble sort for occupied slots
	for i := 0; i < len(slotsByStart); i++ {
		for j := i + 1; j < len(slotsByStart); j++ {
			if slotsByStart[j].StartTime.Before(slotsByStart[i].StartTime) {
				slotsByStart[i], slotsByStart[j] = slotsByStart[j], slotsByStart[i]
			}
		}
	}

	// Find gaps
	for _, occupied := range slotsByStart {
		if currentTime.Add(duration).Before(occupied.StartTime) {
			// Found a gap that fits the duration
			availableSlots = append(availableSlots, &RecurringEventOccurrence{
				StartTime:  currentTime,
				EndTime:    currentTime.Add(duration),
				TimezoneID: occupied.TimezoneID,
			})

			if len(availableSlots) >= limit {
				return availableSlots
			}
		}
		currentTime = occupied.EndTime
	}

	// Check remaining time after last occupied slot
	if currentTime.Add(duration).Before(to) || currentTime.Add(duration).Equal(to) {
		availableSlots = append(availableSlots, &RecurringEventOccurrence{
			StartTime:  currentTime,
			EndTime:    currentTime.Add(duration),
			TimezoneID: "UTC",
		})
	}

	return availableSlots
}
