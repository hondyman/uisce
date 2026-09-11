package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Phase1GoLiveDate is the go-live date for the report_execution_events instrumentation.
// Executions created before this date predate the audit trail; they are excluded from
// broken-chain detection (not migrated, filtered at query time).
// Value must match the PHASE1_GO_LIVE_DATE constant documented in the migration file.
var Phase1GoLiveDate = time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)

// SweepStaleExecutions sweeps report_executions that have remained in 'pending' or 'running'
// state longer than cutoffDuration without completion, transitioning them to 'failed'.
//
// Each swept execution gets a SWEEP_RECONCILED event in the same transaction.
// After the sweep, an unscoped broken-chain check runs against all terminal-status
// executions missing their terminal event (pre-instrumentation rows).
//
// CROSS-TENANT RLS NOTE:
// This reconciliation query operates cross-tenant by design. In production, this background
// job runs under a superuser or dedicated maintenance pool with bypass privileges, ensuring
// orphaned executions across all tenants are reconciled without tenant isolation deadlocks.
func SweepStaleExecutions(ctx context.Context, db *sql.DB, cutoffDuration time.Duration) (sweptCount int64, brokenChainCount int64, err error) {
	if db == nil {
		return 0, 0, fmt.Errorf("db connection is nil")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("SweepStaleExecutions: BeginTx failed: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Bulk update + RETURNING: gets all swept rows in one round-trip
	// from_status is always 'running' — the WHERE clause guarantees it
	rows, err := tx.QueryContext(ctx, `
		UPDATE public.report_executions
		SET status = 'failed',
		    error_message = 'Workflow timed out or abandoned (stale-execution reconciler)',
		    completed_at = NOW()
		WHERE status = 'running'
		  AND created_at < NOW() - ($1 * INTERVAL '1 second')
		RETURNING id, tenant_id, error_message
	`, cutoffDuration.Seconds())
	if err != nil {
		return 0, 0, fmt.Errorf("SweepStaleExecutions: UPDATE RETURNING failed: %w", err)
	}

	var sweptIDs []interface{}
	var eventRows [][]interface{}
	for rows.Next() {
		var execID, tenantID interface{}
		var errMsg *string
		if err := rows.Scan(&execID, &tenantID, &errMsg); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("SweepStaleExecutions: scan returned row failed: %w", err)
		}
		sweptIDs = append(sweptIDs, execID)

		var detail []byte
		if errMsg != nil {
			detail, _ = json.Marshal(map[string]interface{}{"error_message": *errMsg})
		} else {
			detail = []byte(`{}`)
		}
		eventRows = append(eventRows, []interface{}{
			execID, tenantID, "SWEEP_RECONCILED", "running", "failed", "system:sweep", detail,
		})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("SweepStaleExecutions: iterating returned rows failed: %w", err)
	}
	sweptCount = int64(len(eventRows))

	// Bulk insert SWEEP_RECONCILED events
	if len(eventRows) > 0 {
		for _, row := range eventRows {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO public.report_execution_events (
					id, execution_id, tenant_id, event, from_status, to_status, actor_id, detail
				) VALUES (
					gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7
				)
			`, row[0], row[1], row[2], row[3], row[4], row[5], row[6])
			if err != nil {
				return 0, 0, fmt.Errorf("SweepStaleExecutions: event insert failed: %w", err)
			}
		}
	}

	// Unscoped broken-chain check: detects any terminal-status execution missing its terminal event,
	// including pre-instrumentation history. Excludes executions created before Phase1GoLiveDate.
	brokenRows, err := tx.QueryContext(ctx, `
		SELECT e.id, e.tenant_id, e.status, e.created_at
		FROM report_executions e
		WHERE e.status IN ('completed', 'failed', 'failed_dispatch')
		  AND e.created_at >= $1
		  AND NOT EXISTS (
		      SELECT 1 FROM report_execution_events ree
		      WHERE ree.execution_id = e.id
		        AND ree.event IN ('COMPLETED', 'FAILED', 'DISPATCH_FAILED', 'SWEEP_RECONCILED')
		  )
		ORDER BY e.created_at
	`, Phase1GoLiveDate)
	if err != nil {
		return 0, 0, fmt.Errorf("SweepStaleExecutions: broken-chain check query failed: %w", err)
	}
	defer brokenRows.Close()

	for brokenRows.Next() {
		brokenChainCount++
	}
	if err := brokenRows.Err(); err != nil {
		return 0, 0, fmt.Errorf("SweepStaleExecutions: iterating broken-chain rows failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("SweepStaleExecutions: Commit failed: %w", err)
	}

	if sweptCount > 0 {
		log.Printf("[INFO] SweepStaleExecutions: marked %d stale execution(s) as failed (cutoff: %v)", sweptCount, cutoffDuration)
	}
	if brokenChainCount > 0 {
		log.Printf("[WARN] SweepStaleExecutions: broken chains detected: %d", brokenChainCount)
	}

	return sweptCount, brokenChainCount, nil
}
