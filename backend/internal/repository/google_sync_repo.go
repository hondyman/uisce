package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// SyncedGoogleEvent represents a synced event record
type SyncedGoogleEvent struct {
	ConnectionID       string    `json:"connection_id" db:"connection_id"`
	TenantID           string    `json:"tenant_id" db:"tenant_id"`
	GoogleEventID      string    `json:"google_event_id" db:"google_event_id"`
	GoogleCalendarID   string    `json:"google_calendar_id" db:"google_calendar_id"`
	InternalEventID    *string   `json:"internal_event_id" db:"internal_event_id"`
	InternalCalendarID *string   `json:"internal_calendar_id" db:"internal_calendar_id"`
	Title              string    `json:"title" db:"title"`
	Description        *string   `json:"description" db:"description"`
	Location           *string   `json:"location" db:"location"`
	StartTime          time.Time `json:"start_time" db:"start_time"`
	EndTime            time.Time `json:"end_time" db:"end_time"`
	IsAllDay           bool      `json:"is_all_day" db:"is_all_day"`
	IsRecurring        bool      `json:"is_recurring" db:"is_recurring"`
	RecurrenceRule     *string   `json:"recurrence_rule" db:"recurrence_rule"`
	RecurrenceID       *string   `json:"recurrence_id" db:"recurrence_id"`
	SyncStatus         string    `json:"sync_status" db:"sync_status"`
	LastSyncedAt       time.Time `json:"last_synced_at" db:"last_synced_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// ConflictRecord represents a conflict record in the database
type ConflictRecord struct {
	ID                 string      `json:"id" db:"id"`
	TenantID           string      `json:"tenant_id" db:"tenant_id"`
	UserID             string      `json:"user_id" db:"user_id"`
	ConnectionID       string      `json:"connection_id" db:"connection_id"`
	GoogleEventID      string      `json:"google_event_id" db:"google_event_id"`
	GoogleCalendarID   string      `json:"google_calendar_id" db:"google_calendar_id"`
	InternalEventID    *string     `json:"internal_event_id" db:"internal_event_id"`
	ConflictType       string      `json:"conflict_type" db:"conflict_type"`
	Severity           string      `json:"severity" db:"severity"`
	Description        string      `json:"description" db:"description"`
	GoogleEventData    interface{} `json:"google_event_data"`
	InternalEventData  interface{} `json:"internal_event_data"`
	ResolutionStatus   string      `json:"resolution_status" db:"resolution_status"`
	ResolutionStrategy *string     `json:"resolution_strategy" db:"resolution_strategy"`
	DetectedAt         time.Time   `json:"detected_at" db:"detected_at"`
}

// GoogleSyncRepo handles database operations for Google Calendar Sync
type GoogleSyncRepo struct {
	db *sqlx.DB
}

// NewGoogleSyncRepo creates a new repo
func NewGoogleSyncRepo(db *sqlx.DB) *GoogleSyncRepo {
	return &GoogleSyncRepo{db: db}
}

// UpsertSyncedEvent creates or updates a synced event record
func (r *GoogleSyncRepo) UpsertSyncedEvent(ctx context.Context, event *SyncedGoogleEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO synced_google_events (
			connection_id, tenant_id, google_event_id, google_calendar_id,
			internal_event_id, internal_calendar_id, title, description, location,
			start_time, end_time, is_all_day, is_recurring, recurrence_rule, recurrence_id,
			sync_status, last_synced_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15,
			$16, $17, NOW()
		)
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
	`, event.ConnectionID, event.TenantID, event.GoogleEventID, event.GoogleCalendarID,
		event.InternalEventID, event.InternalCalendarID, event.Title, event.Description, event.Location,
		event.StartTime, event.EndTime, event.IsAllDay, event.IsRecurring, event.RecurrenceRule, event.RecurrenceID,
		event.SyncStatus, event.LastSyncedAt)
	return err
}

// GetSyncedEventByGoogleID retrieves a synced event
func (r *GoogleSyncRepo) GetSyncedEventByGoogleID(ctx context.Context, connectionID, googleEventID, googleCalendarID string) (*SyncedGoogleEvent, error) {
	var event SyncedGoogleEvent
	err := r.db.GetContext(ctx, &event, `
		SELECT connection_id, tenant_id, google_event_id, google_calendar_id,
		       internal_event_id, internal_calendar_id, title, description, location,
		       start_time, end_time, is_all_day, is_recurring, recurrence_rule,
		       sync_status, last_synced_at, updated_at
		FROM synced_google_events
		WHERE google_event_id = $1 AND google_calendar_id = $2
		LIMIT 1
	`, googleEventID, googleCalendarID)

	if err != nil {
		return nil, fmt.Errorf("event not found")
	}
	return &event, nil
}

// GetSyncedEventByInternalID retrieves a synced event by internal ID
func (r *GoogleSyncRepo) GetSyncedEventByInternalID(ctx context.Context, internalEventID string) (*SyncedGoogleEvent, error) {
	var event SyncedGoogleEvent
	err := r.db.GetContext(ctx, &event, `
		SELECT connection_id, tenant_id, google_event_id, google_calendar_id,
		       internal_event_id, internal_calendar_id, title, description, location,
		       start_time, end_time, is_all_day, is_recurring, recurrence_rule,
		       sync_status, last_synced_at, updated_at
		FROM synced_google_events
		WHERE internal_event_id = $1
		LIMIT 1
	`, internalEventID)

	if err != nil {
		return nil, nil // Not found
	}
	return &event, nil
}

// FindConflictingEvents finds internal events overlapping with the given time range
func (r *GoogleSyncRepo) FindConflictingEvents(ctx context.Context, tenantID string, start, end time.Time, excludeIDs []string) ([]SyncedGoogleEvent, error) {
	type row struct {
		ID          string    `db:"id"`
		Title       string    `db:"title"`
		Description *string   `db:"description"`
		Location    *string   `db:"location"`
		StartTime   time.Time `db:"start_time"`
		EndTime     time.Time `db:"end_time"`
	}

	var rows []row
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, title, description, location, start_time, end_time
		FROM internal_events
		WHERE tenant_id = $1
		  AND start_time < $3
		  AND end_time > $2
	`, tenantID, start, end)

	if err != nil {
		return []SyncedGoogleEvent{}, nil
	}

	var events []SyncedGoogleEvent
	for _, r := range rows {
		// Skip excluded IDs
		excluded := false
		for _, ex := range excludeIDs {
			if r.ID == ex {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		events = append(events, SyncedGoogleEvent{
			InternalEventID: &r.ID,
			Title:           r.Title,
			Description:     r.Description,
			Location:        r.Location,
			StartTime:       r.StartTime,
			EndTime:         r.EndTime,
			TenantID:        tenantID,
		})
	}
	return events, nil
}

// SaveConflict persists a conflict to the database
func (r *GoogleSyncRepo) SaveConflict(ctx context.Context, conflict interface{}) error {
	conflictMap, ok := conflict.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid conflict data type")
	}

	googleDataJSON, _ := json.Marshal(conflictMap["google_event_data"])
	internalDataJSON, _ := json.Marshal(conflictMap["internal_event_data"])

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_conflicts (
			id, tenant_id, user_id, connection_id, google_event_id, google_calendar_id,
			internal_event_id, conflict_type, severity, description,
			google_event_data, internal_event_data, resolution_status, detected_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, 'pending', NOW()
		)
	`, conflictMap["id"], conflictMap["tenant_id"], conflictMap["user_id"],
		conflictMap["connection_id"], conflictMap["google_event_id"], conflictMap["google_calendar_id"],
		conflictMap["internal_event_id"], conflictMap["conflict_type"], conflictMap["severity"],
		conflictMap["description"], string(googleDataJSON), string(internalDataJSON))

	return err
}

// ListSyncedEvents lists synced events for a user/tenant within a time range
func (r *GoogleSyncRepo) ListSyncedEvents(ctx context.Context, tenantID, userID string, start, end time.Time) ([]SyncedGoogleEvent, error) {
	var events []SyncedGoogleEvent
	err := r.db.SelectContext(ctx, &events, `
		SELECT connection_id, tenant_id, google_event_id, google_calendar_id,
		       internal_event_id, internal_calendar_id, title, description, location,
		       start_time, end_time, is_all_day, is_recurring, recurrence_rule,
		       sync_status, last_synced_at, updated_at
		FROM synced_google_events
		WHERE tenant_id = $1
		  AND start_time < $3
		  AND end_time > $2
		ORDER BY start_time
	`, tenantID, start, end)

	if err != nil {
		return []SyncedGoogleEvent{}, nil
	}
	return events, nil
}

// GetPrimaryCalendarID retrieves the primary Google Calendar ID for a user
func (r *GoogleSyncRepo) GetPrimaryCalendarID(ctx context.Context, tenantID, userID string) (string, error) {
	var calID string
	err := r.db.GetContext(ctx, &calID, `
		SELECT google_calendar_id
		FROM google_calendar_connections
		WHERE tenant_id = $1 AND user_id = $2
		LIMIT 1
	`, tenantID, userID)

	if err != nil {
		return "", fmt.Errorf("no google calendar connection found")
	}
	return calID, nil
}

// CreateInternalEvent creates a new internal event
func (r *GoogleSyncRepo) CreateInternalEvent(ctx context.Context, event *models.InternalEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO internal_events (
			id, tenant_id, user_id, title, description, location,
			start_time, end_time, is_all_day, is_recurring, recurrence_rule,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13
		)
	`, event.ID, event.TenantID, event.UserID, event.Title, event.Description, event.Location,
		event.StartTime, event.EndTime, event.IsAllDay, event.IsRecurring, event.RecurrenceRule,
		event.CreatedAt, event.UpdatedAt)
	return err
}

// UpdateInternalEvent updates an existing internal event
func (r *GoogleSyncRepo) UpdateInternalEvent(ctx context.Context, event *models.InternalEvent) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE internal_events SET
			title = $2, description = $3, location = $4,
			start_time = $5, end_time = $6,
			is_all_day = $7, is_recurring = $8, recurrence_rule = $9,
			updated_at = $10
		WHERE id = $1
	`, event.ID, event.Title, event.Description, event.Location,
		event.StartTime, event.EndTime, event.IsAllDay, event.IsRecurring, event.RecurrenceRule,
		event.UpdatedAt)
	return err
}

// DeleteInternalEvent deletes an internal event
func (r *GoogleSyncRepo) DeleteInternalEvent(ctx context.Context, eventID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM internal_events WHERE id = $1`, eventID)
	return err
}

// GetConflict retrieves a conflict by ID
func (r *GoogleSyncRepo) GetConflict(ctx context.Context, conflictID string) (*ConflictRecord, error) {
	type row struct {
		ID                 string    `db:"id"`
		TenantID           string    `db:"tenant_id"`
		UserID             string    `db:"user_id"`
		ConnectionID       string    `db:"connection_id"`
		GoogleEventID      string    `db:"google_event_id"`
		GoogleCalendarID   string    `db:"google_calendar_id"`
		InternalEventID    *string   `db:"internal_event_id"`
		ConflictType       string    `db:"conflict_type"`
		Severity           string    `db:"severity"`
		Description        string    `db:"description"`
		GoogleEventData    string    `db:"google_event_data"`
		InternalEventData  string    `db:"internal_event_data"`
		ResolutionStatus   string    `db:"resolution_status"`
		ResolutionStrategy *string   `db:"resolution_strategy"`
		DetectedAt         time.Time `db:"detected_at"`
	}

	var r2 row
	err := r.db.GetContext(ctx, &r2, `
		SELECT id, tenant_id, user_id, connection_id, google_event_id, google_calendar_id,
		       internal_event_id, conflict_type, severity, description,
		       COALESCE(google_event_data::text,'{}') as google_event_data,
		       COALESCE(internal_event_data::text,'{}') as internal_event_data,
		       resolution_status, resolution_strategy, detected_at
		FROM sync_conflicts WHERE id = $1
	`, conflictID)

	if err != nil {
		return nil, nil
	}

	var googleData, internalData interface{}
	_ = json.Unmarshal([]byte(r2.GoogleEventData), &googleData)
	_ = json.Unmarshal([]byte(r2.InternalEventData), &internalData)

	return &ConflictRecord{
		ID:                 r2.ID,
		TenantID:           r2.TenantID,
		UserID:             r2.UserID,
		ConnectionID:       r2.ConnectionID,
		GoogleEventID:      r2.GoogleEventID,
		GoogleCalendarID:   r2.GoogleCalendarID,
		InternalEventID:    r2.InternalEventID,
		ConflictType:       r2.ConflictType,
		Severity:           r2.Severity,
		Description:        r2.Description,
		GoogleEventData:    googleData,
		InternalEventData:  internalData,
		ResolutionStatus:   r2.ResolutionStatus,
		ResolutionStrategy: r2.ResolutionStrategy,
		DetectedAt:         r2.DetectedAt,
	}, nil
}

// UpdateConflictStatus updates the status and strategy of a conflict
func (r *GoogleSyncRepo) UpdateConflictStatus(ctx context.Context, conflictID string, status string, strategy *string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE sync_conflicts
		SET resolution_status = $2, resolution_strategy = $3, resolved_at = NOW()
		WHERE id = $1
	`, conflictID, status, strategy)
	return err
}
