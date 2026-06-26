package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// UMARebalanceRulesEngine enforces business rules for UMA rebalancing
type UMARebalanceRulesEngine struct {
	repo   RuleRepository
	engine *RuleEngine

	// Rules configuration
	MaxDriftThreshold     float64 // e.g., 0.05 = 5%
	MinTradeSize          float64 // e.g., 1000
	TaxHarvestingEnabled  bool
	RebalanceWindow       RebalanceWindow // Time window for rebalancing
	ApprovalRequiredAbove float64         // AUM threshold requiring approval
	AUMValueThreshold     float64         // e.g., 5000000
}

// RebalanceWindow defines when rebalancing can occur
type RebalanceWindow struct {
	StartTime time.Time
	EndTime   time.Time
	Timezone  string
}

// UMARebalanceRuleViolation represents a rule violation
type UMARebalanceRuleViolation struct {
	RuleID   string
	RuleName string
	Severity string // "error", "warning", "info"
	Message  string
	Metadata map[string]interface{}
}

// NewUMARebalanceRulesEngine creates a new rules engine
func NewUMARebalanceRulesEngine(repo RuleRepository, engine *RuleEngine) *UMARebalanceRulesEngine {
	return &UMARebalanceRulesEngine{
		repo:                  repo,
		engine:                engine,
		MaxDriftThreshold:     0.05, // 5%
		MinTradeSize:          1000, // $1000
		TaxHarvestingEnabled:  true,
		ApprovalRequiredAbove: 5000000, // $5M
		AUMValueThreshold:     100000,  // $100K
	}
}

// ============================================================================
// DRIFT DETECTION RULES
// ============================================================================

// EvaluateSleeveDrift checks if a sleeve has exceeded drift threshold
func (e *UMARebalanceRulesEngine) EvaluateSleeveDrift(sleeve *models.UMASleeve) *UMARebalanceRuleViolation {
	if sleeve.CurrentAllocation == 0 && sleeve.TargetAllocation == 0 {
		return nil
	}

	drift := sleeve.CurrentAllocation - sleeve.TargetAllocation
	absDrift := drift
	if drift < 0 {
		absDrift = -drift
	}

	// Use sleeve-specific drift threshold if set, otherwise engine default
	threshold := sleeve.MinDriftThreshold
	if threshold == 0 {
		threshold = e.MaxDriftThreshold
	}

	if absDrift > threshold {
		return &UMARebalanceRuleViolation{
			RuleID:   "drift_exceeded",
			RuleName: "Drift Threshold Exceeded",
			Severity: "warning",
			Message:  fmt.Sprintf("Sleeve %s drift %.2f%% exceeds threshold %.2f%%", sleeve.SleeveType, drift*100, threshold*100),
			Metadata: map[string]interface{}{
				"sleeve_id":   sleeve.ID,
				"sleeve_type": sleeve.SleeveType,
				"target":      sleeve.TargetAllocation,
				"current":     sleeve.CurrentAllocation,
				"drift":       drift,
				"threshold":   threshold,
			},
		}
	}

	return nil
}

// EvaluateAllocationBalance checks that sleeve allocations sum to ~100%
func (e *UMARebalanceRulesEngine) EvaluateAllocationBalance(sleeves []*models.UMASleeve) *UMARebalanceRuleViolation {
	totalTarget := 0.0
	totalCurrent := 0.0

	for _, s := range sleeves {
		totalTarget += s.TargetAllocation
		totalCurrent += s.CurrentAllocation
	}

	tolerance := 0.01 // 1%
	if totalTarget < (1.0-tolerance) || totalTarget > (1.0+tolerance) {
		return &UMARebalanceRuleViolation{
			RuleID:   "allocation_balance",
			RuleName: "Allocation Balance Invalid",
			Severity: "error",
			Message:  fmt.Sprintf("Target allocations sum to %.2f%%, should be 100%%", totalTarget*100),
			Metadata: map[string]interface{}{
				"total_target": totalTarget,
				"tolerance":    tolerance,
			},
		}
	}

	return nil
}

// ============================================================================
// TRADE VALIDATION RULES
// ============================================================================

// EvaluateTradeSize checks if a trade meets minimum size requirements
func (e *UMARebalanceRulesEngine) EvaluateTradeSize(trade *models.UMARebalanceTrade) *UMARebalanceRuleViolation {
	if trade.GrossAmount < e.MinTradeSize {
		return &UMARebalanceRuleViolation{
			RuleID:   "trade_too_small",
			RuleName: "Trade Size Below Minimum",
			Severity: "warning",
			Message:  fmt.Sprintf("Trade amount $%.2f below minimum $%.2f", trade.GrossAmount, e.MinTradeSize),
			Metadata: map[string]interface{}{
				"trade_id": trade.ID,
				"amount":   trade.GrossAmount,
				"min_size": e.MinTradeSize,
			},
		}
	}

	return nil
}

// EvaluateTaxLotSufficiency checks that enough tax lots exist for sale
func (e *UMARebalanceRulesEngine) EvaluateTaxLotSufficiency(trade *models.UMARebalanceTrade, holding *models.UMAHolding) *UMARebalanceRuleViolation {
	if trade.TradeType != "sell" {
		return nil
	}

	if trade.Quantity > holding.Quantity {
		return &UMARebalanceRuleViolation{
			RuleID:   "insufficient_holdings",
			RuleName: "Insufficient Holdings for Sale",
			Severity: "error",
			Message:  fmt.Sprintf("Cannot sell %.2f shares; only %.2f available", trade.Quantity, holding.Quantity),
			Metadata: map[string]interface{}{
				"trade_id":      trade.ID,
				"requested_qty": trade.Quantity,
				"available_qty": holding.Quantity,
				"security_id":   holding.SecurityID,
			},
		}
	}

	return nil
}

// EvaluatePriceDeviation checks if quoted price deviates significantly from market
func (e *UMARebalanceRulesEngine) EvaluatePriceDeviation(trade *models.UMARebalanceTrade, currentMarketPrice float64) *UMARebalanceRuleViolation {
	if currentMarketPrice == 0 {
		return nil
	}

	deviationPct := (trade.UnitPrice - currentMarketPrice) / currentMarketPrice
	if deviationPct < 0 {
		deviationPct = -deviationPct
	}

	maxDeviation := 0.02 // 2%
	if deviationPct > maxDeviation {
		return &UMARebalanceRuleViolation{
			RuleID:   "price_deviation",
			RuleName: "Price Deviation Exceeded",
			Severity: "warning",
			Message:  fmt.Sprintf("Quoted price $%.2f deviates %.2f%% from market $%.2f", trade.UnitPrice, deviationPct*100, currentMarketPrice),
			Metadata: map[string]interface{}{
				"trade_id":      trade.ID,
				"quoted_price":  trade.UnitPrice,
				"market_price":  currentMarketPrice,
				"deviation_pct": deviationPct,
				"max_deviation": maxDeviation,
			},
		}
	}

	return nil
}

// ============================================================================
// APPROVAL RULES
// ============================================================================

// EvaluateApprovalRequired determines if a rebalance plan needs approval
func (e *UMARebalanceRulesEngine) EvaluateApprovalRequired(uma *models.UMAAccount, plan *models.UMARebalancePlan) bool {
	// Approval required if:
	// 1. AUM exceeds threshold
	if uma.AUM > e.ApprovalRequiredAbove {
		return true
	}

	// 2. Total trade value exceeds threshold
	if plan.TotalCost > 100000 { // $100K
		return true
	}

	// 3. Tax impact is significant
	if plan.TotalTaxImpact < -50000 { // -$50K (harvesting opportunity)
		return true
	}

	return false
}

// ============================================================================
// TAX HARVESTING RULES
// ============================================================================

// EvaluateTaxHarvestingOpportunity checks if harvesting makes sense
func (e *UMARebalanceRulesEngine) EvaluateTaxHarvestingOpportunity(holding *models.UMAHolding, minThreshold float64) *UMARebalanceRuleViolation {
	if !e.TaxHarvestingEnabled {
		return nil
	}

	if holding.UnrealizedGain >= 0 {
		// No loss to harvest
		return nil
	}

	absLoss := -holding.UnrealizedGain
	if absLoss < minThreshold {
		return &UMARebalanceRuleViolation{
			RuleID:   "tax_loss_immaterial",
			RuleName: "Tax Loss Below Threshold",
			Severity: "info",
			Message:  fmt.Sprintf("Unrealized loss $%.2f below harvesting threshold $%.2f", absLoss, minThreshold),
			Metadata: map[string]interface{}{
				"security_id": holding.SecurityID,
				"loss_amount": absLoss,
				"threshold":   minThreshold,
			},
		}
	}

	return nil
}

// EvaluateWashSaleRisk checks for wash sale violations (simplified)
func (e *UMARebalanceRulesEngine) EvaluateWashSaleRisk(soldAt time.Time, otherTrades []*models.UMARebalanceTrade, sameCUSIP string) *UMARebalanceRuleViolation {
	washSaleWindow := 61 * 24 * time.Hour // 61 days

	for _, trade := range otherTrades {
		if trade.TradeType != "buy" {
			continue
		}
		if trade.CUSIP != sameCUSIP {
			continue
		}
		// Check if trade date is within wash sale window
		// This is simplified; real implementation would check ExecutedAt
		timeDiff := soldAt
		if timeDiff.After(soldAt) {
			timeDiff = timeDiff.Add(-washSaleWindow)
		}
		if timeDiff.Before(soldAt.Add(washSaleWindow)) {
			return &UMARebalanceRuleViolation{
				RuleID:   "wash_sale_risk",
				RuleName: "Potential Wash Sale",
				Severity: "warning",
				Message:  fmt.Sprintf("Buy order for %s within 61-day wash sale window of sale", sameCUSIP),
				Metadata: map[string]interface{}{
					"cusip":  sameCUSIP,
					"window": washSaleWindow.String(),
				},
			}
		}
	}

	return nil
}

// ============================================================================
// EXECUTION TIMING RULES
// ============================================================================

// EvaluateExecutionTiming checks if rebalance can execute during market hours
func (e *UMARebalanceRulesEngine) EvaluateExecutionTiming(now time.Time) *UMARebalanceRuleViolation {
	// For now, always allow; extend with market hours logic
	// Typical NYSE hours: 9:30 AM - 4:00 PM ET
	// This is a simplified placeholder
	return nil
}

// ============================================================================
// COMPREHENSIVE RULE EVALUATION
// ============================================================================

// EvaluateRebalancePlan runs all rules against a complete plan
func (e *UMARebalanceRulesEngine) EvaluateRebalancePlan(ctx context.Context, uma *models.UMAAccount, sleeves []*models.UMASleeve, plan *models.UMARebalancePlan) []UMARebalanceRuleViolation {
	var violations []UMARebalanceRuleViolation

	// 1. Evaluate Dynamic Rules from Database
	if e.repo != nil && e.engine != nil {
		rules, err := e.repo.ListRules(ctx, "")
		if err == nil {
			// Convert structs to map[string]interface{} for CEL
			var umaMap map[string]interface{}
			umaJSON, _ := json.Marshal(uma)
			json.Unmarshal(umaJSON, &umaMap)

			input := map[string]interface{}{
				"uma": umaMap,
				// "sleeves": sleeves, // TODO: Handle slice conversion if needed
				// "plan":    plan,    // TODO: Handle struct conversion if needed
			}
			for _, rule := range rules {
				if !rule.Enabled {
					continue
				}
				// Evaluate: true means compliant, false means violation
				compliant, err := e.engine.Evaluate(ctx, rule.Expression, input)
				if err != nil {
					log.Printf("Error evaluating rule %s: %v", rule.Name, err)
					continue
				}
				if !compliant {
					violations = append(violations, UMARebalanceRuleViolation{
						RuleID:   rule.ID.String(),
						RuleName: rule.Name,
						Severity: rule.Severity,
						Message:  fmt.Sprintf("Violation of rule: %s", rule.Name),
						Metadata: map[string]interface{}{
							"description": rule.Description,
						},
					})
				}
			}
		} else {
			log.Printf("Error listing rules: %v", err)
		}
	}

	// 2. Evaluate Hardcoded Rules (Legacy/Fallback)

	// Sleeve-level rules
	for _, sleeve := range sleeves {
		if v := e.EvaluateSleeveDrift(sleeve); v != nil {
			violations = append(violations, *v)
		}
	}

	// Allocation balance
	if v := e.EvaluateAllocationBalance(sleeves); v != nil {
		violations = append(violations, *v)
	}

	// Trade-level rules
	for _, trade := range plan.Trades {
		if v := e.EvaluateTradeSize(&trade); v != nil {
			violations = append(violations, *v)
		}
	}

	// Approval
	if e.EvaluateApprovalRequired(uma, plan) {
		log.Printf("✅ Approval required for UMA %s (AUM: $%.0f)", uma.ID, uma.AUM)
	}

	return violations
}

// LogRuleEvaluation logs rule evaluation results
func (e *UMARebalanceRulesEngine) LogRuleEvaluation(violations []UMARebalanceRuleViolation, context string) {
	if len(violations) == 0 {
		log.Printf("✅ All rules passed: %s", context)
		return
	}

	for _, v := range violations {
		var level string
		switch v.Severity {
		case "error":
			level = "❌ "
		case "warning":
			level = "⚠️  "
		default:
			level = "ℹ️  "
		}
		log.Printf("%s [%s] %s: %s", level, v.RuleID, v.RuleName, v.Message)
	}
}
