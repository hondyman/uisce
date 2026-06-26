package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/internal/ai"
	"github.com/hondyman/semlayer/internal/drift"
	"github.com/hondyman/semlayer/internal/uar"
)

// Activities bundles dependencies needed by Temporal activities.
type Activities struct {
	UARStore        uar.UARStore
	GeminiClient    *ai.GeminiClient
	DriftCalculator *drift.Calculator
}

// ---------- Activity implementations ----------

// CheckDriftActivity – calculates portfolio drift using StarRocks/Iceberg or mock data.
func (a *Activities) CheckDriftActivity(ctx context.Context, tenantID, portfolioID string) (map[string]any, error) {
	var dr map[string]any
	var err error

	// Use real drift calculator if available, otherwise use fallback mock
	if a.DriftCalculator != nil {
		dr, err = a.DriftCalculator.CalculateDrift(ctx, tenantID, portfolioID)
		if err != nil {
			return nil, fmt.Errorf("calculate drift: %w", err)
		}
	} else {
		// Fallback mock if calculator is not initialized
		dr = map[string]any{
			"has_drift": true,
			"drift_pct": 6.2,
			"is_mock":   true,
		}
	}

	if a.UARStore != nil {
		_, _ = a.UARStore.Write(ctx, tenantID, portfolioID, "DriftCheck", dr)
	}
	return dr, nil
}

// GenerateAIProposalActivity – calls Gemini to produce a structured TradeProposal.
func (a *Activities) GenerateAIProposalActivity(ctx context.Context, tenantID, portfolioID string, drift map[string]any) (map[string]any, error) {
	// Marshal drift report to JSON for the LLM prompt.
	driftBytes, err := json.Marshal(drift)
	if err != nil {
		return nil, fmt.Errorf("marshal drift report: %w", err)
	}
	driftJSON := string(driftBytes)

	// If the Gemini client is not configured, fall back to a deterministic mock.
	if a.GeminiClient == nil {
		proposal := map[string]any{
			"id":          uuid.New().String(),
			"trades":      []map[string]any{{"side": "SELL", "symbol": "IVV", "qty": 50}},
			"explanation": "Fallback mock proposal – Gemini client not initialized.",
			"confidence":  0.5,
			"grounding":   []any{},
		}
		if a.UARStore != nil {
			_, _ = a.UARStore.Write(ctx, tenantID, portfolioID, "AIProposal", proposal)
		}
		return proposal, nil
	}

	// Call Gemini to generate the proposal JSON.
	jsonResp, err := a.GeminiClient.GenerateProposal(ctx, driftJSON)
	if err != nil {
		return nil, fmt.Errorf("gemini proposal generation failed: %w", err)
	}

	var proposal map[string]any
	if err := json.Unmarshal([]byte(jsonResp), &proposal); err != nil {
		return nil, fmt.Errorf("unmarshal gemini response: %w", err)
	}

	if a.UARStore != nil {
		_, _ = a.UARStore.Write(ctx, tenantID, portfolioID, "AIProposal", proposal)
	}
	return proposal, nil
}

// PolicyCheckActivity – deterministic Rego/CEL evaluation (stubbed).
func (a *Activities) PolicyCheckActivity(ctx context.Context, tenantID string, proposal map[string]any) (map[string]any, error) {
	// Simple rule: block if confidence < 0.6.
	conf, _ := proposal["confidence"].(float64)
	if conf < 0.6 {
		return map[string]any{"ok": false, "reasons": []string{"low_confidence"}}, nil
	}
	return map[string]any{"ok": true, "reasons": nil}, nil
}

// NotifyAdvisorActivity – sends proposal to GenUI (mocked) and records UAR.
func (a *Activities) NotifyAdvisorActivity(ctx context.Context, tenantID string, proposal map[string]any, reason string) (map[string]any, error) {
	// In production, POST to GenUI endpoint with JWT + x‑tenant‑id.
	if a.UARStore != nil {
		_, _ = a.UARStore.Write(ctx, tenantID, "", "NotifyAdvisor", map[string]any{
			"proposal_id": proposal["id"],
			"reason":      reason,
		})
	}
	return map[string]any{"notified": true}, nil
}

// ExecuteTradeSagaActivity – linear saga with compensation (mocked).
func (a *Activities) ExecuteTradeSagaActivity(ctx context.Context, tenantID string, proposal map[string]any) (map[string]any, error) {
	trades, ok := proposal["trades"].([]any)
	if !ok {
		// attempt to coerce via JSON round‑trip
		raw, _ := json.Marshal(proposal["trades"]) // ignore error for demo
		var tmp []any
		_ = json.Unmarshal(raw, &tmp)
		trades = tmp
	}

	result := map[string]any{"steps": []any{}}

	for i, t := range trades {
		step := map[string]any{"step": i + 1, "trade": t, "status": "submitted"}
		// Simulate failure on second leg to demonstrate compensation.
		if i == 1 {
			step["status"] = "failed"
			if a.UARStore != nil {
				_, _ = a.UARStore.Write(ctx, tenantID, "", "TradeFailed", step)
			}
			comp := map[string]any{"compensation": "reverse_first_trade", "status": "executed"}
			if a.UARStore != nil {
				_, _ = a.UARStore.Write(ctx, tenantID, "", "Compensation", comp)
			}
			result["saga_status"] = "compensated"
			result["comp"] = comp
			return result, errors.New("simulated trade failure; compensation executed")
		}
		step["status"] = "filled"
		result["steps"] = append(result["steps"].([]any), step)
		if a.UARStore != nil {
			_, _ = a.UARStore.Write(ctx, tenantID, "", "TradeFilled", step)
		}
	}
	result["saga_status"] = "completed"
	return result, nil
}

// PersistUARActivity – generic persistence helper.
func (a *Activities) PersistUARActivity(ctx context.Context, tenantID string, payload map[string]any) (map[string]any, error) {
	if a.UARStore != nil {
		_, _ = a.UARStore.Write(ctx, tenantID, "", "Persist", payload)
	}
	return map[string]any{"ok": true}, nil
}
