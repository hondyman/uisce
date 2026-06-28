package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

type RebalanceActivities struct {
	kafkaWriter   *kafka.Writer
	hasuraURL     string
	xaiAPIKey     string
	finnhubAPIKey string
}

// Activity: Fetch Portfolio
func (a *RebalanceActivities) FetchPortfolio(ctx context.Context, portfolioID string) (*Portfolio, error) {
	return a.fetchPortfolio(ctx, portfolioID)
}

// Activity: Fetch Real-Time Prices from Finnhub
func (a *RebalanceActivities) FetchRealTimePrices(ctx context.Context, symbols []string) (map[string]float64, error) {
	prices := make(map[string]float64)
	client := &http.Client{Timeout: 10 * time.Second}

	for _, symbol := range symbols {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://finnhub.io/api/v1/quote?symbol=%s&token=%s", symbol, a.finnhubAPIKey), nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var result struct {
			CurrentPrice float64 `json:"c"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		prices[symbol] = result.CurrentPrice
	}

	return prices, nil
}

// Activity 1: Calculate Portfolio Drift (Refactored)
func (a *RebalanceActivities) AnalyzeDrift(ctx context.Context, portfolio Portfolio) (float64, error) {
	currentAlloc := make(map[string]float64)
	totalValue := 0.0

	for _, h := range portfolio.Holdings {
		value := h.Shares * h.CurrentPrice
		currentAlloc[h.Sector] += value
		totalValue += value
	}

	// Normalize to percentages
	if totalValue == 0 {
		return 0, nil // Avoid division by zero
	}
	for sector := range currentAlloc {
		currentAlloc[sector] = (currentAlloc[sector] / totalValue) * 100
	}

	// Calculate drift
	drift := 0.0
	for sector, target := range portfolio.TargetModel {
		current := currentAlloc[sector]
		drift += abs(current - target)
	}

	return drift, nil
}

// Activity: Fetch Historical Prices from Finnhub
func (a *RebalanceActivities) FetchHistoricalPrices(ctx context.Context, symbols []string, start, end time.Time) (map[string][]HistoricalPrice, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	from := start.Unix()
	to := end.Unix()
	allPrices := make(map[string][]HistoricalPrice)

	for _, symbol := range symbols {
		url := fmt.Sprintf("https://finnhub.io/api/v1/stock/candle?symbol=%s&resolution=D&from=%d&to=%d&token=%s", symbol, from, to, a.finnhubAPIKey)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for %s: %w", symbol, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch data for %s: %w", symbol, err)
		}
		defer resp.Body.Close()

		var result struct {
			ClosePrices []float64 `json:"c"`
			Timestamps  []int64   `json:"t"`
			Status      string    `json:"s"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response for %s: %w", symbol, err)
		}

		if result.Status != "ok" {
			continue // Skip symbols with no data
		}

		prices := make([]HistoricalPrice, len(result.ClosePrices))
		for i := range result.ClosePrices {
			prices[i] = HistoricalPrice{
				Date:  time.Unix(result.Timestamps[i], 0),
				Price: result.ClosePrices[i],
			}
		}
		allPrices[symbol] = prices
	}

	return allPrices, nil
}

// Activity: Generate Rebalance Summary
func (a *RebalanceActivities) GenerateRebalanceSummary(ctx context.Context, plan RebalancePlan) (string, error) {
	prompt := fmt.Sprintf(`Given the following rebalance plan, write a brief, client-friendly summary (2-3 sentences) explaining what was done and why. Focus on the key outcomes like drift reduction and tax savings.

- Initial Drift: %.2f%%
- Expected Drift after rebalance: %.2f%%
- Tax Savings: $%.2f
- Rationale: %s

Summary:`, plan.CurrentDrift, plan.ExpectedDrift, plan.TaxImpact.TaxSavingsVsRandom, plan.Rationale)

	return a.callXAI(ctx, prompt)
}

// Activity: Update Plan With Summary
func (a *RebalanceActivities) UpdatePlanWithSummary(ctx context.Context, planID, summary string) error {
	mutation := `
		mutation UpdatePlanSummary($id: uuid!, $summary: String!) {
			update_rebalance_plans_by_pk(pk_columns: {id: $id}, _set: {summary: $summary}) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"id":      planID,
		"summary": summary,
	}

	return a.hasuraMutate(ctx, mutation, variables)
}

func buildRebalancePrompt(p Portfolio) string {
	personalizationPrompt := ""
	if p.Constraints.ESGPreference != "" {
		personalizationPrompt += fmt.Sprintf("\n- ESG Preference: %s", p.Constraints.ESGPreference)
	}
	if p.Constraints.RiskAppetite != "" {
		personalizationPrompt += fmt.Sprintf("\n- Risk Appetite: %s", p.Constraints.RiskAppetite)
	}
	if len(p.Constraints.ForbiddenSectors) > 0 {
		personalizationPrompt += fmt.Sprintf("\n- Forbidden Sectors: %s", strings.Join(p.Constraints.ForbiddenSectors, ", "))
	}
	if personalizationPrompt != "" {
		personalizationPrompt = "\n\nPersonalization:" + personalizationPrompt
	}

	policyPrompt := ""
	if p.PolicyDocument != "" {
		policyPrompt = fmt.Sprintf("\n\nCompliance Policy (Must be followed):\n%s", p.PolicyDocument)
	}

	return fmt.Sprintf(`Analyze and rebalance this portfolio. You are a portfolio manager and a compliance officer. Return ONLY valid JSON with no markdown or code blocks.

Portfolio: $%.2f AUM
Current Holdings: %s
Target Model: %s
Constraints: max_trade=$%.0f, min_trade=$%.0f, tax_budget=$%.0f%s%s

Return JSON:
{
  "proposed_trades": [{"symbol": "AAPL", "action": "SELL", "shares": 100, "estimated_price": 150, "notional": 15000, "tax_lot_id": "lot_1", "reason": "reduce tech exposure"}],
  "current_drift": 8.5,
  "expected_drift": 2.1,
  "tax_impact": {"short_term_gains": 5000, "long_term_gains": 10000, "total_tax": 5550, "tax_savings_vs_random": 2000, "strategy": "HIFO"},
  "rationale": "Rebalance to target by selling tech, buying bonds",
  "confidence": 92.5
}`,
		p.AUM,
		toJSON(p.Holdings[:min(5, len(p.Holdings))]), // First 5 holdings
		toJSON(p.TargetModel),
		p.Constraints.MaxTradeSize,
		p.Constraints.MinTradeSize,
		p.Constraints.TaxBudget,
		personalizationPrompt,
		policyPrompt,
	)
}

// Activity: AI Rebalance
// Uses xAI to generate a proposed rebalance plan as JSON.
func (a *RebalanceActivities) AIRebalance(ctx context.Context, portfolio Portfolio) (*RebalancePlan, error) {
	prompt := buildRebalancePrompt(portfolio)
	response, err := a.callXAI(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var plan RebalancePlan
	if err := json.Unmarshal([]byte(response), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	plan.PortfolioID = portfolio.ID
	plan.Timestamp = time.Now()
	if strings.TrimSpace(plan.Status) == "" {
		plan.Status = "proposed"
	}

	return &plan, nil
}

func (a *RebalanceActivities) callXAI(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": "grok-beta",
		"messages": []map[string]string{
			{"role": "system", "content": "You are an expert portfolio manager. Return only valid JSON, no markdown."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.x.ai/v1/chat/completions", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+a.xaiAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// Clean markdown code blocks if present
	content := result.Choices[0].Message.Content
	content = strings.TrimPrefix(content, "```json\n")
	content = strings.TrimPrefix(content, "```\n")
	content = strings.TrimSuffix(content, "\n```")

	content = strings.TrimSpace(content)
	return content, nil
}

// Activity 3: ABAC Authorization
func (a *RebalanceActivities) ABACCheck(ctx context.Context, action, resource, portfolioID string) (bool, error) {
	// Extract from context (set by API layer)
	userID, _ := ctx.Value("user_id").(string)
	tenantID, _ := ctx.Value("tenant_id").(string)

	if userID == "" || tenantID == "" {
		return false, fmt.Errorf("missing auth context")
	}

	// ABAC rules
	rules := []func() bool{
		func() bool { return isPortfolioManager(userID) },
		func() bool { return belongsToTenant(portfolioID, tenantID) },
		func() bool { return isTradingHours() },
	}

	for _, rule := range rules {
		if !rule() {
			return false, nil
		}
	}

	// Audit log
	a.insertAuditLog(ctx, userID, tenantID, action, resource, portfolioID, true)

	return true, nil
}

// Activity 4: Validate Plan
func (a *RebalanceActivities) ValidatePlan(ctx context.Context, plan *RebalancePlan) error {
	portfolio, err := a.fetchPortfolio(ctx, plan.PortfolioID)
	if err != nil {
		return err
	}

	c := portfolio.Constraints

	// Validate trade sizes
	for _, trade := range plan.ProposedTrades {
		if trade.Notional < c.MinTradeSize {
			return fmt.Errorf("trade too small: %s $%.2f", trade.Symbol, trade.Notional)
		}
		if trade.Notional > c.MaxTradeSize {
			return fmt.Errorf("trade too large: %s $%.2f", trade.Symbol, trade.Notional)
		}
	}

	// Validate turnover
	totalTurnover := 0.0
	for _, trade := range plan.ProposedTrades {
		totalTurnover += trade.Notional
	}
	if (totalTurnover / portfolio.AUM * 100) > c.MaxTurnover {
		return fmt.Errorf("turnover exceeds limit")
	}

	// Validate tax budget
	if plan.TaxImpact.TotalTax > c.TaxBudget {
		return fmt.Errorf("tax impact exceeds budget")
	}

	// Validate restricted list
	for _, trade := range plan.ProposedTrades {
		for _, restricted := range c.RestrictedList {
			if trade.Symbol == restricted {
				return fmt.Errorf("cannot trade restricted: %s", trade.Symbol)
			}
		}
	}

	return nil
}

// Activity 5: Execute Trades via Redpanda/Kafka
func (a *RebalanceActivities) ExecuteTrades(ctx context.Context, plan *RebalancePlan) ([]string, error) {
	orderIDs := make([]string, 0, len(plan.ProposedTrades))

	for _, trade := range plan.ProposedTrades {
		order := map[string]interface{}{
			"id":            generateID(),
			"portfolio_id":  plan.PortfolioID,
			"symbol":        trade.Symbol,
			"side":          trade.Action,
			"quantity":      trade.Shares,
			"order_type":    "LIMIT",
			"limit_price":   trade.EstimatedPrice * 1.01, // 1% slippage
			"time_in_force": "DAY",
			"tax_lot_id":    trade.TaxLotID,
			"timestamp":     time.Now().Unix(),
		}

		body, _ := json.Marshal(order)
		if a.kafkaWriter == nil {
			return nil, fmt.Errorf("kafka writer not configured")
		}
		msg := kafka.Message{Topic: "orders", Key: []byte("order.new"), Value: body, Time: time.Now()}
		if err := a.kafkaWriter.WriteMessages(ctx, msg); err != nil {
			return nil, fmt.Errorf("failed to publish order: %w", err)
		}

		orderIDs = append(orderIDs, order["id"].(string))
	}

	return orderIDs, nil
}

// Activity 6: Update Portfolio State in Hasura
func (a *RebalanceActivities) UpdatePortfolioState(ctx context.Context, portfolioID string, plan *RebalancePlan) error {
	mutation := `
        mutation UpdatePortfolio($id: uuid!, $drift: numeric!, $tax_saved: numeric!) {
            update_portfolios_by_pk(
                pk_columns: {id: $id},
                _set: {
                    drift: $drift,
                    last_rebalance: "now()",
                    tax_saved: $tax_saved,
                    rebalance_status: "completed"
                }
            ) {
                id
            }
        }
    `

	variables := map[string]interface{}{
		"id":        portfolioID,
		"drift":     plan.ExpectedDrift,
		"tax_saved": plan.TaxImpact.TaxSavingsVsRandom,
	}

	return a.hasuraMutate(ctx, mutation, variables)
}

// Activity 7: Insert Rebalance Plan to Hasura
func (a *RebalanceActivities) InsertRebalancePlan(ctx context.Context, plan *RebalancePlan) error {
	mutation := `
        mutation InsertPlan($plan: rebalance_plans_insert_input!) {
            insert_rebalance_plans_one(object: $plan) {
                id
            }
        }
    `

	variables := map[string]interface{}{
		"plan": map[string]interface{}{
			"portfolio_id":    plan.PortfolioID,
			"timestamp":       plan.Timestamp,
			"current_drift":   plan.CurrentDrift,
			"expected_drift":  plan.ExpectedDrift,
			"tax_savings":     plan.TaxImpact.TaxSavingsVsRandom,
			"confidence":      plan.Confidence,
			"status":          plan.Status,
			"rationale":       plan.Rationale,
			"proposed_trades": toJSON(plan.ProposedTrades),
			"tax_analysis":    toJSON(plan.TaxImpact),
		},
	}

	return a.hasuraMutate(ctx, mutation, variables)
}

// Activity 8: Send Notification
func (a *RebalanceActivities) NotifyStakeholders(ctx context.Context, plan *RebalancePlan, status string) error {
	notification := map[string]interface{}{
		"type":         "rebalance_completed",
		"portfolio_id": plan.PortfolioID,
		"status":       status,
		"summary": fmt.Sprintf("Drift %.2f%% → %.2f%%, tax saved $%.0f",
			plan.CurrentDrift, plan.ExpectedDrift, plan.TaxImpact.TaxSavingsVsRandom),
		"timestamp": time.Now().Unix(),
	}

	body, _ := json.Marshal(notification)
	if a.kafkaWriter == nil {
		return fmt.Errorf("kafka writer not configured")
	}
	msg := kafka.Message{Topic: "notifications", Key: []byte("notification.rebalance"), Value: body, Time: time.Now()}
	if err := a.kafkaWriter.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish notification: %w", err)
	}
	return nil
}

func (a *RebalanceActivities) fetchPortfolio(ctx context.Context, id string) (*Portfolio, error) {
	query := `
        query GetPortfolio($id: uuid!) {
            portfolios_by_pk(id: $id) {
                id tenant_id name aum drift last_rebalance
                target_model constraints
                holdings { symbol shares current_price cost_basis purchase_date tax_lot_id sector }
            }
        }
    `

	var result struct {
		Data struct {
			Portfolio Portfolio `json:"portfolios_by_pk"`
		} `json:"data"`
	}

	if err := a.hasuraQuery(ctx, query, map[string]interface{}{"id": id}, &result); err != nil {
		return nil, err
	}

	return &result.Data.Portfolio, nil
}

func (a *RebalanceActivities) hasuraQuery(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	body, _ := json.Marshal(map[string]interface{}{"query": query, "variables": variables})
	req, _ := http.NewRequestWithContext(ctx, "POST", a.hasuraURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(result)
}

func (a *RebalanceActivities) hasuraMutate(ctx context.Context, mutation string, variables map[string]interface{}) error {
	body, _ := json.Marshal(map[string]interface{}{"query": mutation, "variables": variables})
	req, _ := http.NewRequestWithContext(ctx, "POST", a.hasuraURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("hasura mutation failed: %d", resp.StatusCode)
	}

	return nil
}

func (a *RebalanceActivities) insertAuditLog(ctx context.Context, userID, tenantID, action, resource, resourceID string, allowed bool) {
	mutation := `
        mutation InsertAudit($log: audit_logs_insert_input!) {
            insert_audit_logs_one(object: $log) { id }
        }
    `

	variables := map[string]interface{}{
		"log": map[string]interface{}{
			"user_id":     userID,
			"tenant_id":   tenantID,
			"action":      action,
			"resource":    resource,
			"resource_id": resourceID,
			"allowed":     allowed,
			"timestamp":   time.Now(),
		},
	}

	a.hasuraMutate(ctx, mutation, variables)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func generateID() string {
	return fmt.Sprintf("order_%d", time.Now().UnixNano())
}

func isPortfolioManager(userID string) bool {
	// Check user roles from database/cache
	return true // Placeholder
}

func belongsToTenant(portfolioID, tenantID string) bool {
	// Verify portfolio belongs to tenant
	return true // Placeholder
}

func isTradingHours() bool {
	now := time.Now().In(time.FixedZone("EST", -5*3600))
	hour := now.Hour()
	minute := now.Minute()

	// 9:30 AM - 4:00 PM ET
	if hour < 9 || (hour == 9 && minute < 30) || hour >= 16 {
		return false
	}

	// No weekends
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}

	return true
}
