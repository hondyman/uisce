package reporting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CalendarMacroResolver struct {
	db *sqlx.DB
}

func NewCalendarMacroResolver(db *sqlx.DB) *CalendarMacroResolver {
	return &CalendarMacroResolver{db: db}
}

// ResolveDateMacro translates relative date tokens into exact deterministic timestamps
func (r *CalendarMacroResolver) ResolveDateMacro(
	ctx context.Context,
	tenantID uuid.UUID,
	macroName string,
	calendarCode string,
	baseTime time.Time,
	offsetDays int,
) (time.Time, time.Time, error) {
	if baseTime.IsZero() {
		baseTime = time.Now().UTC()
	}
	y, m, d := baseTime.Date()
	todayStart := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	todayEnd := time.Date(y, m, d, 23, 59, 59, 999999999, time.UTC)

	switch strings.ToUpper(macroName) {
	case "TODAY":
		return todayStart, todayEnd, nil

	case "YESTERDAY":
		yStart := todayStart.AddDate(0, 0, -1)
		yEnd := todayEnd.AddDate(0, 0, -1)
		return yStart, yEnd, nil

	case "THIS_WEEK":
		weekday := int(baseTime.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday as 7
		}
		wStart := todayStart.AddDate(0, 0, -(weekday - 1))
		return wStart, todayEnd, nil

	case "MTD", "THIS_MONTH":
		mStart := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
		return mStart, todayEnd, nil

	case "QTD", "THIS_QUARTER":
		quarterStartMonth := time.Month(((int(m)-1)/3)*3 + 1)
		qStart := time.Date(y, quarterStartMonth, 1, 0, 0, 0, 0, time.UTC)
		return qStart, todayEnd, nil

	case "YTD", "THIS_YEAR":
		yStart := time.Date(y, time.January, 1, 0, 0, 0, 0, time.UTC)
		return yStart, todayEnd, nil

	case "LAST_N_DAYS":
		nStart := todayStart.AddDate(0, 0, -offsetDays)
		return nStart, todayEnd, nil

	case "PREVIOUS_BUSINESS_DAY", "T-1":
		return r.resolveBusinessDayOffset(ctx, tenantID, calendarCode, todayStart, -1)

	case "T-2":
		return r.resolveBusinessDayOffset(ctx, tenantID, calendarCode, todayStart, -2)

	default:
		return todayStart, todayEnd, fmt.Errorf("unrecognized date macro: %s", macroName)
	}
}

func (r *CalendarMacroResolver) resolveBusinessDayOffset(
	ctx context.Context,
	tenantID uuid.UUID,
	calendarCode string,
	baseDate time.Time,
	offset int,
) (time.Time, time.Time, error) {
	cursor := baseDate
	stepsRemaining := offset
	if stepsRemaining < 0 {
		stepsRemaining = -stepsRemaining
	}
	direction := 1
	if offset < 0 {
		direction = -1
	}

	for stepsRemaining > 0 {
		cursor = cursor.AddDate(0, 0, direction)
		// Check weekends
		if cursor.Weekday() == time.Saturday || cursor.Weekday() == time.Sunday {
			continue
		}

		// Check holiday database if db instance is wired
		if r.db != nil && calendarCode != "" {
			var holidayCount int
			err := r.db.GetContext(ctx, &holidayCount, `
				SELECT COUNT(*) FROM public.tenant_calendar_holidays h
				JOIN public.tenant_exchange_calendars c ON h.calendar_id = c.id
				WHERE c.tenant_id = $1 AND c.calendar_code = $2 AND h.holiday_date = $3;
			`, tenantID, calendarCode, cursor.Format("2006-01-02"))
			if err == nil && holidayCount > 0 {
				continue
			}
		}
		stepsRemaining--
	}

	y, m, d := cursor.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	end := time.Date(y, m, d, 23, 59, 59, 999999999, time.UTC)
	return start, end, nil
}
