package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/rules"
)

// SampleEntityRepository defines how to fetch sample data for testing
type SampleEntityRepository interface {
	SampleEntities(ctx context.Context, tenantID string, entityName string, sampleSize int, filter map[string]interface{}) ([]map[string]interface{}, error)
}

type TestService struct {
	sampleRepo SampleEntityRepository
	engine     *rules.RuleEngine
}

func NewTestService(sampleRepo SampleEntityRepository, engine *rules.RuleEngine) *TestService {
	return &TestService{
		sampleRepo: sampleRepo,
		engine:     engine,
	}
}

// SmokeTestCondition runs a sanity check on a suggested condition (JSON or Starlark)
// It creates a transient TenantValidationRule and evaluates it against sample data.
func (s *TestService) SmokeTestCondition(
	ctx context.Context,
	tenantID, entity string,
	conditionJSON map[string]interface{}, // Optional, for future use with ConditionBuilder
) (failureRate float64, runtimeOK bool) {

	// 1. Fetch small sample
	// Note: We use a small sample size for quick feedback (e.g., 20)
	records, err := s.sampleRepo.SampleEntities(ctx, tenantID, entity, 20, nil)
	if err != nil {
		// Log error?
		return 0, false
	}
	if len(records) == 0 {
		return 0, true // No data to fail on, effectively valid runtime
	}

	// 2. If conditionJSON is provided, do a quick deterministic evaluation over sample records.
	if conditionJSON != nil {
		// Expect simple condition: {"type":"condition","field":"amount","operator":">=","value":1000}
		typeVal, _ := conditionJSON["type"].(string)
		if typeVal == "condition" {
			field, _ := conditionJSON["field"].(string)
			op, _ := conditionJSON["operator"].(string)
			value := conditionJSON["value"]
			valFloat := 0.0
			switch v := value.(type) {
			case float64:
				valFloat = v
			case int:
				valFloat = float64(v)
			case int64:
				valFloat = float64(v)
			}

			failures := 0
			for _, rec := range records {
				if fv, ok := rec[field]; ok {
					switch n := fv.(type) {
					case float64:
						if op == ">=" && n >= valFloat {
							failures++
						}
					case int:
						if op == ">=" && float64(n) >= valFloat {
							failures++
						}
					}
				}
			}
			return float64(failures) / float64(len(records)), true
		}
	}

	now := time.Now()
	tempRule := rules.TenantValidationRule{
		TenantID:    tenantID,
		RuleID:      "ai-smoke-test-" + fmt.Sprintf("%d", now.UnixNano()),
		InheritMode: rules.Custom, // Treat as custom for isolation
		CreatedAt:   now,
	}

	failures := 0
	for _, rec := range records {
		// page, objects := starlib.SplitDataIntoPageAndObjects(rec)
		boCtx := map[string]map[string]interface{}{
			"page": map[string]interface{}{}, // Placeholder
		}
		// for k, v := range objects {
		// 	boCtx[k] = v
		// }
		for k, v := range rec {
			if boCtx["page"] != nil {
				boCtx["page"][k] = v
			}
		}

		passed, err := s.engine.EvaluateTenantRule(ctx, &tempRule, boCtx)
		if err != nil {
			return 0, false
		}
		if !passed {
			failures++
		}
	}

	return float64(failures) / float64(len(records)), true
}
