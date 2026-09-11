package reports

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// SweepStaleExecutions sweeps report_executions that have remained in 'pending' or 'running'
// state longer than cutoffDuration without completion, transitioning them to 'failed'.
//
// CROSS-TENANT RLS NOTE:
// This reconciliation query operates cross-tenant by design. In production, this background
// job runs under a superuser or dedicated maintenance pool with bypass privileges, ensuring
// orphaned executions across all tenants are reconciled without tenant isolation deadlocks.
func SweepStaleExecutions(ctx context.Context, db *sql.DB, cutoffDuration time.Duration) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db connection is nil")
	}

	query := `
		UPDATE public.report_executions
		SET status = 'failed',
		    error_message = 'Workflow timed out or abandoned (stale-execution reconciler)',
		    completed_at = NOW()
		WHERE status IN ('pending', 'running')
		  AND created_at < NOW() - ($1 * INTERVAL '1 second')
	`

	res, err := db.ExecContext(ctx, query, cutoffDuration.Seconds())
	if err != nil {
		return 0, fmt.Errorf("reconciliation sweep failed: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		log.Printf("[INFO] SweepStaleExecutions: marked %d stale execution(s) as failed (cutoff: %v)", rowsAffected, cutoffDuration)
	}

	return rowsAffected, nil
}
