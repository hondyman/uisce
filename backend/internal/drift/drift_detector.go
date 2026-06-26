package drift

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// DriftReport represents a drift analysis report
type DriftReport struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	ModelID           string
	ReportTime        time.Time
	DriftSeverity     string // low, medium, high, critical
	DriftIssues       []DriftIssue
	SuggestedActions  []string
	ResolvedByUserID  *uuid.UUID
	ResolutionComment string
	Status            string // open, investigating, resolved
}

// DriftIssue represents a specific drift problem
type DriftIssue struct {
	ID              uuid.UUID
	ReportID        uuid.UUID
	IssueType       string // schema_drift, logic_drift, freshness_drift, lineage_drift, performance_drift
	Element         string // measure name, dimension name, join name, table name
	Severity        string // low, medium, high, critical
	Description     string
	DetectionMethod string // query comparison, schema inspection, runtime observation
	LastDetectedAt  time.Time
	ProposedFix     string
	DataImpact      *DataImpact
}

// DataImpact measures impact of drift
type DataImpact struct {
	AffectedRows      int64
	AffectedQueries   int
	EstimatedUsers    int
	PerformanceChange float64 // percentage change in query time
}

// DriftDetector analyzes models for drift
type DriftDetector struct {
	db *sqlx.DB
}

// NewDriftDetector creates new detector
func NewDriftDetector(db *sqlx.DB) *DriftDetector {
	return &DriftDetector{db: db}
}

// DetectSchemaDrift checks for schema changes in source tables
func (dd *DriftDetector) DetectSchemaDrift(ctx context.Context, tenantID, modelID string) ([]DriftIssue, error) {
	var issues []DriftIssue

	// Example: detect if columns referenced in measures still exist
	// This is a simplified version - in production, parse model definition properly
	query := `
		WITH model_refs AS (
			SELECT DISTINCT 'customer_id' as column_name  -- Example column refs from model
			UNION ALL
			SELECT 'order_date'
		)
		SELECT mr.column_name 
		FROM model_refs mr
		WHERE NOT EXISTS (
			SELECT 1 FROM information_schema.columns c
			WHERE c.table_schema = 'public' 
			AND c.table_name = $2  -- source table name
			AND c.column_name = mr.column_name
		)
	`

	rows, err := dd.db.QueryContext(ctx, query, tenantID, modelID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("⚠️  Schema drift check failed: %v", err)
		return issues, nil
	}
	defer rows.Close()

	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			continue
		}

		issues = append(issues, DriftIssue{
			ID:              uuid.New(),
			IssueType:       "schema_drift",
			Element:         col,
			Severity:        "high",
			Description:     fmt.Sprintf("Column '%s' referenced in model but no longer exists in source table", col),
			DetectionMethod: "schema_inspection",
			LastDetectedAt:  time.Now(),
			ProposedFix:     fmt.Sprintf("Update model definition to use alternative column or restore '%s' to schema", col),
		})
	}

	return issues, nil
}

// DetectPerformanceDrift compares query performance over time
func (dd *DriftDetector) DetectPerformanceDrift(ctx context.Context, tenantID, modelID string, thresholdMS int64) ([]DriftIssue, error) {
	var issues []DriftIssue

	// Get baseline (first 10 queries)
	var baselineDuration sql.NullFloat64
	err := dd.db.GetContext(ctx, &baselineDuration, `
		SELECT AVG(duration_ms) FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2 AND status = 'success'
		AND created_at > now() - interval '30 days'
		ORDER BY created_at ASC
		LIMIT 10
	`, tenantID, modelID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("⚠️  Baseline query failed: %v", err)
		return issues, nil
	}

	if !baselineDuration.Valid || baselineDuration.Float64 == 0 {
		log.Printf("ℹ️  No baseline data for model %s", modelID)
		return issues, nil
	}

	// Get recent performance (last 10 queries)
	var recentDuration sql.NullFloat64
	err = dd.db.GetContext(ctx, &recentDuration, `
		SELECT AVG(duration_ms) FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2 AND status = 'success'
		ORDER BY created_at DESC
		LIMIT 10
	`, tenantID, modelID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("⚠️  Recent query failed: %v", err)
		return issues, nil
	}

	if !recentDuration.Valid {
		log.Printf("ℹ️  No recent data for model %s", modelID)
		return issues, nil
	}

	percentChange := (recentDuration.Float64 - baselineDuration.Float64) / baselineDuration.Float64 * 100

	if percentChange > 50.0 { // 50% slowdown
		issues = append(issues, DriftIssue{
			ID:              uuid.New(),
			IssueType:       "performance_drift",
			Element:         modelID,
			Severity:        "high",
			Description:     fmt.Sprintf("Query performance degraded by %.1f%% (baseline: %.0fms, recent: %.0fms)", percentChange, baselineDuration.Float64, recentDuration.Float64),
			DetectionMethod: "runtime_observation",
			LastDetectedAt:  time.Now(),
			ProposedFix:     "Check for missing indexes, query plan changes, or increased data volume. Review EXPLAIN ANALYZE output.",
			DataImpact: &DataImpact{
				PerformanceChange: percentChange,
			},
		})
		log.Printf("⚠️  Performance drift detected: %.1f%% slowdown", percentChange)
	}

	return issues, nil
}

// DetectFreshnessDrift checks if data is stale
func (dd *DriftDetector) DetectFreshnessDrift(ctx context.Context, tenantID, modelID string, maxAgeHours int) ([]DriftIssue, error) {
	var issues []DriftIssue

	var lastUpdateTime sql.NullTime
	err := dd.db.GetContext(ctx, &lastUpdateTime, `
		SELECT MAX(created_at) FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2 AND status = 'success'
	`, tenantID, modelID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("⚠️  Freshness check failed: %v", err)
		return issues, nil
	}

	if !lastUpdateTime.Valid {
		log.Printf("ℹ️  No query history for model %s", modelID)
		return issues, nil
	}

	ageHours := time.Since(lastUpdateTime.Time).Hours()
	if ageHours > float64(maxAgeHours) {
		issues = append(issues, DriftIssue{
			ID:              uuid.New(),
			IssueType:       "freshness_drift",
			Element:         modelID,
			Severity:        "medium",
			Description:     fmt.Sprintf("Model data is stale - last query was %.1f hours ago (max age: %d hours)", ageHours, maxAgeHours),
			DetectionMethod: "timestamp_inspection",
			LastDetectedAt:  time.Now(),
			ProposedFix:     "Trigger a refresh of the model or check if source data pipeline is running",
		})
		log.Printf("⚠️  Freshness drift detected: %.1f hours old", ageHours)
	}

	return issues, nil
}

// GenerateDriftReport creates comprehensive drift report
func (dd *DriftDetector) GenerateDriftReport(ctx context.Context, tenantID, modelID string) (*DriftReport, error) {
	report := &DriftReport{
		ID:         uuid.New(),
		TenantID:   uuid.MustParse(tenantID),
		ModelID:    modelID,
		ReportTime: time.Now(),
		Status:     "open",
	}

	// Run all drift detections
	schemaDrift, _ := dd.DetectSchemaDrift(ctx, tenantID, modelID)
	perfDrift, _ := dd.DetectPerformanceDrift(ctx, tenantID, modelID, 1000)
	freshnessDrift, _ := dd.DetectFreshnessDrift(ctx, tenantID, modelID, 24)

	report.DriftIssues = append(report.DriftIssues, schemaDrift...)
	report.DriftIssues = append(report.DriftIssues, perfDrift...)
	report.DriftIssues = append(report.DriftIssues, freshnessDrift...)

	// Calculate overall severity
	maxSeverity := "low"
	for _, issue := range report.DriftIssues {
		if issue.Severity == "critical" {
			maxSeverity = "critical"
			break
		} else if issue.Severity == "high" && maxSeverity != "critical" {
			maxSeverity = "high"
		} else if issue.Severity == "medium" && maxSeverity == "low" {
			maxSeverity = "medium"
		}
	}
	report.DriftSeverity = maxSeverity

	// Add suggested actions
	if len(report.DriftIssues) > 0 {
		report.SuggestedActions = []string{
			fmt.Sprintf("Review %d drift issues (severity: %s)", len(report.DriftIssues), maxSeverity),
			"Update model definition to address identified drift",
			"Re-run query compilation to generate new baseline",
			"Monitor query performance after changes",
		}
	}

	log.Printf("✅ Drift report generated for model %s: %d issues (severity: %s)", modelID, len(report.DriftIssues), maxSeverity)

	return report, nil
}

// SaveDriftReport persists report to database
func (dd *DriftDetector) SaveDriftReport(ctx context.Context, report *DriftReport) error {
	query := `
		INSERT INTO semantic_drift_reports (
			id, tenant_id, model_id, report_time, drift_severity, issue_count, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, now())
	`

	err := dd.db.QueryRowContext(
		ctx, query,
		report.ID, report.TenantID, report.ModelID, report.ReportTime,
		report.DriftSeverity, len(report.DriftIssues), report.Status,
	).Err()

	if err != nil {
		log.Printf("❌ Failed to save drift report: %v", err)
		return err
	}

	// Save individual issues
	for _, issue := range report.DriftIssues {
		issue.ReportID = report.ID
		if err := dd.saveDriftIssue(ctx, &issue); err != nil {
			log.Printf("⚠️  Failed to save drift issue: %v", err)
		}
	}

	log.Printf("✅ Drift report saved with %d issues", len(report.DriftIssues))
	return nil
}

func (dd *DriftDetector) saveDriftIssue(ctx context.Context, issue *DriftIssue) error {
	query := `
		INSERT INTO semantic_drift_issues (
			id, report_id, issue_type, element, severity, description,
			detection_method, last_detected_at, proposed_fix, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
	`

	issue.ID = uuid.New()

	return dd.db.QueryRowContext(
		ctx, query,
		issue.ID, issue.ReportID, issue.IssueType, issue.Element, issue.Severity,
		issue.Description, issue.DetectionMethod, issue.LastDetectedAt, issue.ProposedFix,
	).Err()
}

// GetLatestDriftReport retrieves the most recent drift report for a model
func (dd *DriftDetector) GetLatestDriftReport(ctx context.Context, tenantID, modelID string) (*DriftReport, error) {
	var report DriftReport
	query := `
		SELECT id, tenant_id, model_id, report_time, drift_severity, status, created_at
		FROM semantic_drift_reports
		WHERE tenant_id = $1 AND model_id = $2
		ORDER BY report_time DESC
		LIMIT 1
	`

	err := dd.db.GetContext(ctx, &report, query, tenantID, modelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Load associated issues
	issueQuery := `
		SELECT id, report_id, issue_type, element, severity, description,
		       detection_method, last_detected_at, proposed_fix
		FROM semantic_drift_issues
		WHERE report_id = $1
	`

	err = dd.db.SelectContext(ctx, &report.DriftIssues, issueQuery, report.ID)
	if err != nil {
		log.Printf("⚠️  Failed to load drift issues: %v", err)
	}

	return &report, nil
}

// ScheduleDriftDetection runs drift detection periodically
func (dd *DriftDetector) ScheduleDriftDetection(ctx context.Context, tenantID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("✅ Scheduled drift detection every %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("ℹ️  Drift detection scheduler stopped")
			return
		case <-ticker.C:
			// Get all models for tenant
			var models []string
			query := `
				SELECT DISTINCT model_key FROM fabric_defn
				WHERE tenant_id = $1 AND kind = 'model'
			`
			if err := dd.db.SelectContext(ctx, &models, query, tenantID); err != nil {
				log.Printf("⚠️  Failed to fetch models: %v", err)
				continue
			}

			log.Printf("🔍 Running drift detection for %d models", len(models))
			for _, modelID := range models {
				report, err := dd.GenerateDriftReport(ctx, tenantID, modelID)
				if err != nil {
					log.Printf("⚠️  Drift detection failed for model %s: %v", modelID, err)
					continue
				}

				if err := dd.SaveDriftReport(ctx, report); err != nil {
					log.Printf("⚠️  Failed to save drift report: %v", err)
				}
			}
		}
	}
}
