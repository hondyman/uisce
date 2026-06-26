package activities

import (
	"context"
	"fmt"
	"time"

	"calendar-service/internal/availability"
	"calendar-service/internal/hasura"

	"github.com/sirupsen/logrus"
)

// Activities holds activity dependencies
type Activities struct {
	hasuraClient    *hasura.Client
	availabilityChk *availability.Checker
	logger          *logrus.Entry
}

// NewActivities creates a new Activities instance
func NewActivities(hc *hasura.Client, ac *availability.Checker, logger *logrus.Entry) *Activities {
	return &Activities{
		hasuraClient:    hc,
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

	query := ""
	variables := map[string]interface{}{
		"tenant_id": req.TenantID,
	}

	switch req.EntityType {
	case "calendar":
		// Find all profiles using this calendar
		query = `
		query GetAffectedJobs($calendar_id: uuid!, $tenant_id: uuid!) {
			jobs(where: {calendar_aware: {_eq: true}, tenant_id: {_eq: $tenant_id}, schedule_profile: {profile_calendars: {calendar_id: {_eq: $calendar_id}}}}) {
				id
				next_run
				profile_id
			}
		}
		`
		variables["calendar_id"] = req.EntityID

	case "schedule_profile":
		// Find all jobs using this profile
		query = `
		query GetAffectedJobs($profile_id: uuid!, $tenant_id: uuid!) {
			jobs(where: {profile_id: {_eq: $profile_id}, calendar_aware: {_eq: true}, tenant_id: {_eq: $tenant_id}}) {
				id
				next_run
				profile_id
			}
		}
		`
		variables["profile_id"] = req.EntityID

	case "blackout":
		// Find all jobs in profiles covered by blackout (simplified)
		// In reality would query jobs that overlap with blackout
		query = `
		query GetAffectedJobs($tenant_id: uuid!) {
			jobs(where: {calendar_aware: {_eq: true}, tenant_id: {_eq: $tenant_id}}) {
				id
				next_run
				profile_id
			}
		}
		`
	}

	var response struct {
		Jobs []map[string]interface{} `json:"jobs"`
	}

	if query != "" {
		err := a.hasuraClient.QueryRaw(ctx, query, variables, &response)
		if err != nil {
			logger.WithError(err).Error("Failed to fetch affected jobs")
			return nil, err
		}
	}

	logger.WithField("count", len(response.Jobs)).Info("Fetched affected jobs")
	return response.Jobs, nil
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

	// Query profile name from profile ID
	query := `
	query GetProfileName($id: uuid!) {
		schedule_profiles(where: {id: {_eq: $id}, valid_to: {_is_null: true}}) {
			name
		}
	}
	`
	var profileResp struct {
		ScheduleProfiles []struct {
			Name string `json:"name"`
		} `json:"schedule_profiles"`
	}

	if err := a.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{"id": req.ProfileID}, &profileResp); err != nil {
		logger.WithError(err).Error("Failed to get profile name")
		return false, err
	}

	if len(profileResp.ScheduleProfiles) == 0 {
		logger.Warn("Profile not found")
		return false, nil
	}
	profileName := profileResp.ScheduleProfiles[0].Name

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

	// Query profile name from profile ID
	query := `
	query GetProfileName($id: uuid!) {
		schedule_profiles(where: {id: {_eq: $id}, valid_to: {_is_null: true}}) {
			name
		}
	}
	`
	var profileResp struct {
		ScheduleProfiles []struct {
			Name string `json:"name"`
		} `json:"schedule_profiles"`
	}

	if err := a.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{"id": req.ProfileID}, &profileResp); err != nil {
		logger.WithError(err).Error("Failed to get profile name")
		return time.Time{}, err
	}

	if len(profileResp.ScheduleProfiles) == 0 {
		return time.Time{}, fmt.Errorf("profile not found")
	}
	profileName := profileResp.ScheduleProfiles[0].Name

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

	mutation := `
	mutation RescheduleJob($id: uuid!, $next_run: timestamptz!) {
		update_jobs_by_pk(pk_columns: {id: $id}, _set: {next_run: $next_run}) {
			id
			next_run
		}
	}
	`
	var response struct {
		UpdateJobsByPk struct {
			ID      string    `json:"id"`
			NextRun time.Time `json:"next_run"`
		} `json:"update_jobs_by_pk"`
	}

	err := a.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{
		"id":       req.JobID,
		"next_run": req.NewTime,
	}, &response)
	if err != nil {
		logger.WithError(err).Error("Failed to reschedule job")
		return err
	}

	logger.WithField("newTime", response.UpdateJobsByPk.NextRun).Info("Successfully rescheduled job")
	return nil
}

// ListAffectedProfilesActivity finds all profiles affected by a calendar change
func (a *Activities) ListAffectedProfilesActivity(ctx context.Context, tenantID, calendarID string) ([]map[string]interface{}, error) {
	logger := a.logger.WithField("activity", "ListAffectedProfiles")

	query := `
	query GetAffectedProfiles($calendar_id: uuid!, $tenant_id: uuid!) {
		schedule_profiles(where: {tenant_id: {_eq: $tenant_id}, valid_to: {_is_null: true}, profile_calendars: {calendar_id: {_eq: $calendar_id}}}) {
			id
			name
		}
	}
	`

	var response struct {
		ScheduleProfiles []map[string]interface{} `json:"schedule_profiles"`
	}

	err := a.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{
		"tenant_id":   tenantID,
		"calendar_id": calendarID,
	}, &response)
	if err != nil {
		logger.WithError(err).Error("Failed to list affected profiles")
		return nil, err
	}

	logger.WithField("count", len(response.ScheduleProfiles)).Info("Listed affected profiles")
	return response.ScheduleProfiles, nil
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
