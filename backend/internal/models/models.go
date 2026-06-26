package models

import "time"

// User represents a user in the system
type User struct {
	ID           string            `json:"id"`
	Email        string            `json:"email"`
	Name         string            `json:"name"`
	Role         string            `json:"role"`
	Roles        []string          `json:"roles,omitempty"`
	Organization string            `json:"organization"`
	Permissions  []string          `json:"permissions"`
	IsCoreAdmin  bool              `json:"is_core_admin,omitempty"`
	IsActive     bool              `json:"is_active,omitempty"`
	TenantID     string            `json:"tenant_id,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	PasswordHash string            `json:"-"` // Internal use only
	Salt         string            `json:"-"` // Internal use only
}

// Fund represents a private markets fund
type Fund struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Vintage   int       `json:"vintage"`
	Manager   string    `json:"manager"`
	Strategy  string    `json:"strategy"`
	Geography string    `json:"geography"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FundMetrics represents performance metrics for a fund
type FundMetrics struct {
	FundID        string    `json:"fund_id"`
	TVPI          float64   `json:"tvpi"`
	RVPI          float64   `json:"rvpi"`
	IRR           float64   `json:"irr"`
	XIRR          float64   `json:"xirr"`
	PME           float64   `json:"pme"`
	PaidInCapital float64   `json:"paid_in_capital"`
	Distributions float64   `json:"distributions"`
	ResidualValue float64   `json:"residual_value"`
	AsOfDate      time.Time `json:"as_of_date"`
}

// Bundle represents a metric bundle configuration
type Bundle struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Audience   string           `json:"audience"`
	Version    string           `json:"version"`
	Modules    []BundleModule   `json:"modules"`
	Metrics    []BundleMetric   `json:"metrics"`
	Governance BundleGovernance `json:"governance"`
}

type BundleModule struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

type BundleMetric struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Formula string `json:"formula"`
}

type BundleGovernance struct {
	Status       string    `json:"status"`
	StewardGroup string    `json:"steward_group"`
	SchemaHash   string    `json:"schema_hash"`
	SLA          BundleSLA `json:"sla"`
}

type BundleSLA struct {
	RefreshFrequency string `json:"refresh_frequency"`
	MaxLatency       string `json:"max_latency"`
}

// Dashboard represents a customizable dashboard
type Dashboard struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Widgets     []DashboardWidget `json:"widgets"`
	Layout      string            `json:"layout"`
	Theme       string            `json:"theme"`
	IsPublic    bool              `json:"is_public"`
	CreatedBy   string            `json:"created_by"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// DashboardWidget represents a widget in a dashboard
type DashboardWidget struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Title  string                 `json:"title"`
	Size   WidgetSize             `json:"size"`
	Config map[string]interface{} `json:"config"`
}

// WidgetSize represents the size of a dashboard widget
type WidgetSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DashboardTemplate represents a template for creating dashboards
type DashboardTemplate struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	IsDefault   bool              `json:"is_default"`
	Widgets     []DashboardWidget `json:"widgets"`
}

// ============================================================================
// Phase 4b: Event Sourcing Models
// ============================================================================

// Event represents a domain event (event sourcing pattern)
type Event struct {
	ID            string                 `db:"id" json:"id"`
	EventType     string                 `db:"event_type" json:"event_type"`
	AggregateID   string                 `db:"aggregate_id" json:"aggregate_id"`
	AggregateType string                 `db:"aggregate_type" json:"aggregate_type"`
	Payload       []byte                 `db:"payload" json:"payload"`
	CorrelationID string                 `db:"correlation_id" json:"correlation_id"`
	CausationID   string                 `db:"causation_id" json:"causation_id"`
	CreatedAt     time.Time              `db:"created_at" json:"created_at"`
	CreatedBy     string                 `db:"created_by" json:"created_by"`
	TenantID      string                 `db:"tenant_id" json:"tenant_id"`
	Metadata      map[string]interface{} `db:"metadata" json:"metadata"`
}

// Rule represents a semantic priority rule for gold copy publishing
type Rule struct {
	ID                 string `json:"id"`
	TenantID           string `json:"tenant_id"`
	Name               string `json:"name"`
	BusinessObject     string `json:"business_object"`
	Description        string `json:"description"`
	SemanticTerm       string `json:"semantic_term"`
	Status             string `json:"status"` // draft, testing, staging, production
	Version            int    `json:"version"`
	RuleEngine         string `json:"rule_engine"`         // e.g., "priority", "drools"
	ExpressionLanguage string `json:"expression_language"` // e.g., "JEXL"
	CreatedBy          string `json:"created_by"`
}

// Template represents a rule template for gold copy publishing
type Template struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	TemplateType string   `json:"template_type"`
	Description  string   `json:"description"`
	Status       string   `json:"status"` // draft, approved, retired
	Version      int      `json:"version"`
	RuleIDs      []string `json:"rule_ids"`
	CreatedBy    string   `json:"created_by"`
}
