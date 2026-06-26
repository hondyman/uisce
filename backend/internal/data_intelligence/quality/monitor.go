package quality

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type QualityIssueType string

const (
	IssueAnomaly              QualityIssueType = "anomaly"
	IssueMissingValues        QualityIssueType = "missing_values"
	IssueOutliers             QualityIssueType = "outliers"
	IssueSchemaDrift          QualityIssueType = "schema_drift"
	IssueReferentialIntegrity QualityIssueType = "referential_integrity"
	IssuePreAggInconsistency  QualityIssueType = "preagg_inconsistency"
)

type QualityIssue struct {
	ID           uuid.UUID        `json:"id"`
	Type         QualityIssueType `json:"type"`
	Severity     string           `json:"severity"` // critical, high, medium, low
	TableName    string           `json:"table_name"`
	FieldName    string           `json:"field_name,omitempty"`
	Description  string           `json:"description"`
	Evidence     []string         `json:"evidence"`
	SuggestedFix string           `json:"suggested_fix"`
	DetectedAt   time.Time        `json:"detected_at"`
}

type QualityScore struct {
	TableName         string `json:"table_name"`
	OverallScore      int    `json:"overall_score"` // 0-100
	CompletenessScore int    `json:"completeness_score"`
	AccuracyScore     int    `json:"accuracy_score"`
	ConsistencyScore  int    `json:"consistency_score"`
	TimelinessScore   int    `json:"timeliness_score"`
}

type QualityMonitor struct{}

func NewQualityMonitor() *QualityMonitor {
	return &QualityMonitor{}
}

func (qm *QualityMonitor) DetectIssues(ctx context.Context) ([]QualityIssue, error) {
	// Mock: Generate quality issues
	// Real: Analyze data distributions, detect anomalies, check referential integrity

	issues := []QualityIssue{
		{
			ID:          uuid.New(),
			Type:        IssueMissingValues,
			Severity:    "high",
			TableName:   "positions",
			FieldName:   "price",
			Description: "Field 'price' has 12% missing values for tenant-201",
			Evidence: []string{
				"Historical average: <1% missing values",
				"Current: 12% missing values (last 24 hours)",
				"Affected rows: 1,247 out of 10,392",
			},
			SuggestedFix: "Investigate data pipeline for tenant-201. Check upstream feed from custodian.",
			DetectedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Type:        IssuePreAggInconsistency,
			Severity:    "critical",
			TableName:   "positions",
			Description: "Positions pre-agg is inconsistent with raw data by 3.2%",
			Evidence: []string{
				"Pre-agg total_market_value: $125,450,000",
				"Raw data total_market_value: $129,500,000",
				"Difference: $4,050,000 (3.2%)",
			},
			SuggestedFix: "Force refresh positions_daily pre-aggregation. Likely stale due to failed refresh.",
			DetectedAt:   time.Now().Add(-2 * time.Hour),
		},
		{
			ID:          uuid.New(),
			Type:        IssueAnomaly,
			Severity:    "medium",
			TableName:   "trades",
			FieldName:   "quantity",
			Description: "Sudden spike in trade quantities detected",
			Evidence: []string{
				"Normal average: 500 shares/trade",
				"Current average: 12,000 shares/trade",
				"24x increase in last 6 hours",
			},
			SuggestedFix: "Verify if this is legitimate trading activity or data quality issue.",
			DetectedAt:   time.Now().Add(-30 * time.Minute),
		},
		{
			ID:          uuid.New(),
			Type:        IssueReferentialIntegrity,
			Severity:    "high",
			TableName:   "positions",
			FieldName:   "instrument_id",
			Description: "Orphaned instrument references detected",
			Evidence: []string{
				"42 positions reference instrument_id that no longer exists in instruments table",
				"Affected tenants: tenant-123, tenant-456",
			},
			SuggestedFix: "Add foreign key constraint or implement cascade delete policy.",
			DetectedAt:   time.Now().Add(-1 * time.Hour),
		},
	}

	return issues, nil
}

func (qm *QualityMonitor) ScoreTable(ctx context.Context, tableName string) (*QualityScore, error) {
	// Mock: Generate quality score
	// Real: Calculate scores based on completeness, accuracy, consistency, timeliness

	score := &QualityScore{
		TableName:         tableName,
		OverallScore:      78,
		CompletenessScore: 88, // 12% missing values in price field
		AccuracyScore:     95, // Few outliers
		ConsistencyScore:  65, // Pre-agg inconsistency
		TimelinessScore:   85, // Mostly up-to-date
	}

	return score, nil
}
