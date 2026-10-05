package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PruneReport encapsulates the results of a Hot Tier partition pruning execution
type PruneReport struct {
	PartitionsEvaluated int      `json:"partitions_evaluated"`
	PartitionsPruned    int      `json:"partitions_pruned"`
	SkippedUncertified  int      `json:"skipped_uncertified"`
	PrunedPartitionNames []string `json:"pruned_partition_names"`
	DurationMs          int64    `json:"duration_ms"`
}

// HotTierPruner manages the automated, watermark-gated pruning of historical Hot PG partitions
type HotTierPruner struct {
	db                *sql.DB
	retentionDays     int
	safetyBufferDays  int
}

// NewHotTierPruner creates a new HotTierPruner instance
func NewHotTierPruner(db *sql.DB, retentionDays, safetyBufferDays int) *HotTierPruner {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	if safetyBufferDays <= 0 {
		safetyBufferDays = 5
	}
	return &HotTierPruner{
		db:               db,
		retentionDays:    retentionDays,
		safetyBufferDays: safetyBufferDays,
	}
}

// GetGlobalCertifiedWarmLWM finds the lowest certified WARM watermark across all tenants
func (p *HotTierPruner) GetGlobalCertifiedWarmLWM(ctx context.Context) (int64, error) {
	var minLSN sql.NullInt64
	query := `
		SELECT MIN((current_lsn - '0/0'::pg_lsn)::bigint)
		FROM compliance.compliance_watermark_checkpoint
		WHERE tier = 'WARM'
	`
	err := p.db.QueryRowContext(ctx, query).Scan(&minLSN)
	if err != nil {
		return 0, fmt.Errorf("query global warm lwm: %w", err)
	}
	if !minLSN.Valid {
		return 0, nil
	}
	return minLSN.Int64, nil
}

// PruneEligiblePartitions detaches and drops partitions older than (retention + safety buffer) that are certified in Warm tier
func (p *HotTierPruner) PruneEligiblePartitions(ctx context.Context, dryRun bool) (*PruneReport, error) {
	start := time.Now()
	totalCutoffDays := p.retentionDays + p.safetyBufferDays
	cutoffDate := time.Now().UTC().AddDate(0, 0, -totalCutoffDays)

	globalWarmLWM, err := p.GetGlobalCertifiedWarmLWM(ctx)
	if err != nil {
		return nil, err
	}

	report := &PruneReport{}

	// List child partition tables of compliance_evaluation_event
	query := `
		SELECT c.relname
		FROM pg_inherits i
		JOIN pg_class c ON c.oid = i.inhrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'compliance' 
		  AND i.inhparent = 'compliance.compliance_evaluation_event'::regclass
		  AND c.relname != 'compliance_evaluation_event_default'
		ORDER BY c.relname ASC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query evaluation partitions: %w", err)
	}
	defer rows.Close()

	var partitionNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		partitionNames = append(partitionNames, name)
	}

	report.PartitionsEvaluated = len(partitionNames)

	for _, partName := range partitionNames {
		// Parse partition year and month from name e.g. compliance_evaluation_event_y2026m10
		var year, month int
		n, _ := fmt.Sscanf(partName, "compliance_evaluation_event_y%04dm%02d", &year, &month)
		if n != 2 {
			continue
		}

		partitionEndDate := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
		if partitionEndDate.After(cutoffDate) {
			// Partition is still within active retention window (not eligible for prune)
			continue
		}

		// Check max LSN inside this partition
		var maxPartLSN sql.NullInt64
		lsnQuery := fmt.Sprintf("SELECT MAX(ingest_lsn) FROM compliance.%s", partName)
		_ = p.db.QueryRowContext(ctx, lsnQuery).Scan(&maxPartLSN)

		if maxPartLSN.Valid && maxPartLSN.Int64 > globalWarmLWM {
			// Partition contains data not yet certified in Warm StarRocks -> SKIP
			report.SkippedUncertified++
			continue
		}

		// Partition is eligible for safe prune
		if !dryRun {
			detachSQL := fmt.Sprintf("ALTER TABLE compliance.compliance_evaluation_event DETACH PARTITION compliance.%s", partName)
			if _, err := p.db.ExecContext(ctx, detachSQL); err != nil {
				return nil, fmt.Errorf("detach partition %s: %w", partName, err)
			}

			dropSQL := fmt.Sprintf("DROP TABLE compliance.%s", partName)
			if _, err := p.db.ExecContext(ctx, dropSQL); err != nil {
				return nil, fmt.Errorf("drop partition %s: %w", partName, err)
			}
		}

		report.PartitionsPruned++
		report.PrunedPartitionNames = append(report.PrunedPartitionNames, partName)
	}

	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}
