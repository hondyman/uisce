//go:build !no_parquet

package analytics

import (
	"bytes"
	"context"
	"database/sql"
	"time"

	"github.com/segmentio/parquet-go"
)

type AuditExporter struct {
	db *sql.DB
	// s3Client S3Client // interface placeholder
}

func NewAuditExporter(db *sql.DB) *AuditExporter {
	return &AuditExporter{db: db}
}

type AuditRecord struct {
	ID         string `parquet:"id"`
	InstanceID string `parquet:"instance_id"`
	// TenantID   string   `parquet:"tenant_id"` // Add table column first
	EventType string    `parquet:"event_type"`
	StepKey   string    `parquet:"step_key"`
	ActorID   string    `parquet:"actor_id"`
	ActorRole string    `parquet:"actor_role"`
	OldValue  string    `parquet:"old_value"` // JSON string
	NewValue  string    `parquet:"new_value"` // JSON string
	Reason    string    `parquet:"reason"`
	IPAddress string    `parquet:"ip_address"`
	UserAgent string    `parquet:"user_agent"`
	CreatedAt time.Time `parquet:"created_at"`
}

// ExportAuditLogsToParquet exports logs to a buffer (simulating S3 write)
func (e *AuditExporter) ExportAuditLogsToParquet(
	ctx context.Context,
	since time.Time,
) ([]byte, error) {
	rows, err := e.db.QueryContext(ctx, `
        SELECT
            id, instance_id,
            event_type, step_key, actor_id, actor_role,
            old_value, new_value, reason, ip_address, user_agent,
            created_at
        FROM workflow_audit_log
        WHERE created_at >= $1
        ORDER BY created_at
    `, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []AuditRecord

	for rows.Next() {
		var rec AuditRecord
		var oldJSON, newJSON []byte
		err := rows.Scan(
			&rec.ID, &rec.InstanceID,
			&rec.EventType, &rec.StepKey, &rec.ActorID, &rec.ActorRole,
			&oldJSON, &newJSON, &rec.Reason, &rec.IPAddress, &rec.UserAgent,
			&rec.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Store JSON as string in Parquet for flexibility
		rec.OldValue = string(oldJSON)
		rec.NewValue = string(newJSON)

		records = append(records, rec)
	}

	// Write to Parquet buffer
	var buf bytes.Buffer
	writer := parquet.NewGenericWriter[AuditRecord](&buf)

	_, err = writer.Write(records)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
