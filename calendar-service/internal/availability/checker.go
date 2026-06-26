package availability

import (
	"context"
	"fmt"
	"time"

	"calendar-service/internal/cache"
	"calendar-service/internal/hasura"
	"calendar-service/internal/metrics"

	"github.com/sirupsen/logrus"
	"github.com/teambition/rrule-go"
)

// Checker validates availability based on calendars, holidays, and blackouts
type Checker struct {
	hasuraClient *hasura.Client
	cacheClient  *cache.Client
	cacheTTL     time.Duration
	logger       *logrus.Entry
	metrics      *metrics.MetricsCollector
}

// NewChecker creates a new availability checker
func NewChecker(hc *hasura.Client, cc *cache.Client, ttl time.Duration, logger *logrus.Entry, m *metrics.MetricsCollector) *Checker {
	return &Checker{
		hasuraClient: hc,
		cacheClient:  cc,
		cacheTTL:     ttl,
		logger:       logger.WithField("component", "availability"),
		metrics:      m,
	}
}

// ResolveProfile resolves a calendar profile with caching
func (c *Checker) ResolveProfile(ctx context.Context, tenantID, region, profileName string) (*ResolvedCalendar, error) {
	logger := c.logger.WithFields(logrus.Fields{
		"tenant_id":    tenantID,
		"region":       region,
		"profile_name": profileName,
	})

	// 1. Try Cache
	if c.cacheClient != nil {
		cached, err := c.cacheClient.Get(ctx, tenantID, region, profileName)
		if err == nil && cached != nil {
			logger.Debug("Cache hit for profile resolution")
			if c.metrics != nil {
				c.metrics.RecordCacheHit()
			}
			return cached, nil
		}
		// Cache miss or error
		if c.metrics != nil {
			c.metrics.RecordCacheMiss()
		}
	}

	// 2. Cache Miss - Resolve from DB
	resolved, err := c.computeResolvedProfile(ctx, tenantID, region, profileName)
	if err != nil {
		return nil, err
	}

	// 3. Populate Cache (Async)
	if c.cacheClient != nil && resolved != nil {
		resolved.Region = region
		c.cacheClient.SetAsync(ctx, tenantID, region, profileName, resolved)
	}

	return resolved, nil
}

func (c *Checker) computeResolvedProfile(ctx context.Context, tenantID, region, profileName string) (*ResolvedCalendar, error) {
	logger := c.logger.WithFields(logrus.Fields{
		"tenant_id":    tenantID,
		"region":       region,
		"profile_name": profileName,
	})

	startTime := time.Now()

	// Step 1: Fetch the schedule profile
	profile, err := c.fetchScheduleProfile(ctx, tenantID, profileName)
	if err != nil {
		logger.WithError(err).Error("Failed to fetch schedule profile")
		if c.metrics != nil {
			c.metrics.RecordResolutionError()
		}
		return nil, fmt.Errorf("fetch profile: %w", err)
	}
	if profile == nil {
		logger.Warn("Schedule profile not found")
		// Return empty profile (no holidays/blackouts)
		return &ResolvedCalendar{
			TenantID:    tenantID,
			Region:      region,
			ProfileName: profileName,
			Holidays:    []time.Time{},
			Blackouts:   []TimeRange{},
			Timezone:    "UTC",
			ResolvedAt:  time.Now().UTC(),
			Version:     "v1",
		}, nil
	}

	logger.WithField("calendars_count", len(profile.CalendarIDs)).Debug("Fetched schedule profile")

	// Query time range: 90 days window
	rangeStart := time.Now().UTC()
	rangeEnd := rangeStart.AddDate(0, 0, 90)

	// Step 2: Fetch holidays from all linked calendars
	holidays, err := c.fetchHolidaysForCalendars(ctx, tenantID, profile.CalendarIDs, rangeStart, rangeEnd)
	if err != nil {
		logger.WithError(err).Error("Failed to fetch holidays")
		if c.metrics != nil {
			c.metrics.RecordResolutionError()
		}
		return nil, fmt.Errorf("fetch holidays: %w", err)
	}

	// Step 3: Fetch and expand blackouts
	blackouts, err := c.fetchAndExpandBlackouts(ctx, tenantID, profile.CalendarIDs, rangeStart, rangeEnd)
	if err != nil {
		logger.WithError(err).Error("Failed to fetch blackouts")
		if c.metrics != nil {
			c.metrics.RecordResolutionError()
		}
		return nil, fmt.Errorf("fetch blackouts: %w", err)
	}

	// Step 4: Apply conflict resolution
	resolvedHolidays, resolvedBlackouts := c.applyConflictResolution(
		holidays, blackouts, profile.ConflictResolution, profile.CalendarPriorities,
	)

	// Step 5: Convert to cache format
	cachedHolidays := make([]time.Time, 0, len(resolvedHolidays))
	for _, h := range resolvedHolidays {
		cachedHolidays = append(cachedHolidays, h.Date)
	}

	cachedBlackouts := make([]TimeRange, 0, len(resolvedBlackouts))
	for _, b := range resolvedBlackouts {
		cachedBlackouts = append(cachedBlackouts, TimeRange{
			Start: b.StartTime,
			End:   b.EndTime,
		})
	}

	resolved := &ResolvedCalendar{
		TenantID:    tenantID,
		Region:      region,
		ProfileName: profileName,
		Holidays:    cachedHolidays,
		Blackouts:   cachedBlackouts,
		Timezone:    profile.Timezone,
		ResolvedAt:  time.Now().UTC(),
		Version:     fmt.Sprintf("v1-%d", len(cachedHolidays)+len(cachedBlackouts)),
	}

	duration := time.Since(startTime).Seconds()
	if c.metrics != nil {
		c.metrics.RecordResolutionDuration(duration)
		c.metrics.RecordProfileResolution()
	}

	logger.WithFields(logrus.Fields{
		"holidays":    len(cachedHolidays),
		"blackouts":   len(cachedBlackouts),
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Debug("Successfully resolved profile")

	return resolved, nil
}

// fetchScheduleProfile queries Hasura for a schedule profile with its linked calendars
func (c *Checker) fetchScheduleProfile(ctx context.Context, tenantID, profileName string) (*ScheduleProfile, error) {
	// Use go-graphql-client with struct-based query inference
	var result struct {
		ScheduleProfiles []struct {
			ID                 string     `graphql:"id" json:"id"`
			ProfileName        string     `graphql:"profile_name" json:"profile_name"`
			Timezone           string     `graphql:"timezone" json:"timezone"`
			Region             string     `graphql:"region" json:"region"`
			ConflictResolution string     `graphql:"conflict_resolution" json:"conflict_resolution"`
			ValidFrom          time.Time  `graphql:"valid_from" json:"valid_from"`
			ValidTo            *time.Time `graphql:"valid_to" json:"valid_to"`
			Active             bool       `graphql:"active" json:"active"`
			ProfileCalendars   []struct {
				CalendarID string `graphql:"calendar_id" json:"calendar_id"`
				Weight     int    `graphql:"weight" json:"weight"`
				Calendar   struct {
					ID       string `graphql:"id" json:"id"`
					Name     string `graphql:"name" json:"name"`
					Region   string `graphql:"region" json:"region"`
					Priority int    `graphql:"priority" json:"priority"`
				} `graphql:"calendar" json:"calendar"`
			} `graphql:"profile_calendars" json:"profile_calendars"`
		} `graphql:"schedule_profiles" json:"schedule_profiles"`
	}

	err := c.hasuraClient.Query(ctx, &result, map[string]interface{}{
		"tenantID":    tenantID,
		"profileName": profileName,
	})

	if err != nil {
		return nil, fmt.Errorf("hasura query error: %w", err)
	}

	if len(result.ScheduleProfiles) == 0 {
		return nil, nil // Not found, but not an error
	}

	sp := result.ScheduleProfiles[0]

	// Build calendar IDs and priorities
	calendarIDs := make([]string, 0, len(sp.ProfileCalendars))
	calendarPriorities := make(map[string]int)

	for _, pc := range sp.ProfileCalendars {
		calendarIDs = append(calendarIDs, pc.CalendarID)
		priority := pc.Weight
		if priority == 0 {
			priority = pc.Calendar.Priority
		}
		calendarPriorities[pc.CalendarID] = priority
	}

	return &ScheduleProfile{
		ID:                 sp.ID,
		ProfileName:        sp.ProfileName,
		Timezone:           sp.Timezone,
		Region:             sp.Region,
		ConflictResolution: sp.ConflictResolution,
		CalendarIDs:        calendarIDs,
		CalendarPriorities: calendarPriorities,
		ValidFrom:          sp.ValidFrom,
		ValidTo:            sp.ValidTo,
		Active:             sp.Active,
	}, nil
}

// fetchHolidaysForCalendars fetches holidays from calendar JSONB fields
func (c *Checker) fetchHolidaysForCalendars(ctx context.Context, tenantID string, calendarIDs []string, rangeStart, rangeEnd time.Time) ([]Holiday, error) {
	if len(calendarIDs) == 0 {
		return []Holiday{}, nil
	}

	// Query calendars to get holidays JSONB field
	var result struct {
		Calendars []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Holidays []struct {
				Date     string `json:"date"` // YYYY-MM-DD format
				Name     string `json:"name"`
				Type     string `json:"type"`     // public, observance, bank
				Severity string `json:"severity"` // HIGH, MEDIUM, LOW
				AllDay   bool   `json:"all_day"`
			} `json:"holidays"` // JSONB array
		} `json:"calendars"`
	}

	err := c.hasuraClient.Query(ctx, &result, map[string]interface{}{
		"tenantID":    tenantID,
		"calendarIDs": calendarIDs,
	})

	if err != nil {
		c.logger.WithError(err).WithField("calendar_ids_count", len(calendarIDs)).Warn("Failed to fetch holidays")
		return []Holiday{}, nil // Not fatal - continue without holidays
	}

	// Parse and collect holidays
	holidays := []Holiday{}
	seen := make(map[string]Holiday)

	for _, cal := range result.Calendars {
		for _, h := range cal.Holidays {
			// Parse date string
			holidayDate, err := time.Parse("2006-01-02", h.Date)
			if err != nil {
				c.logger.WithError(err).WithField("date_string", h.Date).Debug("Failed to parse holiday date")
				continue
			}

			holiday := Holiday{
				Date:     holidayDate,
				Name:     h.Name,
				Type:     h.Type,
				Severity: h.Severity,
				AllDay:   true, // Always true for holidays
			}

			// Deduplicate by date+name, keeping highest severity
			key := fmt.Sprintf("%s_%s", holidayDate.Format("2006-01-02"), h.Name)
			if existing, exists := seen[key]; !exists || c.isHigherSeverity(h.Severity, existing.Severity) {
				seen[key] = holiday
			}
		}
	}

	// Convert map to slice
	for _, h := range seen {
		holidays = append(holidays, h)
	}

	c.logger.WithFields(logrus.Fields{
		"calendar_ids_count": len(calendarIDs),
		"holidays_count":     len(holidays),
	}).Debug("Fetched holidays from calendars")

	return holidays, nil
}

// fetchAndExpandBlackouts fetches and expands recurring blackouts
func (c *Checker) fetchAndExpandBlackouts(ctx context.Context, tenantID string, calendarIDs []string, rangeStart, rangeEnd time.Time) ([]Blackout, error) {
	if len(calendarIDs) == 0 {
		return []Blackout{}, nil
	}

	// Query blackouts table
	var result struct {
		Blackouts []struct {
			ID             string    `json:"id"`
			Name           string    `json:"name"`
			StartTime      time.Time `json:"start_time"`
			EndTime        time.Time `json:"end_time"`
			IsRecurring    bool      `json:"is_recurring"`
			RecurrenceRule string    `json:"recurrence_rule"`
			Reason         string    `json:"reason"`
			Severity       string    `json:"severity"`
		} `json:"blackouts"`
	}

	err := c.hasuraClient.Query(ctx, &result, map[string]interface{}{
		"tenantID":    tenantID,
		"calendarIDs": calendarIDs,
		"rangeStart":  rangeStart,
		"rangeEnd":    rangeEnd,
	})

	if err != nil {
		c.logger.WithError(err).WithField("calendar_ids_count", len(calendarIDs)).Warn("Failed to fetch blackouts")
		return []Blackout{}, nil // Not fatal - continue without blackouts
	}

	// Process blackouts - expand recurring ones
	allBlackouts := make([]Blackout, 0)

	for _, b := range result.Blackouts {
		blackout := Blackout{
			ID:             b.ID,
			Name:           b.Name,
			StartTime:      b.StartTime,
			EndTime:        b.EndTime,
			IsRecurring:    b.IsRecurring,
			RecurrenceRule: b.RecurrenceRule,
			Reason:         b.Reason,
			Severity:       b.Severity,
		}

		if b.IsRecurring && b.RecurrenceRule != "" {
			// Expand recurring blackout
			expanded := c.expandRecurringBlackout(blackout, rangeStart, rangeEnd)
			allBlackouts = append(allBlackouts, expanded...)
		} else {
			// One-time blackout
			allBlackouts = append(allBlackouts, blackout)
		}
	}

	// Deduplicate
	allBlackouts = c.deduplicateBlackouts(allBlackouts)

	c.logger.WithFields(logrus.Fields{
		"calendar_ids_count": len(calendarIDs),
		"blackouts_count":    len(allBlackouts),
	}).Debug("Fetched and expanded blackouts")

	return allBlackouts, nil
}

// applyConflictResolution applies conflict resolution strategy
func (c *Checker) applyConflictResolution(holidays []Holiday, blackouts []Blackout, strategy string, priorities map[string]int) ([]Holiday, []Blackout) {
	// For now, deduplicate by severity (same for all strategies)
	// Future: Implement INTERSECTION and PRIORITY-specific logic
	return c.deduplicateHolidays(holidays), c.deduplicateBlackouts(blackouts)
}

// deduplicateHolidays removes duplicates, keeping highest severity
func (c *Checker) deduplicateHolidays(holidays []Holiday) []Holiday {
	if len(holidays) == 0 {
		return holidays
	}

	seen := make(map[string]Holiday)

	for _, h := range holidays {
		key := fmt.Sprintf("%s_%s", h.Date.Format("2006-01-02"), h.Name)
		existing, exists := seen[key]
		if !exists || c.isHigherSeverity(h.Severity, existing.Severity) {
			seen[key] = h
		}
	}

	result := make([]Holiday, 0, len(seen))
	for _, h := range seen {
		result = append(result, h)
	}

	return result
}

// deduplicateBlackouts removes duplicates, keeping highest severity
func (c *Checker) deduplicateBlackouts(blackouts []Blackout) []Blackout {
	if len(blackouts) == 0 {
		return blackouts
	}

	seen := make(map[string]Blackout)

	for _, b := range blackouts {
		key := fmt.Sprintf("%s_%s_%s", b.StartTime.Format(time.RFC3339), b.EndTime.Format(time.RFC3339), b.Name)
		existing, exists := seen[key]
		if !exists || c.isHigherSeverity(b.Severity, existing.Severity) {
			seen[key] = b
		}
	}

	result := make([]Blackout, 0, len(seen))
	for _, b := range seen {
		result = append(result, b)
	}

	return result
}

// isHigherSeverity compares severity levels
func (c *Checker) isHigherSeverity(a, b string) bool {
	order := map[string]int{"LOW": 1, "MEDIUM": 2, "HIGH": 3, "CRITICAL": 4}
	return order[a] > order[b]
}

// CheckAvailability checks if a time range is available
func (c *Checker) CheckAvailability(ctx context.Context, tenantID, region, profileName string, start, end time.Time) (*AvailabilityResult, error) {
	resolved, err := c.ResolveProfile(ctx, tenantID, region, profileName)
	if err != nil {
		return nil, err
	}

	result := &AvailabilityResult{
		Available: true,
		Reasons:   []string{},
		CheckedAt: time.Now().UTC(),
	}

	// Load timezone
	loc, err := time.LoadLocation(resolved.Timezone)
	if err != nil {
		loc = time.UTC
	}

	startLocal := start.In(loc)
	endLocal := end.In(loc)

	// Check Holidays (Date Only Comparison in Local Time)
	for _, h := range resolved.Holidays {
		hLocal := h.In(loc)
		if isSameDay(startLocal, hLocal) || isSameDay(endLocal.Add(-time.Second), hLocal) {
			result.Available = false
			result.Reasons = append(result.Reasons, fmt.Sprintf("Holiday: %s", hLocal.Format("2006-01-02")))
		}
	}

	// Check Blackouts (Absolute Time Comparison in UTC)
	for _, br := range resolved.Blackouts {
		// Check if time range overlaps with blackout
		if start.Before(br.End) && end.After(br.Start) {
			result.Available = false
			result.Reasons = append(result.Reasons, fmt.Sprintf("Blackout: %s to %s", br.Start.Format(time.RFC3339), br.End.Format(time.RFC3339)))
		}
	}

	return result, nil
}

// FindNextAvailableSlot finds the next available time slot
func (c *Checker) FindNextAvailableSlot(ctx context.Context, tenantID, region, profileName string, after time.Time, duration time.Duration) (time.Time, error) {
	logger := c.logger.WithFields(logrus.Fields{
		"tenant_id":    tenantID,
		"region":       region,
		"profile_name": profileName,
		"after":        after,
		"duration":     duration,
	})

	resolved, err := c.ResolveProfile(ctx, tenantID, region, profileName)
	if err != nil {
		return time.Time{}, err
	}

	loc, _ := time.LoadLocation(resolved.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	afterLocal := after.In(loc)
	maxDays := 30 // Search within 30 days

	for i := 0; i < maxDays; i++ {
		// Try start of day in local time
		candidateLocal := afterLocal.AddDate(0, 0, i)
		// Reset to start of business day (9 AM)
		candidateLocal = time.Date(
			candidateLocal.Year(),
			candidateLocal.Month(),
			candidateLocal.Day(),
			9, 0, 0, 0,
			candidateLocal.Location(),
		)

		if candidateLocal.Before(afterLocal) {
			candidateLocal = candidateLocal.AddDate(0, 0, 1)
		}

		// Convert back to UTC
		candidateUTC := candidateLocal.In(time.UTC)
		endUTC := candidateUTC.Add(duration)

		// Check availability
		result, err := c.CheckAvailability(ctx, tenantID, region, profileName, candidateUTC, endUTC)
		if err != nil {
			logger.WithError(err).Warn("Failed to check availability")
			continue
		}

		if result.Available {
			logger.WithField("next_slot", candidateUTC).Info("Found next available slot")
			return candidateUTC, nil
		}
	}

	return time.Time{}, fmt.Errorf("no available slot found within %d days", maxDays)
}

// Helper function: Check if two times are the same day (local time)
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// expandRecurringBlackout expands a recurring blackout into individual occurrences
func (c *Checker) expandRecurringBlackout(blackout Blackout, startTime, endTime time.Time) []Blackout {
	if !blackout.IsRecurring || blackout.RecurrenceRule == "" {
		return []Blackout{blackout}
	}

	expanded := make([]Blackout, 0)

	// Parse RRULE and generate occurrences
	rule, err := rrule.StrToRRule(blackout.RecurrenceRule)
	if err != nil {
		c.logger.WithError(err).WithField("recurrence_rule", blackout.RecurrenceRule).Warn("Failed to parse recurrence rule")
		return []Blackout{blackout}
	}

	// Generate occurrences within our time range
	occurrences := rule.Between(startTime, endTime, true)
	if len(occurrences) == 0 {
		return []Blackout{blackout}
	}

	// Duration of the blackout
	duration := blackout.EndTime.Sub(blackout.StartTime)

	// Create expanded blackout for each occurrence
	for _, occurrence := range occurrences {
		expanded = append(expanded, Blackout{
			ID:             blackout.ID,
			Name:           blackout.Name,
			StartTime:      occurrence,
			EndTime:        occurrence.Add(duration),
			IsRecurring:    false, // Individual expanded instances are not recurring
			RecurrenceRule: "",
			Reason:         blackout.Reason,
			Severity:       blackout.Severity,
		})
	}

	return expanded
}

// InvalidateProfileNameCache invalidates cache for a profile
func (c *Checker) InvalidateProfileNameCache(ctx context.Context, tenantID, profileName string) error {
	cacheKey := fmt.Sprintf("profile:%s:%s", tenantID, profileName)
	return c.cacheClient.DelString(ctx, cacheKey)
}

// ResolveProfileNameForCalendar resolves a profile name for a specific calendar
func (c *Checker) ResolveProfileNameForCalendar(ctx context.Context, tenantID, calendarID string) (string, error) {
	cacheKey := fmt.Sprintf("calendar_profile:%s:%s", tenantID, calendarID)

	// Try cache first
	if cached, err := c.cacheClient.GetString(ctx, cacheKey); err == nil && cached != "" {
		return cached, nil
	}

	// Query Hasura for the profile name
	var result struct {
		Calendars []struct {
			ProfileName string `graphql:"profile_name" json:"profile_name"`
		} `graphql:"calendars" json:"calendars"`
	}

	err := c.hasuraClient.Query(ctx, &result, map[string]interface{}{
		"tenantID":   tenantID,
		"calendarID": calendarID,
	})

	if err != nil {
		return "", fmt.Errorf("failed to resolve profile for calendar %s: %w", calendarID, err)
	}

	// Extract profile_name from result
	profileName := "default"
	if len(result.Calendars) > 0 && result.Calendars[0].ProfileName != "" {
		profileName = result.Calendars[0].ProfileName
	}

	// Cache the result
	_ = c.cacheClient.SetString(ctx, cacheKey, profileName, c.cacheTTL)

	return profileName, nil
}
