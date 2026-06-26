package repository

import (
	"context"
	"fmt"
	"time"

	"calendar-service/internal/hasura"
	"calendar-service/internal/models"
)

// SyncedMicrosoftEvent represents a synced event record
type SyncedMicrosoftEvent struct {
	ConnectionID          string     `json:"connection_id"`
	TenantID              string     `json:"tenant_id"`
	MicrosoftEventID      string     `json:"microsoft_event_id"`
	MicrosoftCalendarID   string     `json:"microsoft_calendar_id"`
	InternalEventID       *string    `json:"internal_event_id"`
	InternalCalendarID    *string    `json:"internal_calendar_id"`
	Title                 string     `json:"title"`
	Description           *string    `json:"description"`
	Location              *string    `json:"location"`
	StartTime             time.Time  `json:"start_time"`
	EndTime               time.Time  `json:"end_time"`
	IsAllDay              bool       `json:"is_all_day"`
	IsRecurring           bool       `json:"is_recurring"`
	RecurrenceRule        *string    `json:"recurrence_rule"`
	RecurrenceID          *string    `json:"recurrence_id"`
	SyncStatus            string     `json:"sync_status"`
	SyncDirection         string     `json:"sync_direction"`
	LastSyncedAt          time.Time  `json:"last_synced_at"`
	LastPushedToMicrosoft *time.Time `json:"last_pushed_to_microsoft"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// MicrosoftSyncRepo handles database operations for Microsoft Sync
type MicrosoftSyncRepo struct {
	hasuraClient HasuraClient
}

// NewMicrosoftSyncRepo creates a new repo
func NewMicrosoftSyncRepo(client *hasura.Client) *MicrosoftSyncRepo {
	return &MicrosoftSyncRepo{
		hasuraClient: &hasuraWrapper{client: client},
	}
}

// UpsertSyncedEvent creates or updates a synced event record
func (r *MicrosoftSyncRepo) UpsertSyncedEvent(ctx context.Context, event *SyncedMicrosoftEvent) error {
	mutation := `
	mutation UpsertMicrosoftSyncedEvent($object: synced_microsoft_events_insert_input!) {
		insert_synced_microsoft_events_one(
			object: $object,
			on_conflict: {
				constraint: synced_microsoft_events_microsoft_event_id_microsoft_calendar_id_key,
				update_columns: [
					title, description, location, start_time, end_time, 
					is_all_day, is_recurring, recurrence_rule, sync_status, last_synced_at
				]
			}
		) {
			id
		}
	}
	`

	object := map[string]interface{}{
		"tenant_id":             event.TenantID,
		"microsoft_event_id":    event.MicrosoftEventID,
		"microsoft_calendar_id": event.MicrosoftCalendarID,
		"internal_event_id":     event.InternalEventID,
		"internal_calendar_id":  event.InternalCalendarID,
		"title":                 event.Title,
		"description":           event.Description,
		"location":              event.Location,
		"start_time":            event.StartTime,
		"end_time":              event.EndTime,
		"is_all_day":            event.IsAllDay,
		"is_recurring":          event.IsRecurring,
		"recurrence_rule":       event.RecurrenceRule,
		"recurrence_id":         event.RecurrenceID,
		"sync_status":           event.SyncStatus,
		"sync_hash":             "0000000000000000000000000000000000000000000000000000000000000000", // placeholder
		"last_synced_at":        event.LastSyncedAt,
	}

	if event.ConnectionID != "" {
		object["connection_id"] = event.ConnectionID
	}

	_, err := r.hasuraClient.Mutate(mutation, map[string]interface{}{"object": object})
	return err
}

// GetPrimaryCalendarID retrieves the primary Microsoft Calendar ID for a user
func (r *MicrosoftSyncRepo) GetPrimaryCalendarID(ctx context.Context, tenantID, userID string) (string, error) {
	query := `
	query GetMicrosoftPrimaryConnection($tenant_id: uuid!, $user_id: uuid!) {
		microsoft_calendar_connections(
			where: {
				tenant_id: {_eq: $tenant_id},
				user_id: {_eq: $user_id}
			},
			limit: 1
		) {
			microsoft_calendar_id
		}
	}
	`
	result, err := r.hasuraClient.Query(query, map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   userID,
	})
	if err != nil {
		return "", err
	}

	conns, ok := result["microsoft_calendar_connections"].([]interface{})
	if !ok || len(conns) == 0 {
		return "", fmt.Errorf("no microsoft calendar connection found")
	}

	conn := conns[0].(map[string]interface{})
	if calID, ok := conn["microsoft_calendar_id"].(string); ok {
		return calID, nil
	}
	return "", fmt.Errorf("microsoft_calendar_id not found in connection")
}

// GetSyncedEventByMicrosoftID retrieves a synced event by its Microsoft ID
func (r *MicrosoftSyncRepo) GetSyncedEventByMicrosoftID(ctx context.Context, microsoftEventID, microsoftCalendarID string) (*SyncedMicrosoftEvent, error) {
	query := `
	query GetSyncedMicrosoftEvent($microsoft_event_id: String!, $microsoft_calendar_id: String!) {
		synced_microsoft_events(
			where: {
				microsoft_event_id: {_eq: $microsoft_event_id},
				microsoft_calendar_id: {_eq: $microsoft_calendar_id}
			},
			limit: 1
		) {
			connection_id tenant_id microsoft_event_id microsoft_calendar_id
			internal_event_id internal_calendar_id title description location
			start_time end_time is_all_day is_recurring recurrence_rule
			sync_status last_synced_at updated_at
		}
	}
	`

	result, err := r.hasuraClient.Query(query, map[string]interface{}{
		"microsoft_event_id":    microsoftEventID,
		"microsoft_calendar_id": microsoftCalendarID,
	})
	if err != nil {
		return nil, err
	}

	events, ok := result["synced_microsoft_events"].([]interface{})
	if !ok || len(events) == 0 {
		return nil, nil // Not found
	}

	raw := events[0].(map[string]interface{})
	return parseSyncedMicrosoftEventMap(raw), nil
}

// GetSyncedEventByInternalID retrieves a synced event by internal ID
func (r *MicrosoftSyncRepo) GetSyncedEventByInternalID(ctx context.Context, internalEventID string) (*SyncedMicrosoftEvent, error) {
	query := `
	query GetSyncedMicrosoftEventByInternalID($internal_event_id: String!) {
		synced_microsoft_events(
			where: {
				internal_event_id: {_eq: $internal_event_id}
			},
			limit: 1
		) {
			connection_id tenant_id microsoft_event_id microsoft_calendar_id
			internal_event_id internal_calendar_id title description location
			start_time end_time is_all_day is_recurring recurrence_rule
			sync_status last_synced_at updated_at
		}
	}
	`

	result, err := r.hasuraClient.Query(query, map[string]interface{}{
		"internal_event_id": internalEventID,
	})
	if err != nil {
		return nil, err
	}

	events, ok := result["synced_microsoft_events"].([]interface{})
	if !ok || len(events) == 0 {
		return nil, nil // Not found
	}

	raw := events[0].(map[string]interface{})
	return parseSyncedMicrosoftEventMap(raw), nil
}

func parseSyncedMicrosoftEventMap(raw map[string]interface{}) *SyncedMicrosoftEvent {
	ev := &SyncedMicrosoftEvent{}
	if val, ok := raw["connection_id"].(string); ok {
		ev.ConnectionID = val
	}
	if val, ok := raw["tenant_id"].(string); ok {
		ev.TenantID = val
	}
	if val, ok := raw["microsoft_event_id"].(string); ok {
		ev.MicrosoftEventID = val
	}
	if val, ok := raw["microsoft_calendar_id"].(string); ok {
		ev.MicrosoftCalendarID = val
	}
	if val, ok := raw["internal_event_id"].(string); ok {
		ev.InternalEventID = &val
	}
	if val, ok := raw["title"].(string); ok {
		ev.Title = val
	}
	if val, ok := raw["sync_status"].(string); ok {
		ev.SyncStatus = val
	}
	if val, ok := raw["last_synced_at"].(string); ok {
		ev.LastSyncedAt, _ = time.Parse(time.RFC3339, val)
	}
	return ev
}

// GetEvent retrieves a single internal event
func (r *MicrosoftSyncRepo) GetEvent(ctx context.Context, eventID string) (*models.InternalEvent, error) {
	query := `
	query GetInternalEvent($id: uuid!) {
		internal_events_by_pk(id: $id) {
			id tenant_id user_id title description location
			start_time end_time is_all_day is_recurring recurrence_rule
			created_at updated_at
		}
	}
	`
	result, err := r.hasuraClient.Query(query, map[string]interface{}{"id": eventID})
	if err != nil {
		return nil, err
	}
	raw, ok := result["internal_events_by_pk"].(map[string]interface{})
	if !ok || raw == nil {
		return nil, fmt.Errorf("event not found")
	}

	return parseInternalEventMap(raw), nil
}
