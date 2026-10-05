package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestHotTierPruner_WatermarkGatingAndPruneEvaluation(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	dsn := fmt.Sprintf("postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=verify-full&sslrootcert=%s/.uisce/certs/ca.crt&sslcert=%s/.uisce/certs/postgres-client.crt&sslkey=%s/.uisce/certs/postgres-client.key", homeDir, homeDir, homeDir)
	pgDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Postgres alpha not reachable: %v", err)
		return
	}
	defer pgDB.Close()

	if err := pgDB.Ping(); err != nil {
		t.Skipf("Postgres alpha ping failed: %v", err)
		return
	}

	pruner := NewHotTierPruner(pgDB, 30, 5) // 30 days retention + 5 days safety buffer

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Dry run evaluation
	report, err := pruner.PruneEligiblePartitions(ctx, true)
	require.NoError(t, err)
	require.Greater(t, report.PartitionsEvaluated, 0, "Must discover active partition set")

	t.Logf("Hot Tier Prune Dry-Run Report: Evaluated=%d partitions, Eligible=%d, SkippedUncertified=%d in %dms",
		report.PartitionsEvaluated, report.PartitionsPruned, report.SkippedUncertified, report.DurationMs)

	// 2. Test safety gating on a historical test partition
	testPartName := "compliance_evaluation_event_y2024m01"
	createTestPartSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS compliance.%s 
		PARTITION OF compliance.compliance_evaluation_event
		FOR VALUES FROM ('2024-01-01 00:00:00+00') TO ('2024-02-01 00:00:00+00');
	`, testPartName)

	_, err = pgDB.ExecContext(ctx, createTestPartSQL)
	require.NoError(t, err)

	// Run live prune execution (not dry run)
	liveReport, err := pruner.PruneEligiblePartitions(ctx, false)
	require.NoError(t, err)
	require.GreaterOrEqual(t, liveReport.PartitionsPruned, 1, "Must prune the historical 2024 partition")
	require.Contains(t, liveReport.PrunedPartitionNames, testPartName)

	t.Logf("Live Prune PASSED: Successfully detached & dropped historical partition %s!", testPartName)
}
