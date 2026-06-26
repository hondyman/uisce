package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/calendar"
	"go.temporal.io/sdk/activity"
)

// CalendarActivities provides Temporal activities for calendar operations
type CalendarActivities struct {
	calendarService *calendar.Service
}

// NewCalendarActivities creates calendar activities
func NewCalendarActivities(calendarService *calendar.Service) *CalendarActivities {
	return &CalendarActivities{
		calendarService: calendarService,
	}
}

// BusinessSleepInput defines input for BusinessSleep activity
type BusinessSleepInput struct {
	CalendarCode string
	TargetTime   time.Time // Time on the next business day to wake up
}

// BusinessSleep sleeps until the next business day at a specific time
// This is a critical activity for scheduling jobs that must run on business days
func (a *CalendarActivities) BusinessSleep(ctx context.Context, input BusinessSleepInput) error {
	logger := activity.GetLogger(ctx)

	now := time.Now()

	// Find next business day
	nextBusiness, err := a.calendarService.NextBusinessDay(ctx, input.CalendarCode, now)
	if err != nil {
		return fmt.Errorf("failed to find next business day: %w", err)
	}

	// Combine next business day with target time
	wakeTime := time.Date(
		nextBusiness.Year(), nextBusiness.Month(), nextBusiness.Day(),
		input.TargetTime.Hour(), input.TargetTime.Minute(), input.TargetTime.Second(),
		0, input.TargetTime.Location(),
	)

	// If wake time is in the past, add one more business day
	if wakeTime.Before(now) {
		nextBusiness, err = a.calendarService.NextBusinessDay(ctx, input.CalendarCode, nextBusiness)
		if err != nil {
			return err
		}
		wakeTime = time.Date(
			nextBusiness.Year(), nextBusiness.Month(), nextBusiness.Day(),
			input.TargetTime.Hour(), input.TargetTime.Minute(), input.TargetTime.Second(),
			0, input.TargetTime.Location(),
		)
	}

	sleepDuration := time.Until(wakeTime)

	logger.Info("Business sleep scheduled",
		"calendar", input.CalendarCode,
		"wake_time", wakeTime,
		"duration", sleepDuration,
	)

	// Sleep (Temporal will handle this as a timer internally)
	time.Sleep(sleepDuration)

	logger.Info("Business sleep completed", "wake_time", wakeTime)

	return nil
}

// IsBusinessDayActivity checks if a date is a business day
func (a *CalendarActivities) IsBusinessDay(ctx context.Context, calendarCode string, date time.Time) (bool, error) {
	return a.calendarService.IsBusinessDay(ctx, calendarCode, date)
}

// NextBusinessDayActivity finds the next business day
func (a *CalendarActivities) NextBusinessDay(ctx context.Context, calendarCode string, from time.Time) (time.Time, error) {
	return a.calendarService.NextBusinessDay(ctx, calendarCode, from)
}

// AddBusinessDaysActivity adds N business days
func (a *CalendarActivities) AddBusinessDays(ctx context.Context, calendarCode string, from time.Time, days int) (time.Time, error) {
	return a.calendarService.AddBusinessDays(ctx, calendarCode, from, days)
}

// AdjustDateActivity adjusts a date per convention
func (a *CalendarActivities) AdjustDate(
	ctx context.Context,
	calendarCode string,
	date time.Time,
	convention calendar.AdjustmentConvention,
) (time.Time, error) {
	return a.calendarService.AdjustDate(ctx, calendarCode, date, convention)
}

// Example workflow using Business Sleep
/*
func MonthEndReportWorkflow(ctx workflow.Context, reportDate time.Time) error {
    logger := workflow.GetLogger(ctx)

    // Adjust to next business day if month-end falls on weekend/holiday
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: time.Minute,
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    var adjustedDate time.Time
    err := workflow.ExecuteActivity(ctx, "AdjustDate",
        "NYSE",
        reportDate,
        calendar.Following,
    ).Get(ctx, &adjustedDate)

    if err != nil {
        return err
    }

    logger.Info("Report scheduled for", "date", adjustedDate)

    // Sleep until 9 AM on the business day
    targetTime := time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC) // Only time matters
    err = workflow.ExecuteActivity(ctx, "BusinessSleep", BusinessSleepInput{
        CalendarCode: "NYSE",
        TargetTime:   targetTime,
    }).Get(ctx, nil)

    if err != nil {
        return err
    }

    // Generate report
    var reportID string
    err = workflow.ExecuteActivity(ctx, "GenerateMonthEndReport", adjustedDate).Get(ctx, &reportID)

    return err
}
*/
