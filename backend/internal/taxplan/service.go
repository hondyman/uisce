package taxplan

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

type Service struct {
	DB           *sqlx.DB
	hasuraClient HasuraClient
}

func NewService(db *sqlx.DB) *Service {
	return &Service{DB: db}
}

// NewServiceWithHasura creates a new tax plan service with Hasura support
func NewServiceWithHasura(db *sqlx.DB, hasuraClient HasuraClient) *Service {
	return &Service{DB: db, hasuraClient: hasuraClient}
}

// DetectOpportunities scans for tax optimization opportunities
func (s *Service) DetectOpportunities(ctx context.Context, clientID uuid.UUID) ([]TaxOpportunity, error) {
	var opportunities []TaxOpportunity

	// 1. Detect Tax-Loss Harvesting Opportunities
	tlhOpps, err := s.detectTaxLossHarvesting(ctx, clientID)
	if err == nil {
		opportunities = append(opportunities, tlhOpps...)
	}

	// 2. Detect Roth Conversion Opportunities
	rothOpp, err := s.detectRothConversion(ctx, clientID)
	if err == nil && rothOpp != nil {
		opportunities = append(opportunities, *rothOpp)
	}

	return opportunities, nil
}

// detectTaxLossHarvesting finds positions with unrealized losses > $3k
func (s *Service) detectTaxLossHarvesting(ctx context.Context, clientID uuid.UUID) ([]TaxOpportunity, error) {
	lots, err := s.getTaxLotsWithLossesRecords(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tax lots: %w", err)
	}

	if len(lots) == 0 {
		return nil, nil
	}

	// Aggregate losses by ticker
	totalLoss := 0.0
	positionsMap := make(map[string]float64)

	for _, lot := range lots {
		totalLoss += lot.UnrealizedGL
		positionsMap[lot.Ticker] += lot.UnrealizedGL
	}

	// Build positions array
	positions := []map[string]interface{}{}
	for ticker, loss := range positionsMap {
		positions = append(positions, map[string]interface{}{
			"ticker": ticker,
			"loss":   loss,
		})
	}

	positionsJSON, _ := json.Marshal(positions)
	actionsJSON, _ := json.Marshal(map[string]interface{}{
		"action":     "SELL_POSITIONS",
		"positions":  positions,
		"total_loss": totalLoss,
	})

	// Assume 37% tax bracket for estimation
	estimatedSavings := -totalLoss * 0.37

	opp := TaxOpportunity{
		OpportunityID:            uuid.New(),
		ClientID:                 clientID,
		OpportunityType:          OpportunityTaxLossHarvest,
		DetectedDate:             time.Now(),
		EstimatedSavings:         estimatedSavings,
		ImplementationComplexity: "MEDIUM",
		TimeSensitivity:          "BEFORE_YEAR_END",
		RecommendedActions:       actionsJSON,
		PositionsAffected:        positionsJSON,
		Status:                   "IDENTIFIED",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	// Save to database
	err = s.saveTaxOpportunityRecord(ctx, &opp)
	if err != nil {
		return nil, fmt.Errorf("failed to save opportunity: %w", err)
	}

	return []TaxOpportunity{opp}, nil
}

// detectRothConversion finds low-income years for Roth conversions
func (s *Service) detectRothConversion(ctx context.Context, clientID uuid.UUID) (*TaxOpportunity, error) {
	profile, err := s.getClientTaxProfileRecord(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// Check if income is down 30% from average
	if profile.CurrentYearIncome >= profile.AvgAnnualIncome*0.7 {
		return nil, nil // No opportunity
	}

	// Must have Traditional IRA and be under 60
	if !profile.HasTraditionalIRA || profile.Age >= 60 {
		return nil, nil
	}

	conversionAmount := 50000.0 // Default conversion amount
	currentBracket := profile.EstimatedBracket
	projectedFutureBracket := profile.AvgTaxBracket

	estimatedSavings := conversionAmount * (projectedFutureBracket - currentBracket)

	actionsJSON, _ := json.Marshal(map[string]interface{}{
		"action":            "ROTH_CONVERSION",
		"conversion_amount": conversionAmount,
		"current_bracket":   currentBracket,
		"future_bracket":    projectedFutureBracket,
	})

	opp := TaxOpportunity{
		OpportunityID:            uuid.New(),
		ClientID:                 clientID,
		OpportunityType:          OpportunityRothConversion,
		DetectedDate:             time.Now(),
		EstimatedSavings:         estimatedSavings,
		ImplementationComplexity: "LOW",
		TimeSensitivity:          "BEFORE_YEAR_END",
		RecommendedActions:       actionsJSON,
		Status:                   "IDENTIFIED",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	err = s.saveTaxOpportunityRecord(ctx, &opp)
	if err != nil {
		return nil, fmt.Errorf("failed to save opportunity: %w", err)
	}

	return &opp, nil
}

// GetClientOpportunities retrieves all opportunities for a client
func (s *Service) GetClientOpportunities(ctx context.Context, clientID uuid.UUID) ([]TaxOpportunity, error) {
	opportunities, err := s.getClientOpportunitiesRecords(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get opportunities: %w", err)
	}

	return opportunities, nil
}

// Helper methods for SQL operations with Hasura fallback

// getTaxLotsWithLossesRecords retrieves tax lots with unrealized losses
// TODO: Migrate to Hasura GraphQL query with where clause:
//
//	query GetTaxLotsWithLosses($client_id: uuid!, $min_loss: numeric!) {
//	  tax_lots(where: {
//	    client_id: {_eq: $client_id},
//	    unrealized_gain_loss: {_lt: $min_loss},
//	    is_wash_sale: {_eq: false}
//	  }) {
//	    lot_id
//	    client_id
//	    account_id
//	    ticker
//	    quantity
//	    purchase_date
//	    purchase_price
//	    current_price
//	    unrealized_gain_loss
//	    is_wash_sale
//	    wash_sale_period_end
//	    created_at
//	  }
//	}
func (s *Service) getTaxLotsWithLossesRecords(ctx context.Context, clientID uuid.UUID) ([]TaxLot, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetTaxLotsWithLosses($clientId: uuid!) {
	//   tax_lots(where: {
	//     client_id: {_eq: $clientId},
	//     unrealized_gain_loss: {_lt: -3000},
	//     is_wash_sale: {_eq: false}
	//   }) {
	//     lot_id account_id ticker quantity purchase_date purchase_price
	//     current_price unrealized_gain_loss is_wash_sale wash_sale_period_end
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var lots []TaxLot
	query := `
		SELECT * FROM tax_lots 
		WHERE client_id = $1 
		  AND unrealized_gain_loss < -3000
		  AND is_wash_sale = FALSE
	`
	err := s.DB.SelectContext(ctx, &lots, query, clientID)
	return lots, err
}

// saveTaxOpportunityRecord inserts a tax optimization opportunity
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation InsertTaxOpportunity($object: tax_optimization_opportunities_insert_input!) {
//	  insert_tax_optimization_opportunities_one(object: $object) {
//	    opportunity_id
//	    client_id
//	    opportunity_type
//	    detected_date
//	    estimated_tax_savings
//	    implementation_complexity
//	    time_sensitivity
//	    recommended_actions
//	    positions_affected
//	    status
//	    created_at
//	    updated_at
//	  }
//	}
func (s *Service) saveTaxOpportunityRecord(ctx context.Context, opp *TaxOpportunity) error {
	// Use SQL for NamedExec pattern with complex struct
	query := `
		INSERT INTO tax_optimization_opportunities (
opportunity_id, client_id, opportunity_type, detected_date,
estimated_tax_savings, implementation_complexity, time_sensitivity,
recommended_actions, positions_affected, status, created_at, updated_at
) VALUES (
:opportunity_id, :client_id, :opportunity_type, :detected_date,
:estimated_tax_savings, :implementation_complexity, :time_sensitivity,
:recommended_actions, :positions_affected, :status, :created_at, :updated_at
)`

	_, err := s.DB.NamedExecContext(ctx, query, opp)
	return err
}

// getClientTaxProfileRecord retrieves a client's tax profile
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetClientTaxProfile($client_id: uuid!) {
//	  client_tax_profiles(where: {client_id: {_eq: $client_id}}, limit: 1) {
//	    client_id
//	    current_year_income
//	    avg_annual_income
//	    estimated_bracket
//	    avg_tax_bracket
//	    has_traditional_ira
//	    has_roth_ira
//	    age
//	    filing_status
//	    created_at
//	    updated_at
//	  }
//	}
func (s *Service) getClientTaxProfileRecord(ctx context.Context, clientID uuid.UUID) (*ClientTaxProfile, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetClientTaxProfile($clientId: uuid!) {
	//   client_tax_profiles(where: {client_id: {_eq: $clientId}}, limit: 1) {
	//     client_id current_year_income avg_annual_income estimated_bracket
	//     avg_tax_bracket has_traditional_ira has_roth_ira age filing_status
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var profile ClientTaxProfile
	query := "SELECT * FROM client_tax_profiles WHERE client_id = $1"
	err := s.DB.GetContext(ctx, &profile, query, clientID)
	return &profile, err
}

// getClientOpportunitiesRecords retrieves all tax opportunities for a client
// TODO: Migrate to Hasura GraphQL query with order_by:
//
//	query GetClientOpportunities($client_id: uuid!) {
//	  tax_optimization_opportunities(
//	    where: {client_id: {_eq: $client_id}},
//	    order_by: {detected_date: desc}
//	  ) {
//	    opportunity_id
//	    client_id
//	    opportunity_type
//	    detected_date
//	    estimated_tax_savings
//	    implementation_complexity
//	    time_sensitivity
//	    recommended_actions
//	    positions_affected
//	    status
//	    created_at
//	    updated_at
//	  }
//	}
func (s *Service) getClientOpportunitiesRecords(ctx context.Context, clientID uuid.UUID) ([]TaxOpportunity, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetClientOpportunities($clientId: uuid!) {
	//   tax_optimization_opportunities(where: {client_id: {_eq: $clientId}}, order_by: {detected_date: desc}) {
	//     opportunity_id client_id opportunity_type detected_date
	//     estimated_tax_savings implementation_complexity time_sensitivity
	//     recommended_actions positions_affected status
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var opportunities []TaxOpportunity
	query := `
SELECT * FROM tax_optimization_opportunities 
WHERE client_id = $1 
ORDER BY detected_date DESC
`
	err := s.DB.SelectContext(ctx, &opportunities, query, clientID)
	return opportunities, err
}
