package audit

import (
	"encoding/json"
	"time"
)

// =============================================================================
// AI Gate & Catalog Mutation Audit Events
// =============================================================================
//
// Typed events for Cardinal Rule 7 (Security Mandate) compliance.
// All AI Gate emissions route to TopicAIGate via RedpandaAuditPublisher
// and are produced synchronously with RequiredAcks=RequireAll.
//
// Cardinal Rule 7 says: "If a regulated event occurs and the audit cannot be
// proven, the system is in violation." These event types cover every decision
// point in the AI Gate so a regulator can replay the full lineage
// (prompt → semantic resolution → ABAC evaluation → masking → query).
//
// Cardinal Rule 6 (Tenant Isolation) says: "TenantID is always set, even if it
// is the empty string." We never allow TenantID to be defaulted to "unknown";
// callers must provide it explicitly.
// =============================================================================

// AIQueryGeneratedEvent is emitted when BOSQLGenerator produces a SQL query
// from an AI / semantic request. This is the root of the lineage chain.
type AIQueryGeneratedEvent struct {
	QueryID        string `json:"queryId"`  // unique per generation request
	TenantID       string `json:"tenantId"` // tenant scope (mandatory)
	UserID         string `json:"userId"`   // actor (from JWT azp)
	UserEmail      string `json:"userEmail,omitempty"`
	FunctionalRole string `json:"functionalRole"` // role at time of generation

	// Source context
	InputPrompt   string `json:"inputPrompt"`                // verbatim user prompt
	PromptHash    string `json:"promptHash"`                 // SHA-256 of prompt for dedup/search
	DatasourceID  string `json:"datasourceId,omitempty"`     // semantic datasource id, if known
	BusinessObjID string `json:"businessObjectId,omitempty"` // resolved BO id, if any
	TechnicalName string `json:"technicalName,omitempty"`    // BO technical name, if resolved

	// Output context
	GeneratedSQL     string `json:"generatedSql"`     // SQL produced
	GeneratedHash    string `json:"generatedHash"`    // SHA-256 of SQL — chain link
	JoinCount        int    `json:"joinCount"`        // number of joins inferred
	FieldCount       int    `json:"fieldCount"`       // number of fields projected
	MaskedFieldCount int    `json:"maskedFieldCount"` // count of PII fields masked

	// Correlation
	CorrelationID string `json:"correlationId,omitempty"`
	SourceIP      string `json:"sourceIp,omitempty"`

	// Timing
	GeneratedAt time.Time `json:"generatedAt"`
	DurationMs  int64     `json:"durationMs"`

	// Free-form
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// AISemanticResolvedEvent is emitted when ResolveSemanticRequest translates
// a natural-language or semantic-term request into a UUID-based SQLGenerationRequest.
// Captures the term → field mapping for governance replay.
type AISemanticResolvedEvent struct {
	QueryID      string `json:"queryId"`
	TenantID     string `json:"tenantId"`
	UserID       string `json:"userId,omitempty"`
	DatasourceID string `json:"datasourceId,omitempty"`

	ResolvedBusinessObjID string `json:"resolvedBusinessObjectId"`
	ResolvedTechnicalName string `json:"resolvedTechnicalName,omitempty"`

	// semantic-term → physical-field mappings produced
	TermResolutions json.RawMessage `json:"termResolutions"` // []TermResolution

	ConfidenceScore float64         `json:"confidenceScore"`
	ResolvedAt      time.Time       `json:"resolvedAt"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
}

// TermResolution captures a single semantic-term → physical-field mapping.
type TermResolution struct {
	SemanticTerm    string  `json:"semanticTerm"`
	PhysicalColumn  string  `json:"physicalColumn"`
	TableName       string  `json:"tableName"`
	MatchMethod     string  `json:"matchMethod"` // "exact", "fuzzy", "embedding"
	MatchConfidence float64 `json:"matchConfidence"`
}

// AIColumnMaskedEvent is emitted every time AIGraphSecurityInterceptor
// rewrites a projected column into a masked expression. Cardinal Rule 7 requires
// that every masking decision is independently auditable, not just the net result.
type AIColumnMaskedEvent struct {
	QueryID  string `json:"queryId"`
	TenantID string `json:"tenantId"`
	UserID   string `json:"userId,omitempty"`

	TableAlias string `json:"tableAlias"`
	ColumnName string `json:"columnName"`
	Original   string `json:"original"` // unmasked expression ("o.email")
	Masked     string `json:"masked"`   // masked expression ("CASE WHEN ... THEN '***' END")
	MaskType   string `json:"maskType"` // e.g. "HASH", "REDACT", "NULLIFY", "TOKENIZE"

	// Context
	ProfileID      string `json:"profileId,omitempty"`      // ABAC profile in scope
	Classification string `json:"classification,omitempty"` // e.g. "PII", "PHI", "PCI"
	LineagePath    string `json:"lineagePath,omitempty"`    // physical → business → semantic path

	MaskedAt time.Time       `json:"maskedAt"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// AIABACEvaluatedEvent records every ABAC evaluation result — including NON-denies.
// This is what regulators ask for: "show me every ABAC decision your system made
// for tenant X on date Y". Masked columns AND allow-decisions both go here.
type AIABACEvaluatedEvent struct {
	QueryID        string `json:"queryId"`
	TenantID       string `json:"tenantId"`
	UserID         string `json:"userId,omitempty"`
	ProfileID      string `json:"profileId"`      // profile against which ABAC was evaluated
	ProfileVersion int    `json:"profileVersion"` // semantic version of profile at eval time
	Classification string `json:"classification"` // tag evaluated against

	Decision string `json:"decision"` // "allow" | "mask:<type>" | "deny"
	Reason   string `json:"reason,omitempty"`

	EvaluatedAt time.Time       `json:"evaluatedAt"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// AIABACDeniedEvent is emitted when an ABAC evaluation results in an
// explicit deny (e.g., analyst probes a highly restricted PII field).
// CRITICAL: this event must be published SYNCHRONOUSLY before the
// deny error is returned to the user, per Cardinal Rule 7.
type AIABACDeniedEvent struct {
	QueryID   string `json:"queryId"`
	TenantID  string `json:"tenantId"`
	UserID    string `json:"userId"`
	UserEmail string `json:"userEmail,omitempty"`

	ProfileID      string `json:"profileId"`
	ProfileVersion int    `json:"profileVersion"`
	Classification string `json:"classification"`

	DeniedResource string `json:"deniedResource"` // "table.column" or path
	DenialReason   string `json:"denialReason"`   // human-readable reason

	// What the user was attempting — captured for forensic replay.
	AttemptedPromptHash string `json:"attemptedPromptHash,omitempty"`
	AttemptedSQLPreview string `json:"attemptedSqlPreview,omitempty"` // truncated SQL

	// Cardinal Rule 7: synchronous emission must succeed before 403 returns.
	// If delivery fails, the caller MUST surface the error so it can be logged.
	EmittedSync bool `json:"emittedSync"`

	DeniedAt time.Time       `json:"deniedAt"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// AILineageResolvedEvent traces lineage from a physical column up through
// business terms to a semantic concept to find classification tags.
// Emitted once per ResolveGraphGovernanceContext call.
type AILineageResolvedEvent struct {
	QueryID  string `json:"queryId"`
	TenantID string `json:"tenantId"`

	PhysicalColumnPath string          `json:"physicalColumnPath"`
	ResolvedTags       []string        `json:"resolvedTags"` // ["PII", "RESTRICTED", ...]
	LineageChain       json.RawMessage `json:"lineageChain"` // [{from, to, type}, ...]
	ResolvedAt         time.Time       `json:"resolvedAt"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
}

// CatalogBOMutatedEvent is emitted whenever a Business Object is created,
// updated, or its fields are modified. Drives both audit replay AND
// Phase A's cache invalidation event listener.
type CatalogBOMutatedEvent struct {
	MutationID string `json:"mutationId"`
	TenantID   string `json:"tenantId"`
	ActorID    string `json:"actorId,omitempty"`

	BusinessObjectID string `json:"businessObjectId"`
	TechnicalName    string `json:"technicalName,omitempty"`
	DatasourceID     string `json:"datasourceId,omitempty"`

	// Mutation semantics
	MutationType  string `json:"mutationType"` // "create" | "update" | "delete" | "field_add" | "field_remove" | "tag_change"
	VersionBefore int    `json:"versionBefore,omitempty"`
	VersionAfter  int    `json:"versionAfter,omitempty"`

	// Cache invalidation hint for Phase A consumer (cmd/cache-invalidator)
	InvalidateCacheKeys []string `json:"invalidateCacheKeys,omitempty"` // e.g. ["md:v1:t:{tenantID}:bo:by-name:{ds}:{name}"]

	MutatedAt time.Time       `json:"mutatedAt"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// =============================================================================
// AI Gate Topics
// =============================================================================
//
// Cardinal Rule 6 routes tenant-scoped events into tenant-keyed partitions
// inside these topics so consumer-side filtering is exact.

const (
	// TopicAIGate is the topic for AI-Gate lifecycle events (B1–B4, B6).
	// Cardinal Rule 7 mandates synchronous, acked publish for everything on this topic.
	TopicAIGate = "audit.ai.gate"

	// TopicAIDenials is a dedicated topic for ABAC denials (B7).
	// Separate so SIEM/SOC alerting can subscribe to denial stream alone.
	TopicAIDenials = "audit.ai.denials"

	// TopicCatalogMutations is the topic for catalog BO mutations (B5).
	// Phase A's cmd/cache-invalidator consumes this to drain invalidations.
	TopicCatalogMutations = "audit.catalog.mutations"

	// TopicCacheInvalidations is a thin wrapper event type used by the
	// cache-invalidator consumer to track its drain attempts (for replay).
	TopicCacheInvalidations = "audit.cache.invalidations"
)

// =============================================================================
// AI Gate Event-Type Constants
// =============================================================================

const (
	EventTypeAIQueryGenerated   = "AI_QUERY_GENERATED"
	EventTypeAISemanticResolved = "AI_SEMANTIC_RESOLVED"
	EventTypeAIColumnMasked     = "AI_COLUMN_MASKED"
	EventTypeAIABACEvaluated    = "AI_ABAC_EVALUATED"
	EventTypeAIABACDenied       = "AI_ABAC_DENIED" // CRITICAL — Cardinal Rule 7
	EventTypeAILineageResolved  = "AI_LINEAGE_RESOLVED"
	EventTypeCatalogBOMutated   = "CATALOG_BO_MUTATED"
)
