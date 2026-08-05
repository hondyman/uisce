package repository

import (
	"context"
	"database/sql"
	"time"

	"calendar-service/internal/database"

	"github.com/google/uuid"
)

type SyncedMicrosoftEvent struct {
	ConnectionID         string
	TenantID             string
	MicrosoftEventID   string
	MicrosoftCalendarID string
	InternalEventID     *string
	InternalCalendarID  *string
	Title               string
	Description         *string
	Location            *string
	StartTime           time.Time
	EndTime             time.Time
	IsAllDay            bool
	IsRecurring         bool
	RecurrenceRule      *string
	RecurrenceID        *string
	SyncStatus          string
	SyncDirection       string
	LastSyncedAt        time.Time
	LastPushedToMicrosoft *time.Time
	UpdatedAt           time.Time
}

type MicrosoftSyncRepo struct {
	db *database.Client
}

func NewMicrosoftSyncRepo(db *database.Client) *MicrosoftSyncRepo {
	return &MicrosoftSyncRepo{db: db}
}

func (r *MicrosoftSyncRepo) UpsertSyncedEvent(ctx context.Context, event *SyncedMicrosoftEvent) error {
	query := `
		INSERT INTO synced_microsoft_events (
			id, tenant_id, microsoft_event_id, microsoft_calendar_id, internal_event_id, internal_calendar_id,
			title, description, location, start_time, end_time, is_all_day, is_recurring,
			recurrence_rule, recurrence_id, sync_status, sync_direction, last_synced_at, last_pushed_to_microsoft, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (microsoft_event_id, microsoft_calendar_id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			location = EXCLUDED.location,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			is_all_day = EXCLUDED.is_all_day,
			is_recurring = EXCLUDED.is_recurring,
			recurrence_rule = EXCLUDED.recurrence_rule,
			sync_status = EXCLUDED.sync_status,
			last_synced_at = EXCLUDED.last_synced_at,
			updated_at = NOW()
	`

	id := uuid.New()
	_, err := r.db.Pool().Exec(ctx, query,
		id, event.TenantID, event.MicrosoftEventID, event.MicrosoftCalendarID, event.InternalEventID, event.InternalCalendarID,
		event.Title, event.Description, event.Location, event.StartTime, event.EndTime, event.IsAllDay, event.IsRecurring,
		event.RecurrenceRule, event.RecurrenceID, event.SyncStatus, event.SyncDirection, event.LastSyncedAt, event.LastPushedToMicrosoft, time.Now(),
	)
	return err
}

func (r *MicrosoftSyncRepo) GetPrimaryCalendarID(ctx context.Context, tenantID, userID string) (string, error) {
	query := `
		SELECT microsoft_calendar_id
		FROM microsoft_calendar_connections
		WHERE tenant_id = $1 AND user_id = $2
		LIMIT 1
	`

	var calendarID string
	err := r.db.Pool().QueryRow(ctx, query, tenantID, userID).Scan(&calendarID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return calendarID, err
}

func (r *MicrosoftSyncRepo) GetSyncedEventByMicrosoftID(ctx context.Context, microsoftEventID, microsoftCalendarID string) (*SyncedMicrosoftEvent, error) {
	query := `
		SELECT id, tenant_id, microsoft_event_id, microsoft_calendar_id, internal_event_id, internal_calendar_id,
			title, description, location, start_time, end_time, is_all_day, is_recurring,
			recurrence_rule, recurrence_id, sync_status, sync_direction, last_synced_at, last_pushed_to_microsoft, updated_at
		FROM synced_microsoft_events
		WHERE microsoft_event_id = $1 AND microsoft_calendar_id = $2
	`

	var event SyncedMicrosoftEvent
	err := r.db.Pool().QueryRow(ctx, query, microsoftEventID, microsoftCalendarID).Scan(
		&event.ConnectionID, &event.TenantID, &event.MicrosoftEventID, &event.MicrosoftCalendarID,
		&event.InternalEventID, &event.InternalCalendarID, &event.Title, &event.Description,
		&event.Location, &event.StartTime, &event.EndTime, &event.IsAllDay, &event.IsRecurring,
		&event.RecurrenceRule, &event.RecurrenceID, &event.SyncStatus, &event.SyncDirection,
		&event.LastSyncedAt, &event.LastPushedToMicrosoft, &event.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *MicrosoftSyncRepo) GetSyncedEventByInternalID(ctx context.Context, internalEventID string) (*SyncedMicrosoftEvent, error) {
	query := `
		SELECT id, tenant_id, microsoft_event_id, microsoft_calendar_id, internal_event_id, internal_calendar_id,
			title, description, location, start_time, end_time, is_all_day, is_recurring,
			recurrence_rule, recurrence_id, sync_status, sync_direction, last_synced_at, last_pushed_to_microsoft, updated_at
		FROM synced_microsoft_events
		WHERE internal_event_id = $1
	`

	var event SyncedMicrosoftEvent
	err := r.db.Pool().QueryRow(ctx, query, internalEventID).Scan(
		&event.ConnectionID, &event.TenantID, &event.MicrosoftEventID, &event.MicrosoftCalendarID,
		&event.InternalEventID, &event.InternalCalendarID, &event.Title, &event.Description,
		&event.Location, &event.StartTime, &event.EndTime, &event.IsAllDay, &event.IsRecurring,
		&event.RecurrenceRule, &event.RecurrenceID, &event.SyncStatus, &event.SyncDirection,
		&event.LastSyncedAt, &event.LastPushedToMicrosoft, &event.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *MicrosoftSyncRepo) DeleteSyncedEvent(ctx context.Context, microsoftEventID, microsoftCalendarID string) error {
	query := `DELETE FROM synced_microsoft_events WHERE microsoft_event_id = $1 AND microsoft_calendar_id = $2`
	_, err := r.db.Pool().Exec(ctx, query, microsoftEventID, microsoftCalendarID)
	return err
}

func (r *MicrosoftSyncRepo) UpdateSyncStatus(ctx context.Context, microsoftEventID, microsoftCalendarID, status string) error {
	query := `UPDATE synced_microsoft_events SET sync_status = $3, updated_at = NOW() WHERE microsoft_event_id = $1 AND microsoft_calendar_id = $2`
	_, err := r.db.Pool().Exec(ctx, query, microsoftEventID, microsoftCalendarID, status)
	return err
}
