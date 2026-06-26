package audit

import (
	"time"
)

// TenantScope represents the tenants a user can access
type TenantScope []string

// IsGlobal returns true if scope includes "global" or "*"
func (ts TenantScope) IsGlobal() bool {
	for _, t := range ts {
		if t == "global" || t == "*" {
			return true
		}
	}
	return false
}

// Contains checks if a tenant is in scope
func (ts TenantScope) Contains(tenantID string) bool {
	if ts.IsGlobal() {
		return true
	}
	for _, t := range ts {
		if t == tenantID {
			return true
		}
	}
	return false
}

// Intersect returns the intersection of two tenant scopes
func (ts TenantScope) Intersect(other TenantScope) TenantScope {
	if ts.IsGlobal() {
		return other
	}
	if other.IsGlobal() {
		return ts
	}
	result := TenantScope{}
	for _, t := range ts {
		if other.Contains(t) {
			result = append(result, t)
		}
	}
	return result
}

// AuditEvent represents a single audit event from any source
type AuditEvent struct {
	ID                string                 `json:"id"`
	Type              string                 `json:"type"` // job_run, dag_run, changeset, semantic_snapshot, compliance_violation, etc.
	TenantID          string                 `json:"tenantId"`
	Timestamp         time.Time              `json:"timestamp"`
	Status            string                 `json:"status,omitempty"`       // SUCCESS, FAILED, COMPLIANCE_BLOCK, PENDING, APPROVED, etc.
	ArtifactType      string                 `json:"artifactType,omitempty"` // job, dag, semantic_term, business_term, workflow, etc.
	ArtifactID        string                 `json:"artifactId,omitempty"`
	Title             string                 `json:"title,omitempty"`     // Human-readable title
	Actor             string                 `json:"actor,omitempty"`     // User ID or system name
	RiskLevel         string                 `json:"riskLevel,omitempty"` // LOW, MEDIUM, HIGH, CRITICAL
	SemanticContext   map[string]interface{} `json:"semanticContext,omitempty"`
	ComplianceContext map[string]interface{} `json:"complianceContext,omitempty"`
	AINarrative       map[string]interface{} `json:"aiNarrative,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// EntityAudit represents all audit events for a specific entity
type EntityAudit struct {
	EntityType  string                 `json:"entityType"` // semantic_term, business_term, job, dag, workflow, tenant
	EntityID    string                 `json:"entityId"`
	EntityName  string                 `json:"entityName"`
	TenantIDs   []string               `json:"tenantIds"`
	Timeline    []AuditEvent           `json:"timeline"`
	Changes     []AuditEvent           `json:"changes"`
	Compliance  []AuditEvent           `json:"compliance"`
	AIInsights  map[string]interface{} `json:"aiInsights,omitempty"`
	LastUpdated time.Time              `json:"lastUpdated"`
}

// IncidentCluster represents a group of related failures
type IncidentCluster struct {
	ID              string      `json:"id"`
	TimeWindow      TimeWindow  `json:"timeWindow"`
	AffectedTenants []string    `json:"affectedTenants"`
	AffectedJobs    []string    `json:"affectedJobs"`
	AffectedDAGs    []string    `json:"affectedDAGs"`
	Status          string      `json:"status"` // open, mitigated, closed
	EventCount      int         `json:"eventCount"`
	AIRootCause     string      `json:"aiRootCause,omitempty"`
	AINarrative     string      `json:"aiNarrative,omitempty"`
	SLOImpact       string      `json:"sloImpact,omitempty"`
	BlastRadius     BlastRadius `json:"blastRadius,omitempty"`
	SuggestedFix    string      `json:"suggestedFix,omitempty"`
}

type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type BlastRadius struct {
	JobsImpacted    int      `json:"jobsImpacted"`
	DAGsImpacted    int      `json:"dagsImpacted"`
	TenantsImpacted []string `json:"tenantsImpacted"`
	SemanticTerms   []string `json:"semanticTerms,omitempty"`
	BusinessTerms   []string `json:"businessTerms,omitempty"`
	DownstreamJobs  []string `json:"downstreamJobs,omitempty"`
}

// ComplianceEvent represents a compliance-related audit event
type ComplianceEvent struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenantId"`
	Timestamp         time.Time              `json:"timestamp"`
	ViolationType     string                 `json:"violationType"` // PII_VIOLATION, RESIDENCY_BLOCK, SENSITIVITY_CHANGE, POLICY_CHANGE
	Status            string                 `json:"status"`        // resolved, unresolved
	ArtifactType      string                 `json:"artifactType"`  // job, semantic_term, business_term
	ArtifactID        string                 `json:"artifactId"`
	Severity          string                 `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	AIExplanation     string                 `json:"aiExplanation,omitempty"`
	RelatedChangeSets []string               `json:"relatedChangeSets,omitempty"`
	RemediationPath   string                 `json:"remediationPath,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ExplainRequest is sent to the AI explanation service
type ExplainRequest struct {
	TenantScope       []string               `json:"tenantScope"`
	AuditRecords      []AuditEvent           `json:"auditRecords"`
	EntityContext     map[string]interface{} `json:"entityContext,omitempty"`
	SemanticContext   map[string]interface{} `json:"semanticContext,omitempty"`
	ComplianceContext map[string]interface{} `json:"complianceContext,omitempty"`
}

// ExplainResponse contains AI-generated insights
type ExplainResponse struct {
	Narrative                 string   `json:"narrative"`
	RootCause                 string   `json:"rootCause"`
	BlastRadius               string   `json:"blastRadius"`
	RecommendedFix            string   `json:"recommendedFix"`
	SuggestedChangeSetSummary string   `json:"suggestedChangeSetSummary"`
	AffectedEntities          []string `json:"affectedEntities,omitempty"`
	RiskScore                 float64  `json:"riskScore"`  // 0.0 - 1.0
	Confidence                float64  `json:"confidence"` // 0.0 - 1.0
}

// QueryFilters represents audit query filters
type QueryFilters struct {
	TimeRange     TimeRange
	ArtifactTypes []string
	Statuses      []string
	RiskLevels    []string
	Actors        []string
	ErrorTypes    []string // for ops-specific filtering
	SearchTerm    string   // for entity name search
	Limit         int
	Offset        int
}

type TimeRange struct {
	From time.Time
	To   time.Time
}

// ListEventsRequest is the API request for listing audit events
type ListEventsRequest struct {
	TenantFilter  []string  `json:"tenantFilter"`
	TimeRange     TimeRange `json:"timeRange"`
	ArtifactTypes []string  `json:"artifactTypes"`
	Statuses      []string  `json:"statuses"`
	RiskLevels    []string  `json:"riskLevels"`
	Actors        []string  `json:"actors"`
	Limit         int       `json:"limit"`
	Offset        int       `json:"offset"`
}

// ListEventsResponse wraps audit events with metadata
type ListEventsResponse struct {
	Events  []AuditEvent `json:"events"`
	Total   int          `json:"total"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
	HasMore bool         `json:"hasMore"`
}

// Role-specific dashboard summaries

// GlobalAdminDashboard shows platform-wide metrics
type GlobalAdminDashboard struct {
	TenantCount          int                    `json:"tenantCount"`
	FailedRunsLastDay    map[string]int         `json:"failedRunsLastDay"`    // by tenant
	ComplianceViolations map[string]int         `json:"complianceViolations"` // by tenant
	SLOBreachRisk        map[string]float64     `json:"sloBreachRisk"`        // by tenant
	HighRiskChangeSets   []AuditEvent           `json:"highRiskChangeSets"`
	TopCrossTenantRisks  []string               `json:"topCrossTenantRisks"`
	PlatformHealth       map[string]interface{} `json:"platformHealth"`
}

// GlobalOpsDashboard shows multi-tenant ops metrics
type GlobalOpsDashboard struct {
	AssignedTenants          []string           `json:"assignedTenants"`
	IncidentClustersByTenant map[string]int     `json:"incidentClustersByTenant"`
	SLOPressure              map[string]float64 `json:"sloPressure"` // by tenant
	ForecastedBreaches       []string           `json:"forecastedBreaches"`
	JobsAtRisk               int                `json:"jobsAtRisk"`
	DAGsUnderStress          int                `json:"dagsUnderStress"`
}

// TenantAdminDashboard shows tenant-specific metrics
type TenantAdminDashboard struct {
	TenantID             string                 `json:"tenantId"`
	FailedRunsLastDay    int                    `json:"failedRunsLastDay"`
	ComplianceViolations int                    `json:"complianceViolations"`
	PendingApprovals     []AuditEvent           `json:"pendingApprovals"`
	HighRiskChangeSets   []AuditEvent           `json:"highRiskChangeSets"`
	PiiViolations        int                    `json:"piiViolations"`
	ResidencyBlocks      int                    `json:"residencyBlocks"`
	TenantHealth         map[string]interface{} `json:"tenantHealth"`
}

// TenantOpsDashboard shows operational metrics for tenant ops
type TenantOpsDashboard struct {
	TenantID             string                 `json:"tenantId"`
	FailedRunsLastDay    int                    `json:"failedRunsLastDay"`
	FailedDAGsLastDay    int                    `json:"failedDAGsLastDay"`
	OpenIncidents        int                    `json:"openIncidents"`
	RecentFailures       []AuditEvent           `json:"recentFailures"`
	RetryStormCount      int                    `json:"retryStormCount"`
	ComplianceBlockCount int                    `json:"complianceBlockCount"`
	OperationalHealth    map[string]interface{} `json:"operationalHealth"`
}
