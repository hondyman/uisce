package availability

import (
	"context"
	"fmt"
	"time"

	"calendar-service/internal/cache"
	"calendar-service/internal/database"
	"calendar-service/internal/metrics"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/teambition/rrule-go"
)

type Checker struct {
	dbClient    *database.Client
	cacheClient *cache.Client
	cacheTTL    time.Duration
	logger      *logrus.Entry
	metrics     *metrics.MetricsCollector
}

func NewChecker(db *database.Client, cc *cache.Client, ttl time.Duration, logger *logrus.Entry, m *metrics.MetricsCollector) *Checker {
	return &Checker{
		dbClient:    db,
		cacheClient: cc,
		cacheTTL:    ttl,
		logger:      logger.WithField("component", "availability"),
		metrics:     m,
	}
}

func (c *Checker) ResolveProfile(ctx context.Context, tenantID, region, profileName string) (*ResolvedCalendar, error) {
	logger := c.logger.WithFields(logrus.Fields{
		"tenant_id":    tenantID,
		"region":       region,
		"profile_name": profileName,
	})

	if c.cacheClient != nil {
		cached, err := c.cacheClient.Get(ctx, tenantID, region, profileName)
		if err == nil && cached != nil {
			logger.Debug("Cache hit for profile resolution")
			if c.metrics != nil {
				c.metrics.RecordCacheHit()
			}
			return cached, nil
		}
		if c.metrics != nil {
			c.metrics.RecordCacheMiss()
		}
	}

	resolved, err := c.computeResolvedProfile(ctx, tenantID, region, profileName)
	if err != nil {
		return nil, err
	}

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

	profile, err := c.fetchScheduleProfile(ctx, tenantID, profileName)
	if err != nil {
		logger.WithError(err).Error("Failed to fetch schedule profile")
		if c.metrics != nil {
			c.metrics.RecordResolutionError()
		}
		return nil, fmt.Errorf("fetch profile: %w", err)
	}
	if profile == nil {
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

	rangeStart := time.Now().UTC()
	rangeEnd := rangeStart.AddDate(0, 0, 90)

	holidays, err := c.fetchHolidaysForCalendars(ctx, tenantID, profile.CalendarIDs, rangeStart, rangeEnd)
	if err != nil {
		logger.WithError(err).Error("Failed to fetch holidays")
		if c.metrics != nil {
			c.metrics.RecordResolutionError()
		}
		return nil, fmt.Errorf("fetch holidays: %w", err)
	}

	blackouts, err := c.fetchAndExpandBlackouts(ctx, tenantID, profile.CalendarIDs, rangeStart, rangeEnd)
	if err != nil {
		logger.WithError(err).Error("Failed to fetch blackouts")
		if c.metrics != nil {
			c.metrics.RecordResolutionError()
		}
		return nil, fmt.Errorf("fetch blackouts: %w", err)
	}

	resolvedHolidays, resolvedBlackouts := c.applyConflictResolution(
		holidays, blackouts, profile.ConflictResolution, profile.CalendarPriorities,
	)

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

func (c *Checker) fetchScheduleProfile(ctx context.Context, tenantID, profileName string) (*ScheduleProfile, error) {
	query := `
		SELECT sp.id, sp.profile_name, sp.timezone, sp.region, sp.conflict_resolution,
			sp.valid_from, sp.valid_to, sp.active
		FROM schedule_profiles sp
		WHERE sp.tenant_id = $1 AND sp.profile_name = $2 AND sp.active = true
	`

	var id, name, timezone, region, conflictRes string
	var validFrom time.Time
	var validTo *time.Time
	var active bool

	err := c.dbClient.Pool().QueryRow(ctx, query, tenantID, profileName).Scan(
		&id, &name, &timezone, &region, &conflictRes, &validFrom, &validTo, &active,
	)

	if err != nil {
		return nil, nil
	}

	return &ScheduleProfile{
		ID:                  id,
		ProfileName:         name,
		Timezone:            timezone,
		Region:              region,
		ConflictResolution:  conflictRes,
		CalendarIDs:         []string{},
		CalendarPriorities:  map[string]int{},
		ValidFrom:           validFrom,
		ValidTo:             validTo,
		Active:              active,
	}, nil
}

func (c *Checker) fetchHolidaysForCalendars(ctx context.Context, tenantID string, calendarIDs []string, rangeStart, rangeEnd time.Time) ([]Holiday, error) {
	if len(calendarIDs) == 0 {
		return []Holiday{}, nil
	}

	query := `
		SELECT id, name, holidays
		FROM calendars
		WHERE tenant_id = $1 AND id = ANY($2)
	`

	rows, err := c.dbClient.Pool().Query(ctx, query, tenantID, calendarIDs)
	if err != nil {
		c.logger.WithError(err).WithField("calendar_ids_count", len(calendarIDs)).Warn("Failed to fetch holidays")
		return []Holiday{}, nil
	}
	defer rows.Close()

	holidays := []Holiday{}
	seen := make(map[string]Holiday)

	for rows.Next() {
		var id, name string
		var holidaysJSON []byte
		if err := rows.Scan(&id, &name, &holidaysJSON); err != nil {
			continue
		}

		var parsedHolidays []struct {
			Date     string `json:"date"`
			Name     string `json:"name"`
			Type     string `json:"type"`
			Severity string `json:"severity"`
			AllDay   bool   `json:"all_day"`
		}
		if err := parseJSON(holidaysJSON, &parsedHolidays); err != nil {
			continue
		}

		for _, h := range parsedHolidays {
			holidayDate, err := time.Parse("2006-01-02", h.Date)
			if err != nil {
				continue
			}

			holiday := Holiday{
				Date:     holidayDate,
				Name:     h.Name,
				Type:     h.Type,
				Severity: h.Severity,
				AllDay:   true,
			}

			key := fmt.Sprintf("%s_%s", holidayDate.Format("2006-01-02"), h.Name)
			if existing, exists := seen[key]; !exists || c.isHigherSeverity(h.Severity, existing.Severity) {
				seen[key] = holiday
			}
		}
	}

	for _, h := range seen {
		holidays = append(holidays, h)
	}

	return holidays, nil
}

func (c *Checker) fetchAndExpandBlackouts(ctx context.Context, tenantID string, calendarIDs []string, rangeStart, rangeEnd time.Time) ([]Blackout, error) {
	if len(calendarIDs) == 0 {
		return []Blackout{}, nil
	}

	query := `
		SELECT id, name, start_time, end_time, is_recurring, recurrence_rule, reason, severity
		FROM blackouts
		WHERE tenant_id = $1 AND calendar_id = ANY($2)
			AND start_time < $4 AND end_time > $3
	`

	rows, err := c.dbClient.Pool().Query(ctx, query, tenantID, calendarIDs, rangeStart, rangeEnd)
	if err != nil {
		c.logger.WithError(err).WithField("calendar_ids_count", len(calendarIDs)).Warn("Failed to fetch blackouts")
		return []Blackout{}, nil
	}
	defer rows.Close()

	allBlackouts := make([]Blackout, 0)

	for rows.Next() {
		var id, name, reason, severity, recurrenceRule string
		var startTime, endTime time.Time
		var isRecurring bool

		if err := rows.Scan(&id, &name, &startTime, &endTime, &isRecurring, &recurrenceRule, &reason, &severity); err != nil {
			continue
		}

		blackout := Blackout{
			ID:             id,
			Name:           name,
			StartTime:      startTime,
			EndTime:        endTime,
			IsRecurring:    isRecurring,
			RecurrenceRule: recurrenceRule,
			Reason:         reason,
			Severity:       severity,
		}

		if isRecurring && recurrenceRule != "" {
			expanded := c.expandRecurringBlackout(blackout, rangeStart, rangeEnd)
			allBlackouts = append(allBlackouts, expanded...)
		} else {
			allBlackouts = append(allBlackouts, blackout)
		}
	}

	allBlackouts = c.deduplicateBlackouts(allBlackouts)

	return allBlackouts, nil
}

func (c *Checker) applyConflictResolution(holidays []Holiday, blackouts []Blackout, strategy string, priorities map[string]int) ([]Holiday, []Blackout) {
	return c.deduplicateHolidays(holidays), c.deduplicateBlackouts(blackouts)
}

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

func (c *Checker) isHigherSeverity(a, b string) bool {
	order := map[string]int{"LOW": 1, "MEDIUM": 2, "HIGH": 3, "CRITICAL": 4}
	return order[a] > order[b]
}

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

	loc, err := time.LoadLocation(resolved.Timezone)
	if err != nil {
		loc = time.UTC
	}

	startLocal := start.In(loc)
	endLocal := end.In(loc)

	for _, h := range resolved.Holidays {
		hLocal := h.In(loc)
		if isSameDay(startLocal, hLocal) || isSameDay(endLocal.Add(-time.Second), hLocal) {
			result.Available = false
			result.Reasons = append(result.Reasons, fmt.Sprintf("Holiday: %s", hLocal.Format("2006-01-02")))
		}
	}

	for _, br := range resolved.Blackouts {
		if start.Before(br.End) && end.After(br.Start) {
			result.Available = false
			result.Reasons = append(result.Reasons, fmt.Sprintf("Blackout: %s to %s", br.Start.Format(time.RFC3339), br.End.Format(time.RFC3339)))
		}
	}

	return result, nil
}

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
	maxDays := 30

	for i := 0; i < maxDays; i++ {
		candidateLocal := afterLocal.AddDate(0, 0, i)
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

		candidateUTC := candidateLocal.In(time.UTC)
		endUTC := candidateUTC.Add(duration)

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

func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func (c *Checker) expandRecurringBlackout(blackout Blackout, startTime, endTime time.Time) []Blackout {
	if !blackout.IsRecurring || blackout.RecurrenceRule == "" {
		return []Blackout{blackout}
	}

	expanded := make([]Blackout, 0)

	rule, err := rrule.StrToRRule(blackout.RecurrenceRule)
	if err != nil {
		c.logger.WithError(err).WithField("recurrence_rule", blackout.RecurrenceRule).Warn("Failed to parse recurrence rule")
		return []Blackout{blackout}
	}

	occurrences := rule.Between(startTime, endTime, true)
	if len(occurrences) == 0 {
		return []Blackout{blackout}
	}

	duration := blackout.EndTime.Sub(blackout.StartTime)

	for _, occurrence := range occurrences {
		expanded = append(expanded, Blackout{
			ID:             uuid.New().String(),
			Name:           blackout.Name,
			StartTime:      occurrence,
			EndTime:        occurrence.Add(duration),
			IsRecurring:    false,
			RecurrenceRule: "",
			Reason:         blackout.Reason,
			Severity:       blackout.Severity,
		})
	}

	return expanded
}

func (c *Checker) InvalidateProfileNameCache(ctx context.Context, tenantID, profileName string) error {
	cacheKey := fmt.Sprintf("profile:%s:%s", tenantID, profileName)
	return c.cacheClient.DelString(ctx, cacheKey)
}

func (c *Checker) ResolveProfileNameForCalendar(ctx context.Context, tenantID, calendarID string) (string, error) {
	cacheKey := fmt.Sprintf("calendar_profile:%s:%s", tenantID, calendarID)

	if cached, err := c.cacheClient.GetString(ctx, cacheKey); err == nil && cached != "" {
		return cached, nil
	}

	query := `SELECT profile_name FROM calendars WHERE tenant_id = $1 AND id = $2`

	var profileName string
	err := c.dbClient.Pool().QueryRow(ctx, query, tenantID, calendarID).Scan(&profileName)
	if err != nil {
		return "default", nil
	}

	if profileName == "" {
		profileName = "default"
	}

	_ = c.cacheClient.SetString(ctx, cacheKey, profileName, c.cacheTTL)

	return profileName, nil
}

func parseJSON(data []byte, v interface{}) error {
	return nil
}
