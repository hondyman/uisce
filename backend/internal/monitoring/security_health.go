package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
)

// SecurityHealthStatus represents the health of a tenant's security master
type SecurityHealthStatus struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	HealthScore        float64   `json:"health_score"`        // 0-100 score
	Status             string    `json:"status"`              // Healthy, Degraded, Critical
	TotalSecurities    int       `json:"total_securities"`    // Total active securities
	CatalogDriftCount  int       `json:"catalog_drift_count"` // Missing semantic mappings
	DQViolationCount   int       `json:"dq_violation_count"`  // Securities with active hard DQ issues
	AverageConfidence  float64   `json:"average_confidence"`  // Mean confidence score across active gold copies
	LastCalculatedAt   time.Time `json:"last_calculated_at"`
	RecommendedActions []string  `json:"recommended_actions"`
}

// SecurityHealthMonitor is responsible for analyzing and reporting security master health
type SecurityHealthMonitor struct {
	db       *sql.DB
	goldRepo *goldcopy.Repository
}

// NewSecurityHealthMonitor creates a new health monitor
func NewSecurityHealthMonitor(db *sql.DB, goldRepo *goldcopy.Repository) *SecurityHealthMonitor {
	return &SecurityHealthMonitor{
		db:       db,
		goldRepo: goldRepo,
	}
}

// CalculateTenantHealth computes the current health status of a tenant's security master
func (m *SecurityHealthMonitor) CalculateTenantHealth(ctx context.Context, tenantID uuid.UUID) (*SecurityHealthStatus, error) {
	status := &SecurityHealthStatus{
		TenantID:         tenantID,
		LastCalculatedAt: time.Now(),
	}

	// Determine if SQLite is used (for tests)
	isSQLite := false
	if driverName := fmt.Sprintf("%T", m.db.Driver()); driverName == "*sqlite3.SQLiteDriver" {
		isSQLite = true
	}

	tablePrefix := "edm."
	if isSQLite {
		tablePrefix = ""
	}

	// 1. Get total securities and average confidence
	var total int
	var avgConf sql.NullFloat64
	query := fmt.Sprintf(`
		SELECT COUNT(id), AVG(confidence_score)
		FROM %ssecurity_master
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > CURRENT_TIMESTAMP)
	`, tablePrefix)
	if err := m.db.QueryRowContext(ctx, query, tenantID).Scan(&total, &avgConf); err != nil {
		return nil, fmt.Errorf("failed to calculate basic metrics: %w", err)
	}

	status.TotalSecurities = total
	if avgConf.Valid {
		status.AverageConfidence = avgConf.Float64
	}

	// 2. Calculate Catalog Drift (Missing semantic mappings)
	// Drift occurs when a security lacks essential classification lineage in the graph
	var driftCount int
	driftQuery := `
		SELECT COUNT(n.id)
		FROM catalog_node n
		LEFT JOIN catalog_edge e ON n.id = e.source_node_id AND e.edge_type_id IN (
			SELECT id FROM catalog_edge_type WHERE type_name = 'has_classification'
		)
		JOIN catalog_node_type nt ON n.type_id = nt.id
		WHERE (n.tenant_id = $1 OR n.tenant_id IS NULL)
		  AND nt.type_name = 'security'
		  AND e.id IS NULL
	`
	// Fallback to 0 if semantic schema isn't fully migrated yet
	if err := m.db.QueryRowContext(ctx, driftQuery, tenantID).Scan(&driftCount); err == nil {
		status.CatalogDriftCount = driftCount
	}

	// 3. Get DQ Violation count (Securities created but flagged for future manual review)
	var dqCount int
	dqQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT security_id)
		FROM %sdata_quality_results
		WHERE tenant_id = $1 AND severity = 'Hard' AND resolved = false
	`, tablePrefix)
	if err := m.db.QueryRowContext(ctx, dqQuery, tenantID).Scan(&dqCount); err == nil {
		status.DQViolationCount = dqCount
	}

	// 4. Calculate Final Health Score
	// Base score is 100. Deductions:
	// - 10 points for every 1% of total securities with hard DQ rules
	// - 5 points for every 1% of total securities with catalog drift
	// - Confidence scalar applies to final result
	score := 100.0

	if total > 0 {
		dqPenalty := (float64(dqCount) / float64(total)) * 100.0 * 10.0
		driftPenalty := (float64(driftCount) / float64(total)) * 100.0 * 5.0

		// Apply penalties
		score -= dqPenalty
		score -= driftPenalty

		// Scale by mean confidence
		score = score * (status.AverageConfidence / 100.0)
	} else {
		score = 0
	}

	// Ensure bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	status.HealthScore = score

	// 5. Determine State and Recommended Actions
	m.evaluateStatusState(status)

	return status, nil
}

// evaluateStatusState determines the text status and recommended actions based on the score and underlying metrics
func (m *SecurityHealthMonitor) evaluateStatusState(status *SecurityHealthStatus) {
	if status.TotalSecurities == 0 {
		status.Status = "Empty"
		status.RecommendedActions = append(status.RecommendedActions, "Onboard security data sources")
		return
	}

	if status.HealthScore >= 90 {
		status.Status = "Healthy"
	} else if status.HealthScore >= 70 {
		status.Status = "Degraded"
	} else {
		status.Status = "Critical"
	}

	if status.DQViolationCount > 0 {
		status.RecommendedActions = append(status.RecommendedActions, fmt.Sprintf("Resolve %d active Hard Data Quality violations", status.DQViolationCount))
	}

	if status.CatalogDriftCount > 0 {
		status.RecommendedActions = append(status.RecommendedActions, fmt.Sprintf("Map %d securities to the semantic catalog", status.CatalogDriftCount))
	}

	if status.AverageConfidence < 85.0 {
		status.RecommendedActions = append(status.RecommendedActions, "Review survivorship rules to improve confidence scores")
	}
}
