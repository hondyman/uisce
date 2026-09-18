package tiles

import (
	"context"
	"fmt"
)

type ComplianceConfig struct {
	RuleSetIDs        []string `json:"rule_set_ids"`
	SeverityThreshold string   `json:"severity_threshold"` // "WARNING" | "HARD_BLOCK"
}

// ComplianceEvaluator abstracts the rule engine (for testing).
type ComplianceEvaluator interface {
	Evaluate(ctx context.Context, tenantID string, ruleSetIDs []string, record map[string]any) (passed bool, violations []string, err error)
}

func NewComplianceValidator(cfg ComplianceConfig, evaluator ComplianceEvaluator) TileFunc {
	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		out := make([]Record, 0, len(records))
		var errs []string

		tctx := TenantFromContext(ctx)

		for _, rec := range records {
			passed, violations, err := evaluator.Evaluate(ctx, tctx.TenantID, cfg.RuleSetIDs, rec)
			if err != nil {
				errs = append(errs, fmt.Sprintf("compliance: evaluate error: %v", err))
				continue
			}

			if !passed {
				if cfg.SeverityThreshold == "HARD_BLOCK" {
					errs = append(errs, fmt.Sprintf("compliance: HARD_BLOCK violations: %v", violations))
					continue // DO NOT include in output
				}
				// WARNING threshold -> include, but add warnings
				newRec := make(Record, len(rec)+1)
				for k, v := range rec {
					newRec[k] = v
				}
				newRec["compliance_warnings"] = violations
				out = append(out, newRec)
			} else {
				out = append(out, rec)
			}
		}

		return out, errs, nil
	}
}
