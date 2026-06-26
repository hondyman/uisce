package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// WorkloadAnalyzer analyzes query telemetry to build workload profiles.
type WorkloadAnalyzer struct {
	db *sqlx.DB
}

func NewWorkloadAnalyzer(db *sqlx.DB) *WorkloadAnalyzer {
	return &WorkloadAnalyzer{db: db}
}

// AnalyzeBO returns aggregated workload metrics for a specific BO.
func (a *WorkloadAnalyzer) AnalyzeBO(ctx context.Context, tenantID, boName string, window time.Duration) (*models.BOWorkloadProfile, error) {
	profile := &models.BOWorkloadProfile{
		TenantID: tenantID,
		BOName:   boName,
	}

	interval := fmt.Sprintf("%d seconds", int(window.Seconds()))

	// Summary stats
	var totalQ, slowQ int
	var avgDur, p95Dur, avgRows, p95Rows float64
	err := a.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total_queries,
			COUNT(*) FILTER (WHERE duration_ms > 1000) AS slow_queries,
			COALESCE(AVG(duration_ms), 0) AS avg_duration_ms,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms), 0) AS p95_duration_ms,
			COALESCE(AVG(rows_scanned), 0) AS avg_rows_scanned,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY rows_scanned), 0) AS p95_rows_scanned
		FROM semantic.query_telemetry
		WHERE tenant_id = $1
		  AND bo_name = $2
		  AND started_at >= now() - $3::interval
	`, tenantID, boName, interval).Scan(&totalQ, &slowQ, &avgDur, &p95Dur, &avgRows, &p95Rows)
	if err != nil {
		return nil, fmt.Errorf("summary stats: %w", err)
	}
	profile.TotalQueries = totalQ
	profile.SlowQueries = slowQ
	profile.AvgDurationMs = avgDur
	profile.P95DurationMs = p95Dur
	profile.AvgRowsScanned = avgRows
	profile.P95RowsScanned = p95Rows

	// Top group-bys
	profile.TopGroupBys, _ = a.getTopGroupBys(ctx, tenantID, boName, interval)

	// Top measures
	profile.TopMeasures, _ = a.getTopMeasures(ctx, tenantID, boName, interval)

	// Top filters
	profile.TopFilters, _ = a.getTopFilters(ctx, tenantID, boName, interval)

	return profile, nil
}

func (a *WorkloadAnalyzer) getTopGroupBys(ctx context.Context, tenantID, boName, interval string) ([]models.GroupByProfile, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT group_by_terms,
			   COUNT(*) AS query_count,
			   AVG(duration_ms) AS avg_duration_ms,
			   PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) AS p95_duration_ms,
			   AVG(rows_scanned) AS avg_rows_scanned
		FROM semantic.query_telemetry
		WHERE tenant_id = $1
		  AND bo_name = $2
		  AND started_at >= now() - $3::interval
		  AND group_by_terms IS NOT NULL
		GROUP BY group_by_terms
		ORDER BY query_count DESC
		LIMIT 10
	`, tenantID, boName, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.GroupByProfile
	for rows.Next() {
		var gbJSON []byte
		var gb models.GroupByProfile
		if err := rows.Scan(&gbJSON, &gb.QueryCount, &gb.AvgDurationMs, &gb.P95DurationMs, &gb.AvgRowsScanned); err != nil {
			continue
		}
		_ = json.Unmarshal(gbJSON, &gb.Terms)
		result = append(result, gb)
	}
	return result, nil
}

func (a *WorkloadAnalyzer) getTopMeasures(ctx context.Context, tenantID, boName, interval string) ([]models.MeasureProfile, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT jsonb_array_elements_text(measures) AS measure,
			   COUNT(*) AS query_count,
			   AVG(duration_ms) AS avg_duration_ms
		FROM semantic.query_telemetry
		WHERE tenant_id = $1
		  AND bo_name = $2
		  AND started_at >= now() - $3::interval
		  AND measures IS NOT NULL
		GROUP BY measure
		ORDER BY query_count DESC
		LIMIT 10
	`, tenantID, boName, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.MeasureProfile
	for rows.Next() {
		var m models.MeasureProfile
		if err := rows.Scan(&m.Name, &m.QueryCount, &m.AvgDurationMs); err != nil {
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

func (a *WorkloadAnalyzer) getTopFilters(ctx context.Context, tenantID, boName, interval string) ([]models.FilterProfile, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT (f->>'term') AS term,
			   (f->>'op') AS op,
			   COUNT(*) AS query_count,
			   AVG(duration_ms) AS avg_duration_ms
		FROM semantic.query_telemetry,
			 jsonb_array_elements(filters) AS f
		WHERE tenant_id = $1
		  AND bo_name = $2
		  AND started_at >= now() - $3::interval
		  AND filters IS NOT NULL
		GROUP BY term, op
		ORDER BY query_count DESC
		LIMIT 10
	`, tenantID, boName, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.FilterProfile
	for rows.Next() {
		var f models.FilterProfile
		if err := rows.Scan(&f.Term, &f.Operator, &f.QueryCount, &f.AvgDurationMs); err != nil {
			continue
		}
		result = append(result, f)
	}
	return result, nil
}

// AnalyzeAll returns workload profiles for all active BOs.
func (a *WorkloadAnalyzer) AnalyzeAll(ctx context.Context, window time.Duration) ([]models.BOWorkloadProfile, error) {
	interval := fmt.Sprintf("%d seconds", int(window.Seconds()))

	rows, err := a.db.QueryContext(ctx, `
		SELECT DISTINCT tenant_id, bo_name
		FROM semantic.query_telemetry
		WHERE started_at >= now() - $1::interval
	`, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []models.BOWorkloadProfile
	for rows.Next() {
		var tenantID, boName string
		if err := rows.Scan(&tenantID, &boName); err != nil {
			continue
		}
		p, err := a.AnalyzeBO(ctx, tenantID, boName, window)
		if err != nil {
			continue
		}
		profiles = append(profiles, *p)
	}
	return profiles, nil
}
