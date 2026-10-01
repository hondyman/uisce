package analytics

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// ContextLoader is an optional function for loading related-row context from the DB.
type ContextLoader func(ctx context.Context, tenantID, boKey, recordID string) (map[string]any, error)

// ServerContextRequiredError is returned when rules reference context/collection
// fields that were not supplied in an ad-hoc payload and could not be loaded from DB.
type ServerContextRequiredError struct {
	MissingFields []string `json:"missing_context_fields"`
	Hint          string   `json:"hint"`
}

func (e *ServerContextRequiredError) Error() string {
	return fmt.Sprintf("ERR_SERVER_CONTEXT_REQUIRED: missing context fields: %s", strings.Join(e.MissingFields, ", "))
}

// RecordEvaluation represents the result of evaluating one record against a rule snapshot.
type RecordEvaluation struct {
	Valid          bool                     `json:"valid"`
	Blocked        bool                     `json:"blocked"`
	EvaluatedCount int                      `json:"evaluated_rules_count"`
	SnapshotID     string                   `json:"snapshot_id,omitempty"`
	Violations     []models.ViolationRecord `json:"violations"`
	RuleErrors     []RuleErrorEntry         `json:"rule_errors"`
}

// RuleErrorEntry represents a rule execution error (e.g., malformed AST or arithmetic error).
type RuleErrorEntry struct {
	RuleID   string `json:"rule_id"`
	RuleKey  string `json:"rule_key"`
	RuleName string `json:"rule_name"`
	Message  string `json:"message"`
}

// EvaluateRecord evaluates a single ad-hoc data record against the rules in snap.
//
// Intentional Design Divergence:
//   - In shadow_evaluation.go (write path), unresolvedFieldRefs flags ANY missing top-level field
//     as RuleError = true because real committed records are expected to supply all table columns.
//   - In this ad-hoc API (EvaluateRecord), partial payloads are legitimate for client-side forms
//     and partial checks. Plain-field omissions follow evaluator-natural semantics (conditions evaluate
//     to false/is_null=true). Only server-context/collection fields trigger ServerContextRequiredError.
func EvaluateRecord(ctx context.Context, snap *RuleSnapshot, record map[string]any, loader ContextLoader) (*RecordEvaluation, error) {
	return EvaluateRecordWithEvaluator(ctx, snap, record, loader, vm.NewAdvancedEvaluator())
}

// EvaluateRecordWithEvaluator evaluates a single ad-hoc record using a supplied AdvancedEvaluator.
func EvaluateRecordWithEvaluator(ctx context.Context, snap *RuleSnapshot, record map[string]any, loader ContextLoader, ae *vm.AdvancedEvaluator) (*RecordEvaluation, error) {
	if snap == nil {
		return nil, fmt.Errorf("rule snapshot is nil")
	}
	if ae == nil {
		ae = vm.NewAdvancedEvaluator()
	}

	evalData := make(map[string]any, len(record))
	for k, v := range record {
		evalData[k] = v
	}

	// 1. Identify any referenced context or collection fields missing from payload
	var missingContext []string
	contextNeeded := map[string]bool{}
	for _, r := range snap.Rules {
		for _, f := range r.FieldRefs {
			rootField := strings.Split(f, ".")[0]
			if models.IsKnownContextField(rootField) {
				if _, ok := evalData[rootField]; !ok {
					contextNeeded[rootField] = true
				}
			}
		}
	}
	for f := range contextNeeded {
		missingContext = append(missingContext, f)
	}

	// 2. If context is missing, attempt to load via loader if a record ID is present
	if len(missingContext) > 0 {
		var recordID string
		for _, idKey := range []string{"id", "record_id", "order_id", "execution_id"} {
			if v, ok := evalData[idKey]; ok && v != nil {
				recordID = fmt.Sprintf("%v", v)
				break
			}
		}

		if recordID != "" && loader != nil {
			loadedCtx, err := loader(ctx, snap.TenantID, snap.BOName, recordID)
			if err == nil && loadedCtx != nil {
				for k, v := range loadedCtx {
					if _, exists := evalData[k]; !exists {
						evalData[k] = v
					}
				}
				// Re-verify if all missing context fields were satisfied
				var remaining []string
				for _, f := range missingContext {
					if _, ok := evalData[f]; !ok {
						remaining = append(remaining, f)
					}
				}
				missingContext = remaining
			}
		}

		if len(missingContext) > 0 {
			return nil, &ServerContextRequiredError{
				MissingFields: missingContext,
				Hint:          "provide referenced context fields directly or provide a valid record ID for server lookup",
			}
		}
	}

	// 3. Evaluate rules
	var violations []models.ViolationRecord
	var ruleErrors []RuleErrorEntry
	var hasBlock bool

	var recordIDStr string
	if v, ok := evalData["id"]; ok && v != nil {
		recordIDStr = fmt.Sprintf("%v", v)
	}

	for _, r := range snap.Rules {
		if r.AST == nil {
			continue
		}

		passed, err := ae.Evaluate(*r.AST, evalData)
		if err != nil {
			ruleErrors = append(ruleErrors, RuleErrorEntry{
				RuleID:   r.RuleID,
				RuleKey:  r.RuleKey,
				RuleName: r.RuleName,
				Message:  err.Error(),
			})
			violations = append(violations, models.ViolationRecord{
				TenantID:     snap.TenantID,
				RuleID:       r.RuleID,
				RuleKey:      r.RuleKey,
				RuleVersion:  r.RuleVersion,
				RuleName:     r.RuleName,
				BOKey:        r.BOName,
				Severity:     r.Severity,
				RecordID:     recordIDStr,
				Message:      err.Error(),
				Fields:       r.FieldRefs,
				Context:      evalData,
				WriteBlocked: r.Severity == models.ValidationRuleSeverityBlock,
				RuleError:    true,
			})
			if r.Severity == models.ValidationRuleSeverityBlock {
				hasBlock = true
			}
			continue
		}

		if !passed {
			isBlock := r.Severity == models.ValidationRuleSeverityBlock
			if isBlock {
				hasBlock = true
			}
			violations = append(violations, models.ViolationRecord{
				TenantID:     snap.TenantID,
				RuleID:       r.RuleID,
				RuleKey:      r.RuleKey,
				RuleVersion:  r.RuleVersion,
				RuleName:     r.RuleName,
				BOKey:        r.BOName,
				Severity:     r.Severity,
				RecordID:     recordIDStr,
				Message:      fmt.Sprintf("validation rule %q (%s) failed", r.RuleName, r.RuleKey),
				Fields:       r.FieldRefs,
				Context:      evalData,
				WriteBlocked: isBlock,
				RuleError:    false,
			})
		}
	}

	valid := len(violations) == 0 && len(ruleErrors) == 0

	return &RecordEvaluation{
		Valid:          valid,
		Blocked:        hasBlock,
		EvaluatedCount: len(snap.Rules),
		SnapshotID:     snap.SnapshotID,
		Violations:     violations,
		RuleErrors:     ruleErrors,
	}, nil
}
