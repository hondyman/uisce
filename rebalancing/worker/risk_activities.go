package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"go.temporal.io/sdk/activity"
)

// AI Risk Scoring Activity (xAI)
func (a *RebalanceActivities) AIRiskScore(ctx context.Context, portfolioID string) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Scoring risk for portfolio", "portfolioID", portfolioID)

	prompt := fmt.Sprintf(`Score risk for portfolio %s. Your response must be a valid JSON object with the keys "risk_score" (float), "risk_summary" (string), and "mitigation_action" (string). Example: {"risk_score": 7.8, "risk_summary": "High concentration in tech sector.", "mitigation_action": "Sell 10%% of AAPL and MSFT"}. The risk score should be between 0 and 10.`, portfolioID)

	// Using the existing callXAI helper
	response, err := a.callXAI(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI risk score response: %w", err)
	}

	return result, nil
}

// Execute Mitigation Activity (Placeholder)
func (a *RebalanceActivities) ExecuteMitigation(ctx context.Context, riskResult map[string]interface{}) error {
	logger := activity.GetLogger(ctx)
	mitigationAction, ok := riskResult["mitigation_action"].(string)
	if !ok {
		return fmt.Errorf("invalid mitigation action in risk result")
	}

	logger.Info("Executing mitigation action", "action", mitigationAction)
	// In a real implementation, this would publish trades to Redpanda/Kafka or call another service.
	return nil
}

// Update Risk Status Activity
func (a *RebalanceActivities) UpdateRiskStatus(ctx context.Context, portfolioID string, riskResult map[string]interface{}) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating risk status", "portfolioID", portfolioID)

	id, err := parseUUID(portfolioID)
	if err != nil {
		return err
	}

	_, err = a.db.pool.Exec(ctx, `
		UPDATE portfolios
		SET risk_score = $2,
			mitigation_action = $3,
			rebalance_status = $4,
			updated_at = NOW()
		WHERE id = $1
	`, id, riskResult["risk_score"], riskResult["mitigation_action"], riskResult["risk_summary"])
	return err
}

// ============================================================================
// RISK ALPHA COMPREHENSIVE ACTIVITIES
// ============================================================================

// AIRiskScoreComprehensive - Full xAI Grok analysis (BEATS ADDEPAR)
func (a *RebalanceActivities) AIRiskScoreComprehensive(ctx context.Context, portfolioID string) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing comprehensive AI risk analysis", "portfolioID", portfolioID)

	// Fetch portfolio data
	portfolioData, err := a.FetchPortfolio(ctx, portfolioID)
	if err != nil {
		logger.Error("Failed to fetch portfolio", "error", err)
		return nil, err
	}

	// Get top 5 holdings
	topHoldings := make([]string, 0)
	for i, h := range portfolioData.Holdings {
		if i >= 5 {
			break
		}
		topHoldings = append(topHoldings, fmt.Sprintf("%s: %.2f%%", h.Symbol, (h.Shares*h.CurrentPrice)/portfolioData.AUM*100))
	}

	// Build comprehensive prompt
	prompt := fmt.Sprintf(`Analyze portfolio %s for COMPREHENSIVE risk. Return a JSON object with EXACTLY these keys:

Portfolio Context:
- Total AUM: $%.2f
- Positions: %d
- Top 5 Holdings: %v
- Constraints: Max Turnover %.1f%%, Tax Budget $%.2f

Analyze and return JSON:
{
  "risk_score": <0-10 float>,
  "confidence": <0-1 float>,
  "var_95": <percentage float>,
  "cvar_95": <percentage float>,
  "primary_risk_type": "CONCENTRATION|VAR_BREACH|LIQUIDITY|ESG|GEOPOLITICAL|OPERATIONAL",
  "severity": "LOW|MEDIUM|HIGH|CRITICAL",
  "reasoning": "detailed explanation of top 3 risks",
  "concentration_pct": <top_10_pct float>,
  "liquidity_ratio": <0-1 float>,
  "recommendations": {
    "actions": ["action1", "action2"],
    "urgency": "immediate|high|medium|low",
    "estimated_time_to_execute_hours": <integer>
  }
}

Be conservative with scores. Only score 9+ for truly critical situations.`,
		portfolioID,
		portfolioData.AUM,
		len(portfolioData.Holdings),
		topHoldings,
		portfolioData.Constraints.MaxTurnover*100,
		portfolioData.Constraints.TaxBudget,
	)

	response, err := a.callXAI(ctx, prompt)
	if err != nil {
		logger.Error("xAI call failed", "error", err)
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		logger.Error("Failed to parse AI response", "error", err, "response", response)
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	logger.Info("Risk analysis complete", "score", result["risk_score"], "severity", result["severity"])
	return result, nil
}

// AIMitigationStrategy - Generate optimal mitigation (tax-aware, liquidity-aware)
func (a *RebalanceActivities) AIMitigationStrategy(ctx context.Context, portfolioID string, riskAnalysis map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating mitigation strategy", "portfolioID", portfolioID)

	// Fetch portfolio constraints
	portfolio, err := a.FetchPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Generate optimal MITIGATION STRATEGY for portfolio %s:

Risk Assessment:
- Risk Score: %.1f/10
- Primary Risk: %s
- VaR 95%%: %.2f%%
- Concentration: %.1f%%
- Reasoning: %s

Portfolio Constraints:
- Total AUM: $%.2f
- Max Turnover: %.1f%%
- Tax-Aware: true
- Tax Budget: $%.2f
- Min Liquidity Ratio: 0.25

Return JSON:
{
  "strategy": "narrative strategy description",
  "actions": [
    {
      "type": "REBALANCE|HEDGE|LIQUIDATE",
      "ticker": "AAPL",
      "current_weight": 0.10,
      "target_weight": 0.05,
      "shares_to_trade": 1000,
      "rationale": "reduce concentration"
    }
  ],
  "estimated_tax_impact": <dollar_amount>,
  "estimated_risk_reduction": <0-10_points>,
  "estimated_execution_hours": <integer>,
  "confidence": <0-1>
}`,
		portfolioID,
		riskAnalysis["risk_score"].(float64),
		riskAnalysis["primary_risk_type"].(string),
		riskAnalysis["var_95"].(float64),
		riskAnalysis["concentration_pct"].(float64),
		riskAnalysis["reasoning"].(string),
		portfolio.AUM,
		portfolio.Constraints.MaxTurnover*100,
		portfolio.Constraints.TaxBudget,
	)

	response, err := a.callXAI(ctx, prompt)
	if err != nil {
		logger.Error("xAI mitigation call failed", "error", err)
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		logger.Error("Failed to parse mitigation strategy", "error", err)
		return nil, fmt.Errorf("failed to parse mitigation strategy: %w", err)
	}

	logger.Info("Mitigation strategy generated", "actions_count", len(result["actions"].([]interface{})))
	return result, nil
}

// ExecuteRiskMitigation - Execute mitigation trades with audit trail
func (a *RebalanceActivities) ExecuteRiskMitigation(ctx context.Context, portfolioID string, strategy map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing risk mitigation", "portfolioID", portfolioID)

	result := map[string]interface{}{
		"success":               true,
		"executed_trades":       []string{},
		"actual_risk_reduction": 0.0,
		"execution_cost":        0.0,
	}

	// For each action in strategy, execute
	actions, ok := strategy["actions"].([]interface{})
	if !ok {
		logger.Warn("No actions in strategy")
		return result, nil
	}

	for _, actionRaw := range actions {
		action := actionRaw.(map[string]interface{})
		actionType := action["type"].(string)
		ticker := action["ticker"].(string)
		shares := action["shares_to_trade"].(float64)

		// Execute trade via Kafka or broker API
		logger.Info("Executing trade", "type", actionType, "ticker", ticker, "shares", shares)

		// Publish to portfolio.trades exchange
		tradeMessage := map[string]interface{}{
			"portfolio_id": portfolioID,
			"action_type":  actionType,
			"ticker":       ticker,
			"shares":       shares,
			"timestamp":    "NOW()",
			"reason":       "risk_mitigation",
		}

		if err := a.publishToKafka(ctx, "portfolio.events", "trade.execute", tradeMessage); err != nil {
			logger.Error("Failed to publish trade", "error", err)
			result["success"] = false
			return result, err
		}

		executed := fmt.Sprintf("%s %d shares of %s", actionType, int(shares), ticker)
		result["executed_trades"] = append(result["executed_trades"].([]string), executed)
	}

	// Insert risk_mitigation_actions
	if err := a.recordMitigationActions(ctx, portfolioID, strategy); err != nil {
		logger.Error("Failed to record mitigation actions", "error", err)
		// Don't fail—trades already executed
	}

	logger.Info("Risk mitigation complete", "trades_executed", len(result["executed_trades"].([]string)))
	return result, nil
}

// CreateRiskEvent - Insert risk_events record
func (a *RebalanceActivities) CreateRiskEvent(ctx context.Context, portfolioID, tenantID, eventType, severity string, riskAnalysis map[string]interface{}, workflowID string) (string, error) {
	logger := activity.GetLogger(ctx)

	tid, err := parseUUID(tenantID)
	if err != nil {
		return "", err
	}

	reasoningJSON, _ := json.Marshal(riskAnalysis["reasoning"])
	recommendationsJSON, _ := json.Marshal(riskAnalysis["recommendations"])

	var riskEventID string
	err = a.db.pool.QueryRow(ctx, `
		INSERT INTO risk_events (
			tenant_id, portfolio_id, event_type, severity,
			risk_score, confidence_score, var_95, cvar_95,
			concentration_pct, liquidity_ratio, ai_reasoning,
			ai_recommendations, status, workflow_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'DETECTED', $13)
		RETURNING id
	`, tid, portfolioID, eventType, severity,
		riskAnalysis["risk_score"], riskAnalysis["confidence"],
		riskAnalysis["var_95"], riskAnalysis["cvar_95"],
		riskAnalysis["concentration_pct"], riskAnalysis["liquidity_ratio"],
		string(reasoningJSON), string(recommendationsJSON), workflowID,
	).Scan(&riskEventID)

	if err != nil {
		logger.Error("Failed to create risk event", "error", err)
		return "", err
	}

	logger.Info("Risk event created", "id", riskEventID)
	return riskEventID, nil
}

// UpdateRiskEventMitigated - Update risk event status to MITIGATED
func (a *RebalanceActivities) UpdateRiskEventMitigated(ctx context.Context, riskEventID string, mitigationResult map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	id, err := parseUUID(riskEventID)
	if err != nil {
		return err
	}

	mitigationJSON, _ := json.Marshal(mitigationResult["executed_trades"])

	_, err = a.db.pool.Exec(ctx, `
		UPDATE risk_events
		SET status = 'MITIGATED',
			resolved_at = NOW(),
			resolved = true,
			mitigation_actions = $2,
			updated_at = NOW()
		WHERE id = $1
	`, id, string(mitigationJSON))

	if err != nil {
		logger.Error("Failed to update risk event", "error", err)
		return err
	}

	logger.Info("Risk event updated to MITIGATED", "id", riskEventID)
	return nil
}

// Helper: recordMitigationActions
func (a *RebalanceActivities) recordMitigationActions(ctx context.Context, portfolioID string, strategy map[string]interface{}) error {
	logger := activity.GetLogger(ctx)

	actions, ok := strategy["actions"].([]interface{})
	if !ok {
		return nil
	}

	for _, actionRaw := range actions {
		action := actionRaw.(map[string]interface{})
		actionParamsJSON, _ := json.Marshal(action)

		_, err := a.db.pool.Exec(ctx, `
			INSERT INTO risk_mitigation_actions (
				risk_event_id, action_type, action_description, action_parameters, status
			) VALUES ($1, $2, $3, $4, 'PENDING')
		`, portfolioID, action["type"],
			fmt.Sprintf("Mitigation trade: %v shares of %v", action["shares_to_trade"], action["ticker"]),
			string(actionParamsJSON))

		if err != nil {
			logger.Error("Failed to record mitigation action", "error", err)
			return err
		}
	}

	logger.Info("Mitigation actions recorded", "count", len(actions))
	return nil
}

// Helper: publishToKafka publishes a message to a Kafka topic
func (a *RebalanceActivities) publishToKafka(ctx context.Context, exchange, routingKey string, message map[string]interface{}) error {
	body, _ := json.Marshal(message)
	if a.kafkaWriter == nil {
		return fmt.Errorf("kafka writer not configured")
	}
	msg := kafka.Message{Topic: exchange, Key: []byte(routingKey), Value: body, Time: time.Now()}
	if err := a.kafkaWriter.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish to kafka topic %s: %w", exchange, err)
	}
	return nil
}
