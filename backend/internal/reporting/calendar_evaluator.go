package reporting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CalendarEvaluator struct {
	db *sqlx.DB
}

func NewCalendarEvaluator(db *sqlx.DB) *CalendarEvaluator {
	return &CalendarEvaluator{db: db}
}

// IsExecutionAllowedOnDate checks calendar holidays and business day conventions
func (c *CalendarEvaluator) IsExecutionAllowedOnDate(
	ctx context.Context,
	tenantID, calendarID uuid.UUID,
	targetDate time.Time,
	unscheduledBehavior string,
) (bool, time.Time, error) {
	if calendarID == uuid.Nil {
		return true, targetDate, nil // No calendar configured; follow standard UTC cron
	}

	// 1. Check Weekend Convention
	weekday := targetDate.Weekday()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	// 2. Check Holiday Database
	var holidayCount int
	err := c.db.GetContext(ctx, &holidayCount, `
		SELECT COUNT(*) 
		FROM public.tenant_calendar_holidays 
		WHERE calendar_id = $1 AND holiday_date = $2
	`, calendarID, targetDate.Format("2006-01-02"))
	if err != nil {
		return false, targetDate, err
	}

	isHoliday := holidayCount > 0

	if !isWeekend && !isHoliday {
		return true, targetDate, nil
	}

	// 3. Resolve Unscheduled / Holiday Behavior
	switch strings.ToUpper(unscheduledBehavior) {
	case "SKIP":
		return false, targetDate, nil
	case "RUN_PREVIOUS_BUS_DAY":
		prevDay := targetDate.AddDate(0, 0, -1)
		for prevDay.Weekday() == time.Saturday || prevDay.Weekday() == time.Sunday {
			prevDay = prevDay.AddDate(0, 0, -1)
		}
		return true, prevDay, nil
	case "RUN_NEXT_BUS_DAY":
		nextDay := targetDate.AddDate(0, 0, 1)
		for nextDay.Weekday() == time.Saturday || nextDay.Weekday() == time.Sunday {
			nextDay = nextDay.AddDate(0, 0, 1)
		}
		return true, nextDay, nil
	case "WARN_HALT":
		return false, targetDate, fmt.Errorf("execution halted: %s is a non-business day", targetDate.Format("2006-01-02"))
	default:
		return false, targetDate, nil
	}
}
