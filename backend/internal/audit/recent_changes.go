package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// RecentChange represents a summary of a recent entity change
type RecentChange struct {
	EntityType   string    `json:"entity_type"`
	EntityID     string    `json:"entity_id"`
	EntityName   string    `json:"entity_name,omitempty"` // Extracted from entity_data if available
	ChangeType   string    `json:"change_type"`
	ChangedBy    string    `json:"changed_by"`
	SystemFrom   time.Time `json:"system_from"`
	VersionCount int       `json:"version_count"` // Total versions for this entity
}

// GetRecentChanges retrieves a summary of recent changes across all entity types
func (bt *BitemporalTracker) GetRecentChanges(ctx context.Context, from, to *time.Time, entityType string, limit int) ([]RecentChange, error) {
	if limit <= 0 {
		limit = 100
	}

	// Trino timestamp format
	const trinoTSFormat = "2006-01-02 15:04:05.000000 UTC"

	var changes []RecentChange
	entityTypes := []string{"tenant", "instance", "connection", "product"}

	// Filter to specific entity type if provided
	if entityType != "" {
		entityTypes = []string{entityType}
	}

	for _, eType := range entityTypes {
		tableName := bt.getTableName(eType)
		idColumn := bt.getEntityIDColumn(eType)

		query := fmt.Sprintf(`
			SELECT 
				'%s' as entity_type,
				%s as entity_id,
				change_type,
				changed_by,
				system_from,
				entity_data
			FROM %s
			WHERE is_current = true
		`, eType, idColumn, tableName)

		if from != nil {
			query += fmt.Sprintf(" AND system_from >= CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE)", from.UTC().Format(trinoTSFormat))
		}

		if to != nil {
			query += fmt.Sprintf(" AND system_from <= CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE)", to.UTC().Format(trinoTSFormat))
		}

		query += fmt.Sprintf(" ORDER BY system_from DESC LIMIT %d", limit)

		rows, err := bt.trinoClient.Query(ctx, query)
		if err != nil {
			// Log error but continue with other entity types
			continue
		}

		for rows.Next() {
			var change RecentChange
			var entityDataJSON string
			var systemFromTime time.Time

			err := rows.Scan(
				&change.EntityType,
				&change.EntityID,
				&change.ChangeType,
				&change.ChangedBy,
				&systemFromTime,
				&entityDataJSON,
			)

			if err != nil {
				continue
			}

			change.SystemFrom = systemFromTime

			// Try to extract entity name from entity_data
			var entityData map[string]interface{}
			if err := json.Unmarshal([]byte(entityDataJSON), &entityData); err == nil {
				// Try common name fields
				if name, ok := entityData["name"].(string); ok {
					change.EntityName = name
				} else if displayName, ok := entityData["display_name"].(string); ok {
					change.EntityName = displayName
				}
			}

			// Get version count for this entity
			countQuery := fmt.Sprintf(`
				SELECT COUNT(*) FROM %s WHERE %s = '%s'
			`, tableName, idColumn, change.EntityID)

			countRow := bt.trinoClient.QueryRow(ctx, countQuery)
			var count int
			if err := countRow.Scan(&count); err == nil {
				change.VersionCount = count
			}

			changes = append(changes, change)
		}
		rows.Close()
	}

	return changes, nil
}

// Helper to scan a row into EntitySnapshot
func (bt *BitemporalTracker) scanRow(row *sql.Row) (*EntitySnapshot, error) {
	var snapshot EntitySnapshot
	var entityDataJSON string
	var validFromStr, systemFromStr string
	var validToStr, systemToStr sql.NullString

	err := row.Scan(
		&snapshot.VersionID,
		&validFromStr,
		&validToStr,
		&systemFromStr,
		&systemToStr,
		&snapshot.ChangeType,
		&snapshot.ChangedBy,
		&snapshot.ChangeReason,
		&entityDataJSON,
		&snapshot.IsCurrent,
		&snapshot.IsDeleted,
	)

	if err != nil {
		return nil, err
	}

	// Parse timestamps
	snapshot.ValidFrom, _ = time.Parse(time.RFC3339Nano, validFromStr)
	snapshot.SystemFrom, _ = time.Parse(time.RFC3339Nano, systemFromStr)

	if validToStr.Valid {
		t, _ := time.Parse(time.RFC3339Nano, validToStr.String)
		snapshot.ValidTo = &t
	}

	if systemToStr.Valid {
		t, _ := time.Parse(time.RFC3339Nano, systemToStr.String)
		snapshot.SystemTo = &t
	}

	// Parse entity data
	if err := json.Unmarshal([]byte(entityDataJSON), &snapshot.EntityData); err != nil {
		return nil, err
	}

	return &snapshot, nil
}
