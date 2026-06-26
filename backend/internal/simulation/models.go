package simulation

import (
	"encoding/json"
	"time"
)

// ScenarioType defines the category of a simulation scenario
type ScenarioType string

const (
	ScenarioTypePosition  ScenarioType = "POSITION"
	ScenarioTypePortfolio ScenarioType = "PORTFOLIO"
	ScenarioTypeMarket    ScenarioType = "MARKET"
	ScenarioTypeClient    ScenarioType = "CLIENT"
	ScenarioTypeMixed     ScenarioType = "MIXED"
)

// ScenarioStatus defines the lifecycle state of a scenario
type ScenarioStatus string

const (
	ScenarioStatusDraft     ScenarioStatus = "DRAFT"
	ScenarioStatusRunning   ScenarioStatus = "RUNNING"
	ScenarioStatusCompleted ScenarioStatus = "COMPLETED"
	ScenarioStatusFailed    ScenarioStatus = "FAILED"
)

// DeltaType defines the type of change applied in a delta
type DeltaType string

const (
	DeltaTypePosition  DeltaType = "POSITION_DELTA"
	DeltaTypePortfolio DeltaType = "PORTFOLIO_DELTA"
	DeltaTypeMarket    DeltaType = "MARKET_DELTA"
	DeltaTypeClient    DeltaType = "CLIENT_DELTA"
	DeltaTypeRebalance DeltaType = "REBALANCE_RULE" // Rule to generate position deltas
)

// SimulationScenario represents a single What-If scenario definition
type SimulationScenario struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenantId"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	ScenarioType ScenarioType   `json:"scenarioType"`
	Status       ScenarioStatus `json:"status"`
	BaseAsOf     time.Time      `json:"baseAsOf"`
	CreatedBy    string         `json:"createdBy"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// SimulationDelta represents a specific modification applied in a scenario
type SimulationDelta struct {
	ID         string          `json:"id"`
	ScenarioID string          `json:"scenarioId"`
	BOID       string          `json:"boId"` // Business Object ID (e.g., position:TSLA)
	DeltaType  DeltaType       `json:"deltaType"`
	Changes    json.RawMessage `json:"changes"` // e.g., {"quantityPct": -0.3}
	CreatedAt  time.Time       `json:"createdAt"`
}

// SimulationRun represents an execution instance of a scenario
type SimulationRun struct {
	ID            string          `json:"id"`
	ScenarioID    string          `json:"scenarioId"`
	Status        ScenarioStatus  `json:"status"`
	StartedAt     time.Time       `json:"startedAt"`
	CompletedAt   time.Time       `json:"completedAt,omitempty"`
	EngineVersion string          `json:"engineVersion"`
	Parameters    json.RawMessage `json:"parameters"`
}

// SimulationResult captures the outcome of a simulation run
type SimulationResult struct {
	ID                string             `json:"id"`
	RunID             string             `json:"runId"`
	ScenarioID        string             `json:"scenarioId"`
	TenantID          string             `json:"tenantId"`
	Summary           json.RawMessage    `json:"summary"`           // e.g. { "navDelta": -1000, "varDelta": 500 }
	ComplianceSummary json.RawMessage    `json:"complianceSummary"` // e.g. { "violations": 2 }
	ImpactedEntities  []string           `json:"impactedEntities"`
	CreatedAt         time.Time          `json:"createdAt"`
	Metrics           []SimulationMetric `json:"metrics,omitempty"` // Detailed breakdown
}

// SimulationMetric represents a granular calculation result
type SimulationMetric struct {
	ID             string  `json:"id"`
	ResultID       string  `json:"resultId"`
	BOID           string  `json:"boId"`
	MetricName     string  `json:"metricName"` // e.g., "NAV", "VaR_95"
	BaselineValue  float64 `json:"baselineValue"`
	SimulatedValue float64 `json:"simulatedValue"`
	DeltaValue     float64 `json:"deltaValue"`
	Unit           string  `json:"unit"`
}

// SimulationComplianceIssue represents a regulatory or governance finding
type SimulationComplianceIssue struct {
	ID          string `json:"id"`
	ResultID    string `json:"resultId"`
	RuleID      string `json:"ruleId"`
	Severity    string `json:"severity"` // INFO, WARN, CRITICAL
	Description string `json:"description"`
	BOID        string `json:"boId,omitempty"`
}

// ----------------------------------------------------------------------------
// Rebalancing Extensions
// ----------------------------------------------------------------------------

// RebalanceRule defines how a portfolio should be rebalanced
type RebalanceRule struct {
	Type        string                `json:"type"` // TO_TARGET_WEIGHTS, TO_NEW_TARGET, SHIFT_ALLOCATION
	Targets     map[string]float64    `json:"targets,omitempty"`
	Shifts      map[string]float64    `json:"shifts,omitempty"`
	Constraints *RebalanceConstraints `json:"constraints,omitempty"`
}

// RebalanceConstraints defines limits on the rebalancing
type RebalanceConstraints struct {
	MaxSectorWeight     map[string]float64 `json:"maxSectorWeight,omitempty"`
	MinSectorWeight     map[string]float64 `json:"minSectorWeight,omitempty"`
	MaxAssetWeight      map[string]float64 `json:"maxAssetWeight,omitempty"`
	AvoidShortTermGains *bool              `json:"avoidShortTermGains,omitempty"`
	ESGConstraints      map[string]any     `json:"esgConstraints,omitempty"`
	Liquidity           map[string]any     `json:"liquidityConstraints,omitempty"`
}

// ----------------------------------------------------------------------------
// Market Shock Extensions
// ----------------------------------------------------------------------------

// MarketShock defines parameters for a market scenario
type MarketShock struct {
	ParallelShiftBps float64 `json:"parallelShiftBps,omitempty"` // e.g., 50 for +50bps
	EquityShockPct   float64 `json:"equityShockPct,omitempty"`   // e.g., -0.10 for -10%
	VolShockPct      float64 `json:"volShockPct,omitempty"`      // e.g., 0.20 for +20%
	FXShockPct       float64 `json:"fxShockPct,omitempty"`       // e.g., 0.05 for +5% USD strength
}

// ----------------------------------------------------------------------------
// Compliance Extensions
// ----------------------------------------------------------------------------

// ComplianceRequest defines input for compliance checking
type ComplianceRequest struct {
	TenantID  string             `json:"tenantId"`
	AsOf      time.Time          `json:"asOf"`
	Positions map[string]float64 `json:"positions"` // AssetID -> Quantity
}

// ComplianceResult details the outcome of compliance checks
// ComplianceResult details the outcome of compliance checks
type ComplianceResult struct {
	Status         string             `json:"status"` // PASSED, PASSED_WITH_WARNINGS, FAILED
	NewIssues      []ComplianceIssue  `json:"newIssues"`
	ResolvedIssues []ComplianceIssue  `json:"resolvedIssues"`
	ChangedIssues  []ComplianceIssue  `json:"changedIssues"`
	Metrics        []SimulationMetric `json:"metrics"` // Compliance-specific metrics (Concentration, Liquidity)
}

// ComplianceIssue represents a specific rule violation
// Rule expression DSL: sectorWeight("TECH") <= 0.25
type ComplianceIssue struct {
	RuleID      string `json:"ruleId"`
	Severity    string `json:"severity"` // INFO, WARN, CRITICAL
	Description string `json:"description"`
	Expression  string `json:"expression,omitempty"`
	EntityID    string `json:"entityId,omitempty"` // Affected position/portfolio
}

// ----------------------------------------------------------------------------
// Workflow / Engine Contract
// ----------------------------------------------------------------------------

// SimulationPlan represents the output from NL Intelligence
type SimulationPlan struct {
	ScenarioType string            `json:"scenarioType"` // PORTFOLIO_SIMULATION, POSITION..., etc.
	PrimaryBOID  string            `json:"primaryBoId"`
	Deltas       []SimulationDelta `json:"deltas"`
	Metrics      []string          `json:"metrics"`     // Requested metrics
	Constraints  map[string]any    `json:"constraints"` // Runtime constraints
	Explain      string            `json:"explain"`
}

// SnapshotResult represents the baseline state fetched in Step 1
type SnapshotResult struct {
	PortfolioID string             `json:"portfolioId"`
	Positions   map[string]float64 `json:"positions"`
	MarketData  map[string]float64 `json:"marketData"`
	RiskModelID string             `json:"riskModelId"`
}

// SandboxState represents the state after applying deltas in Step 2
type SandboxState struct {
	Snapshot           SnapshotResult     `json:"snapshot"`
	SimulatedPositions map[string]float64 `json:"simulatedPositions"`
	SimulatedMarket    map[string]float64 `json:"simulatedMarket"`
	ClientAdjustments  map[string]any     `json:"clientAdjustments"`
}

// ----------------------------------------------------------------------------
// Governance Contract (ChangeSets)
// ----------------------------------------------------------------------------

// ChangeSet represents the master governance record
type ChangeSet struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenantId"`
	Type             string     `json:"type"`   // REBALANCE, CONFIG, SCHEMA
	Status           string     `json:"status"` // DRAFT, PENDING_APPROVAL, APPROVED...
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	SourceScenarioID string     `json:"sourceScenarioId"`
	CreatedBy        string     `json:"createdBy"`
	CreatedAt        time.Time  `json:"createdAt"`
	ApprovedBy       *string    `json:"approvedBy,omitempty"`
	ApprovedAt       *time.Time `json:"approvedAt,omitempty"`
}

// ChangesetRebalance represents a governance proposal for a rebalance
// Linked to ChangeSet via ChangesetID
type ChangesetRebalance struct {
	ChangesetID               string  `json:"changeset_id"`
	PortfolioID               string  `json:"portfolio_id"`
	RebalanceRule             []byte  `json:"rebalance_rule"` // JSONB
	SimulationResultID        string  `json:"simulation_result_id"`
	EstimatedNAVDelta         float64 `json:"estimated_nav_delta"`
	EstimatedVaR95Delta       float64 `json:"estimated_var95_delta"`
	EstimatedTransactionCosts float64 `json:"estimated_transaction_costs"`
	EstimatedTaxImpact        float64 `json:"estimated_tax_impact"`
	EstimatedLiquidityCost    float64 `json:"estimated_liquidity_cost"`
}

// ChangesetRebalanceTrade represents a proposed trade
type ChangesetRebalanceTrade struct {
	ID             string  `json:"id"`
	ChangesetID    string  `json:"changeset_id"`
	InstrumentID   string  `json:"instrument_id"`
	Side           string  `json:"side"` // BUY, SELL
	Quantity       float64 `json:"quantity"`
	EstimatedPrice float64 `json:"estimated_price"`
	EstimatedValue float64 `json:"estimated_value"`
	EstimatedCosts float64 `json:"estimated_costs"`
	EstimatedTax   float64 `json:"estimated_tax"`
	LiquidityFlag  string  `json:"liquidity_flag"` // OK, LIMITED, CRITICAL
}
