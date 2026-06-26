package sharedtypes

import (
	"context"
	"time"
)

// Permission represents access permissions
type Permission string

const (
	PermRead   Permission = "read"
	PermWrite  Permission = "write"
	PermUpdate Permission = "update"
	PermDelete Permission = "delete"
	PermAdmin  Permission = "admin"
)

// AccessEvaluationRequest represents an access evaluation request
type AccessEvaluationRequest struct {
	UserID   string                 `json:"user_id"`
	Action   string                 `json:"action"`
	Resource string                 `json:"resource"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// AccessEvaluationResponse represents the response from access evaluation
type AccessEvaluationResponse struct {
	Allowed  bool     `json:"allowed"`
	Reason   string   `json:"reason"`
	Policies []string `json:"policies,omitempty"`
}

// EffectiveClaim represents an effective claim for a user
type EffectiveClaim struct {
	AssetID    string     `json:"asset_id"`
	Permission Permission `json:"permission"`
	Scope      []string   `json:"scope"`
	Source     string     `json:"source"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// EvaluationRequest represents an access evaluation request
type EvaluationRequest struct {
	UserID   string                 `json:"user_id"`
	TenantID string                 `json:"tenant_id"`
	AssetID  string                 `json:"asset_id"`
	Action   Permission             `json:"action"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Timestamp time.Time              `json:"timestamp"`
	Result    string                 `json:"result"`
	Reason    string                 `json:"reason,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// Policy represents an access control policy
type Policy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  map[string]interface{} `json:"conditions"`
	Actions     []string               `json:"actions"`
	Effect      string                 `json:"effect"` // "allow" or "deny"
}

// SemanticCalculationRequest represents a request for semantic model calculation
type SemanticCalculationRequest struct {
	UserID   string                 `json:"user_id"`
	TenantID string                 `json:"tenant_id"`
	ModelID  string                 `json:"model_id"`
	Query    string                 `json:"query"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// SemanticCalculationResponse represents the response from semantic model calculation
type SemanticCalculationResponse struct {
	ModelID     string    `json:"model_id"`
	Result      string    `json:"result"`
	ProcessedAt time.Time `json:"processed_at"`
}

// SemanticMapping represents a semantic field mapping
type SemanticMapping struct {
	ID              string  `json:"id"`
	SourceField     string  `json:"source_field"`
	TargetField     string  `json:"target_field"`
	MappingType     string  `json:"mapping_type"`
	ConfidenceScore float64 `json:"confidence_score"`
	CreatedAt       string  `json:"created_at"`
}

// APIResponse represents a standardized API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError represents an API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta contains metadata for API responses
type Meta struct {
	RequestID      string    `json:"request_id"`
	Timestamp      time.Time `json:"timestamp"`
	ProcessingTime string    `json:"processing_time,omitempty"`
}

// Evaluator defines the interface for access evaluation
type Evaluator interface {
	Evaluate(ctx context.Context, req EvaluationRequest) (bool, string, []EffectiveClaim, error)
}

// PolicyChecker defines the interface for policy checking
type PolicyChecker interface {
	Check(ctx context.Context, req EvaluationRequest, claims []EffectiveClaim) (bool, string, []map[string]interface{}, []string, error)
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogAccess(ctx context.Context, entry *AuditEntry) error
}

// ABACCheckRequest represents an ABAC (Attribute-Based Access Control) check request
type ABACCheckRequest struct {
	UserID     string `json:"user_id"`
	Action     string `json:"action"`
	ResourceID string `json:"resource_id"`
	TenantID   string `json:"tenant_id,omitempty"`
}

// TaxHarvestResult represents the result of a tax harvesting simulation
type TaxHarvestResult struct {
	EstimatedTaxSavings float64 `json:"estimated_tax_savings"`
	HarvestedLotsCount  int     `json:"harvested_lots_count"`
	TotalProceeds       float64 `json:"total_proceeds,omitempty"`
	RealizedLosses      float64 `json:"realized_losses,omitempty"`
}

// UMARebalanceRequest represents a request to rebalance a UMA account
type UMARebalanceRequest struct {
	UMAAccountID string                 `json:"uma_account_id"`
	UserID       string                 `json:"user_id"`
	RequestType  string                 `json:"request_type"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

// UMARebalanceWorkflowResult represents the result of a UMA rebalance workflow
type UMARebalanceWorkflowResult struct {
	UMAAccountID   string    `json:"uma_account_id"`
	Status         string    `json:"status"`
	PlanID         string    `json:"plan_id,omitempty"`
	ExecutedTrades int       `json:"executed_trades,omitempty"`
	TaxSavings     float64   `json:"tax_savings,omitempty"`
	CompletedAt    time.Time `json:"completed_at"`
	Error          string    `json:"error,omitempty"`
}

// ApprovalSignal represents an approval signal for workflow continuation
type ApprovalSignal struct {
	Approved   bool   `json:"approved"`
	ApprovedBy string `json:"approved_by"`
	Comments   string `json:"comments,omitempty"`
}

// Calculation represents a financial or analytical calculation definition
type Calculation struct {
	ID             string                 `json:"id"`
	NodeID         string                 `json:"node_id"`
	Name           string                 `json:"name"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Formula        string                 `json:"formula"`
	EngineType     string                 `json:"engine_type"`
	ReturnType     string                 `json:"return_type"`
	Arguments      map[string]interface{} `json:"arguments"`
	Category       string                 `json:"category"`
	Subcategory    string                 `json:"subcategory"`
	DomainID       *string                `json:"domain_id"`      // UUID string
	ExecutionType  string                 `json:"execution_type"` // realtime, batch
	Engine         string                 `json:"engine"`         // internal, cube, spark
	IsMaterialized bool                   `json:"is_materialized"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}
