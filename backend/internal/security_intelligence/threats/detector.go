package threats

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ThreatType string

const (
	ThreatSuspiciousAPI    ThreatType = "suspicious_api"
	ThreatSuspiciousPage   ThreatType = "suspicious_page"
	ThreatCrossTenant      ThreatType = "cross_tenant_anomaly"
	ThreatCredentialMisuse ThreatType = "credential_misuse"
	ThreatDataExfiltration ThreatType = "data_exfiltration"
)

type Threat struct {
	ID          uuid.UUID  `json:"id"`
	Type        ThreatType `json:"type"`
	Severity    string     `json:"severity"` // critical, high, medium, low
	UserID      string     `json:"user_id"`
	TenantID    string     `json:"tenant_id"`
	Description string     `json:"description"`
	Evidence    []string   `json:"evidence"`
	Timestamp   time.Time  `json:"timestamp"`
	Mitigated   bool       `json:"mitigated"`
}

type ThreatDetector struct{}

func NewThreatDetector() *ThreatDetector {
	return &ThreatDetector{}
}

func (td *ThreatDetector) DetectThreats(ctx context.Context) ([]Threat, error) {
	// Mock: Generate threat detections
	// Real: Analyze API logs, page telemetry, usage patterns, ML anomaly detection

	threats := []Threat{
		{
			ID:          uuid.New(),
			Type:        ThreatDataExfiltration,
			Severity:    "critical",
			UserID:      "user-482",
			TenantID:    "tenant-123",
			Description: "User accessing PII fields at 10× normal rate",
			Evidence: []string{
				"Normal rate: 5 PII accesses/hour",
				"Current rate: 52 PII accesses/hour",
				"Fields accessed: client_ssn, account_number, tax_id",
				"Time window: last 30 minutes",
			},
			Timestamp: time.Now(),
			Mitigated: false,
		},
		{
			ID:          uuid.New(),
			Type:        ThreatSuspiciousAPI,
			Severity:    "high",
			UserID:      "user-891",
			TenantID:    "tenant-77",
			Description: "High-volume queries outside normal business hours",
			Evidence: []string{
				"Time: 02:15 AM (outside 8 AM - 6 PM business hours)",
				"Query count: 847 in last hour",
				"Normal count: 12 queries/hour",
				"API: positions_api",
			},
			Timestamp: time.Now().Add(-2 * time.Hour),
			Mitigated: false,
		},
		{
			ID:          uuid.New(),
			Type:        ThreatCredentialMisuse,
			Severity:    "critical",
			UserID:      "user-234",
			TenantID:    "tenant-456",
			Description: "Impossible travel detected",
			Evidence: []string{
				"Login from New York at 09:00 AM",
				"Login from London at 09:15 AM",
				"Physical travel time: ~7 hours",
				"Actual time elapsed: 15 minutes",
			},
			Timestamp: time.Now().Add(-1 * time.Hour),
			Mitigated: false,
		},
		{
			ID:          uuid.New(),
			Type:        ThreatSuspiciousPage,
			Severity:    "medium",
			UserID:      "user-567",
			TenantID:    "tenant-789",
			Description: "Rapid page navigation with unusual filter patterns",
			Evidence: []string{
				"Pages visited: 42 in 3 minutes",
				"Normal rate: 8 pages/3 minutes",
				"Unusual filters: account_balance > 1000000",
				"Pattern suggests automated scraping",
			},
			Timestamp: time.Now().Add(-30 * time.Minute),
			Mitigated: false,
		},
	}

	return threats, nil
}
