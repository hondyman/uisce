package ops

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ========== Timeline and Incident Management ==========

// InsertEvent inserts a new event into the ops_events table
func (p *PostgresStore) InsertEvent(ctx context.Context, e Event) error {
	query := `
		INSERT INTO ops_events (id, incident_id, event_type, scope, tenant_id, endpoint_path, region, 
		                        fingerprint_id, alert_id, severity, title, details, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := p.db.ExecContext(ctx, query,
		e.ID, e.IncidentID, e.EventType, e.Scope, e.TenantID, e.EndpointPath, e.Region,
		e.FingerprintID, e.AlertID, e.Severity, e.Title, e.Details, e.OccurredAt, time.Now().UTC(),
	)
	return err
}

// ListEvents retrieves events since a given time, ordered by occurred_at DESC
func (p *PostgresStore) ListEvents(ctx context.Context, since time.Time, limit int) ([]Event, error) {
	query := `
		SELECT id, incident_id, event_type, scope, tenant_id, endpoint_path, region, 
		       fingerprint_id, alert_id, severity, title, details, occurred_at, created_at
		FROM ops_events
		WHERE occurred_at >= $1
		ORDER BY occurred_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID, &e.IncidentID, &e.EventType, &e.Scope, &e.TenantID, &e.EndpointPath, &e.Region,
			&e.FingerprintID, &e.AlertID, &e.Severity, &e.Title, &e.Details, &e.OccurredAt, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// GetIncident retrieves an incident with all its related events
func (p *PostgresStore) GetIncident(ctx context.Context, id uuid.UUID) (*Incident, []Event, error) {
	query := `
		SELECT id, status, severity, title, summary, root_cause, region, started_at, ended_at, created_at, updated_at
		FROM ops_incidents
		WHERE id = $1
	`

	var inc Incident
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&inc.ID, &inc.Status, &inc.Severity, &inc.Title, &inc.Summary, &inc.RootCause, &inc.Region,
		&inc.StartedAt, &inc.EndedAt, &inc.CreatedAt, &inc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("incident not found")
	}
	if err != nil {
		return nil, nil, err
	}

	// Get related events
	eventQuery := `
		SELECT id, incident_id, event_type, scope, tenant_id, endpoint_path, region,
		       fingerprint_id, alert_id, severity, title, details, occurred_at, created_at
		FROM ops_events
		WHERE incident_id = $1
		ORDER BY occurred_at ASC
	`

	rows, err := p.db.QueryContext(ctx, eventQuery, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID, &e.IncidentID, &e.EventType, &e.Scope, &e.TenantID, &e.EndpointPath, &e.Region,
			&e.FingerprintID, &e.AlertID, &e.Severity, &e.Title, &e.Details, &e.OccurredAt, &e.CreatedAt,
		); err != nil {
			return nil, nil, err
		}
		events = append(events, e)
	}

	inc.Events = events
	return &inc, events, rows.Err()
}

// UpsertIncidentForEvent correlates an event to existing incident or creates new one
// Strategy: Look back 60 minutes for open incidents matching event scope
func (p *PostgresStore) UpsertIncidentForEvent(ctx context.Context, e Event) (*Incident, error) {
	lookbackTime := time.Now().UTC().Add(-60 * time.Minute)

	// Try to find matching incident
	var incidentID *uuid.UUID

	switch e.Scope {
	case "tenant":
		if e.TenantID != nil {
			correlationQuery := `
				SELECT DISTINCT i.id FROM ops_incidents i
				JOIN ops_events ev ON i.id = ev.incident_id
				WHERE i.status = 'open'
				AND i.started_at >= $1
				AND (ev.tenant_id = $2 OR ev.event_type = $3)
				ORDER BY i.started_at DESC
				LIMIT 1
			`
			var id uuid.UUID
			err := p.db.QueryRowContext(ctx, correlationQuery, lookbackTime, e.TenantID, EventTenantHealth).Scan(&id)
			if err == nil {
				incidentID = &id
			} else if err != sql.ErrNoRows {
				return nil, err
			}
		}

	case "endpoint":
		if e.EndpointPath != nil {
			correlationQuery := `
				SELECT DISTINCT i.id FROM ops_incidents i
				JOIN ops_events ev ON i.id = ev.incident_id
				WHERE i.status = 'open'
				AND i.started_at >= $1
				AND (ev.endpoint_path = $2 OR ev.event_type = $3)
				ORDER BY i.started_at DESC
				LIMIT 1
			`
			var id uuid.UUID
			err := p.db.QueryRowContext(ctx, correlationQuery, lookbackTime, e.EndpointPath, EventEndpointHealth).Scan(&id)
			if err == nil {
				incidentID = &id
			} else if err != sql.ErrNoRows {
				return nil, err
			}
		}

	case "region":
		if e.Region != nil {
			correlationQuery := `
				SELECT DISTINCT i.id FROM ops_incidents i
				JOIN ops_events ev ON i.id = ev.incident_id
				WHERE i.status = 'open'
				AND i.started_at >= $1
				AND ev.region = $2
				ORDER BY i.started_at DESC
				LIMIT 1
			`
			var id uuid.UUID
			err := p.db.QueryRowContext(ctx, correlationQuery, lookbackTime, e.Region).Scan(&id)
			if err == nil {
				incidentID = &id
			} else if err != sql.ErrNoRows {
				return nil, err
			}
		}

	default:
		// global scope - only correlate if same alert
		if e.AlertID != nil {
			correlationQuery := `
				SELECT DISTINCT i.id FROM ops_incidents i
				JOIN ops_events ev ON i.id = ev.incident_id
				WHERE i.status = 'open'
				AND i.started_at >= $1
				AND ev.alert_id = $2
				ORDER BY i.started_at DESC
				LIMIT 1
			`
			var id uuid.UUID
			err := p.db.QueryRowContext(ctx, correlationQuery, lookbackTime, e.AlertID).Scan(&id)
			if err == nil {
				incidentID = &id
			} else if err != sql.ErrNoRows {
				return nil, err
			}
		}
	}

	// If found matching incident, retrieve and return it
	if incidentID != nil {
		return p.getIncidentByID(ctx, *incidentID)
	}

	// Create new incident with region from event
	newIncID := uuid.New()
	insertQuery := `
		INSERT INTO ops_incidents (id, status, severity, title, region, started_at, created_at, updated_at)
		VALUES ($1, 'open', $2, $3, $4, $5, $6, $7)
	`
	_, err := p.db.ExecContext(ctx, insertQuery,
		newIncID, e.Severity, e.Title, e.Region, e.OccurredAt, time.Now().UTC(), time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}

	return p.getIncidentByID(ctx, newIncID)
}

// CloseIncident closes an incident with optional summary and root cause
func (p *PostgresStore) CloseIncident(ctx context.Context, id uuid.UUID, summary, rootCause *string) error {
	query := `
		UPDATE ops_incidents
		SET status = 'closed', summary = $2, root_cause = $3, ended_at = $4, updated_at = $5
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, summary, rootCause, time.Now().UTC(), time.Now().UTC())
	return err
}

// getIncidentByID is a helper to fetch incident by ID
func (p *PostgresStore) getIncidentByID(ctx context.Context, id uuid.UUID) (*Incident, error) {
	query := `
		SELECT id, status, severity, title, summary, root_cause, region, started_at, ended_at, created_at, updated_at
		FROM ops_incidents
		WHERE id = $1
	`

	var inc Incident
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&inc.ID, &inc.Status, &inc.Severity, &inc.Title, &inc.Summary, &inc.RootCause, &inc.Region,
		&inc.StartedAt, &inc.EndedAt, &inc.CreatedAt, &inc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &inc, nil
}

// ========== Action History ==========

// InsertActionHistory inserts a new action history record
func (p *PostgresStore) InsertActionHistory(ctx context.Context, history ActionHistory) error {
	query := `
		INSERT INTO ops_action_history (id, incident_id, action_type, status, parameters, result, error_msg, executed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := p.db.ExecContext(ctx, query,
		history.ID, history.IncidentID, history.ActionType, history.Status, history.Parameters,
		history.Result, history.ErrorMsg, history.ExecutedAt, history.CreatedAt, history.UpdatedAt,
	)
	return err
}

// UpdateActionHistory updates an action history record with result or error
func (p *PostgresStore) UpdateActionHistory(ctx context.Context, id uuid.UUID, status string, result []byte, errorMsg *string) error {
	query := `
		UPDATE ops_action_history
		SET status = $1, result = $2, error_msg = $3, executed_at = $4, updated_at = $5
		WHERE id = $6
	`
	now := time.Now().UTC()
	_, err := p.db.ExecContext(ctx, query, status, result, errorMsg, now, now, id)
	return err
}

// GetActionHistory retrieves an action history record by ID
func (p *PostgresStore) GetActionHistory(ctx context.Context, id uuid.UUID) (*ActionHistory, error) {
	query := `
		SELECT id, incident_id, action_type, status, parameters, result, error_msg, executed_at, created_at, updated_at
		FROM ops_action_history
		WHERE id = $1
	`
	var history ActionHistory
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&history.ID, &history.IncidentID, &history.ActionType, &history.Status, &history.Parameters,
		&history.Result, &history.ErrorMsg, &history.ExecutedAt, &history.CreatedAt, &history.UpdatedAt,
	)
	return &history, err
}

// ListIncidentActions retrieves all actions for an incident
func (p *PostgresStore) ListIncidentActions(ctx context.Context, incidentID uuid.UUID, limit int) ([]ActionHistory, error) {
	query := `
		SELECT id, incident_id, action_type, status, parameters, result, error_msg, executed_at, created_at, updated_at
		FROM ops_action_history
		WHERE incident_id = $1
		ORDER BY executed_at DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, incidentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []ActionHistory
	for rows.Next() {
		var a ActionHistory
		if err := rows.Scan(
			&a.ID, &a.IncidentID, &a.ActionType, &a.Status, &a.Parameters,
			&a.Result, &a.ErrorMsg, &a.ExecutedAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}

	return actions, rows.Err()
}

// ========== Phase 3.9: Incident Listing Methods ==========

// ListIncidents returns all incidents up to a limit
func (p *PostgresStore) ListIncidents(ctx context.Context, limit int) ([]Incident, error) {
	query := `
		SELECT id, status, severity, title, summary, root_cause, region, started_at, ended_at, created_at, updated_at
		FROM ops_incidents
		ORDER BY started_at DESC
		LIMIT $1
	`

	rows, err := p.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var inc Incident
		if err := rows.Scan(
			&inc.ID, &inc.Status, &inc.Severity, &inc.Title, &inc.Summary, &inc.RootCause, &inc.Region,
			&inc.StartedAt, &inc.EndedAt, &inc.CreatedAt, &inc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		incidents = append(incidents, inc)
	}

	return incidents, rows.Err()
}

// ListIncidentsByRegion returns incidents for a specific region
func (p *PostgresStore) ListIncidentsByRegion(ctx context.Context, region string, limit int) ([]Incident, error) {
	query := `
		SELECT id, status, severity, title, summary, root_cause, region, started_at, ended_at, created_at, updated_at
		FROM ops_incidents
		WHERE region = $1
		ORDER BY started_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, region, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var inc Incident
		if err := rows.Scan(
			&inc.ID, &inc.Status, &inc.Severity, &inc.Title, &inc.Summary, &inc.RootCause, &inc.Region,
			&inc.StartedAt, &inc.EndedAt, &inc.CreatedAt, &inc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		incidents = append(incidents, inc)
	}

	return incidents, rows.Err()
}
