package uisce

import (
	"context"
	"errors"
)

// SanctionsFilter checks if a counterparty is on a sanctions list (mock)
type SanctionsFilter struct {
	// In real impl, this would connect to a sanctions database
}

func (f *SanctionsFilter) Name() string {
	return "Sanctions Filter"
}

func (f *SanctionsFilter) Purify(ctx context.Context, data map[string]interface{}) error {
	// Mock implementation - in reality, this would check against OFAC SDN list
	counterparty, ok := data["counterparty"].(string)
	if !ok {
		return nil // No counterparty to check
	}

	// Mock: Block any counterparty named "BLOCKED_ENTITY"
	if counterparty == "BLOCKED_ENTITY" {
		return errors.New("counterparty is on OFAC sanctions list")
	}
	return nil
}

// LimitFilter checks if a transaction exceeds a configured limit
type LimitFilter struct {
	Limit float64
}

func (f *LimitFilter) Name() string {
	return "Limit Filter"
}

func (f *LimitFilter) Purify(ctx context.Context, data map[string]interface{}) error {
	amount, ok := data["amount"].(float64)
	if !ok {
		// Try to convert from int
		if amtInt, ok := data["amount"].(int); ok {
			amount = float64(amtInt)
		} else {
			return nil // No amount to check
		}
	}

	if amount > f.Limit {
		return errors.New("transaction amount exceeds configured limit")
	}
	return nil
}

// AIAnomalyFilter simulates an AI-based anomaly detection
type AIAnomalyFilter struct {
	// In real impl, this would call an ML model
}

func (f *AIAnomalyFilter) Name() string {
	return "AI_Anomaly Filter"
}

func (f *AIAnomalyFilter) Purify(ctx context.Context, data map[string]interface{}) error {
	// Mock: Flag if "anomaly_score" exceeds 0.8
	score, ok := data["anomaly_score"].(float64)
	if !ok {
		return nil
	}

	if score > 0.8 {
		return errors.New("AI detected anomalous trading pattern")
	}
	return nil
}
