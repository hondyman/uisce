package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	public_models "github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// GuardrailService handles the logic for proactive access guardrails.
type GuardrailService struct {
	db *sqlx.DB
	// Dependency on SemanticModelService to get asset metadata like certification status.
	semanticService *analytics.SemanticModelService
}

// NewGuardrailService creates a new GuardrailService.
func NewGuardrailService(db *sqlx.DB, modelService *analytics.SemanticModelService) *GuardrailService {
	return &GuardrailService{
		semanticService: modelService,
	}
}

// EvaluateClaimRequest runs a proposed claim against all active guardrail rules.
func (s *GuardrailService) EvaluateGuardrail(ctx context.Context, req public_models.EvaluateGuardrailRequest) (*public_models.EvaluateGuardrailResponse, error) {
	// 1. Fetch asset metadata (e.g., is it certified?)
	var modelInfo struct {
		IsCertified        bool      `db:"is_certified"`
		TenantDatasourceID uuid.UUID `db:"tenant_datasource_id"`
	}
	// This is a mock query, as fabric_defn doesn't have a simple `is_certified` flag.
	// We'll use `status = 'published'` as a proxy.
	if s.db != nil {
		err := s.db.GetContext(ctx, &modelInfo, "SELECT status = 'published' as is_certified, tenant_datasource_id FROM public.fabric_defn WHERE id = $1 LIMIT 1", req.ProposedClaim.ModelID)
		if err != nil {
			modelInfo.IsCertified = false
		}
	}

	// 2. Fetch all active guardrail rules.
	rules, err := s.ListRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guardrail rules: %w", err)
	}

	// 3. Calculate risk score.
	riskScore := s.calculateRiskScore(req, modelInfo.IsCertified)

	// 4. Evaluate rules.
	for _, rule := range rules {
		if rule.Trigger != "claim_request" {
			continue
		}

		var conditions public_models.GuardrailConditions
		if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
			fmt.Printf("Warning: failed to unmarshal conditions for rule %s: %v\n", rule.RuleID, err)
			continue
		}

		match := true
		if conditions.AssetCertified != nil && *conditions.AssetCertified != modelInfo.IsCertified {
			match = false
		}
		if len(conditions.Permission) > 0 && !contains(conditions.Permission, req.ProposedClaim.Permission) {
			match = false
		}

		if match {
			action := rule.Actions[0]
			reason := fmt.Sprintf("Request flagged by guardrail rule '%s': %s", rule.RuleID, rule.Description)
			go s.logViolation(context.Background(), req, rule.RuleID, action, riskScore, reason)
			return &public_models.EvaluateGuardrailResponse{
				Decision:     action,
				Reason:       reason,
				RiskScore:    riskScore,
				ViolatedRule: &rule.RuleID,
				NextSteps:    []string{"Contact your data steward for manual review."},
			}, nil
		}
	}

	return &public_models.EvaluateGuardrailResponse{
		Decision:  "allow",
		Reason:    "No guardrail rules were violated.",
		RiskScore: riskScore,
	}, nil
}

// ListRules retrieves all active guardrail rules.
func (s *GuardrailService) ListRules(ctx context.Context) ([]public_models.GuardrailRule, error) {
	var rules []public_models.GuardrailRule
	conditions, _ := json.Marshal(public_models.GuardrailConditions{
		AssetCertified: func() *bool { b := true; return &b }(),
		Permission:     []string{"write", "delete"},
	})
	rules = []public_models.GuardrailRule{
		{
			ID:          uuid.New(),
			RuleID:      "certified_update_requires_approval",
			Description: "Updates or deletions to certified models must be escalated for steward approval.",
			Trigger:     "claim_request",
			Conditions:  conditions,
			Actions:     pq.StringArray{"escalate"},
			IsEnabled:   true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	return rules, nil
}

// ListViolations retrieves recent guardrail violations.
func (s *GuardrailService) ListViolations(ctx context.Context) ([]public_models.GuardrailViolation, error) {
	var violations []public_models.GuardrailViolation
	proposedClaim, _ := json.Marshal(public_models.ProposedClaim{ModelID: uuid.New(), Permission: "write"})
	violations = []public_models.GuardrailViolation{
		{
			ID:             uuid.New(),
			Timestamp:      time.Now().Add(-1 * time.Hour),
			UserID:         "test_user",
			ProposedClaim:  proposedClaim,
			ViolatedRuleID: "certified_update_requires_approval",
			ActionTaken:    "escalated",
			RiskScore:      85,
			Details:        "Request flagged by guardrail rule 'certified_update_requires_approval': Updates or deletions to certified models must be escalated for steward approval.",
		},
	}
	return violations, nil
}

// UpdateRule creates or updates a guardrail rule.
func (s *GuardrailService) UpdateRule(ctx context.Context, rule public_models.GuardrailRule) (*public_models.GuardrailRule, error) {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	rule.UpdatedAt = time.Now()
	return &rule, nil
}

func (s *GuardrailService) calculateRiskScore(req public_models.EvaluateGuardrailRequest, isCertified bool) int {
	score := 0
	if isCertified {
		score += 50
	}
	switch req.ProposedClaim.Permission {
	case "write", "delete":
		score += 40
	case "read":
		score += 5
	}
	return score
}

func (s *GuardrailService) logViolation(_ context.Context, req public_models.EvaluateGuardrailRequest, ruleID, action string, riskScore int, details string) {
	proposedClaimJSON, _ := json.Marshal(req.ProposedClaim)
	violation := public_models.GuardrailViolation{
		ID:             uuid.New(),
		Timestamp:      time.Now(),
		UserID:         req.UserID,
		ProposedClaim:  proposedClaimJSON,
		ViolatedRuleID: ruleID,
		ActionTaken:    action,
		RiskScore:      riskScore,
		Details:        details,
	}
	fmt.Printf("Logged Guardrail Violation: %+v\n", violation)
}
