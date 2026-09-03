package semantic_bridge

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

// OutboxSyncWorker listens for ON_PUBLISH events and synchronizes active targets
type OutboxSyncWorker struct {
	db            *sqlx.DB
	cortexExp     *CortexExporter
	databricksExp *DatabricksExporter
	vault         *CredentialVault
	ledger        *Ledger
	databricksPsh *DatabricksPusher
	snowflakePsh  *SnowflakePusher
	stopChan      chan struct{}
}

// NewOutboxSyncWorker wires the worker to a real credential vault and
// tamper-evident ledger. Returns an error if the shared server key
// (API_TOKEN_ENCRYPTION_KEY) isn't configured — the worker refuses to run
// rather than push targets it can't authenticate or log honestly.
func NewOutboxSyncWorker(db *sqlx.DB) (*OutboxSyncWorker, error) {
	vault, err := NewCredentialVault()
	if err != nil {
		return nil, err
	}
	hmacKey, err := security.LoadKeyFromEnv(CredentialVaultKeyEnv, CredentialVaultDevFallbackEnv)
	if err != nil {
		return nil, err
	}
	return &OutboxSyncWorker{
		db:            db,
		cortexExp:     NewCortexExporter(db),
		databricksExp: NewDatabricksExporter(db),
		vault:         vault,
		ledger:        NewLedger(db, hmacKey),
		databricksPsh: NewDatabricksPusher(),
		snowflakePsh:  NewSnowflakePusher(),
		stopChan:      make(chan struct{}),
	}, nil
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
		SELECT id, tenant_id, vendor_type, target_name, is_active, config_payload, credentials_vaulted,
		       sync_frequency, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at
		FROM catalog_ai.ai_bridge_targets
		WHERE tenant_id = $1 AND is_active = TRUE AND sync_frequency IN ('ON_PUBLISH', 'ALWAYS');`

	if err := w.db.SelectContext(ctx, &targets, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed fetching on_publish targets: %w", err)
	}

	results := make([]SyncLog, 0, len(targets))

	for _, target := range targets {
		start := time.Now()
		status, httpStatus, responseBody, payload := w.pushTarget(ctx, tenantID, &target)
		duration := int(time.Since(start).Milliseconds())

		logID, err := w.ledger.Append(ctx, tenantID, &target.ID, string(target.VendorType),
			"ON_PUBLISH_AUTO_SYNC: "+eventReason, payload, status, httpStatus, responseBody, duration)
		if err != nil {
			log.Printf("[Semantic Bridge Outbox] failed writing audit ledger for target %s: %v", target.ID, err)
			continue
		}

		_, _ = w.db.ExecContext(ctx, `
			UPDATE catalog_ai.ai_bridge_targets
			SET last_sync_at = NOW(), last_sync_status = $1, last_sync_error = $2, updated_at = NOW()
			WHERE id = $3;`, status, responseBody, target.ID)

		results = append(results, SyncLog{
			ID:              logID,
			TenantID:        tenantID,
			TargetID:        &target.ID,
			VendorType:      string(target.VendorType),
			Action:          "ON_PUBLISH_AUTO_SYNC: " + eventReason,
			Status:          status,
			HTTPStatus:      httpStatus,
			ResponseBody:    responseBody,
			ExecutionTimeMS: duration,
			CreatedAt:       time.Now(),
		})
	}

	return results, nil
}

func (w *OutboxSyncWorker) pushTarget(ctx context.Context, tenantID uuid.UUID, target *BridgeTarget) (status string, httpStatus int, responseBody string, payload []byte) {
	creds := w.vault.Open(target.CredentialsVaulted)

	switch target.VendorType {
	case VendorSnowflakeCortex:
		var err error
		payload, err = w.cortexExp.CompileFullCortexModel(ctx, tenantID)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		ddl, err := w.cortexExp.GenerateSnowflakeGovernanceDDL(ctx, tenantID)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		statements := make([]string, 0, len(ddl))
		for _, d := range ddl {
			statements = append(statements, d.SQL)
		}
		if len(statements) == 0 {
			return "SUCCESS", 0, "compiled model only — no governance-tagged columns to push", payload
		}
		res, err := w.snowflakePsh.Push(ctx, target.ConfigPayload, creds, statements)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		if !res.Success {
			if res.HTTPStatus == 0 {
				return "NOT_CONFIGURED", 0, res.ResponseBody, payload
			}
			return "ERROR", res.HTTPStatus, res.ResponseBody, payload
		}
		return "SUCCESS", res.HTTPStatus, res.ResponseBody, payload

	case VendorDatabricksGenie:
		var err error
		payload, err = w.databricksExp.CompileGenieModel(ctx, tenantID)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		ddlScript, err := w.databricksExp.GenerateUnityCatalogSQL(ctx, tenantID)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		res, err := w.databricksPsh.Push(ctx, target.ConfigPayload, creds, ddlScript)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		if !res.Success {
			if res.HTTPStatus == 0 {
				return "NOT_CONFIGURED", 0, res.ResponseBody, payload
			}
			return "ERROR", res.HTTPStatus, res.ResponseBody, payload
		}
		return "SUCCESS", res.HTTPStatus, res.ResponseBody, payload

	default:
		return "UNSUPPORTED_VENDOR", 0, "no push implementation for vendor " + string(target.VendorType), nil
	}
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

	// NOTE: this generic `outbox` table (as opposed to
	// catalog_ai.ai_bridge_sync_logs, which this package owns) is referenced
	// by internal/events/outbox.go elsewhere in the codebase but has no
	// tenant_id column confirmed in any migration here. Rule 7 requires a
	// real tenant_id to call SyncTenantOnPublish safely, so until that
	// column is confirmed/added this poller only drains the queue — it does
	// NOT call SyncTenantOnPublish. Use TriggerSync (manual) or call
	// SyncTenantOnPublish directly from a caller that already has a
	// tenant-scoped BUSINESS_OBJECT publish event for real auto-sync today.
	type EventRow struct {
		ID        uuid.UUID `db:"id"`
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
		log.Printf("[Semantic Bridge Outbox] BUSINESS_OBJECT publish event seen (%s) but not auto-synced: outbox table has no confirmed tenant_id column. See NOTE above.", e.EventType)
		_, _ = w.db.ExecContext(ctx, `UPDATE outbox SET published = TRUE, published_at = NOW() WHERE id = $1`, e.ID)
	}
}
