package exceptions

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ExceptionType string

const (
	ExceptionSLOBreach           ExceptionType = "slo_breach"
	ExceptionSemanticDrift       ExceptionType = "semantic_drift"
	ExceptionSecurityAnomaly     ExceptionType = "security_anomaly"
	ExceptionDataQuality         ExceptionType = "data_quality"
	ExceptionResidencyViolation  ExceptionType = "residency_violation"
	ExceptionAccessibility       ExceptionType = "accessibility_violation"
	ExceptionAPIInconsistency    ExceptionType = "api_inconsistency"
	ExceptionPreAggInconsistency ExceptionType = "preagg_inconsistency"
	ExceptionTenantAnomaly       ExceptionType = "tenant_anomaly"
	ExceptionPIIExposure         ExceptionType = "pii_exposure"
)

type Exception struct {
	ID          uuid.UUID     `json:"id"`
	Type        ExceptionType `json:"type"`
	Severity    string        `json:"severity"` // critical, high, medium, low
	Source      string        `json:"source"`   // page_id, api_id, tenant_id, etc.
	Description string        `json:"description"`
	Evidence    []string      `json:"evidence"`
	DetectedAt  time.Time     `json:"detected_at"`
	Resolved    bool          `json:"resolved"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
}

type ExceptionSummary struct {
	TotalExceptions    int                   `json:"total_exceptions"`
	CriticalCount      int                   `json:"critical_count"`
	HighCount          int                   `json:"high_count"`
	MediumCount        int                   `json:"medium_count"`
	LowCount           int                   `json:"low_count"`
	ByType             map[ExceptionType]int `json:"by_type"`
	RecentExceptions   []Exception           `json:"recent_exceptions"`
	TopAffectedTenants []string              `json:"top_affected_tenants"`
	TopAffectedPages   []string              `json:"top_affected_pages"`
	TopAffectedAPIs    []string              `json:"top_affected_apis"`
}

type ExceptionAggregator struct{}

func NewExceptionAggregator() *ExceptionAggregator {
	return &ExceptionAggregator{}
}

func (ea *ExceptionAggregator) GetAllExceptions(ctx context.Context) ([]Exception, error) {
	// Mock: Generate exceptions
	// Real: Aggregate from all platform layers (SLO engine, semantic intelligence, security, data quality, etc.)

	exceptions := []Exception{
		{
			ID:          uuid.New(),
			Type:        ExceptionSLOBreach,
			Severity:    "high",
			Source:      "page:positions_dashboard",
			Description: "Positions Dashboard p95 latency exceeded 300ms threshold",
			Evidence:    []string{"p95: 342ms", "threshold: 300ms", "breach duration: 15 minutes"},
			DetectedAt:  time.Now().Add(-15 * time.Minute),
			Resolved:    false,
		},
		{
			ID:          uuid.New(),
			Type:        ExceptionSemanticDrift,
			Severity:    "medium",
			Source:      "bo:Position",
			Description: "Field 'market_value' type changed from float to decimal",
			Evidence:    []string{"Previous type: float", "Current type: decimal", "Affected pages: 3"},
			DetectedAt:  time.Now().Add(-2 * time.Hour),
			Resolved:    false,
		},
		{
			ID:          uuid.New(),
			Type:        ExceptionSecurityAnomaly,
			Severity:    "critical",
			Source:      "tenant:tenant-123",
			Description: "User accessing PII fields at 10× normal rate",
			Evidence:    []string{"Normal rate: 5/hour", "Current rate: 52/hour", "User: user-482"},
			DetectedAt:  time.Now().Add(-30 * time.Minute),
			Resolved:    false,
		},
		{
			ID:          uuid.New(),
			Type:        ExceptionDataQuality,
			Severity:    "high",
			Source:      "table:positions",
			Description: "Field 'price' has 12% missing values for tenant-201",
			Evidence:    []string{"Historical: <1%", "Current: 12%", "Affected rows: 1,247"},
			DetectedAt:  time.Now().Add(-1 * time.Hour),
			Resolved:    false,
		},
		{
			ID:          uuid.New(),
			Type:        ExceptionResidencyViolation,
			Severity:    "critical",
			Source:      "api:positions_detail",
			Description: "EU tenant accessed US-residency data",
			Evidence:    []string{"Tenant: tenant-456 (EU)", "API: positions_detail (US-only)", "Timestamp: 2026-01-16 22:30:00"},
			DetectedAt:  time.Now().Add(-20 * time.Minute),
			Resolved:    false,
		},
		{
			ID:          uuid.New(),
			Type:        ExceptionAccessibility,
			Severity:    "medium",
			Source:      "page:client_profile",
			Description: "WCAG AA contrast violation detected",
			Evidence:    []string{"Element: .header-title", "Contrast ratio: 3.2:1", "Required: 4.5:1"},
			DetectedAt:  time.Now().Add(-45 * time.Minute),
			Resolved:    false,
		},
	}

	return exceptions, nil
}

func (ea *ExceptionAggregator) GetSummary(ctx context.Context) (*ExceptionSummary, error) {
	// Mock: Generate summary
	// Real: Aggregate statistics from all exceptions

	exceptions, _ := ea.GetAllExceptions(ctx)

	summary := &ExceptionSummary{
		TotalExceptions: len(exceptions),
		CriticalCount:   2,
		HighCount:       2,
		MediumCount:     2,
		LowCount:        0,
		ByType: map[ExceptionType]int{
			ExceptionSLOBreach:          1,
			ExceptionSemanticDrift:      1,
			ExceptionSecurityAnomaly:    1,
			ExceptionDataQuality:        1,
			ExceptionResidencyViolation: 1,
			ExceptionAccessibility:      1,
		},
		RecentExceptions:   exceptions[:3],
		TopAffectedTenants: []string{"tenant-123", "tenant-201", "tenant-456"},
		TopAffectedPages:   []string{"positions_dashboard", "client_profile"},
		TopAffectedAPIs:    []string{"positions_detail", "positions_api"},
	}

	return summary, nil
}

func (ea *ExceptionAggregator) GetByType(ctx context.Context, exceptionType ExceptionType) ([]Exception, error) {
	// Mock: Filter by type
	// Real: Query exceptions by type

	allExceptions, _ := ea.GetAllExceptions(ctx)
	filtered := []Exception{}
	for _, ex := range allExceptions {
		if ex.Type == exceptionType {
			filtered = append(filtered, ex)
		}
	}
	return filtered, nil
}
