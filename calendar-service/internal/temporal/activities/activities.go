package activities

import (
	"context"
	"fmt"
	"time"

	"calendar-service/internal/availability"
	"calendar-service/internal/database"

	"github.com/sirupsen/logrus"
)

type Activities struct {
	dbClient        *database.Client
	availabilityChk *availability.Checker
	logger          *logrus.Entry
}

func NewActivities(dc *database.Client, ac *availability.Checker, logger *logrus.Entry) *Activities {
	return &Activities{
		dbClient:        dc,
		availabilityChk: ac,
		logger:          logger.WithField("component", "temporal_activities"),
	}
}

// FetchAffectedJobsRequest specifies which jobs to fetch
type FetchAffectedJobsRequest struct {
	TenantID   string
	EntityID   string
	EntityType string // "calendar", "schedule_profile", "blackout"
}

// FetchAffectedJobsActivity returns jobs affected by calendar changes
func (a *Activities) FetchAffectedJobsActivity(ctx context.Context, req FetchAffectedJobsRequest) ([]map[string]interface{}, error) {
	logger := a.logger.WithFields(logrus.Fields{
		"activity": "FetchAffectedJobs",
		"entity":   req.EntityID,
		"type":     req.EntityType,
	})
	logger.Info("Fetching affected jobs")

	var sqlQuery string
	var args []interface{}

	switch req.EntityType {
	case "calendar":
		sqlQuery = `
			SELECT j.id, j.next_run::text, j.profile_id
			FROM jobs j
			JOIN schedule_profiles sp ON j.profile_id = sp.id
			JOIN profile_calendars pc ON sp.id = pc.profile_id
			WHERE j.calendar_aware = true
			  AND j.tenant_id = $1
			  AND pc.calendar_id = $2`
		args = []interface{}{req.TenantID, req.EntityID}

	case "schedule_profile":
		sqlQuery = `
			SELECT id, next_run::text, profile_id
			FROM jobs
			WHERE profile_id = $1 AND calendar_aware = true AND tenant_id = $2`
		args = []interface{}{req.EntityID, req.TenantID}

	case "blackout":
		sqlQuery = `
			SELECT id, next_run::text, profile_id
			FROM jobs
			WHERE calendar_aware = true AND tenant_id = $1`
		args = []interface{}{req.TenantID}
	}

	rows, err := a.dbClient.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		logger.WithError(err).Error("Failed to fetch affected jobs")
		return nil, err
	}
	defer rows.Close()

	result := []map[string]interface{}{}
	for rows.Next() {
		var id, nextRun, profileID string
		if err := rows.Scan(&id, &nextRun, &profileID); err != nil {
			logger.WithError(err).Error("Failed to scan row")
			continue
		}
		result = append(result, map[string]interface{}{
			"id":         id,
			"next_run":   nextRun,
			"profile_id": profileID,
		})
	}

	logger.WithField("count", len(result)).Info("Fetched affected jobs")
	return result, nil
}

// CheckAvailabilityRequest checks if a time slot is available
type CheckAvailabilityRequest struct {
	TenantID  string
	Region    string
	ProfileID string
	Start     time.Time
	End       time.Time
}

// CheckAvailabilityActivity validates availability for a time range
func (a *Activities) CheckAvailabilityActivity(ctx context.Context, req CheckAvailabilityRequest) (bool, error) {
	logger := a.logger.WithField("activity", "CheckAvailability")

	var profileName string
	query := `SELECT name FROM schedule_profiles WHERE id = $1 AND valid_to IS NULL LIMIT 1`
	if err := a.dbClient.Pool().QueryRow(ctx, query, req.ProfileID).Scan(&profileName); err != nil {
		logger.WithError(err).Error("Failed to get profile name")
		return false, err
	}

	result, err := a.availabilityChk.CheckAvailability(ctx, req.TenantID, profileName, req.Region, req.Start, req.End)
	if err != nil {
		logger.WithError(err).Error("Failed to check availability")
		return false, err
	}

	return result.Available, nil
}

// FindNextSlotRequest finds the next available time slot
type FindNextSlotRequest struct {
	TenantID  string
	Region    string
	ProfileID string
	After     time.Time
	Duration  time.Duration
}

// FindNextSlotActivity finds the next available slot for job execution
func (a *Activities) FindNextSlotActivity(ctx context.Context, req FindNextSlotRequest) (time.Time, error) {
	logger := a.logger.WithField("activity", "FindNextSlot")

	var profileName string
	query := `SELECT name FROM schedule_profiles WHERE id = $1 AND valid_to IS NULL LIMIT 1`
	if err := a.dbClient.Pool().QueryRow(ctx, query, req.ProfileID).Scan(&profileName); err != nil {
		logger.WithError(err).Error("Failed to get profile name")
		return time.Time{}, err
	}

	nextSlot, err := a.availabilityChk.FindNextAvailableSlot(ctx, req.TenantID, profileName, req.Region, req.After, req.Duration)
	if err != nil {
		logger.WithError(err).Error("Failed to find next slot", "after", req.After, "duration", req.Duration)
		return time.Time{}, err
	}

	logger.WithField("nextSlot", nextSlot.Format(time.RFC3339)).Info("Found next available slot")
	return nextSlot, nil
}

// RescheduleRequest updates a job's scheduled time
type RescheduleRequest struct {
	JobID     string
	TenantID  string
	ProfileID string
	NewTime   time.Time
}

// RescheduleJobActivity updates job next_run time and records audit
func (a *Activities) RescheduleJobActivity(ctx context.Context, req RescheduleRequest) error {
	logger := a.logger.WithFields(logrus.Fields{
		"activity": "RescheduleJob",
		"job_id":   req.JobID,
	})
	logger.WithField("newTime", req.NewTime.Format(time.RFC3339)).Info("Rescheduling job")

	var nextRun time.Time
	query := `UPDATE jobs SET next_run = $2 WHERE id = $1 RETURNING next_run`
	if err := a.dbClient.Pool().QueryRow(ctx, query, req.JobID, req.NewTime).Scan(&nextRun); err != nil {
		logger.WithError(err).Error("Failed to reschedule job")
		return err
	}

	logger.WithField("newTime", nextRun).Info("Successfully rescheduled job")
	return nil
}

// ListAffectedProfilesActivity finds all profiles affected by a calendar change
func (a *Activities) ListAffectedProfilesActivity(ctx context.Context, tenantID, calendarID string) ([]map[string]interface{}, error) {
	logger := a.logger.WithField("activity", "ListAffectedProfiles")

	sqlQuery := `
		SELECT sp.id, sp.name
		FROM schedule_profiles sp
		JOIN profile_calendars pc ON sp.id = pc.profile_id
		WHERE sp.tenant_id = $1 AND sp.valid_to IS NULL AND pc.calendar_id = $2`

	rows, err := a.dbClient.Pool().Query(ctx, sqlQuery, tenantID, calendarID)
	if err != nil {
		logger.WithError(err).Error("Failed to list affected profiles")
		return nil, err
	}
	defer rows.Close()

	result := []map[string]interface{}{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			logger.WithError(err).Error("Failed to scan row")
			continue
		}
		result = append(result, map[string]interface{}{
			"id":   id,
			"name": name,
		})
	}

	logger.WithField("count", len(result)).Info("Listed affected profiles")
	return result, nil
}

// RegisterActivities registers all activities with a Temporal worker
func RegisterActivities(act *Activities) map[string]interface{} {
	return map[string]interface{}{
		"FetchAffectedJobsActivity":    act.FetchAffectedJobsActivity,
		"CheckAvailabilityActivity":    act.CheckAvailabilityActivity,
		"FindNextSlotActivity":         act.FindNextSlotActivity,
		"RescheduleJobActivity":        act.RescheduleJobActivity,
		"ListAffectedProfilesActivity": act.ListAffectedProfilesActivity,
	}
}
