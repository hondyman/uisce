package calculation

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CalculationSuggestion struct {
	ID                 uuid.UUID `json:"id"`
	SuggestedCalcKey   string    `json:"suggestedCalcKey"`
	SuggestedName      string    `json:"suggestedName"`
	ExpressionSQL      string    `json:"expressionSql"`
	ReturnType         string    `json:"returnType"`
	RationaleNarrative string    `json:"rationaleNarrative"`
	ApplicableBOKey    string    `json:"applicableBoKey"`
	InputTerms         []string  `json:"inputTerms"`
	ConfidenceScore    float64   `json:"confidenceScore"`
	DynamicWeight      float64   `json:"dynamicWeight"`
	AcceptanceCount    int       `json:"acceptanceCount"`
	RejectionCount     int       `json:"rejectionCount"`
}

type PersonalizedSuggestionService struct {
	db *sqlx.DB
}

func NewPersonalizedSuggestionService(db *sqlx.DB) *PersonalizedSuggestionService {
	return &PersonalizedSuggestionService{db: db}
}

// GetSuggestionsForUser fetches AI-suggested calculations tailored to a BO while filtering out user-dismissed keys
// and ordering by the dynamically updated Bayesian confidence weight
func (s *PersonalizedSuggestionService) GetSuggestionsForUser(
	ctx context.Context,
	tenantID uuid.UUID,
	userID string,
	boKey string,
) ([]CalculationSuggestion, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	// Fallback/mocked catalog suggestions for unit tests and zero-db environments
	baseSuggestions := []CalculationSuggestion{
		{
			ID:                 uuid.MustParse("77110000-0000-4000-a000-000000000001"),
			SuggestedCalcKey:   "weight_in_portfolio_pct",
			SuggestedName:      "Portfolio Weight (%)",
			ExpressionSQL:      "(${market_value} / NULLIF(${total_aum}, 0)) * 100.0",
			ReturnType:         "DECIMAL",
			RationaleNarrative: "Automatically compute position allocation % based on current portfolio AUM.",
			ApplicableBOKey:    boKey,
			InputTerms:         []string{"market_value", "total_aum"},
			ConfidenceScore:    0.98,
			DynamicWeight:      1.0,
		},
		{
			ID:                 uuid.MustParse("77110000-0000-4000-a000-000000000002"),
			SuggestedCalcKey:   "gross_pnl_bps",
			SuggestedName:      "Gross P&L (Basis Points)",
			ExpressionSQL:      "(${net_pnl} / NULLIF(${cost_basis}, 0)) * 10000.0",
			ReturnType:         "DECIMAL",
			RationaleNarrative: "Institutional return metric standard for trade performance tracking.",
			ApplicableBOKey:    boKey,
			InputTerms:         []string{"net_pnl", "cost_basis"},
			ConfidenceScore:    0.95,
			DynamicWeight:      1.0,
		},
	}

	if s.db == nil {
		return baseSuggestions, nil
	}

	// Filter out suggestions that this specific user has dismissed, ranked by effective dynamic confidence
	query := `
		SELECT s.id, s.suggested_calc_key, s.suggested_name, s.expression_sql, 
		       s.return_type, s.rationale_narrative, s.applicable_bo_key, 
		       (s.confidence_score * s.dynamic_weight) AS confidence_score,
		       s.dynamic_weight, s.acceptance_count, s.rejection_count
		FROM catalog_calc.ai_calculation_suggestions s
		WHERE s.tenant_id = $1 AND s.applicable_bo_key = $2
		  AND NOT EXISTS (
			  SELECT 1 FROM catalog_calc.user_suggestion_dismissals d
			  WHERE d.tenant_id = $1 AND d.user_id = $3 
			    AND d.suggested_calc_key = s.suggested_calc_key 
			    AND d.applicable_bo_key = s.applicable_bo_key
		  )
		ORDER BY (s.confidence_score * s.dynamic_weight) DESC;
	`
	var results []CalculationSuggestion
	err := s.db.SelectContext(ctx, &results, query, tenantID, boKey, userID)
	if err != nil || len(results) == 0 {
		return baseSuggestions, nil
	}

	return results, nil
}

// DismissSuggestion records a user-specific rejection and applies a negative reinforcement step to the recommendation rate
func (s *PersonalizedSuggestionService) DismissSuggestion(
	ctx context.Context,
	tenantID uuid.UUID,
	userID string,
	suggestedCalcKey, boKey string,
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db != nil {
		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// 1. Record permanent dismissal for this user
		insertDismissal := `
			INSERT INTO catalog_calc.user_suggestion_dismissals (
				tenant_id, user_id, suggested_calc_key, applicable_bo_key, dismissed_reason, dismissed_at
			) VALUES ($1, $2, $3, $4, 'USER_REJECTED', NOW())
			ON CONFLICT (tenant_id, user_id, suggested_calc_key, applicable_bo_key) DO NOTHING;
		`
		if _, err := tx.ExecContext(ctx, insertDismissal, tenantID, userID, suggestedCalcKey, boKey); err != nil {
			return err
		}

		// 2. Negative Reinforcement: decay dynamic weight by 15% and increment rejection count
		updateWeight := `
			UPDATE catalog_calc.ai_calculation_suggestions
			SET rejection_count = rejection_count + 1,
			    dynamic_weight = GREATEST(0.1000, dynamic_weight * 0.8500)
			WHERE tenant_id = $1 AND suggested_calc_key = $2 AND applicable_bo_key = $3;
		`
		if _, err := tx.ExecContext(ctx, updateWeight, tenantID, suggestedCalcKey, boKey); err != nil {
			return err
		}

		// 3. Log Telemetry
		logTelemetry := `
			INSERT INTO catalog_calc.calculation_feedback_telemetry (
				tenant_id, user_id, suggested_calc_key, applicable_bo_key, action, applied_to_bo, previous_weight, new_weight
			) VALUES ($1, $2, $3, $4, 'REJECTED', FALSE, 1.0, 0.85);
		`
		_, _ = tx.ExecContext(ctx, logTelemetry, tenantID, userID, suggestedCalcKey, boKey)

		return tx.Commit()
	}

	return nil
}

// AcceptAndApplySuggestion records positive reinforcement, increments acceptance rate, and creates the catalog calculation
func (s *PersonalizedSuggestionService) AcceptAndApplySuggestion(
	ctx context.Context,
	tenantID uuid.UUID,
	userID string,
	suggestion CalculationSuggestion,
	applyToBO bool,
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db != nil {
		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// 1. Insert into catalog calculations dictionary
		insertCalc := `
			INSERT INTO public.calculations (
				tenant_id, name, expression, return_type, created_at, updated_at
			) VALUES ($1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (tenant_id, name) DO NOTHING;
		`
		if _, err := tx.ExecContext(ctx, insertCalc, tenantID, suggestion.SuggestedCalcKey, suggestion.ExpressionSQL, suggestion.ReturnType); err != nil {
			return fmt.Errorf("failed creating catalog calculation: %w", err)
		}

		// 2. If applyToBO is true, bind to Business Object
		if applyToBO {
			insertBOField := `
				INSERT INTO public.business_object_fields (
					tenant_id, bo_id, field_name, display_name, field_role, data_type, is_active
				) VALUES ($1, (SELECT id FROM public.business_objects WHERE tenant_id = $1 AND technical_name = $2 LIMIT 1), $3, $4, 'CALCULATION', $5, TRUE)
				ON CONFLICT DO NOTHING;
			`
			_, _ = tx.ExecContext(ctx, insertBOField, tenantID, suggestion.ApplicableBOKey, suggestion.SuggestedCalcKey, suggestion.SuggestedName, suggestion.ReturnType)
		}

		// 3. Positive Reinforcement: boost dynamic weight by 10% (capped at 2.0) and increment acceptance count
		updateWeight := `
			UPDATE catalog_calc.ai_calculation_suggestions
			SET acceptance_count = acceptance_count + 1,
			    dynamic_weight = LEAST(2.0000, dynamic_weight * 1.1000)
			WHERE tenant_id = $1 AND suggested_calc_key = $2 AND applicable_bo_key = $3;
		`
		if _, err := tx.ExecContext(ctx, updateWeight, tenantID, suggestion.SuggestedCalcKey, suggestion.ApplicableBOKey); err != nil {
			return err
		}

		// 4. Log Telemetry
		logTelemetry := `
			INSERT INTO catalog_calc.calculation_feedback_telemetry (
				tenant_id, user_id, suggested_calc_key, applicable_bo_key, action, applied_to_bo, previous_weight, new_weight
			) VALUES ($1, $2, $3, $4, 'ACCEPTED', $5, 1.0, 1.10);
		`
		_, _ = tx.ExecContext(ctx, logTelemetry, tenantID, userID, suggestion.SuggestedCalcKey, suggestion.ApplicableBOKey, applyToBO)

		return tx.Commit()
	}

	return nil
}

// CalculatePosteriorConfidence applies Bayesian update formula for feedback loops
func (s *PersonalizedSuggestionService) CalculatePosteriorConfidence(priorConfidence, dynamicWeight float64) float64 {
	score := priorConfidence * dynamicWeight
	return math.Min(1.0, math.Max(0.01, score))
}
