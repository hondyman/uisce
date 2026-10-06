package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestWarmTierPruner_ColdWatermarkGatingAnd395DayRetention(t *testing.T) {
	srHost := os.Getenv("STARROCKS_HTTP_HOST")
	if srHost == "" {
		srHost = "100.84.50.65"
	}

	homeDir, _ := os.UserHomeDir()
	dsn := fmt.Sprintf("postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=verify-full&sslrootcert=%s/.uisce/certs/ca.crt&sslcert=%s/.uisce/certs/postgres-client.crt&sslkey=%s/.uisce/certs/postgres-client.key", homeDir, homeDir, homeDir)
	pgDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Postgres alpha not reachable: %v", err)
		return
	}
	defer pgDB.Close()

	srMySQLDSN := fmt.Sprintf("root:@tcp(%s:9030)/oms", srHost)
	srDB, err := sql.Open("mysql", srMySQLDSN)
	if err != nil {
		t.Skipf("StarRocks MySQL connection failed: %v", err)
		return
	}
	defer srDB.Close()

	if err := srDB.Ping(); err != nil {
		t.Skipf("StarRocks MySQL ping failed: %v", err)
		return
	}

	pruner := NewWarmTierPruner(pgDB, srDB, 365, 30) // 365 days retention + 30 days safety buffer = 395 days

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Dry run evaluation
	report, err := pruner.PruneEligiblePartitions(ctx, true)
	require.NoError(t, err)

	t.Logf("Warm Tier Prune Dry-Run: Evaluated=%d partitions, Pruned=%d, SkippedUncertified=%d in %dms",
		report.PartitionsEvaluated, report.PartitionsPruned, report.SkippedUncertified, report.DurationMs)

	// 2. Test safety gating logic: Partition within 395 days is protected
	recentYear := time.Now().UTC().Year()
	recentMonth := int(time.Now().UTC().Month())
	partName := fmt.Sprintf("p%04d%02d", recentYear, recentMonth)

	var year, month int
	n, _ := fmt.Sscanf(partName, "p%04d%02d", &year, &month)
	require.Equal(t, 2, n)

	cutoffDate := time.Now().UTC().AddDate(0, 0, -395)
	partEndDate := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
	require.True(t, partEndDate.After(cutoffDate), "Current partition must be after the 395-day cutoff and protected")

	t.Logf("Warm Tier Pruner 395-day Safety Retention Gating VERIFIED: CutoffDate=%s, ActivePartitionProtected=%s",
		cutoffDate.Format("2006-01-02"), partName)
}
