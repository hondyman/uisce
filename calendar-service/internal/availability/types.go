package availability

import (
	"time"

	"calendar-service/internal/cache"
)

// ResolvedCalendar is an alias to cache.ResolvedCalendar
type ResolvedCalendar = cache.ResolvedCalendar

// TimeRange is an alias to cache.TimeRange
type TimeRange = cache.TimeRange

// AvailabilityResult represents the result of an availability check
type AvailabilityResult struct {
	Available bool      `json:"available"`
	Reasons   []string  `json:"reasons,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
	Region    string    `json:"region,omitempty"`
}

// AvailabilityMetrics represents availability metrics
type AvailabilityMetrics struct {
	TenantID           string                 `json:"tenant_id"`
	CalendarID         string                 `json:"calendar_id"`
	AvailableSlots     int                    `json:"available_slots"`
	BlockedSlots       int                    `json:"blocked_slots"`
	AvailabilityRate   float32                `json:"availability_rate"`
	SLAComplianceRate  float32                `json:"sla_compliance_rate"`
	LastUpdated        time.Time              `json:"last_updated"`
	AverageFulfillTime string                 `json:"average_fulfill_time"`
	Breakdown          map[string]interface{} `json:"breakdown,omitempty"`
}

// Holiday represents a single holiday entry
type Holiday struct {
	Date     time.Time `json:"date"` // Date only (time component ignored)
	Name     string    `json:"name"`
	Type     string    `json:"type"`     // public, observance, bank
	Severity string    `json:"severity"` // HIGH, MEDIUM, LOW
	AllDay   bool      `json:"all_day"`  // Always true for holidays
}

// Blackout represents a time range when jobs should not run
type Blackout struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	IsRecurring    bool      `json:"is_recurring"`
	RecurrenceRule string    `json:"recurrence_rule,omitempty"` // RRULE format
	Reason         string    `json:"reason"`
	Severity       string    `json:"severity"` // CRITICAL, HIGH, NORMAL, LOW
}

// ConflictRules defines how to merge multiple calendars
type ConflictRules struct {
	Strategy   string         `json:"strategy"`             // UNION, INTERSECTION, PRIORITY
	Priorities map[string]int `json:"priorities,omitempty"` // calendar_id -> priority
}

// ScheduleProfile represents a fetched schedule profile with linked calendars
type ScheduleProfile struct {
	ID                 string         `json:"id"`
	ProfileName        string         `json:"profile_name"`
	Timezone           string         `json:"timezone"`
	Region             string         `json:"region"`
	ConflictResolution string         `json:"conflict_resolution"` // UNION, INTERSECTION, PRIORITY
	CalendarIDs        []string       `json:"calendar_ids"`
	CalendarPriorities map[string]int `json:"calendar_priorities"` // calendar_id -> priority
	ConflictRules      ConflictRules  `json:"conflict_rules"`
	ValidFrom          time.Time      `json:"valid_from"`
	ValidTo            *time.Time     `json:"valid_to"`
	Active             bool           `json:"active"`
}
