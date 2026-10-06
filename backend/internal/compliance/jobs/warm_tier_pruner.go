package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// WarmTierPruneReport summarizes the StarRocks warm tier partition prune run
type WarmTierPruneReport struct {
	PartitionsEvaluated  int      `json:"partitions_evaluated"`
	PartitionsPruned     int      `json:"partitions_pruned"`
	SkippedUncertified   int      `json:"skipped_uncertified"`
	PrunedPartitionNames []string `json:"pruned_partition_names"`
	DurationMs           int64    `json:"duration_ms"`
}

// WarmTierPruner manages the automated, COLD-watermark-gated pruning of historical StarRocks Warm tier partitions
type WarmTierPruner struct {
	postgresDB       *sql.DB
	starRocksDB      *sql.DB
	retentionDays    int
	safetyBufferDays int
}

// NewWarmTierPruner creates a new WarmTierPruner instance
func NewWarmTierPruner(postgresDB, starRocksDB *sql.DB, retentionDays, safetyBufferDays int) *WarmTierPruner {
	if retentionDays <= 0 {
		retentionDays = 365 // 1 year retention
	}
	if safetyBufferDays <= 0 {
		safetyBufferDays = 30 // 30-day safety buffer (total 395 days)
	}
	return &WarmTierPruner{
		postgresDB:       postgresDB,
		starRocksDB:      starRocksDB,
		retentionDays:    retentionDays,
		safetyBufferDays: safetyBufferDays,
	}
}

// GetGlobalCertifiedColdLWM retrieves the minimum certified COLD watermark across all tenants
func (p *WarmTierPruner) GetGlobalCertifiedColdLWM(ctx context.Context) (int64, error) {
	var minLSN sql.NullInt64
	query := `
		SELECT MIN((current_lsn - '0/0'::pg_lsn)::bigint)
		FROM compliance.compliance_watermark_checkpoint
		WHERE tier = 'COLD'
	`
	err := p.postgresDB.QueryRowContext(ctx, query).Scan(&minLSN)
	if err != nil {
		return 0, fmt.Errorf("query global cold lwm: %w", err)
	}
	if !minLSN.Valid {
		return 0, nil
	}
	return minLSN.Int64, nil
}

// PruneEligiblePartitions checks partition ages against the 395-day threshold and drops partitions certified in Cold tier
func (p *WarmTierPruner) PruneEligiblePartitions(ctx context.Context, dryRun bool) (*WarmTierPruneReport, error) {
	start := time.Now()
	totalCutoffDays := p.retentionDays + p.safetyBufferDays
	cutoffDate := time.Now().UTC().AddDate(0, 0, -totalCutoffDays)

	globalColdLWM, err := p.GetGlobalCertifiedColdLWM(ctx)
	if err != nil {
		return nil, err
	}

	report := &WarmTierPruneReport{}

	// Query StarRocks partitions for table oms.compliance_evaluations
	query := "SHOW PARTITIONS FROM oms.compliance_evaluations"
	rows, err := p.starRocksDB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("show starrocks partitions: %w", err)
	}
	defer rows.Close()

	type partMeta struct {
		name string
	}
	var partitions []partMeta

	cols, err := rows.Columns()
	if err == nil && len(cols) > 0 {
		for rows.Next() {
			rowVals := make([]interface{}, len(cols))
			rowValPtrs := make([]interface{}, len(cols))
			for i := range rowVals {
				rowValPtrs[i] = &rowVals[i]
			}
			if err := rows.Scan(rowValPtrs...); err == nil {
				if len(rowVals) > 1 {
					var name string
					switch v := rowVals[1].(type) {
					case []byte:
						name = string(v)
					case string:
						name = v
					}
					if name != "" {
						partitions = append(partitions, partMeta{name: name})
					}
				}
			}
		}
	}

	report.PartitionsEvaluated = len(partitions)

	for _, part := range partitions {
		// Example StarRocks monthly partition name format: p202501 or similar
		var year, month int
		n, _ := fmt.Sscanf(part.name, "p%04d%02d", &year, &month)
		if n != 2 {
			continue
		}

		partEndDate := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
		if partEndDate.After(cutoffDate) {
			// Still within warm tier retention window (not eligible)
			continue
		}

		// Check max ingest_lsn inside StarRocks partition
		var maxLSN sql.NullInt64
		lsnQ := fmt.Sprintf("SELECT MAX(ingest_lsn) FROM oms.compliance_evaluations PARTITION (%s)", part.name)
		_ = p.starRocksDB.QueryRowContext(ctx, lsnQ).Scan(&maxLSN)

		if maxLSN.Valid && maxLSN.Int64 > globalColdLWM {
			// Partition contains records not yet certified into WORM Cold tier -> SKIP
			report.SkippedUncertified++
			continue
		}

		if !dryRun {
			dropSQL := fmt.Sprintf("ALTER TABLE oms.compliance_evaluations DROP PARTITION %s", part.name)
			if _, err := p.starRocksDB.ExecContext(ctx, dropSQL); err != nil {
				return nil, fmt.Errorf("drop starrocks partition %s: %w", part.name, err)
			}
		}

		report.PartitionsPruned++
		report.PrunedPartitionNames = append(report.PrunedPartitionNames, part.name)
	}

	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}
