package enterprise

import (
	"time"
)

// Phase 8: Enterprise Architecture

// --- DATA QUALITY ---

// FreshnessGate defines data freshness requirements
type FreshnessGate struct {
	EntityPath      string        `json:"entity_path"`
	MaxStaleness    time.Duration `json:"max_staleness"` // e.g., 3 hours
	GateLevel       string        `json:"gate_level"` // green, amber, red
	LastUpdated     time.Time     `json:"last_updated"`
	UpdateFrequency time.Duration `json:"update_frequency"` // Expected refresh rate
}

// QualityGate represents data quality thresholds
type QualityGate struct {
	MetricName      string  `json:"metric_name"` // null_rate, completeness, etc.
	Threshold       float64 `json:"threshold"`
	CurrentValue    float64 `json:"current_value"`
	Status          string  `json:"status"` // pass, warning, fail
	ImpactLevel     string  `json:"impact_level"` // low, medium, high, critical
}

// DataQualityCheck validates data quality before use
type DataQualityCheck struct {
	EntityPath   string         `json:"entity_path"`
	CheckedAt    time.Time      `json:"checked_at"`
	FreshnessGate *FreshnessGate `json:"freshness_gate"`
	QualityGates []QualityGate  `json:"quality_gates"`
	OverallStatus string        `json:"overall_status"` // pass, warning, fail
	BlockResponse bool          `json:"block_response"` // Block if critical failure
}

// SLADefinition defines service level agreements
type SLADefinition struct {
	EntityPath          string        `json:"entity_path"`
	AvailabilityTarget  float64       `json:"availability_target"` // e.g., 99.9%
	FreshnessTarget     time.Duration `json:"freshness_target"`
	AccuracyTarget      float64       `json:"accuracy_target"` // e.g., 99.5%
	ResponseTimeTarget  time.Duration `json:"response_time_target"`
}

// --- SECURITY & ISOLATION ---

// TenantIsolation enforces multi-tenant data separation
type TenantIsolation struct {
	TenantID        string   `json:"tenant_id"`
	AllowedSchemas  []string `json:"allowed_schemas"`  // DB schemas accessible
	RowLevelFilters map[string]string `json:"row_level_filters"` // Table -> filter clause
	EncryptionKey   string   `json:"encryption_key,omitempty"`
}

// PIIRedactionRule defines rules for redacting sensitive information
type PIIRedactionRule struct {
	FieldPattern string `json:"field_pattern"` // Regex pattern
	RedactionType string `json:"redaction_type"` // mask, hash, remove
	Replacement  string `json:"replacement,omitempty"` // e.g., "***"
}

// AuditLog represents immutable audit trail
type AuditLog struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	TenantID      string                 `json:"tenant_id"`
	UserID        string                 `json:"user_id"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource"`
	Question      string                 `json:"question,omitempty"`
	ContextIDs    []string               `json:"context_ids"` // DAG node IDs used
	Provider      string                 `json:"provider"` // LLM provider used
	ResponseHash  string                 `json:"response_hash"` // Hash of response
	Sources       []string               `json:"sources"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	SessionID     string                 `json:"session_id"`
	StatusCode    int                    `json:"status_code"`
	LatencyMs     int64                  `json:"latency_ms"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityContext aggregates security information for a request
type SecurityContext struct {
	TenantID       string            `json:"tenant_id"`
	UserID         string            `json:"user_id"`
	Roles          []string          `json:"roles"`
	Scopes         []string          `json:"scopes"`
	IPAddress      string            `json:"ip_address"`
	SessionID      string            `json:"session_id"`
	SecurityLevel  string            `json:"security_level"` // public, internal, confidential, restricted
	Isolation      TenantIsolation   `json:"isolation"`
}

// DataQualityService manages data quality checks
type DataQualityService struct{}

// CheckFreshness evaluates data freshness against SLA
func (s *DataQualityService) CheckFreshness(entityPath string, lastUpdated time.Time, maxStaleness time.Duration) *FreshnessGate {
	staleness := time.Since(lastUpdated)
	
	var gateLevel string
	if staleness <= maxStaleness/2 {
		gateLevel = "green"
	} else if staleness <= maxStaleness {
		gateLevel = "amber"
	} else {
		gateLevel = "red"
	}
	
	return &FreshnessGate{
		EntityPath:   entityPath,
		MaxStaleness: maxStaleness,
		GateLevel:    gateLevel,
		LastUpdated:  lastUpdated,
	}
}

// EnforceSLA checks if data meets SLA requirements
func (s *DataQualityService) EnforceSLA(check *DataQualityCheck, sla *SLADefinition) bool {
	// Check freshness
	if check.FreshnessGate != nil && check.FreshnessGate.GateLevel == "red" {
		return false
	}
	
	// Check quality gates
	for _, gate := range check.QualityGates {
		if gate.ImpactLevel == "critical" && gate.Status == "fail" {
			return false
		}
	}
	
	return true
}

// AuditService manages audit logging
type AuditService struct{}

// LogQuery logs an NLQ query to the audit trail
func (s *AuditService) LogQuery(ctx SecurityContext, question string, sources []string, responseHash string, latencyMs int64) error {
	// TODO: Implement actual audit log persistence
	log := AuditLog{
		Timestamp:    time.Now(),
		TenantID:     ctx.TenantID,
		UserID:       ctx.UserID,
		Action:       "nlq_query",
		Question:     question,
		Sources:      sources,
		ResponseHash: responseHash,
		IPAddress:    ctx.IPAddress,
		SessionID:    ctx.SessionID,
		LatencyMs:    latencyMs,
	}
	
	// In production: write to immutable log store (append-only DB, S3, etc.)
	_ = log
	return nil
}
