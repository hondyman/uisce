// Package household provides multi-level household aggregation for wealth management.
// It supports complex entity hierarchies, tax-lot tracking, and consolidated reporting.
package household

import (
"context"
"encoding/json"
"fmt"
"sort"
"sync"
"time"

"github.com/google/uuid"
"github.com/jmoiron/sqlx"
)

// EntityType defines the type of entity in a household hierarchy
type EntityType string

const (
EntityHousehold   EntityType = "household"
EntityFamily      EntityType = "family"
EntityIndividual  EntityType = "individual"
EntityTrust       EntityType = "trust"
EntityFoundation  EntityType = "foundation"
EntityCorporation EntityType = "corporation"
EntityPartnership EntityType = "partnership"
EntityAccount     EntityType = "account"
EntitySubAccount  EntityType = "sub_account"
)

// AggregationLevel defines the level of aggregation
type AggregationLevel string

const (
LevelHousehold   AggregationLevel = "household"
LevelFamily      AggregationLevel = "family"
LevelEntity      AggregationLevel = "entity"
LevelAccount     AggregationLevel = "account"
LevelAssetClass  AggregationLevel = "asset_class"
LevelSector      AggregationLevel = "sector"
LevelSecurity    AggregationLevel = "security"
LevelTaxLot      AggregationLevel = "tax_lot"
)

// HouseholdEntity represents an entity in the household hierarchy
type HouseholdEntity struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	ParentID      *uuid.UUID             `json:"parent_id,omitempty" db:"parent_id"`
	Name          string                 `json:"name" db:"name"`
	Type          EntityType             `json:"type" db:"type"`
	TaxStatus     string                 `json:"tax_status" db:"tax_status"`
	TaxJurisdiction string              `json:"tax_jurisdiction" db:"tax_jurisdiction"`
	Currency      string                 `json:"currency" db:"currency"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
	Children      []*HouseholdEntity     `json:"children,omitempty"`
	Accounts      []*Account             `json:"accounts,omitempty"`
}

// Account represents a financial account
type Account struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	EntityID      uuid.UUID  `json:"entity_id" db:"entity_id"`
	AccountNumber string     `json:"account_number" db:"account_number"`
	AccountType   string     `json:"account_type" db:"account_type"`
	Custodian     string     `json:"custodian" db:"custodian"`
	Currency      string     `json:"currency" db:"currency"`
	TaxStatus     string     `json:"tax_status" db:"tax_status"`
	OpenDate      time.Time  `json:"open_date" db:"open_date"`
	CloseDate     *time.Time `json:"close_date,omitempty" db:"close_date"`
	IsActive      bool       `json:"is_active" db:"is_active"`
}

// AggregatedPosition represents an aggregated position across accounts
type AggregatedPosition struct {
	SecurityID     string    `json:"security_id"`
	SecurityName   string    `json:"security_name"`
	AssetClass     string    `json:"asset_class"`
	Sector         string    `json:"sector,omitempty"`
	TotalQuantity  float64   `json:"total_quantity"`
	TotalMarketValue float64 `json:"total_market_value"`
	TotalCostBasis float64   `json:"total_cost_basis"`
	UnrealizedGain float64   `json:"unrealized_gain"`
	UnrealizedGainPct float64 `json:"unrealized_gain_pct"`
	Weight         float64   `json:"weight"`
	TaxLots        []TaxLot  `json:"tax_lots,omitempty"`
	AccountBreakdown []AccountPosition `json:"account_breakdown,omitempty"`
}

// TaxLot represents a single tax lot
type TaxLot struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	AccountID     uuid.UUID  `json:"account_id" db:"account_id"`
	SecurityID    string     `json:"security_id" db:"security_id"`
	AcquisitionDate time.Time `json:"acquisition_date" db:"acquisition_date"`
	Quantity      float64    `json:"quantity" db:"quantity"`
	CostBasis     float64    `json:"cost_basis" db:"cost_basis"`
	CurrentValue  float64    `json:"current_value" db:"current_value"`
	UnrealizedGain float64   `json:"unrealized_gain"`
	HoldingPeriod string     `json:"holding_period"` // short_term, long_term
	DaysHeld      int        `json:"days_held"`
}

// AccountPosition represents a position in a specific account
type AccountPosition struct {
	AccountID     uuid.UUID `json:"account_id"`
	AccountName   string    `json:"account_name"`
	Quantity      float64   `json:"quantity"`
	MarketValue   float64   `json:"market_value"`
	CostBasis     float64   `json:"cost_basis"`
	Weight        float64   `json:"weight"`
}

// AggregationResult contains the aggregated household data
type AggregationResult struct {
	ID              uuid.UUID                   `json:"id"`
	TenantID        string                      `json:"tenant_id"`
	HouseholdID     uuid.UUID                   `json:"household_id"`
	AsOfDate        time.Time                   `json:"as_of_date"`
	Currency        string                      `json:"currency"`
	
	// Total values
	TotalMarketValue float64                   `json:"total_market_value"`
	TotalCostBasis  float64                    `json:"total_cost_basis"`
	TotalUnrealizedGain float64                `json:"total_unrealized_gain"`
	TotalRealizedGain float64                  `json:"total_realized_gain"`
	
	// Hierarchy breakdown
	EntityBreakdown []EntitySummary            `json:"entity_breakdown"`
	AccountBreakdown []AccountSummary          `json:"account_breakdown"`
	
	// Asset allocation
	AssetClassAllocation []AllocationItem      `json:"asset_class_allocation"`
	SectorAllocation []AllocationItem          `json:"sector_allocation"`
	GeographyAllocation []AllocationItem       `json:"geography_allocation"`
	
	// Positions
	Positions       []AggregatedPosition       `json:"positions"`
	
	// Tax analysis
	TaxAnalysis     *TaxAnalysis               `json:"tax_analysis,omitempty"`
	
	// Metadata
	GeneratedAt     time.Time                  `json:"generated_at"`
	Metadata        map[string]interface{}     `json:"metadata,omitempty"`
}

// EntitySummary summarizes an entity's holdings
type EntitySummary struct {
EntityID      uuid.UUID `json:"entity_id"`
EntityName    string    `json:"entity_name"`
EntityType    EntityType `json:"entity_type"`
MarketValue   float64   `json:"market_value"`
CostBasis     float64   `json:"cost_basis"`
UnrealizedGain float64  `json:"unrealized_gain"`
Weight        float64   `json:"weight"`
AccountCount  int       `json:"account_count"`
}

// AccountSummary summarizes an account's holdings
type AccountSummary struct {
	AccountID     uuid.UUID `json:"account_id"`
	AccountName   string    `json:"account_name"`
	AccountType   string    `json:"account_type"`
	Custodian     string    `json:"custodian"`
	MarketValue   float64   `json:"market_value"`
	CostBasis     float64   `json:"cost_basis"`
	UnrealizedGain float64  `json:"unrealized_gain"`
	Weight        float64   `json:"weight"`
	PositionCount int       `json:"position_count"`
}

// AllocationItem represents an allocation bucket
type AllocationItem struct {
	Name        string  `json:"name"`
	MarketValue float64 `json:"market_value"`
	Weight      float64 `json:"weight"`
	Target      float64 `json:"target,omitempty"`
	Deviation   float64 `json:"deviation,omitempty"`
}

// TaxAnalysis provides tax-related analysis
type TaxAnalysis struct {
	ShortTermGains float64           `json:"short_term_gains"`
	ShortTermLosses float64          `json:"short_term_losses"`
	LongTermGains  float64           `json:"long_term_gains"`
	LongTermLosses float64           `json:"long_term_losses"`
	NetShortTerm   float64           `json:"net_short_term"`
	NetLongTerm    float64           `json:"net_long_term"`
	TaxLossHarvestingOpportunities []TaxLossOpportunity `json:"tax_loss_harvesting,omitempty"`
	WashSaleRisk   []WashSaleRisk    `json:"wash_sale_risk,omitempty"`
}

// TaxLossOpportunity identifies tax-loss harvesting opportunities
type TaxLossOpportunity struct {
	SecurityID    string  `json:"security_id"`
	SecurityName  string  `json:"security_name"`
	UnrealizedLoss float64 `json:"unrealized_loss"`
	CostBasis     float64 `json:"cost_basis"`
	MarketValue   float64 `json:"market_value"`
	HoldingPeriod string  `json:"holding_period"`
	TaxSavings    float64 `json:"estimated_tax_savings"`
}

// WashSaleRisk identifies positions at risk of wash sale
type WashSaleRisk struct {
	SecurityID      string    `json:"security_id"`
	SecurityName    string    `json:"security_name"`
	RecentSaleDate  time.Time `json:"recent_sale_date"`
	DaysSinceSale   int       `json:"days_since_sale"`
	RiskLevel       string    `json:"risk_level"` // high, medium, low
}

// AggregationConfig configures the aggregation
type AggregationConfig struct {
	HouseholdID      uuid.UUID          `json:"household_id"`
	AsOfDate         time.Time          `json:"as_of_date"`
	IncludeTaxLots   bool               `json:"include_tax_lots"`
	IncludeTaxAnalysis bool             `json:"include_tax_analysis"`
	Levels           []AggregationLevel `json:"levels"`
	Currency         string             `json:"currency"`
	IncludeInactive  bool               `json:"include_inactive"`
}

// HouseholdEngine provides household aggregation capabilities
type HouseholdEngine struct {
	db    *sqlx.DB
	cache sync.Map
}

// NewHouseholdEngine creates a new household aggregation engine
func NewHouseholdEngine(db *sqlx.DB) *HouseholdEngine {
	return &HouseholdEngine{db: db}
}

// Aggregate performs household-level aggregation
func (e *HouseholdEngine) Aggregate(ctx context.Context, tenantID string, config AggregationConfig) (*AggregationResult, error) {
	// Load household hierarchy
	household, err := e.getHouseholdHierarchy(ctx, tenantID, config.HouseholdID)
	if err != nil {
		return nil, fmt.Errorf("failed to load household hierarchy: %w", err)
	}

	// Load all accounts for the household
	accounts, err := e.getHouseholdAccounts(ctx, tenantID, config.HouseholdID, config.IncludeInactive)
	if err != nil {
		return nil, fmt.Errorf("failed to load accounts: %w", err)
	}

	// Load positions for all accounts
	positions, err := e.getPositions(ctx, tenantID, accounts, config.AsOfDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load positions: %w", err)
	}

	// Load tax lots if requested
	var taxLots []TaxLot
	if config.IncludeTaxLots {
		taxLots, err = e.getTaxLots(ctx, tenantID, accounts)
		if err != nil {
			return nil, fmt.Errorf("failed to load tax lots: %w", err)
		}
	}

	// Aggregate positions
	aggregatedPositions := e.aggregatePositions(positions, taxLots, config)

	// Calculate totals
	totalMarketValue := 0.0
	totalCostBasis := 0.0
	for _, pos := range aggregatedPositions {
		totalMarketValue += pos.TotalMarketValue
		totalCostBasis += pos.TotalCostBasis
	}

	// Calculate weights
	for i := range aggregatedPositions {
		if totalMarketValue > 0 {
			aggregatedPositions[i].Weight = aggregatedPositions[i].TotalMarketValue / totalMarketValue * 100
		}
	}

	// Build entity breakdown
	entityBreakdown := e.buildEntityBreakdown(household, positions, totalMarketValue)

	// Build account breakdown
	accountBreakdown := e.buildAccountBreakdown(accounts, positions, totalMarketValue)

	// Build asset allocation
	assetAllocation := e.buildAssetAllocation(aggregatedPositions, totalMarketValue)
	sectorAllocation := e.buildSectorAllocation(aggregatedPositions, totalMarketValue)

	result := &AggregationResult{
		ID:                uuid.New(),
		TenantID:          tenantID,
		HouseholdID:       config.HouseholdID,
		AsOfDate:          config.AsOfDate,
		Currency:          config.Currency,
		TotalMarketValue:  totalMarketValue,
		TotalCostBasis:    totalCostBasis,
		TotalUnrealizedGain: totalMarketValue - totalCostBasis,
		EntityBreakdown:   entityBreakdown,
		AccountBreakdown:  accountBreakdown,
		AssetClassAllocation: assetAllocation,
		SectorAllocation:  sectorAllocation,
		Positions:         aggregatedPositions,
		GeneratedAt:       time.Now(),
		Metadata: map[string]interface{}{
			"account_count":  len(accounts),
			"position_count": len(aggregatedPositions),
		},
	}

	// Add tax analysis if requested
	if config.IncludeTaxAnalysis && len(taxLots) > 0 {
		result.TaxAnalysis = e.analyzeTax(taxLots, config.AsOfDate)
	}

	return result, nil
}

// getHouseholdHierarchy loads the household entity hierarchy
func (e *HouseholdEngine) getHouseholdHierarchy(ctx context.Context, tenantID string, householdID uuid.UUID) (*HouseholdEntity, error) {
	query := `
		WITH RECURSIVE hierarchy AS (
SELECT id, tenant_id, parent_id, name, type, tax_status, tax_jurisdiction, currency, metadata, 0 as depth
FROM household_entities
WHERE id = $1 AND tenant_id = $2
UNION ALL
SELECT e.id, e.tenant_id, e.parent_id, e.name, e.type, e.tax_status, e.tax_jurisdiction, e.currency, e.metadata, h.depth + 1
FROM household_entities e
INNER JOIN hierarchy h ON e.parent_id = h.id
WHERE e.tenant_id = $2
)
		SELECT * FROM hierarchy ORDER BY depth
	`

	var entities []HouseholdEntity
	err := e.db.SelectContext(ctx, &entities, query, householdID, tenantID)
	if err != nil {
		return nil, err
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("household not found: %s", householdID)
	}

	// Build tree structure
	entityMap := make(map[uuid.UUID]*HouseholdEntity)
	for i := range entities {
		entityMap[entities[i].ID] = &entities[i]
	}

	for i := range entities {
		if entities[i].ParentID != nil {
			parent := entityMap[*entities[i].ParentID]
			if parent != nil {
				parent.Children = append(parent.Children, &entities[i])
			}
		}
	}

	return &entities[0], nil
}

// getHouseholdAccounts loads all accounts for a household
func (e *HouseholdEngine) getHouseholdAccounts(ctx context.Context, tenantID string, householdID uuid.UUID, includeInactive bool) ([]Account, error) {
	query := `
		WITH RECURSIVE entity_ids AS (
SELECT id FROM household_entities WHERE id = $1 AND tenant_id = $2
UNION ALL
SELECT e.id FROM household_entities e
INNER JOIN entity_ids p ON e.parent_id = p.id
WHERE e.tenant_id = $2
)
		SELECT a.* FROM accounts a
		INNER JOIN entity_ids e ON a.entity_id = e.id
		WHERE ($3 = true OR a.is_active = true)
		ORDER BY a.account_number
	`

	var accounts []Account
	err := e.db.SelectContext(ctx, &accounts, query, householdID, tenantID, includeInactive)
	return accounts, err
}

// getPositions loads positions for accounts
func (e *HouseholdEngine) getPositions(ctx context.Context, tenantID string, accounts []Account, asOfDate time.Time) (map[uuid.UUID][]AccountPosition, error) {
	if len(accounts) == 0 {
		return make(map[uuid.UUID][]AccountPosition), nil
	}

	accountIDs := make([]uuid.UUID, len(accounts))
	for i, a := range accounts {
		accountIDs[i] = a.ID
	}

	query := `
		SELECT 
			p.account_id,
			p.security_id,
			s.name as security_name,
			s.asset_class,
			s.sector,
			p.quantity,
			p.market_value,
			p.cost_basis
		FROM positions p
		JOIN securities s ON p.security_id = s.id
		WHERE p.account_id = ANY($1) AND p.as_of_date = $2
	`

	type positionRow struct {
		AccountID    uuid.UUID `db:"account_id"`
		SecurityID   string    `db:"security_id"`
		SecurityName string    `db:"security_name"`
		AssetClass   string    `db:"asset_class"`
		Sector       string    `db:"sector"`
		Quantity     float64   `db:"quantity"`
		MarketValue  float64   `db:"market_value"`
		CostBasis    float64   `db:"cost_basis"`
	}

	var rows []positionRow
	err := e.db.SelectContext(ctx, &rows, query, accountIDs, asOfDate)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]AccountPosition)
	for _, r := range rows {
		result[r.AccountID] = append(result[r.AccountID], AccountPosition{
AccountID:   r.AccountID,
Quantity:    r.Quantity,
MarketValue: r.MarketValue,
CostBasis:   r.CostBasis,
})
	}

	return result, nil
}

// getTaxLots loads tax lots for accounts
func (e *HouseholdEngine) getTaxLots(ctx context.Context, tenantID string, accounts []Account) ([]TaxLot, error) {
	if len(accounts) == 0 {
		return nil, nil
	}

	accountIDs := make([]uuid.UUID, len(accounts))
	for i, a := range accounts {
		accountIDs[i] = a.ID
	}

	query := `
		SELECT id, account_id, security_id, acquisition_date, quantity, cost_basis, current_value
		FROM tax_lots
		WHERE account_id = ANY($1) AND quantity > 0
		ORDER BY acquisition_date
	`

	var lots []TaxLot
	err := e.db.SelectContext(ctx, &lots, query, accountIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for i := range lots {
		lots[i].DaysHeld = int(now.Sub(lots[i].AcquisitionDate).Hours() / 24)
		if lots[i].DaysHeld > 365 {
			lots[i].HoldingPeriod = "long_term"
		} else {
			lots[i].HoldingPeriod = "short_term"
		}
		lots[i].UnrealizedGain = lots[i].CurrentValue - lots[i].CostBasis
	}

	return lots, nil
}

// aggregatePositions aggregates positions across accounts
func (e *HouseholdEngine) aggregatePositions(positions map[uuid.UUID][]AccountPosition, taxLots []TaxLot, config AggregationConfig) []AggregatedPosition {
	securityMap := make(map[string]*AggregatedPosition)

	for accountID, acctPositions := range positions {
		for _, pos := range acctPositions {
			secID := fmt.Sprintf("%s", pos.AccountID) // simplified - would use security_id
			if existing, ok := securityMap[secID]; ok {
				existing.TotalQuantity += pos.Quantity
				existing.TotalMarketValue += pos.MarketValue
				existing.TotalCostBasis += pos.CostBasis
				existing.AccountBreakdown = append(existing.AccountBreakdown, AccountPosition{
AccountID:   accountID,
Quantity:    pos.Quantity,
MarketValue: pos.MarketValue,
CostBasis:   pos.CostBasis,
})
			} else {
				securityMap[secID] = &AggregatedPosition{
					SecurityID:       secID,
					TotalQuantity:    pos.Quantity,
					TotalMarketValue: pos.MarketValue,
					TotalCostBasis:   pos.CostBasis,
					AccountBreakdown: []AccountPosition{{
						AccountID:   accountID,
						Quantity:    pos.Quantity,
						MarketValue: pos.MarketValue,
						CostBasis:   pos.CostBasis,
					}},
				}
			}
		}
	}

	// Add tax lots to positions
	if config.IncludeTaxLots {
		lotsByAccount := make(map[uuid.UUID][]TaxLot)
		for _, lot := range taxLots {
			lotsByAccount[lot.AccountID] = append(lotsByAccount[lot.AccountID], lot)
		}
		// Assign lots to positions...
	}

	// Convert to slice and calculate gains
	result := make([]AggregatedPosition, 0, len(securityMap))
	for _, pos := range securityMap {
		pos.UnrealizedGain = pos.TotalMarketValue - pos.TotalCostBasis
		if pos.TotalCostBasis > 0 {
			pos.UnrealizedGainPct = pos.UnrealizedGain / pos.TotalCostBasis * 100
		}
		result = append(result, *pos)
	}

	// Sort by market value descending
	sort.Slice(result, func(i, j int) bool {
return result[i].TotalMarketValue > result[j].TotalMarketValue
	})

	return result
}

// buildEntityBreakdown builds entity-level summary
func (e *HouseholdEngine) buildEntityBreakdown(household *HouseholdEntity, positions map[uuid.UUID][]AccountPosition, totalValue float64) []EntitySummary {
	summaries := make([]EntitySummary, 0)
	e.buildEntityBreakdownRecursive(household, positions, totalValue, &summaries)
	return summaries
}

func (e *HouseholdEngine) buildEntityBreakdownRecursive(entity *HouseholdEntity, positions map[uuid.UUID][]AccountPosition, totalValue float64, summaries *[]EntitySummary) {
	marketValue := 0.0
	costBasis := 0.0
	accountCount := 0

	for _, account := range entity.Accounts {
		if acctPositions, ok := positions[account.ID]; ok {
			for _, pos := range acctPositions {
				marketValue += pos.MarketValue
				costBasis += pos.CostBasis
			}
			accountCount++
		}
	}

	for _, child := range entity.Children {
		e.buildEntityBreakdownRecursive(child, positions, totalValue, summaries)
	}

	if marketValue > 0 || len(entity.Children) > 0 {
		weight := 0.0
		if totalValue > 0 {
			weight = marketValue / totalValue * 100
		}
		*summaries = append(*summaries, EntitySummary{
EntityID:       entity.ID,
EntityName:     entity.Name,
EntityType:     entity.Type,
MarketValue:    marketValue,
CostBasis:      costBasis,
UnrealizedGain: marketValue - costBasis,
Weight:         weight,
AccountCount:   accountCount,
})
	}
}

// buildAccountBreakdown builds account-level summary
func (e *HouseholdEngine) buildAccountBreakdown(accounts []Account, positions map[uuid.UUID][]AccountPosition, totalValue float64) []AccountSummary {
	summaries := make([]AccountSummary, 0, len(accounts))

	for _, account := range accounts {
		marketValue := 0.0
		costBasis := 0.0
		posCount := 0

		if acctPositions, ok := positions[account.ID]; ok {
			for _, pos := range acctPositions {
				marketValue += pos.MarketValue
				costBasis += pos.CostBasis
				posCount++
			}
		}

		weight := 0.0
		if totalValue > 0 {
			weight = marketValue / totalValue * 100
		}

		summaries = append(summaries, AccountSummary{
AccountID:      account.ID,
AccountName:    account.AccountNumber,
AccountType:    account.AccountType,
Custodian:      account.Custodian,
MarketValue:    marketValue,
CostBasis:      costBasis,
UnrealizedGain: marketValue - costBasis,
Weight:         weight,
PositionCount:  posCount,
})
	}

	sort.Slice(summaries, func(i, j int) bool {
return summaries[i].MarketValue > summaries[j].MarketValue
	})

	return summaries
}

// buildAssetAllocation builds asset class allocation
func (e *HouseholdEngine) buildAssetAllocation(positions []AggregatedPosition, totalValue float64) []AllocationItem {
	allocationMap := make(map[string]float64)

	for _, pos := range positions {
		assetClass := pos.AssetClass
		if assetClass == "" {
			assetClass = "Other"
		}
		allocationMap[assetClass] += pos.TotalMarketValue
	}

	allocation := make([]AllocationItem, 0, len(allocationMap))
	for name, value := range allocationMap {
		weight := 0.0
		if totalValue > 0 {
			weight = value / totalValue * 100
		}
		allocation = append(allocation, AllocationItem{
Name:        name,
MarketValue: value,
Weight:      weight,
})
	}

	sort.Slice(allocation, func(i, j int) bool {
return allocation[i].MarketValue > allocation[j].MarketValue
	})

	return allocation
}

// buildSectorAllocation builds sector allocation
func (e *HouseholdEngine) buildSectorAllocation(positions []AggregatedPosition, totalValue float64) []AllocationItem {
	allocationMap := make(map[string]float64)

	for _, pos := range positions {
		sector := pos.Sector
		if sector == "" {
			sector = "Other"
		}
		allocationMap[sector] += pos.TotalMarketValue
	}

	allocation := make([]AllocationItem, 0, len(allocationMap))
	for name, value := range allocationMap {
		weight := 0.0
		if totalValue > 0 {
			weight = value / totalValue * 100
		}
		allocation = append(allocation, AllocationItem{
Name:        name,
MarketValue: value,
Weight:      weight,
})
	}

	sort.Slice(allocation, func(i, j int) bool {
return allocation[i].MarketValue > allocation[j].MarketValue
	})

	return allocation
}

// analyzeTax performs tax analysis on tax lots
func (e *HouseholdEngine) analyzeTax(taxLots []TaxLot, asOfDate time.Time) *TaxAnalysis {
	analysis := &TaxAnalysis{
		TaxLossHarvestingOpportunities: make([]TaxLossOpportunity, 0),
		WashSaleRisk:                   make([]WashSaleRisk, 0),
	}

	for _, lot := range taxLots {
		gain := lot.CurrentValue - lot.CostBasis

		if lot.HoldingPeriod == "short_term" {
			if gain > 0 {
				analysis.ShortTermGains += gain
			} else {
				analysis.ShortTermLosses += -gain
			}
		} else {
			if gain > 0 {
				analysis.LongTermGains += gain
			} else {
				analysis.LongTermLosses += -gain
			}
		}

		// Identify tax loss harvesting opportunities
		if gain < -100 { // Minimum threshold
			analysis.TaxLossHarvestingOpportunities = append(analysis.TaxLossHarvestingOpportunities, TaxLossOpportunity{
SecurityID:     lot.SecurityID,
UnrealizedLoss: -gain,
CostBasis:      lot.CostBasis,
MarketValue:    lot.CurrentValue,
HoldingPeriod:  lot.HoldingPeriod,
TaxSavings:     -gain * 0.35, // Estimated at 35% marginal rate
})
		}
	}

	analysis.NetShortTerm = analysis.ShortTermGains - analysis.ShortTermLosses
	analysis.NetLongTerm = analysis.LongTermGains - analysis.LongTermLosses

	// Sort tax loss opportunities by size
	sort.Slice(analysis.TaxLossHarvestingOpportunities, func(i, j int) bool {
return analysis.TaxLossHarvestingOpportunities[i].UnrealizedLoss > analysis.TaxLossHarvestingOpportunities[j].UnrealizedLoss
	})

	return analysis
}

// ToJSON marshals result to JSON
func (r *AggregationResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// GetHierarchyPath returns the path from root to a specific entity
func (e *HouseholdEngine) GetHierarchyPath(ctx context.Context, tenantID string, entityID uuid.UUID) ([]HouseholdEntity, error) {
	query := `
		WITH RECURSIVE path AS (
SELECT id, tenant_id, parent_id, name, type, tax_status, tax_jurisdiction, currency, metadata, 0 as depth
FROM household_entities
WHERE id = $1 AND tenant_id = $2
UNION ALL
SELECT e.id, e.tenant_id, e.parent_id, e.name, e.type, e.tax_status, e.tax_jurisdiction, e.currency, e.metadata, p.depth - 1
FROM household_entities e
INNER JOIN path p ON e.id = p.parent_id
WHERE e.tenant_id = $2
)
		SELECT * FROM path ORDER BY depth
	`

	var entities []HouseholdEntity
	err := e.db.SelectContext(ctx, &entities, query, entityID, tenantID)
	return entities, err
}
