package scheduler_intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/calendar"
)

// ConstraintEngine evaluates scheduling constraints and computes optimal execution windows
type ConstraintEngine struct {
	calendarSvc *calendar.Service
	logger      *slog.Logger
}

// NewConstraintEngine creates a new constraint engine
func NewConstraintEngine(calendarSvc *calendar.Service, logger *slog.Logger) *ConstraintEngine {
	return &ConstraintEngine{
		calendarSvc: calendarSvc,
		logger:      logger,
	}
}

// ============================================================================
// Constraint Types
// ============================================================================

// SchedulingConstraints defines all constraints for job scheduling
type SchedulingConstraints struct {
	// Calendar constraints
	CalendarCodes        []string `json:"calendar_codes,omitempty"`
	RequireBusinessDay   bool     `json:"require_business_day"`
	AdjustmentConvention string   `json:"adjustment_convention,omitempty"` // FOLLOWING, PRECEDING, etc.

	// Time window constraints
	AllowedStartHour  int   `json:"allowed_start_hour,omitempty"`   // 0-23
	AllowedEndHour    int   `json:"allowed_end_hour,omitempty"`     // 0-23
	AllowedDaysOfWeek []int `json:"allowed_days_of_week,omitempty"` // 0=Sunday, 6=Saturday

	// Blackout windows
	BlackoutWindows []BlackoutWindow `json:"blackout_windows,omitempty"`

	// Resource constraints
	MaxConcurrentRuns    int            `json:"max_concurrent_runs,omitempty"`
	ResourceRequirements map[string]int `json:"resource_requirements,omitempty"`

	// SLO constraints
	SLOTargetMS     int64 `json:"slo_target_ms,omitempty"`
	SLOCritical     bool  `json:"slo_critical"`
	MaxDelayMinutes int   `json:"max_delay_minutes,omitempty"`

	// Tenant-specific rules
	TenantRules []TenantConstraintRule `json:"tenant_rules,omitempty"`
}

// TenantConstraintRule defines a tenant-specific scheduling rule
type TenantConstraintRule struct {
	RuleID      string                 `json:"rule_id"`
	RuleType    string                 `json:"rule_type"` // "blackout", "priority_boost", "resource_limit"
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Conditions  map[string]interface{} `json:"conditions"`
	Actions     map[string]interface{} `json:"actions"`
	Priority    int                    `json:"priority"`
	Active      bool                   `json:"active"`
}

// ConstraintViolation represents a failed constraint check
type ConstraintViolation struct {
	ConstraintType string     `json:"constraint_type"`
	Description    string     `json:"description"`
	Severity       string     `json:"severity"` // "error", "warning", "info"
	SuggestedTime  *time.Time `json:"suggested_time,omitempty"`
}

// ExecutionWindow represents a valid time window for job execution
type ExecutionWindow struct {
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	Score         float64   `json:"score"` // 0-1, higher is better
	Reason        string    `json:"reason,omitempty"`
	IsBusinessDay bool      `json:"is_business_day"`
	SLOCompliant  bool      `json:"slo_compliant"`
}

// ============================================================================
// Constraint Evaluation
// ============================================================================

// EvaluateConstraints checks if a proposed execution time satisfies all constraints
func (ce *ConstraintEngine) EvaluateConstraints(
	ctx context.Context,
	job *Job,
	proposedTime time.Time,
) (bool, []ConstraintViolation, error) {
	var violations []ConstraintViolation

	// Parse job constraints
	var constraints SchedulingConstraints
	if len(job.Constraints) > 0 {
		if err := json.Unmarshal(job.Constraints, &constraints); err != nil {
			ce.logger.Warn("Failed to parse job constraints", "job_id", job.ID, "error", err)
		}
	}

	// Parse blackout windows
	var blackoutWindows []BlackoutWindow
	if len(job.BlackoutWindows) > 0 {
		json.Unmarshal(job.BlackoutWindows, &blackoutWindows)
	}
	constraints.BlackoutWindows = append(constraints.BlackoutWindows, blackoutWindows...)

	// Set SLO constraints from job
	constraints.SLOCritical = job.SLOCritical

	// Check business day constraint
	if constraints.RequireBusinessDay && len(job.CalendarIDs) > 0 {
		for _, calendarID := range job.CalendarIDs {
			calendarCode := calendarID.String() // In practice, would look up code
			isBusinessDay, err := ce.calendarSvc.IsBusinessDay(ctx, calendarCode, proposedTime)
			if err != nil {
				ce.logger.Warn("Failed to check business day", "error", err)
				continue
			}
			if !isBusinessDay {
				nextBizDay, _ := ce.calendarSvc.NextBusinessDay(ctx, calendarCode, proposedTime)
				violations = append(violations, ConstraintViolation{
					ConstraintType: "business_day",
					Description:    fmt.Sprintf("Proposed time is not a business day on calendar %s", calendarCode),
					Severity:       "error",
					SuggestedTime:  &nextBizDay,
				})
			}
		}
	}

	// Check time window constraint
	if constraints.AllowedStartHour > 0 || constraints.AllowedEndHour > 0 {
		hour := proposedTime.Hour()
		startHour := constraints.AllowedStartHour
		endHour := constraints.AllowedEndHour
		if endHour == 0 {
			endHour = 24
		}

		if hour < startHour || hour >= endHour {
			// Calculate next allowed time
			suggestedTime := time.Date(
				proposedTime.Year(), proposedTime.Month(), proposedTime.Day(),
				startHour, 0, 0, 0, proposedTime.Location(),
			)
			if hour >= endHour {
				// Move to next day
				suggestedTime = suggestedTime.Add(24 * time.Hour)
			}
			violations = append(violations, ConstraintViolation{
				ConstraintType: "time_window",
				Description:    fmt.Sprintf("Proposed time %d:00 is outside allowed window %d:00-%d:00", hour, startHour, endHour),
				Severity:       "error",
				SuggestedTime:  &suggestedTime,
			})
		}
	}

	// Check day of week constraint
	if len(constraints.AllowedDaysOfWeek) > 0 {
		currentDay := int(proposedTime.Weekday())
		allowed := false
		for _, day := range constraints.AllowedDaysOfWeek {
			if day == currentDay {
				allowed = true
				break
			}
		}
		if !allowed {
			// Find next allowed day
			suggestedTime := proposedTime
			for i := 0; i < 7; i++ {
				suggestedTime = suggestedTime.Add(24 * time.Hour)
				nextDay := int(suggestedTime.Weekday())
				for _, day := range constraints.AllowedDaysOfWeek {
					if day == nextDay {
						violations = append(violations, ConstraintViolation{
							ConstraintType: "day_of_week",
							Description:    fmt.Sprintf("Day %s is not in allowed days", proposedTime.Weekday()),
							Severity:       "error",
							SuggestedTime:  &suggestedTime,
						})
						break
					}
				}
				break
			}
		}
	}

	// Check blackout windows
	for _, window := range constraints.BlackoutWindows {
		if ce.isInBlackout(proposedTime, window) {
			endTime := window.End
			violations = append(violations, ConstraintViolation{
				ConstraintType: "blackout_window",
				Description:    fmt.Sprintf("Proposed time falls within blackout window: %s", window.Reason),
				Severity:       "error",
				SuggestedTime:  &endTime,
			})
		}
	}

	// Check SLO constraints (warning only)
	if constraints.SLOCritical && constraints.MaxDelayMinutes > 0 {
		// Check if we're past SLO acceptable delay
		if job.NextRunAt != nil && proposedTime.After(job.NextRunAt.Add(time.Duration(constraints.MaxDelayMinutes)*time.Minute)) {
			violations = append(violations, ConstraintViolation{
				ConstraintType: "slo_delay",
				Description:    fmt.Sprintf("Job is %d minutes past SLO acceptable delay", int(proposedTime.Sub(*job.NextRunAt).Minutes())),
				Severity:       "warning",
			})
		}
	}

	// Determine if constraints are satisfied (only errors block execution)
	hasErrors := false
	for _, v := range violations {
		if v.Severity == "error" {
			hasErrors = true
			break
		}
	}

	return !hasErrors, violations, nil
}

// ============================================================================
// Blackout Window Handling
// ============================================================================

// isInBlackout checks if a time falls within a blackout window
func (ce *ConstraintEngine) isInBlackout(t time.Time, window BlackoutWindow) bool {
	// Simple check: is t between Start and End?
	return (t.Equal(window.Start) || t.After(window.Start)) && t.Before(window.End)
}

// parseTimeOfDay parses HH:MM format to minutes since midnight
func parseTimeOfDay(timeStr string) (int, error) {
	var hour, minute int
	_, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if err != nil {
		return 0, err
	}
	return hour*60 + minute, nil
}

// ============================================================================
// Execution Window Computation
// ============================================================================

// ComputeNextExecutionWindow finds the next valid execution window for a job
func (ce *ConstraintEngine) ComputeNextExecutionWindow(
	ctx context.Context,
	job *Job,
	fromTime time.Time,
	maxLookahead time.Duration,
) (*ExecutionWindow, error) {
	if maxLookahead == 0 {
		maxLookahead = 7 * 24 * time.Hour // Default 7 days
	}

	endTime := fromTime.Add(maxLookahead)
	currentTime := fromTime

	// Parse constraints
	var constraints SchedulingConstraints
	if len(job.Constraints) > 0 {
		json.Unmarshal(job.Constraints, &constraints)
	}

	// Iterate through time slots
	for currentTime.Before(endTime) {
		satisfied, violations, err := ce.EvaluateConstraints(ctx, job, currentTime)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate constraints: %w", err)
		}

		if satisfied {
			// Found a valid window, determine its end
			windowEnd := ce.findWindowEnd(ctx, job, currentTime, constraints)

			// Calculate window score
			score := ce.calculateWindowScore(ctx, job, currentTime, constraints)

			// Check business day status
			isBusinessDay := true
			if len(job.CalendarIDs) > 0 {
				calendarCode := job.CalendarIDs[0].String()
				isBusinessDay, _ = ce.calendarSvc.IsBusinessDay(ctx, calendarCode, currentTime)
			}

			return &ExecutionWindow{
				Start:         currentTime,
				End:           windowEnd,
				Score:         score,
				Reason:        "First available valid window",
				IsBusinessDay: isBusinessDay,
				SLOCompliant:  !job.SLOCritical || score > 0.8,
			}, nil
		}

		// Find next suggested time from violations
		var nextTime *time.Time
		for _, v := range violations {
			if v.SuggestedTime != nil {
				if nextTime == nil || v.SuggestedTime.Before(*nextTime) {
					nextTime = v.SuggestedTime
				}
			}
		}

		if nextTime != nil && nextTime.After(currentTime) {
			currentTime = *nextTime
		} else {
			// Default: try next hour
			currentTime = currentTime.Add(time.Hour)
		}
	}

	return nil, fmt.Errorf("no valid execution window found within %v", maxLookahead)
}

// findWindowEnd determines when the current valid window ends
func (ce *ConstraintEngine) findWindowEnd(
	ctx context.Context,
	job *Job,
	windowStart time.Time,
	constraints SchedulingConstraints,
) time.Time {
	// Default window length based on job timeout
	windowEnd := windowStart.Add(time.Duration(job.TimeoutSeconds) * time.Second)

	// Check for upcoming blackout windows
	var blackoutWindows []BlackoutWindow
	json.Unmarshal(job.BlackoutWindows, &blackoutWindows)
	blackoutWindows = append(blackoutWindows, constraints.BlackoutWindows...)

	for _, window := range blackoutWindows {
		// Check if blackout starts after our window start but before our current end
		if window.Start.After(windowStart) && window.Start.Before(windowEnd) {
			windowEnd = window.Start
		}
	}

	// Check for end of allowed time window
	if constraints.AllowedEndHour > 0 {
		dayEnd := time.Date(
			windowStart.Year(), windowStart.Month(), windowStart.Day(),
			constraints.AllowedEndHour, 0, 0, 0, windowStart.Location(),
		)
		if dayEnd.Before(windowEnd) {
			windowEnd = dayEnd
		}
	}

	return windowEnd
}

// calculateWindowScore computes a quality score for an execution window
func (ce *ConstraintEngine) calculateWindowScore(
	ctx context.Context,
	job *Job,
	windowStart time.Time,
	constraints SchedulingConstraints,
) float64 {
	score := 1.0

	// Penalize delay from scheduled time
	if job.NextRunAt != nil {
		delayMinutes := windowStart.Sub(*job.NextRunAt).Minutes()
		if delayMinutes > 0 {
			// Reduce score based on delay
			score -= delayMinutes / 60.0 * 0.1 // -10% per hour delay
			if score < 0.1 {
				score = 0.1
			}
		}
	}

	// Boost score for business hours (9 AM - 5 PM)
	hour := windowStart.Hour()
	if hour >= 9 && hour < 17 {
		score *= 1.1
		if score > 1.0 {
			score = 1.0
		}
	}

	// Boost score for SLO-critical jobs that are on time
	if constraints.SLOCritical && job.NextRunAt != nil {
		if windowStart.Before(*job.NextRunAt) || windowStart.Equal(*job.NextRunAt) {
			score = 1.0 // Force high score for on-time SLO jobs
		}
	}

	return score
}

// ============================================================================
// SLO-Aware Scheduling
// ============================================================================

// SLOSchedulingResult represents the result of SLO-aware scheduling
type SLOSchedulingResult struct {
	RecommendedTime  time.Time   `json:"recommended_time"`
	AlternativeTimes []time.Time `json:"alternative_times,omitempty"`
	SLOAtRisk        bool        `json:"slo_at_risk"`
	RiskLevel        string      `json:"risk_level"` // "low", "medium", "high", "critical"
	RiskReason       string      `json:"risk_reason,omitempty"`
	EstimatedLatency int64       `json:"estimated_latency_ms,omitempty"`
}

// ComputeSLOAwareSchedule computes the optimal schedule considering SLO requirements
func (ce *ConstraintEngine) ComputeSLOAwareSchedule(
	ctx context.Context,
	jobs []*Job,
	fromTime time.Time,
) (map[uuid.UUID]*SLOSchedulingResult, error) {
	results := make(map[uuid.UUID]*SLOSchedulingResult)

	// Sort jobs by SLO criticality and deadline
	sortedJobs := make([]*Job, len(jobs))
	copy(sortedJobs, jobs)
	sort.Slice(sortedJobs, func(i, j int) bool {
		// SLO-critical jobs come first
		if sortedJobs[i].SLOCritical != sortedJobs[j].SLOCritical {
			return sortedJobs[i].SLOCritical
		}
		// Then by next run time
		if sortedJobs[i].NextRunAt != nil && sortedJobs[j].NextRunAt != nil {
			return sortedJobs[i].NextRunAt.Before(*sortedJobs[j].NextRunAt)
		}
		return sortedJobs[i].Priority > sortedJobs[j].Priority
	})

	scheduledTimes := make(map[time.Time]int) // Track concurrent jobs

	for _, job := range sortedJobs {
		window, err := ce.ComputeNextExecutionWindow(ctx, job, fromTime, 24*time.Hour)
		if err != nil {
			// No valid window found
			results[job.ID] = &SLOSchedulingResult{
				RecommendedTime: fromTime,
				SLOAtRisk:       true,
				RiskLevel:       "critical",
				RiskReason:      "No valid execution window found within 24 hours",
			}
			continue
		}

		result := &SLOSchedulingResult{
			RecommendedTime: window.Start,
			SLOAtRisk:       !window.SLOCompliant,
		}

		// Calculate risk level
		if job.SLOCritical {
			if job.NextRunAt != nil {
				delay := window.Start.Sub(*job.NextRunAt)
				if delay > time.Hour {
					result.RiskLevel = "critical"
					result.RiskReason = fmt.Sprintf("Job delayed by %v", delay)
				} else if delay > 30*time.Minute {
					result.RiskLevel = "high"
					result.RiskReason = fmt.Sprintf("Job delayed by %v", delay)
				} else if delay > 10*time.Minute {
					result.RiskLevel = "medium"
				} else {
					result.RiskLevel = "low"
				}
			} else {
				result.RiskLevel = "low"
			}
		} else {
			result.RiskLevel = "low"
		}

		// Track for concurrency
		scheduledTimes[window.Start.Truncate(time.Hour)]++

		// Generate alternative times
		for i := 0; i < 3; i++ {
			altTime := window.Start.Add(time.Duration(i+1) * 30 * time.Minute)
			if altTime.Before(window.End) {
				result.AlternativeTimes = append(result.AlternativeTimes, altTime)
			}
		}

		results[job.ID] = result
	}

	return results, nil
}

// ============================================================================
// Tenant Constraint Rules
// ============================================================================

// EvaluateTenantRules applies tenant-specific constraint rules
func (ce *ConstraintEngine) EvaluateTenantRules(
	ctx context.Context,
	tenantID uuid.UUID,
	job *Job,
	proposedTime time.Time,
	rules []TenantConstraintRule,
) ([]ConstraintViolation, []string) {
	var violations []ConstraintViolation
	var appliedRules []string

	for _, rule := range rules {
		if !rule.Active {
			continue
		}

		// Check if rule conditions match
		if !ce.ruleConditionsMatch(job, proposedTime, rule.Conditions) {
			continue
		}

		// Apply rule based on type
		switch rule.RuleType {
		case "blackout":
			// Create a blackout violation
			violations = append(violations, ConstraintViolation{
				ConstraintType: "tenant_rule",
				Description:    fmt.Sprintf("Tenant rule '%s': %s", rule.Name, rule.Description),
				Severity:       "error",
			})
			appliedRules = append(appliedRules, rule.RuleID)

		case "priority_boost":
			// Adjust priority (would be applied to the job)
			appliedRules = append(appliedRules, rule.RuleID)
			ce.logger.Info("Applied priority boost rule",
				"rule_id", rule.RuleID,
				"job_id", job.ID,
			)

		case "resource_limit":
			// Check resource requirements
			if resources, ok := rule.Conditions["resources"].(map[string]interface{}); ok {
				for resource, limit := range resources {
					ce.logger.Debug("Checking resource limit",
						"resource", resource,
						"limit", limit,
					)
				}
			}
			appliedRules = append(appliedRules, rule.RuleID)
		}
	}

	return violations, appliedRules
}

// ruleConditionsMatch checks if a rule's conditions match the current context
func (ce *ConstraintEngine) ruleConditionsMatch(
	job *Job,
	proposedTime time.Time,
	conditions map[string]interface{},
) bool {
	// Check job category
	if categories, ok := conditions["categories"].([]interface{}); ok {
		matched := false
		for _, cat := range categories {
			if catStr, ok := cat.(string); ok && catStr == job.Category {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check time of day
	if timeRange, ok := conditions["time_range"].(map[string]interface{}); ok {
		startHour, hasStart := timeRange["start_hour"].(float64)
		endHour, hasEnd := timeRange["end_hour"].(float64)
		if hasStart && hasEnd {
			hour := proposedTime.Hour()
			if hour < int(startHour) || hour >= int(endHour) {
				return false
			}
		}
	}

	// Check job type
	if jobTypes, ok := conditions["job_types"].([]interface{}); ok {
		matched := false
		for _, jt := range jobTypes {
			if jtStr, ok := jt.(string); ok && jtStr == job.JobType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}
