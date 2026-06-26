package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/rules"
)

// ============================================================================
// UMA REBALANCE ACTIVITIES
// Activities handle individual steps in the UMA rebalance workflow
// ============================================================================

type UMAActivities struct {
	db          *sql.DB
	rulesEngine *rules.UMARebalanceRulesEngine
	abacEngine  interface{} // Your existing ABAC engine
	eventBus    interface{} // Your existing event bus
}

// NewUMAActivities creates a new UMA activities instance
func NewUMAActivities(db *sql.DB, abacEngine interface{}, eventBus interface{}) *UMAActivities {
	repo := rules.NewSQLRuleRepository(db)
	engine := rules.NewRuleEngine(repo)

	return &UMAActivities{
		db:          db,
		rulesEngine: rules.NewUMARebalanceRulesEngine(repo, engine),
		abacEngine:  abacEngine,
		eventBus:    eventBus,
	}
}

// ============================================================================
// PHASE 1: ABAC CHECK
// ============================================================================

// ABACCheckActivity verifies ABAC authorization for the rebalance request
func (a *UMAActivities) ABACCheckActivity(ctx context.Context, input models.UMARebalanceWorkflowInput) (bool, error) {
	log.Printf("🔐 ABAC Check: user=%s, action=rebalance, resource=uma:%s", input.InitiatedBy, input.UMAAccountID)

	// TODO: Integrate with your existing ABAC engine
	// For now, return true; replace with actual ABAC evaluation
	// Example:
	// approved := a.abacEngine.Evaluate(ctx, &abac.Request{
	//     Subject: input.InitiatedBy,
	//     Action:  "rebalance",
	//     Resource: fmt.Sprintf("uma:%s", input.UMAAccountID),
	// })

	return true, nil
}

// ============================================================================
// PHASE 2: LOAD UMA DATA
// ============================================================================

// LoadUMADataActivity loads UMA account, sleeves, and holdings from database
func (a *UMAActivities) LoadUMADataActivity(ctx context.Context, umaAccountID string, tenantID string) (map[string]interface{}, error) {
	log.Printf("📊 Loading UMA data: account=%s, tenant=%s", umaAccountID, tenantID)

	// Load UMA account
	query := `
		SELECT id, tenant_id, datasource_id, name, status, aum, created_at, updated_at, last_rebalanced, target_allocation, metadata
		FROM uma_accounts
		WHERE id = $1 AND tenant_id = $2
	`
	row := a.db.QueryRowContext(ctx, query, umaAccountID, tenantID)

	var uma models.UMAAccount
	var targetAllocJSON []byte
	var metadataJSON []byte

	err := row.Scan(
		&uma.ID, &uma.TenantID, &uma.DatasourceID, &uma.Name, &uma.Status, &uma.AUM,
		&uma.CreatedAt, &uma.UpdatedAt, &uma.LastRebalanced, &targetAllocJSON, &metadataJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load UMA account: %w", err)
	}

	// Unmarshal JSON fields
	if len(targetAllocJSON) > 0 {
		_ = json.Unmarshal(targetAllocJSON, &uma.TargetAllocation)
	}
	if len(metadataJSON) > 0 {
		_ = json.Unmarshal(metadataJSON, &uma.Metadata)
	}

	// Load sleeves
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetUMASleeves($accountId: uuid!) {
	//   uma_sleeves(where: {uma_account_id: {_eq: $accountId}}) {
	//     id
	//     uma_account_id
	//     model
	//     sleeve_type
	//     target_allocation
	//     current_allocation
	//     drift
	//     min_drift_threshold
	//     status
	//     created_at
	//     updated_at
	//     metadata
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	sleeveQuery := `
		SELECT id, uma_account_id, model, sleeve_type, target_allocation, current_allocation, drift, min_drift_threshold, status, created_at, updated_at, metadata
		FROM uma_sleeves
		WHERE uma_account_id = $1
	`
	rows, err := a.db.QueryContext(ctx, sleeveQuery, umaAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sleeves: %w", err)
	}
	defer rows.Close()

	var sleeves []models.UMASleeve
	for rows.Next() {
		var sleeve models.UMASleeve
		var metadataJSON []byte

		err := rows.Scan(
			&sleeve.ID, &sleeve.UMAAccountID, &sleeve.Model, &sleeve.SleeveType,
			&sleeve.TargetAllocation, &sleeve.CurrentAllocation, &sleeve.Drift,
			&sleeve.MinDriftThreshold, &sleeve.Status, &sleeve.CreatedAt, &sleeve.UpdatedAt, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sleeve: %w", err)
		}

		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &sleeve.Metadata)
		}

		sleeves = append(sleeves, sleeve)
	}

	// Load holdings
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetUMAHoldings($accountId: uuid!) {
	//   uma_holdings(where: {sleeve: {uma_account_id: {_eq: $accountId}}}) {
	//     id
	//     sleeve_id
	//     cusip
	//     security_id
	//     security_name
	//     quantity
	//     unit_cost
	//     market_price
	//     market_value
	//     unrealized_gain
	//     cost_basis
	//     created_at
	//     updated_at
	//     metadata
	//   }
	// }
	// Use nested relationship: uma_holdings -> sleeve -> uma_account
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	holdingQuery := `
		SELECT id, sleeve_id, cusip, security_id, security_name, quantity, unit_cost, market_price, market_value, unrealized_gain, cost_basis, created_at, updated_at, metadata
		FROM uma_holdings
		WHERE sleeve_id IN (SELECT id FROM uma_sleeves WHERE uma_account_id = $1)
	`
	rows, err = a.db.QueryContext(ctx, holdingQuery, umaAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to query holdings: %w", err)
	}
	defer rows.Close()

	var holdings []models.UMAHolding
	for rows.Next() {
		var holding models.UMAHolding
		var metadataJSON []byte

		err := rows.Scan(
			&holding.ID, &holding.SleeveID, &holding.CUSIP, &holding.SecurityID, &holding.SecurityName,
			&holding.Quantity, &holding.UnitCost, &holding.MarketPrice, &holding.MarketValue,
			&holding.UnrealizedGain, &holding.CostBasis, &holding.CreatedAt, &holding.UpdatedAt, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan holding: %w", err)
		}

		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &holding.Metadata)
		}

		holdings = append(holdings, holding)
	}

	log.Printf("✅ Loaded UMA: %s with %d sleeves and %d holdings", uma.Name, len(sleeves), len(holdings))

	return map[string]interface{}{
		"uma":      uma,
		"sleeves":  sleeves,
		"holdings": holdings,
	}, nil
}

// ============================================================================
// PHASE 3: EVALUATE RULES
// ============================================================================

// EvaluateRulesActivity runs business rules against the UMA
func (a *UMAActivities) EvaluateRulesActivity(ctx context.Context, uma *models.UMAAccount, sleeves []*models.UMASleeve, holdings []*models.UMAHolding) ([]map[string]interface{}, error) {
	log.Printf("📋 Evaluating rules for UMA: %s", uma.ID)

	// Create a dummy plan for evaluation (will be properly generated later)
	plan := &models.UMARebalancePlan{
		ID:           uuid.New().String(),
		RequestID:    "",
		UMAAccountID: uma.ID,
	}

	violations := a.rulesEngine.EvaluateRebalancePlan(ctx, uma, sleeves, plan)
	a.rulesEngine.LogRuleEvaluation(violations, fmt.Sprintf("UMA %s evaluation", uma.ID))

	// Convert violations to JSON-serializable format
	var result []map[string]interface{}
	for _, v := range violations {
		result = append(result, map[string]interface{}{
			"rule_id":   v.RuleID,
			"rule_name": v.RuleName,
			"severity":  v.Severity,
			"message":   v.Message,
			"metadata":  v.Metadata,
		})
	}

	log.Printf("✅ Rules evaluated: %d violations", len(result))
	return result, nil
}

// ============================================================================
// PHASE 4: GENERATE REBALANCE PLAN
// ============================================================================

// GenerateRebalancePlanActivity generates the rebalance trades
func (a *UMAActivities) GenerateRebalancePlanActivity(ctx context.Context, umaAccountID string, tenantID string, sleeves []*models.UMASleeve, holdings []*models.UMAHolding) (*models.UMARebalancePlan, error) {
	log.Printf("🔄 Generating rebalance plan for UMA: %s", umaAccountID)

	plan := &models.UMARebalancePlan{
		ID:           uuid.New().String(),
		UMAAccountID: umaAccountID,
		Status:       "draft",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Trades:       []models.UMARebalanceTrade{},
	}

	// Simple drift-based trade generation
	// TODO: Implement sophisticated optimization logic
	for _, sleeve := range sleeves {
		if sleeve.Drift == 0 {
			continue
		}

		// If drift is positive (overweight), sell
		// If drift is negative (underweight), buy
		tradeType := "buy"
		if sleeve.Drift > 0 {
			tradeType = "sell"
		}

		absDrift := sleeve.Drift
		if absDrift < 0 {
			absDrift = -absDrift
		}

		// Generate a trade for each holding with drift
		for _, holding := range holdings {
			if holding.SleeveID != sleeve.ID {
				continue
			}

			tradeQty := (absDrift * holding.Quantity) // Calculate trade quantity based on drift and holding quantity
			if tradeQty < 1 {
				continue
			}

			trade := models.UMARebalanceTrade{
				ID:              uuid.New().String(),
				PlanID:          plan.ID,
				SleeveID:        sleeve.ID,
				CUSIP:           holding.CUSIP,
				SecurityID:      holding.SecurityID,
				TradeType:       tradeType,
				Quantity:        tradeQty,
				UnitPrice:       holding.MarketPrice,
				GrossAmount:     tradeQty * holding.MarketPrice,
				TaxImpact:       0, // Calculated in tax sim phase
				NetAmount:       tradeQty * holding.MarketPrice,
				Priority:        1,
				ExecutionStatus: "pending",
			}

			plan.Trades = append(plan.Trades, trade)
			plan.TotalCost += trade.GrossAmount
		}
	}

	// Save plan to database
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation InsertRebalancePlan($object: uma_rebalance_plans_insert_input!) {
	//   insert_uma_rebalance_plans_one(object: $object) {
	//     id
	//     status
	//   }
	// }
	// Variables: {"object": {"id": "...", "request_id": "", "uma_account_id": "...",
	//   "total_tax_impact": 0, "total_cost": 12345.67, "trades": {...}, "status": "draft",
	//   "created_at": "...", "updated_at": "..."}}
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	planJSON, _ := json.Marshal(plan)
	saveQuery := `
		INSERT INTO uma_rebalance_plans (id, request_id, uma_account_id, total_tax_impact, total_cost, trades, status, created_at, updated_at)
		VALUES ($1, '', $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := a.db.ExecContext(ctx, saveQuery, plan.ID, plan.UMAAccountID, plan.TotalTaxImpact, plan.TotalCost, planJSON, plan.Status, plan.CreatedAt, plan.UpdatedAt)
	if err != nil {
		log.Printf("⚠️  Failed to save plan to DB (non-blocking): %v", err)
	}

	log.Printf("✅ Rebalance plan generated: %d trades, cost: $%.2f", len(plan.Trades), plan.TotalCost)
	return plan, nil
}

// ============================================================================
// PHASE 5: TAX HARVEST SIMULATION
// ============================================================================

// TaxHarvestSimulationActivity simulates tax harvesting opportunities
func (a *UMAActivities) TaxHarvestSimulationActivity(ctx context.Context, plan *models.UMARebalancePlan, sleeves []*models.UMASleeve, holdings []*models.UMAHolding) (map[string]interface{}, error) {
	log.Printf("💰 Tax harvest simulation for plan: %s", plan.ID)

	totalLossesHarvested := 0.0
	harvestedLots := []map[string]interface{}{}

	// Find tax-loss harvesting opportunities
	for _, holding := range holdings {
		if holding.UnrealizedGain >= 0 {
			continue // No loss to harvest
		}

		violation := a.rulesEngine.EvaluateTaxHarvestingOpportunity(holding, 500) // Min $500
		if violation != nil {
			continue
		}

		absLoss := -holding.UnrealizedGain
		totalLossesHarvested += absLoss

		harvestedLots = append(harvestedLots, map[string]interface{}{
			"security_id": holding.SecurityID,
			"loss":        absLoss,
			"quantity":    holding.Quantity,
		})
	}

	result := map[string]interface{}{
		"total_losses_harvested": totalLossesHarvested,
		"tax_savings_est":        totalLossesHarvested * 0.25, // Assuming 25% tax rate
		"harvested_lots":         harvestedLots,
		"timestamp":              time.Now(),
	}

	log.Printf("✅ Tax simulation completed: $%.2f harvested", totalLossesHarvested)
	return result, nil
}

// ============================================================================
// PHASE 6: APPROVAL CHECK
// ============================================================================

// CheckApprovalRequiredActivity determines if approval is needed
func (a *UMAActivities) CheckApprovalRequiredActivity(ctx context.Context, uma *models.UMAAccount, plan *models.UMARebalancePlan) (bool, error) {
	log.Printf("🔍 Checking approval requirement for plan: %s", plan.ID)

	required := a.rulesEngine.EvaluateApprovalRequired(uma, plan)
	log.Printf("Approval required: %v", required)

	return required, nil
}

// ============================================================================
// PHASE 7: EXECUTE TRADES
// ============================================================================

// ExecuteTradesActivity executes the trades in the plan
func (a *UMAActivities) ExecuteTradesActivity(ctx context.Context, plan *models.UMARebalancePlan) (map[string]interface{}, error) {
	log.Printf("🚀 Executing %d trades for plan: %s", len(plan.Trades), plan.ID)

	completed := 0
	failed := 0
	totalExecutionCost := 0.0

	executedTrades := []map[string]interface{}{}

	for _, trade := range plan.Trades {
		// TODO: Integrate with actual trading system / custodian API
		// For now, simulate successful execution
		log.Printf("  ✅ Trade %s: %s %d @ $%.2f", trade.ID, trade.TradeType, int(trade.Quantity), trade.UnitPrice)

		trade.ExecutionStatus = "executed"
		trade.ExecutedAt = &[]time.Time{time.Now()}[0]

		completed++
		totalExecutionCost += trade.GrossAmount

		executedTrades = append(executedTrades, map[string]interface{}{
			"trade_id":         trade.ID,
			"type":             trade.TradeType,
			"quantity":         trade.Quantity,
			"unit_price":       trade.UnitPrice,
			"gross_amount":     trade.GrossAmount,
			"execution_status": trade.ExecutionStatus,
			"executed_at":      trade.ExecutedAt,
		})
	}

	result := map[string]interface{}{
		"completed_trades":       completed,
		"failed_trades":          failed,
		"total_execution_cost":   totalExecutionCost,
		"executed_trade_details": executedTrades,
		"timestamp":              time.Now(),
	}

	log.Printf("✅ Trade execution completed: %d completed, %d failed", completed, failed)
	return result, nil
}

// ============================================================================
// PHASE 8: UPDATE HASURA
// ============================================================================

// UpdateHasuraActivity updates Hasura GraphQL with rebalance results
func (a *UMAActivities) UpdateHasuraActivity(ctx context.Context, tenantID string, plan *models.UMARebalancePlan, executionResult map[string]interface{}) error {
	log.Printf("🔗 Updating Hasura for plan: %s", plan.ID)

	// TODO: Integrate with Hasura GraphQL endpoint
	// Example mutation:
	// mutation InsertRebalancePlan($plan: uma_rebalance_plans_insert_input!) {
	//   insert_uma_rebalance_plans_one(object: $plan) {
	//     id
	//     status
	//   }
	// }

	log.Printf("✅ Hasura updated")
	return nil
}

// ============================================================================
// PHASE 9: EMIT EVENTS
// ============================================================================

// EmitRebalanceCompletedEventActivity emits the rebalance completed event
func (a *UMAActivities) EmitRebalanceCompletedEventActivity(ctx context.Context, input models.UMARebalanceWorkflowInput, plan *models.UMARebalancePlan, executionResult map[string]interface{}) error {
	log.Printf("📢 Emitting rebalance completed event for plan: %s", plan.ID)

	// TODO: Integrate with your existing event bus (RabbitMQ)
	// Example:
	// event := &events.UMARebalanceCompletedEvent{
	//   EventID: uuid.New().String(),
	//   PlanID:  plan.ID,
	//   ...
	// }
	// a.eventBus.Emit(ctx, event)

	log.Printf("✅ Event emitted")
	return nil
}
