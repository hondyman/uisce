package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/internal/models"
)

// Reconciler handles AI-driven trade reconciliation
type Reconciler struct {
	xaiClient *xAIClient
}

// NewReconciler creates a new reconciler
func NewReconciler() *Reconciler {
	return &Reconciler{
		xaiClient: NewxAIClient(),
	}
}

// ReconcileInput is the input to the reconciliation
type ReconcileInput struct {
	Trades   []models.Trade        `json:"trades"`
	Confirms []models.TradeConfirm `json:"confirms"`
}

// ReconcileOutput is the output from AI matching
type ReconcileOutput struct {
	Matched           []models.TradeMatch  `json:"matched"`
	UnmatchedTrades   []string             `json:"unmatched_trades"`
	UnmatchedConfirms []string             `json:"unmatched_confirms"`
	Discrepancies     []models.Discrepancy `json:"discrepancies"`
	MatchRate         float64              `json:"match_rate"`
	Reasoning         string               `json:"reasoning"`
}

// Reconcile performs AI-driven matching of trades to confirms
func (r *Reconciler) Reconcile(ctx context.Context, trades []models.Trade, confirms []models.TradeConfirm) (*ReconcileOutput, error) {
	prompt := buildReconciliationPrompt(trades, confirms)

	response, err := r.xaiClient.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI call failed: %w", err)
	}

	var output ReconcileOutput
	if err := json.Unmarshal([]byte(response), &output); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w\nResponse: %s", err, response)
	}

	// Calculate match rate
	totalMatches := len(output.Matched)
	totalTrades := len(trades)
	if totalTrades > 0 {
		output.MatchRate = float64(totalMatches) / float64(totalTrades)
	}

	return &output, nil
}

// buildReconciliationPrompt constructs the prompt for xAI
func buildReconciliationPrompt(trades []models.Trade, confirms []models.TradeConfirm) string {
	tradesJSON, _ := json.MarshalIndent(trades, "", "  ")
	confirmsJSON, _ := json.MarshalIndent(confirms, "", "  ")

	prompt := fmt.Sprintf(`You are an expert trade reconciliation AI. Your task is to match trades to trade confirmations with high precision.

YESTERDAY'S TRADES (from our system):
%s

TRADE CONFIRMATIONS RECEIVED (from custodians):
%s

MATCHING RULES:
1. Symbol must match exactly (case-insensitive OK)
2. Shares within ±0.1%% tolerance (e.g., 1000 ± 1 share)
3. Price within ±0.5%% or $0.01 (whichever is greater)
4. Trade date must be same day or ±1 business day from confirm date
5. Custodian must match (if both have custodian info)
6. Action (buy/sell) must match

TASK:
1. Identify all matches between trades and confirms
2. Flag any discrepancies (mismatched fields, unmatched items)
3. For discrepancies, assign severity: low (rounding), medium (processing error), high (potential fraud/error)
4. Suggest fixes where appropriate
5. Return ONLY valid JSON matching the schema below (no markdown, no explanation)

RESPONSE SCHEMA (must be valid JSON):
{
  "matched": [
    {
      "trade_id": "uuid",
      "confirm_id": "uuid", 
      "confidence": 0.95,
      "match_fields": ["symbol", "shares", "price", "date"]
    }
  ],
  "unmatched_trades": ["trade_id1", "trade_id2"],
  "unmatched_confirms": ["confirm_id1"],
  "discrepancies": [
    {
      "trade_id": "uuid or null",
      "confirm_id": "uuid or null",
      "discrepancy_type": "unmatched_trade|unmatched_confirm|mismatch",
      "field": "shares|price|date|symbol|custodian or null",
      "trade_value": "value from trade or null",
      "confirm_value": "value from confirm or null",
      "severity": "low|medium|high",
      "suggested_fix": "description of recommended action"
    }
  ],
  "match_rate": 0.98,
  "reasoning": "brief summary of matching logic applied"
}

Begin matching now.`, string(tradesJSON), string(confirmsJSON))

	return prompt
}
