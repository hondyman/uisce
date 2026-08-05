package repository

import (
	"context"
	"database/sql"
	"time"

	"calendar-service/internal/database"

	"github.com/google/uuid"
)

type SyncedGoogleEvent struct {
	ConnectionID        string
	TenantID            string
	GoogleEventID       string
	GoogleCalendarID    string
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
	LastPushedToGoogle *time.Time
	UpdatedAt           time.Time
}

type GoogleSyncRepo struct {
	db *database.Client
}

func NewGoogleSyncRepo(db *database.Client) *GoogleSyncRepo {
	return &GoogleSyncRepo{db: db}
}

func (r *GoogleSyncRepo) UpsertSyncedEvent(ctx context.Context, event *SyncedGoogleEvent) error {
	query := `
		INSERT INTO synced_google_events (
			id, tenant_id, google_event_id, google_calendar_id, internal_event_id, internal_calendar_id,
			title, description, location, start_time, end_time, is_all_day, is_recurring,
			recurrence_rule, recurrence_id, sync_status, sync_direction, last_synced_at, last_pushed_to_google, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (google_event_id, google_calendar_id) DO UPDATE SET
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
		id, event.TenantID, event.GoogleEventID, event.GoogleCalendarID, event.InternalEventID, event.InternalCalendarID,
		event.Title, event.Description, event.Location, event.StartTime, event.EndTime, event.IsAllDay, event.IsRecurring,
		event.RecurrenceRule, event.RecurrenceID, event.SyncStatus, event.SyncDirection, event.LastSyncedAt, event.LastPushedToGoogle, time.Now(),
	)
	return err
}

func (r *GoogleSyncRepo) GetSyncedEventByGoogleID(ctx context.Context, googleEventID, googleCalendarID string) (*SyncedGoogleEvent, error) {
	query := `
		SELECT id, tenant_id, google_event_id, google_calendar_id, internal_event_id, internal_calendar_id,
			title, description, location, start_time, end_time, is_all_day, is_recurring,
			recurrence_rule, recurrence_id, sync_status, sync_direction, last_synced_at, last_pushed_to_google, updated_at
		FROM synced_google_events
		WHERE google_event_id = $1 AND google_calendar_id = $2
	`

	var event SyncedGoogleEvent
	err := r.db.Pool().QueryRow(ctx, query, googleEventID, googleCalendarID).Scan(
		&event.ConnectionID, &event.TenantID, &event.GoogleEventID, &event.GoogleCalendarID,
		&event.InternalEventID, &event.InternalCalendarID, &event.Title, &event.Description,
		&event.Location, &event.StartTime, &event.EndTime, &event.IsAllDay, &event.IsRecurring,
		&event.RecurrenceRule, &event.RecurrenceID, &event.SyncStatus, &event.SyncDirection,
		&event.LastSyncedAt, &event.LastPushedToGoogle, &event.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *GoogleSyncRepo) GetSyncedEventsByConnection(ctx context.Context, connectionID string) ([]*SyncedGoogleEvent, error) {
	query := `
		SELECT id, tenant_id, google_event_id, google_calendar_id, internal_event_id, internal_calendar_id,
			title, description, location, start_time, end_time, is_all_day, is_recurring,
			recurrence_rule, recurrence_id, sync_status, sync_direction, last_synced_at, last_pushed_to_google, updated_at
		FROM synced_google_events
		WHERE connection_id = $1
		ORDER BY start_time ASC
	`

	rows, err := r.db.Pool().Query(ctx, query, connectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*SyncedGoogleEvent
	for rows.Next() {
		var event SyncedGoogleEvent
		if err := rows.Scan(
			&event.ConnectionID, &event.TenantID, &event.GoogleEventID, &event.GoogleCalendarID,
			&event.InternalEventID, &event.InternalCalendarID, &event.Title, &event.Description,
			&event.Location, &event.StartTime, &event.EndTime, &event.IsAllDay, &event.IsRecurring,
			&event.RecurrenceRule, &event.RecurrenceID, &event.SyncStatus, &event.SyncDirection,
			&event.LastSyncedAt, &event.LastPushedToGoogle, &event.UpdatedAt,
		); err != nil {
			continue
		}
		events = append(events, &event)
	}
	return events, nil
}

func (r *GoogleSyncRepo) DeleteSyncedEvent(ctx context.Context, googleEventID, googleCalendarID string) error {
	query := `DELETE FROM synced_google_events WHERE google_event_id = $1 AND google_calendar_id = $2`
	_, err := r.db.Pool().Exec(ctx, query, googleEventID, googleCalendarID)
	return err
}

func (r *GoogleSyncRepo) GetUnsyncedEvents(ctx context.Context, connectionID string, limit int) ([]*SyncedGoogleEvent, error) {
	query := `
		SELECT id, tenant_id, google_event_id, google_calendar_id, internal_event_id, internal_calendar_id,
			title, description, location, start_time, end_time, is_all_day, is_recurring,
			recurrence_rule, recurrence_id, sync_status, sync_direction, last_synced_at, last_pushed_to_google, updated_at
		FROM synced_google_events
		WHERE connection_id = $1 AND sync_status = 'pending'
		ORDER BY start_time ASC
		LIMIT $2
	`

	rows, err := r.db.Pool().Query(ctx, query, connectionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*SyncedGoogleEvent
	for rows.Next() {
		var event SyncedGoogleEvent
		if err := rows.Scan(
			&event.ConnectionID, &event.TenantID, &event.GoogleEventID, &event.GoogleCalendarID,
			&event.InternalEventID, &event.InternalCalendarID, &event.Title, &event.Description,
			&event.Location, &event.StartTime, &event.EndTime, &event.IsAllDay, &event.IsRecurring,
			&event.RecurrenceRule, &event.RecurrenceID, &event.SyncStatus, &event.SyncDirection,
			&event.LastSyncedAt, &event.LastPushedToGoogle, &event.UpdatedAt,
		); err != nil {
			continue
		}
		events = append(events, &event)
	}
	return events, nil
}

func (r *GoogleSyncRepo) UpdateSyncStatus(ctx context.Context, googleEventID, googleCalendarID, status string) error {
	query := `UPDATE synced_google_events SET sync_status = $3, updated_at = NOW() WHERE google_event_id = $1 AND google_calendar_id = $2`
	_, err := r.db.Pool().Exec(ctx, query, googleEventID, googleCalendarID, status)
	return err
}

func (r *GoogleSyncRepo) GetSyncStats(ctx context.Context, connectionID string) (map[string]int, error) {
	stats := make(map[string]int)

	query := `SELECT sync_status, COUNT(*) FROM synced_google_events WHERE connection_id = $1 GROUP BY sync_status`
	rows, err := r.db.Pool().Query(ctx, query, connectionID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		stats[status] = count
	}
	return stats, nil
}
