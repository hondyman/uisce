package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AdjustmentConvention represents ISDA business day conventions
type AdjustmentConvention string

const (
	Following         AdjustmentConvention = "FOLLOWING"
	ModifiedFollowing AdjustmentConvention = "MODIFIED_FOLLOWING"
	Preceding         AdjustmentConvention = "PRECEDING"
	Unadjusted        AdjustmentConvention = "UNADJUSTED"
)

// Calendar represents a business calendar
type Calendar struct {
	ID          uuid.UUID   `db:"calendar_id"`
	TenantID    *uuid.UUID  `db:"tenant_id"`
	Code        string      `db:"calendar_code"`
	Name        string      `db:"calendar_name"`
	Description string      `db:"description"`
	ParentIDs   []uuid.UUID `db:"parent_calendar_ids"`
	Timezone    string      `db:"timezone"`
	WeekendDays []int       `db:"weekend_days"`
	Active      bool        `db:"active"`
	IsGlobal    bool        `db:"is_global"`
}

// Holiday represents a calendar holiday
type Holiday struct {
	ID               uuid.UUID  `db:"holiday_id"`
	CalendarID       uuid.UUID  `db:"calendar_id"`
	Date             time.Time  `db:"holiday_date"`
	Name             string     `db:"holiday_name"`
	Type             string     `db:"holiday_type"`
	IsHalfDay        bool       `db:"is_half_day"`
	HalfDayCloseTime *time.Time `db:"half_day_close_time"`
	IsRecurring      bool       `db:"is_recurring"`
	RecurrenceRule   *string    `db:"recurrence_rule"`
	ObservedDate     *time.Time `db:"observed_date"`
}

// Service provides calendar operations
type Service struct {
	db            *sqlx.DB
	calendarCache map[string]*Calendar       // calendar_code -> Calendar
	holidayCache  map[string]map[string]bool // calendar_code -> date -> is_holiday
}

// NewService creates a new calendar service
func NewService(db *sqlx.DB) *Service {
	return &Service{
		db:            db,
		calendarCache: make(map[string]*Calendar),
		holidayCache:  make(map[string]map[string]bool),
	}
}

// IsBusinessDay checks if a date is a business day
// TODO: Migrate to Hasura GraphQL with custom function or materialized view:
//
//	query IsBusinessDay($calendar_code: String!, $date: date!) {
//	  is_business_day(args: {calendar_code: $calendar_code, check_date: $date})
//	}
//
// Note: Stored procedure call includes calendar inheritance logic
func (s *Service) IsBusinessDay(ctx context.Context, calendarCode string, date time.Time) (bool, error) {
	// Prefer database function (includes inheritance), but fall back in sqlite/dev environments.
	if s.db != nil {
		var isBusiness bool
		query := "SELECT is_business_day($1, $2::DATE)"
		err := s.db.GetContext(ctx, &isBusiness, query, calendarCode, date.Format("2006-01-02"))
		if err == nil {
			return isBusiness, nil
		}
		if !shouldFallbackToInMemoryCalendar(err) {
			return false, err
		}
	}

	return isBusinessDayFallback(calendarCode, date), nil
}

// NextBusinessDay finds the next business day after the given date
// TODO: Migrate to Hasura GraphQL with custom function:
//
//	query NextBusinessDay($calendar_code: String!, $from_date: date!) {
//	  next_business_day(args: {calendar_code: $calendar_code, from_date: $from_date})
//	}
func (s *Service) NextBusinessDay(ctx context.Context, calendarCode string, from time.Time) (time.Time, error) {
	if s.db != nil {
		var nextDay time.Time
		query := "SELECT next_business_day($1, $2::DATE)"
		err := s.db.GetContext(ctx, &nextDay, query, calendarCode, from.Format("2006-01-02"))
		if err == nil {
			return nextDay, nil
		}
		if !shouldFallbackToInMemoryCalendar(err) {
			return time.Time{}, err
		}
	}

	current := from.AddDate(0, 0, 1)
	for i := 0; i < 370; i++ {
		ok, _ := s.IsBusinessDay(ctx, calendarCode, current)
		if ok {
			return current, nil
		}
		current = current.AddDate(0, 0, 1)
	}
	return time.Time{}, fmt.Errorf("failed to find next business day for %s", calendarCode)
}

// PreviousBusinessDay finds the previous business day before the given date
// TODO: Migrate to Hasura GraphQL with custom function:
//
//	query PreviousBusinessDay($calendar_code: String!, $from_date: date!) {
//	  previous_business_day(args: {calendar_code: $calendar_code, from_date: $from_date})
//	}
func (s *Service) PreviousBusinessDay(ctx context.Context, calendarCode string, from time.Time) (time.Time, error) {
	if s.db != nil {
		var prevDay time.Time
		query := "SELECT previous_business_day($1, $2::DATE)"
		err := s.db.GetContext(ctx, &prevDay, query, calendarCode, from.Format("2006-01-02"))
		if err == nil {
			return prevDay, nil
		}
		if !shouldFallbackToInMemoryCalendar(err) {
			return time.Time{}, err
		}
	}

	current := from.AddDate(0, 0, -1)
	for i := 0; i < 370; i++ {
		ok, _ := s.IsBusinessDay(ctx, calendarCode, current)
		if ok {
			return current, nil
		}
		current = current.AddDate(0, 0, -1)
	}
	return time.Time{}, fmt.Errorf("failed to find previous business day for %s", calendarCode)
}

// AddBusinessDays adds N business days to a date
// TODO: Migrate to Hasura GraphQL with custom function:
//
//	query AddBusinessDays($calendar_code: String!, $from_date: date!, $days: Int!) {
//	  add_business_days(args: {calendar_code: $calendar_code, from_date: $from_date, days: $days})
//	}
func (s *Service) AddBusinessDays(ctx context.Context, calendarCode string, from time.Time, days int) (time.Time, error) {
	if s.db != nil {
		var result time.Time
		query := "SELECT add_business_days($1, $2::DATE, $3)"
		err := s.db.GetContext(ctx, &result, query, calendarCode, from.Format("2006-01-02"), days)
		if err == nil {
			return result, nil
		}
		if !shouldFallbackToInMemoryCalendar(err) {
			return time.Time{}, err
		}
	}

	current := from
	remaining := days
	step := 1
	if remaining < 0 {
		step = -1
		remaining = -remaining
	}
	for remaining > 0 {
		current = current.AddDate(0, 0, step)
		ok, _ := s.IsBusinessDay(ctx, calendarCode, current)
		if ok {
			remaining--
		}
	}
	return current, nil
}

// AdjustDate adjusts a date according to business day convention
// TODO: Migrate to Hasura GraphQL with custom function:
//
//	query AdjustDate($calendar_code: String!, $date: date!, $convention: String!) {
//	  adjust_date(args: {calendar_code: $calendar_code, date: $date, convention: $convention})
//	}
//
// Note: Supports ISDA conventions (FOLLOWING, MODIFIED_FOLLOWING, PRECEDING, UNADJUSTED)
func (s *Service) AdjustDate(
	ctx context.Context,
	calendarCode string,
	date time.Time,
	convention AdjustmentConvention,
) (time.Time, error) {
	if s.db != nil {
		var adjusted time.Time
		query := "SELECT adjust_date($1, $2::DATE, $3::adjustment_convention)"
		err := s.db.GetContext(ctx, &adjusted, query, calendarCode, date.Format("2006-01-02"), string(convention))
		if err == nil {
			return adjusted, nil
		}
		if !shouldFallbackToInMemoryCalendar(err) {
			return time.Time{}, err
		}
	}

	if convention == Unadjusted {
		return date, nil
	}

	originalMonth := date.Month()
	adjusted := date
	switch convention {
	case Following, ModifiedFollowing:
		for i := 0; i < 370; i++ {
			ok, _ := s.IsBusinessDay(ctx, calendarCode, adjusted)
			if ok {
				break
			}
			adjusted = adjusted.AddDate(0, 0, 1)
		}
		if convention == ModifiedFollowing && adjusted.Month() != originalMonth {
			adjusted = date
			for i := 0; i < 370; i++ {
				ok, _ := s.IsBusinessDay(ctx, calendarCode, adjusted)
				if ok {
					break
				}
				adjusted = adjusted.AddDate(0, 0, -1)
			}
		}
	case Preceding:
		for i := 0; i < 370; i++ {
			ok, _ := s.IsBusinessDay(ctx, calendarCode, adjusted)
			if ok {
				break
			}
			adjusted = adjusted.AddDate(0, 0, -1)
		}
	default:
		return time.Time{}, fmt.Errorf("unsupported convention: %s", convention)
	}
	return adjusted, nil
}

func shouldFallbackToInMemoryCalendar(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Common sqlite incompatibilities: ::DATE casts, missing functions.
	if strings.Contains(msg, "unrecognized token") {
		return true
	}
	if strings.Contains(msg, "no such function") {
		return true
	}
	if strings.Contains(msg, "syntax error") {
		return true
	}
	return false
}

func isBusinessDayFallback(calendarCode string, date time.Time) bool {
	// Weekend rule (default: Sat/Sun)
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		return false
	}
	if isHolidayFallback(calendarCode, date) {
		return false
	}
	return true
}

func isHolidayFallback(calendarCode string, date time.Time) bool {
	code := strings.ToUpper(strings.TrimSpace(calendarCode))
	day := date.Format("2006-01-02")
	year := date.Year()

	calHolidays := holidaySetFallback(code, year)
	if calHolidays[day] {
		return true
	}
	// NYSE inherits US_FEDERAL in this simplified fallback model.
	if code == "NYSE" {
		fed := holidaySetFallback("US_FEDERAL", year)
		return fed[day]
	}
	return false
}

func holidaySetFallback(calendarCode string, year int) map[string]bool {
	set := map[string]bool{}

	switch calendarCode {
	case "US_FEDERAL", "NYSE":
		addFixedObserved(set, year, time.January, 1)   // New Year's Day
		addFixedObserved(set, year, time.December, 25) // Christmas
		// Martin Luther King Jr. Day: third Monday in January
		set[nthWeekdayOfMonth(year, time.January, time.Monday, 3).Format("2006-01-02")] = true
	}

	if calendarCode == "NYSE" {
		// Good Friday: 2 days before Easter Sunday
		easter := easterSunday(year)
		goodFriday := easter.AddDate(0, 0, -2)
		set[goodFriday.Format("2006-01-02")] = true
	}

	return set
}

func addFixedObserved(set map[string]bool, year int, month time.Month, day int) {
	d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	set[d.Format("2006-01-02")] = true
	// Basic observed rule: if holiday falls on weekend, observe closest weekday.
	if d.Weekday() == time.Saturday {
		set[d.AddDate(0, 0, -1).Format("2006-01-02")] = true
	}
	if d.Weekday() == time.Sunday {
		set[d.AddDate(0, 0, 1).Format("2006-01-02")] = true
	}
}

func nthWeekdayOfMonth(year int, month time.Month, weekday time.Weekday, n int) time.Time {
	// Find the first day of the month.
	d := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	// Advance to first desired weekday.
	for d.Weekday() != weekday {
		d = d.AddDate(0, 0, 1)
	}
	// Add (n-1) weeks.
	d = d.AddDate(0, 0, (n-1)*7)
	return d
}

// easterSunday computes Easter Sunday for the given year (Gregorian calendar).
// Uses the Anonymous Gregorian algorithm.
func easterSunday(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// GetCalendar retrieves a calendar by code
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetCalendar($code: String!) {
//	  business_calendars(where: {calendar_code: {_eq: $code}, active: {_eq: true}}, limit: 1) {
//	    calendar_id
//	    tenant_id
//	    calendar_code
//	    calendar_name
//	    description
//	    parent_calendar_ids
//	    timezone
//	    weekend_days
//	    active
//	    is_global
//	  }
//	}
func (s *Service) GetCalendar(ctx context.Context, code string) (*Calendar, error) {
	// Check cache
	if cal, ok := s.calendarCache[code]; ok {
		return cal, nil
	}

	// Query database
	var calendar Calendar
	query := `
		SELECT calendar_id, tenant_id, calendar_code, calendar_name, description,
		       parent_calendar_ids, timezone, weekend_days, active, is_global
		FROM business_calendars
		WHERE calendar_code = $1 AND active = TRUE
	`
	err := s.db.GetContext(ctx, &calendar, query, code)
	if err != nil {
		return nil, fmt.Errorf("calendar %s not found: %w", code, err)
	}

	// Cache
	s.calendarCache[code] = &calendar

	return &calendar, nil
}

// ListCalendars returns all active calendars
// TODO: Migrate to Hasura GraphQL query with _or filtering:
//
//	query ListCalendars($tenant_id: uuid) {
//	  business_calendars(
//	    where: {
//	      active: {_eq: true},
//	      _or: [
//	        {is_global: {_eq: true}},
//	        {tenant_id: {_eq: $tenant_id}}
//	      ]
//	    },
//	    order_by: {calendar_name: asc}
//	  ) {
//	    calendar_id
//	    tenant_id
//	    calendar_code
//	    calendar_name
//	    description
//	    parent_calendar_ids
//	    timezone
//	    weekend_days
//	    active
//	    is_global
//	  }
//	}
func (s *Service) ListCalendars(ctx context.Context, tenantID *uuid.UUID) ([]Calendar, error) {
	var calendars []Calendar

	query := `
		SELECT calendar_id, tenant_id, calendar_code, calendar_name, description,
		       parent_calendar_ids, timezone, weekend_days, active, is_global
		FROM business_calendars
		WHERE active = TRUE
		AND (is_global = TRUE OR ($1::UUID IS NOT NULL AND tenant_id = $1))
		ORDER BY calendar_name
	`

	err := s.db.SelectContext(ctx, &calendars, query, tenantID)
	return calendars, err
}

// GetHolidays returns holidays for a calendar in a date range
// TODO: Migrate to Hasura GraphQL - may need custom SQL function for recursive CTE:
//
//	query GetHolidays($calendar_id: uuid!, $start_date: date!, $end_date: date!) {
//	  get_calendar_holidays_with_inheritance(
//	    args: {cal_id: $calendar_id, start_dt: $start_date, end_dt: $end_date}
//	  ) {
//	    holiday_id
//	    calendar_id
//	    holiday_date
//	    holiday_name
//	    holiday_type
//	    is_half_day
//	    half_day_close_time
//	    is_recurring
//	    recurrence_rule
//	    observed_date
//	  }
//	}
//
// Note: Uses recursive CTE for parent calendar hierarchy traversal
func (s *Service) GetHolidays(ctx context.Context, calendarCode string, startDate, endDate time.Time) ([]Holiday, error) {
	calendar, err := s.GetCalendar(ctx, calendarCode)
	if err != nil {
		return nil, err
	}

	// Query holidays with inheritance
	query := `
		WITH RECURSIVE calendar_hierarchy AS (
			SELECT calendar_id FROM business_calendars WHERE calendar_id = $1
			UNION
			SELECT unnest(parent_calendar_ids) 
			FROM business_calendars bc
			JOIN calendar_hierarchy ch ON bc.calendar_id = ch.calendar_id
			WHERE bc.parent_calendar_ids IS NOT NULL
		)
		SELECT h.holiday_id, h.calendar_id, h.holiday_date, h.holiday_name,
		       h.holiday_type, h.is_half_day, h.half_day_close_time,
		       h.is_recurring, h.recurrence_rule, h.observed_date
		FROM calendar_holidays h
		WHERE h.calendar_id IN (SELECT calendar_id FROM calendar_hierarchy)
		AND h.holiday_date BETWEEN $2::DATE AND $3::DATE
		ORDER BY h.holiday_date
	`

	var holidays []Holiday
	err = s.db.SelectContext(ctx, &holidays, query, calendar.ID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	return holidays, err
}

// CreateCalendar creates a new calendar
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation CreateCalendar($object: business_calendars_insert_input!) {
//	  insert_business_calendars_one(object: $object) {
//	    calendar_id
//	    tenant_id
//	    calendar_code
//	    calendar_name
//	    description
//	    parent_calendar_ids
//	    timezone
//	    weekend_days
//	    is_global
//	  }
//	}
func (s *Service) CreateCalendar(ctx context.Context, calendar *Calendar) error {
	query := `
		INSERT INTO business_calendars (
			tenant_id, calendar_code, calendar_name, description,
			parent_calendar_ids, timezone, weekend_days, is_global
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING calendar_id
	`

	return s.db.GetContext(ctx, &calendar.ID, query,
		calendar.TenantID,
		calendar.Code,
		calendar.Name,
		calendar.Description,
		calendar.ParentIDs,
		calendar.Timezone,
		calendar.WeekendDays,
		calendar.IsGlobal,
	)
}

// AddHoliday adds a holiday to a calendar
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation AddHoliday($object: calendar_holidays_insert_input!) {
//	  insert_calendar_holidays_one(object: $object) {
//	    holiday_id
//	    calendar_id
//	    holiday_date
//	    holiday_name
//	    holiday_type
//	    is_half_day
//	    half_day_close_time
//	    is_recurring
//	    recurrence_rule
//	  }
//	}
func (s *Service) AddHoliday(ctx context.Context, holiday *Holiday) error {
	query := `
		INSERT INTO calendar_holidays (
			calendar_id, holiday_date, holiday_name, holiday_type,
			is_half_day, half_day_close_time, is_recurring, recurrence_rule
		) VALUES (
			$1, $2::DATE, $3, $4, $5, $6, $7, $8
		) RETURNING holiday_id
	`

	return s.db.GetContext(ctx, &holiday.ID, query,
		holiday.CalendarID,
		holiday.Date.Format("2006-01-02"),
		holiday.Name,
		holiday.Type,
		holiday.IsHalfDay,
		holiday.HalfDayCloseTime,
		holiday.IsRecurring,
		holiday.RecurrenceRule,
	)
}

// CountBusinessDays counts business days between two dates (inclusive)
func (s *Service) CountBusinessDays(ctx context.Context, calendarCode string, startDate, endDate time.Time) (int, error) {
	count := 0
	current := startDate

	for !current.After(endDate) {
		isBusiness, err := s.IsBusinessDay(ctx, calendarCode, current)
		if err != nil {
			return 0, err
		}
		if isBusiness {
			count++
		}
		current = current.AddDate(0, 0, 1)
	}

	return count, nil
}

// ClearCache clears the internal cache
func (s *Service) ClearCache() {
	s.calendarCache = make(map[string]*Calendar)
	s.holidayCache = make(map[string]map[string]bool)
}
