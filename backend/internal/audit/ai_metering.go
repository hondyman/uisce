package audit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AIMeterPayload struct {
	TenantID             uuid.UUID  `json:"tenantId"`
	ProposalHash         string     `json:"proposalHash"`
	DrivingNodeID        *uuid.UUID `json:"drivingNodeId,omitempty"`
	PromptType           string     `json:"promptType"`
	PromptChars          int        `json:"promptChars"`
	TokensPrompt         int        `json:"tokensPrompt"`
	TokensCompletion     int        `json:"tokensCompletion"`
	IsHeuristic          bool       `json:"isHeuristic"`
	RetryCount           int        `json:"retryCount"`
	WasSuccessful        bool       `json:"wasSuccessful"`
	TokenRatePerThousand float64    `json:"tokenRatePerThousand"`
}

type AIMeterService struct {
	db *sqlx.DB
}

func NewAIMeterService(db *sqlx.DB) *AIMeterService {
	return &AIMeterService{db: db}
}

// RecordAITokenUsage applies deduplication checks, 3-way token accounting, and retry penalties
func (s *AIMeterService) RecordAITokenUsage(
	ctx context.Context,
	payload AIMeterPayload,
) (float64, error) {
	if payload.TenantID == uuid.Nil {
		return 0, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	// 1. Check Deduplication Window (Rule 8)
	if s.db != nil {
		var existingCount int
		err := s.db.GetContext(ctx, &existingCount, `
			SELECT COUNT(*) FROM audit.ai_query_metering_ledger
			WHERE tenant_id = $1 AND proposal_hash = $2 AND executed_at > NOW() - INTERVAL '15 minutes';
		`, payload.TenantID, payload.ProposalHash)
		if err == nil && existingCount > 0 {
			return 0.0, nil // Cached plan reuse: zero charge
		}
	}

	// 2. Three-Way Token Attribution Resolution
	promptTokens := payload.TokensPrompt
	completionTokens := payload.TokensCompletion

	if payload.IsHeuristic || (promptTokens == 0 && payload.PromptChars > 0) {
		promptTokens = payload.PromptChars / 4 // Heuristic fallback estimation
		payload.IsHeuristic = true
	}

	totalTokens := (promptTokens + completionTokens) * (1 + payload.RetryCount) // Retry penalty accounting
	rate := payload.TokenRatePerThousand
	if rate <= 0 {
		rate = 0.002 // Default $0.002 per 1k tokens
	}
	totalCostUSD := (float64(totalTokens) / 1000.0) * rate

	// 3. Persist to Metering Ledger
	if s.db != nil {
		insertSQL := `
			INSERT INTO audit.ai_query_metering_ledger (
				tenant_id, proposal_hash, driving_node_id, prompt_type,
				tokens_prompt, tokens_completion, tokens_total,
				is_heuristic_estimation, retry_count, was_successful,
				total_token_cost_usd, executed_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW());
		`
		_, err := s.db.ExecContext(ctx, insertSQL,
			payload.TenantID, payload.ProposalHash, payload.DrivingNodeID, payload.PromptType,
			promptTokens, completionTokens, totalTokens,
			payload.IsHeuristic, payload.RetryCount, payload.WasSuccessful,
			totalCostUSD,
		)
		if err != nil {
			return 0, fmt.Errorf("failed recording AI token metering: %w", err)
		}
	}

	return totalCostUSD, nil
}
