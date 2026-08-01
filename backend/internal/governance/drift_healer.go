package governance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

const minDriftHealConfidence = 0.6

type SelfHealingService struct {
	db *sql.DB
}

func NewSelfHealingService(db *sql.DB) *SelfHealingService {
	return &SelfHealingService{db: db}
}

type DriftRepairProposal struct {
	RuleID           string  `json:"rule_id"`
	MissingFieldPath string  `json:"missing_field_path"`
	ProposedField    string  `json:"proposed_field"`
	ConfidenceScore  float64 `json:"confidence_score"`
}

func (s *SelfHealingService) HandleCompileFailure(
	ctx context.Context,
	tenantID uuid.UUID,
	boID uuid.UUID,
	ruleID string,
	missingSymbol string,
) error {
	if s.db == nil {
		return fmt.Errorf("self-healing service: db is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	candidateQuery := `
		SELECT attribute_name, similarity(attribute_name, $1) AS score
		FROM public.tenant_custom_attributes
		WHERE tenant_id = $2 AND bo_id = $3
		ORDER BY score DESC
		LIMIT 1;
	`

	var proposedField string
	var score float64
	err := s.db.QueryRowContext(ctx, candidateQuery, missingSymbol, tenantID, boID).
		Scan(&proposedField, &score)
	if err == sql.ErrNoRows {
		return fmt.Errorf("no candidate attributes found for tenant=%s bo=%s", tenantID, boID)
	}
	if err != nil {
		return fmt.Errorf("drift healer candidate lookup failed: %w", err)
	}
	if score < minDriftHealConfidence {
		return fmt.Errorf("no suitable self-healing field candidate found for missing symbol %q (best score %.2f < %.2f)",
			missingSymbol, score, minDriftHealConfidence)
	}

	proposal := DriftRepairProposal{
		RuleID:           ruleID,
		MissingFieldPath: missingSymbol,
		ProposedField:    proposedField,
		ConfidenceScore:  score,
	}

	proposalJSON, err := json.Marshal(proposal)
	if err != nil {
		return fmt.Errorf("failed to marshal drift proposal: %w", err)
	}

	validationJSON, err := json.Marshal(map[string]any{
		"drift_detected": true,
		"reason": fmt.Sprintf(
			"Symbol %q unresolvable during VM compile; candidate %q matched with %.2f confidence",
			missingSymbol, proposedField, score),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal validation payload: %w", err)
	}

	ticketQuery := `
		INSERT INTO public.agent_approval_tickets
		(tenant_id, agent_id, target_bo_id, action_type, proposed_payload, compliance_validation_results, status)
		VALUES ($1, 'SelfHealingCopilot-v1', $2, 'SCHEMA_DRIFT_REPAIR', $3, $4, 'PENDING_CHECKER')
	`

	if _, err := s.db.ExecContext(ctx, ticketQuery,
		tenantID, boID, proposalJSON, validationJSON); err != nil {
		return fmt.Errorf("failed to queue self-healing ticket: %w", err)
	}

	return nil
}
