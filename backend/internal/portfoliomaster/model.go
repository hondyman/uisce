package portfoliomaster

import (
	"time"

	"github.com/google/uuid"
)

// SourceRegistry is a vendor of market/portfolio data (Bloomberg, Refinitiv, etc.)
type SourceRegistry struct {
	ID             uuid.UUID  `json:"id"`
	SourceName     string     `json:"source_name"`
	SourceCode     string     `json:"source_code"`
	SourceType     string     `json:"source_type"`
	EndpointURL    string     `json:"endpoint_url,omitempty"`
	IsActive       bool       `json:"is_active"`
	PriorityScore  int        `json:"priority_score"`
	ConfidenceBase int        `json:"confidence_base"`
	AccountTypes   []string   `json:"account_types"`
	AssetClasses   []string   `json:"asset_classes"`
	Regions        []string   `json:"regions"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	CoreID         *uuid.UUID `json:"core_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PortfolioSource is raw data ingested from a single source system
// for one portfolio position (security). This is the staging record
// before the golden record is synthesised.
type PortfolioSource struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	SourceRegistryID uuid.UUID  `json:"source_registry_id"`
	PortfolioID      string     `json:"portfolio_id"`
	AccountType      string     `json:"account_type"` // retail | institutional | private_wealth | private_markets
	SecurityID       string     `json:"security_id"`
	SecurityName     string     `json:"security_name,omitempty"`
	Quantity         *float64   `json:"quantity,omitempty"`
	Price            *float64   `json:"price,omitempty"`
	MarketValue      *float64   `json:"market_value,omitempty"`
	Currency         string     `json:"currency"`
	AssetClass       string     `json:"asset_class,omitempty"`
	Country          string     `json:"country,omitempty"`
	Region           string     `json:"region,omitempty"`
	ConfidenceScore  int        `json:"confidence_score"`
	IngestionJobID   *uuid.UUID `json:"ingestion_job_id,omitempty"`
	SourceTimestamp  *time.Time `json:"source_timestamp,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	ValidFrom        time.Time  `json:"valid_from"`
	ValidTo          *time.Time `json:"valid_to,omitempty"`
}

// PortfolioGolden is the authoritative golden record for one portfolio position.
// It is synthesised from PortfolioSource rows using source preferences.
type PortfolioGolden struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	PortfolioID     string    `json:"portfolio_id"`
	AccountType     string    `json:"account_type"`
	SecurityID      string    `json:"security_id"`
	SecurityName    string    `json:"security_name,omitempty"`
	Quantity        float64   `json:"quantity"`
	Price           float64   `json:"price"`
	MarketValue     float64   `json:"market_value"` // Derived: Quantity * Price
	Currency        string    `json:"currency"`
	AssetClass      string    `json:"asset_class,omitempty"`
	Country         string    `json:"country,omitempty"`
	Region          string    `json:"region,omitempty"`
	ConfidenceScore int       `json:"confidence_score"`
	// SourceSystems maps field name → source that supplied it
	// e.g. {"price": "Bloomberg", "quantity": "FactSet"}
	SourceSystems       map[string]string `json:"source_systems"`
	ContributingSources []uuid.UUID       `json:"contributing_sources,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	CreatedBy           uuid.UUID         `json:"created_by"`
	UpdatedBy           *uuid.UUID        `json:"updated_by,omitempty"`
	ValidFrom           time.Time         `json:"valid_from"`
	ValidTo             *time.Time        `json:"valid_to,omitempty"`
}

// ImpactSimulationResult describes the effect of changing a source preference
// on the current set of golden records.
type ImpactSimulationResult struct {
	PreferenceID      uuid.UUID        `json:"preference_id"`
	AsOfDate          time.Time        `json:"as_of_date"`
	AffectedPositions int              `json:"affected_positions"`
	ConfidenceBefore  float64          `json:"confidence_before"`
	ConfidenceAfter   float64          `json:"confidence_after"`
	ConfidenceDelta   float64          `json:"confidence_delta"`
	BusinessImpact    string           `json:"business_impact"` // none | low | moderate | high
	ChangedPositions  []PositionChange `json:"changed_positions"`
	SimulatedAt       time.Time        `json:"simulated_at"`
}

// PositionChange describes how one golden record would change under the simulation
type PositionChange struct {
	PortfolioID   string `json:"portfolio_id"`
	SecurityID    string `json:"security_id"`
	Field         string `json:"field"` // price | quantity
	OldSource     string `json:"old_source"`
	NewSource     string `json:"new_source"`
	OldConfidence int    `json:"old_confidence"`
	NewConfidence int    `json:"new_confidence"`
}
