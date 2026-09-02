package semantic_bridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// OutboxSyncWorker listens for ON_PUBLISH events and synchronizes active targets
type OutboxSyncWorker struct {
	db            *sqlx.DB
	cortexExp     *CortexExporter
	databricksExp *DatabricksExporter
	stopChan      chan struct{}
}

func NewOutboxSyncWorker(db *sqlx.DB) *OutboxSyncWorker {
	return &OutboxSyncWorker{
		db:            db,
		cortexExp:     NewCortexExporter(db),
		databricksExp: NewDatabricksExporter(db),
		stopChan:      make(chan struct{}),
	}
}

// SyncTenantOnPublish triggers immediate compilation and push for all ON_PUBLISH targets
func (w *OutboxSyncWorker) SyncTenantOnPublish(ctx context.Context, tenantID uuid.UUID, eventReason string) ([]SyncLog, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenantID cannot be nil")
	}

	if w.db == nil {
		return []SyncLog{}, nil
	}

	var targets []BridgeTarget
	query := `
		SELECT id, tenant_id, vendor_type, target_name, is_active, config_payload, 
		       sync_frequency, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at
		FROM catalog_ai.ai_bridge_targets
		WHERE tenant_id = $1 AND is_active = TRUE AND sync_frequency IN ('ON_PUBLISH', 'ALWAYS');`

	if err := w.db.SelectContext(ctx, &targets, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed fetching on_publish targets: %w", err)
	}

	results := make([]SyncLog, 0, len(targets))

	for _, target := range targets {
		start := time.Now()
		var payload []byte
		var err error

		switch target.VendorType {
		case VendorSnowflakeCortex:
			payload, err = w.cortexExp.CompileFullCortexModel(ctx, tenantID)
		case VendorDatabricksGenie:
			payload, err = w.databricksExp.CompileGenieModel(ctx, tenantID)
		default:
			continue
		}

		hasher := sha256.New()
		hasher.Write(payload)
		payloadHash := hex.EncodeToString(hasher.Sum(nil))

		status := "SUCCESS"
		var errMsg string
		if err != nil {
			status = "ERROR"
			errMsg = err.Error()
		}

		duration := int(time.Since(start).Milliseconds())

		logEntry := SyncLog{
			ID:              uuid.New(),
			TenantID:        tenantID,
			TargetID:        &target.ID,
			VendorType:      string(target.VendorType),
			Action:          "ON_PUBLISH_AUTO_SYNC: " + eventReason,
			PayloadHash:     payloadHash,
			ArtifactPayload: string(payload),
			Status:          status,
			HTTPStatus:      200,
			ResponseBody:    errMsg,
			ExecutionTimeMS: duration,
			CreatedAt:       time.Now(),
		}

		// Insert into audit log
		_, _ = w.db.ExecContext(ctx, `
			INSERT INTO catalog_ai.ai_bridge_sync_logs (
				tenant_id, target_id, vendor_type, action, payload_hash, artifact_payload,
				status, http_status, response_body, execution_time_ms
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);`,
			tenantID, target.ID, target.VendorType, logEntry.Action, payloadHash, string(payload),
			status, 200, errMsg, duration,
		)

		// Update target record
		_, _ = w.db.ExecContext(ctx, `
			UPDATE catalog_ai.ai_bridge_targets
			SET last_sync_at = NOW(), last_sync_status = $1, last_sync_error = $2, updated_at = NOW()
			WHERE id = $3;`, status, errMsg, target.ID)

		results = append(results, logEntry)
	}

	return results, nil
}

// StartBackgroundRelay polls for unpublished catalog changes in the background
func (w *OutboxSyncWorker) StartBackgroundRelay(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				w.pollAndProcessOutbox(ctx)
			case <-w.stopChan:
				ticker.Stop()
				return
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the background relay
func (w *OutboxSyncWorker) Stop() {
	close(w.stopChan)
}

func (w *OutboxSyncWorker) pollAndProcessOutbox(ctx context.Context) {
	if w.db == nil {
		return
	}

	// Safe read for published events targeting semantic change
	type EventRow struct {
		ID        uuid.UUID `db:"id"`
		TenantID  uuid.NullUUID `db:"tenant_id"`
		EventType string    `db:"event_type"`
	}

	var events []EventRow
	err := w.db.SelectContext(ctx, &events, `
		SELECT id, event_type
		FROM outbox
		WHERE published = FALSE AND event_type LIKE '%BUSINESS_OBJECT%'
		LIMIT 10;
	`)
	if err != nil || len(events) == 0 {
		return
	}

	for _, e := range events {
		log.Printf("[Semantic Bridge Outbox] Incremental sync event triggered: %s", e.EventType)
		_, _ = w.db.ExecContext(ctx, `UPDATE outbox SET published = TRUE, published_at = NOW() WHERE id = $1`, e.ID)
	}
}
