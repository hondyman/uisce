package models

import (
	"time"
)

// UMAAccount represents a Unified Managed Account
type UMAAccount struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	DatasourceID     string                 `json:"datasource_id"`
	Name             string                 `json:"name"`
	Status           string                 `json:"status"` // active, inactive, archived
	AUM              float64                `json:"aum"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	LastRebalanced   *time.Time             `json:"last_rebalanced,omitempty"`
	TargetAllocation map[string]float64     `json:"target_allocation"` // e.g., {"equities": 0.60, "fixed_income": 0.30}
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// UMASleeve represents a sleeve within a UMA account (e.g., US Equities, Alts, Fixed Income)
type UMASleeve struct {
	ID                string                 `json:"id"`
	UMAAccountID      string                 `json:"uma_account_id"`
	Model             string                 `json:"model"`             // e.g., "Growth", "Conservative", "Alternatives"
	SleeveType        string                 `json:"sleeve_type"`       // e.g., "equities", "fixed_income", "alternatives"
	TargetAllocation  float64                `json:"target_allocation"` // 0.6 = 60%
	CurrentAllocation float64                `json:"current_allocation"`
	Drift             float64                `json:"drift"`               // current - target
	MinDriftThreshold float64                `json:"min_drift_threshold"` // e.g., 0.05 = 5%
	Status            string                 `json:"status"`              // active, pending, rebalancing
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// UMAHolding represents a single holding (security) within a sleeve
type UMAHolding struct {
	ID             string                 `json:"id"`
	SleeveID       string                 `json:"sleeve_id"`
	CUSIP          string                 `json:"cusip"`
	SecurityID     string                 `json:"security_id"`
	SecurityName   string                 `json:"security_name"`
	Quantity       float64                `json:"quantity"`
	UnitCost       float64                `json:"unit_cost"`
	MarketPrice    float64                `json:"market_price"`
	MarketValue    float64                `json:"market_value"`
	UnrealizedGain float64                `json:"unrealized_gain"`
	CostBasis      float64                `json:"cost_basis"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UMARebalanceRequest represents a request to rebalance a UMA account
type UMARebalanceRequest struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	UMAAccountID string                 `json:"uma_account_id"`
	RequestType  string                 `json:"request_type"` // "drift", "manual", "scheduled"
	Reason       string                 `json:"reason"`
	InitiatedBy  string                 `json:"initiated_by"`
	Status       string                 `json:"status"` // pending, approved, executing, completed, failed
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UMARebalancePlan represents the proposed rebalancing trades
type UMARebalancePlan struct {
	ID             string                 `json:"id"`
	RequestID      string                 `json:"request_id"`
	UMAAccountID   string                 `json:"uma_account_id"`
	TotalTaxImpact float64                `json:"total_tax_impact"`
	TotalCost      float64                `json:"total_cost"`
	Trades         []UMARebalanceTrade    `json:"trades"`
	ApprovedAt     *time.Time             `json:"approved_at,omitempty"`
	ApprovedBy     string                 `json:"approved_by,omitempty"`
	ExecutedAt     *time.Time             `json:"executed_at,omitempty"`
	Status         string                 `json:"status"` // draft, pending_approval, approved, executing, completed
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UMARebalanceTrade represents a single trade in a rebalance plan
type UMARebalanceTrade struct {
	ID              string                 `json:"id"`
	PlanID          string                 `json:"plan_id"`
	SleeveID        string                 `json:"sleeve_id"`
	CUSIP           string                 `json:"cusip"`
	SecurityID      string                 `json:"security_id"`
	TradeType       string                 `json:"trade_type"` // "buy", "sell"
	Quantity        float64                `json:"quantity"`
	UnitPrice       float64                `json:"unit_price"`
	GrossAmount     float64                `json:"gross_amount"`
	TaxImpact       float64                `json:"tax_impact"`
	NetAmount       float64                `json:"net_amount"`
	Priority        int                    `json:"priority"`
	LotSelection    []TaxLot               `json:"lot_selection,omitempty"` // For tax-aware lot selection
	ExecutionStatus string                 `json:"execution_status"`        // pending, executed, failed
	ExecutedAt      *time.Time             `json:"executed_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// TaxLot represents a specific tax lot for tax-loss harvesting
type TaxLot struct {
	ID              string    `json:"id"`
	HoldingID       string    `json:"holding_id"`
	AcquisitionDate time.Time `json:"acquisition_date"`
	Quantity        float64   `json:"quantity"`
	CostBasis       float64   `json:"cost_basis"`
	UnrealizedGain  float64   `json:"unrealized_gain"`
	SelectedFor     string    `json:"selected_for,omitempty"` // "sell", "harvest"
	Reason          string    `json:"reason,omitempty"`       // tax_loss, gain_realization, drift
}

// UMARebalanceHistory tracks completed rebalances
type UMARebalanceHistory struct {
	ID              string                 `json:"id"`
	PlanID          string                 `json:"plan_id"`
	UMAAccountID    string                 `json:"uma_account_id"`
	CompletedAt     time.Time              `json:"completed_at"`
	TotalTradeCount int                    `json:"total_trade_count"`
	SuccessCount    int                    `json:"success_count"`
	FailureCount    int                    `json:"failure_count"`
	TotalTaxImpact  float64                `json:"total_tax_impact"`
	TotalCost       float64                `json:"total_cost"`
	PreDrift        map[string]float64     `json:"pre_drift"`  // Pre-rebalance drift by sleeve
	PostDrift       map[string]float64     `json:"post_drift"` // Post-rebalance drift by sleeve
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// UMARebalanceWorkflowInput is the input to the UMA Rebalance workflow
type UMARebalanceWorkflowInput struct {
	RequestID    string                 `json:"request_id"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	UMAAccountID string                 `json:"uma_account_id"`
	RequestType  string                 `json:"request_type"` // "drift", "manual", "scheduled"
	Reason       string                 `json:"reason"`
	InitiatedBy  string                 `json:"initiated_by"`
	EventData    map[string]interface{} `json:"event_data,omitempty"`
}

// UMARebalanceWorkflowState tracks workflow state
type UMARebalanceWorkflowState struct {
	RequestID        string
	UMAAccountID     string
	CurrentPhase     string // abac_check, load_data, tax_simulate, generate_plan, approval, execution, completion
	ABACApproved     bool
	PlanID           string
	ApprovalStatus   string
	ExecutionDetails map[string]interface{}
	Errors           []string
	LastUpdated      time.Time
}
