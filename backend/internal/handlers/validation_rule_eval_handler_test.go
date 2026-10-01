package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// stubEvalService returns hand-built snapshots; the handler exercises the
// REAL analytics.EvaluateRecord/EvaluateBatch against them.
type stubEvalService struct{ snap *analytics.RuleSnapshot }

func (s *stubEvalService) LoadRuleSnapshot(_ context.Context, _, _, _, _ string) (*analytics.RuleSnapshot, error) {
	return s.snap, nil
}

func (s *stubEvalService) EvaluateBatch(ctx context.Context, snap *analytics.RuleSnapshot, records []map[string]any, l analytics.ContextLoader) (*analytics.EvaluateBatchResult, error) {
	return analytics.EvaluateBatchForTest(ctx, snap, records, l)
}

func conditionAST(field string, val float64) vm.RuleNode {
	return vm.RuleNode{
		Type: vm.NodeTypeCondition,
		Condition: &vm.RuleCondition{
			ID:       "c1",
			Field:    field,
			Operator: "greater_than",
			Value:    val,
		},
	}
}

func snapshotWith(t *testing.T, rules ...analytics.EvaluableRule) *analytics.RuleSnapshot {
	t.Helper()
	return &analytics.RuleSnapshot{
		TenantID:   "t1",
		BOName:     "order",
		SnapshotID: "test-snap",
		Rules:      rules,
	}
}

func ruleWithAST(t *testing.T, id string, ast vm.RuleNode) analytics.EvaluableRule {
	t.Helper()
	return analytics.EvaluableRule{
		RuleID:      id,
		RuleKey:     id,
		RuleName:    id,
		BOName:      "order",
		Severity:    models.ValidationRuleSeverityBlock,
		RuleVersion: "1",
		AST:         &ast,
		FieldRefs:   vm.FieldRefs(ast),
	}
}

func postJSON(t *testing.T, path string, body any) *http.Request {
	t.Helper()
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(b)))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestEvaluateRecordViolationAndPass(t *testing.T) {
	h := &ValidationRuleHandler{
		evaluator: &stubEvalService{snap: snapshotWith(t, ruleWithAST(t, "qty-limit", conditionAST("TargetQuantity", 100)))},
	}

	// Violation
	rec := httptest.NewRecorder()
	h.handleEvaluateRecord(rec, postJSON(t, "/evaluate-record", map[string]any{
		"bo_name": "order",
		"record":  map[string]any{"TargetQuantity": 50.0},
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var res struct {
		Valid      bool                     `json:"valid"`
		Violations []models.ViolationRecord `json:"violations"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Valid || len(res.Violations) != 1 || res.Violations[0].RuleKey != "qty-limit" {
		t.Fatalf("unexpected: %+v", res)
	}
	if res.Violations[0].RuleVersion != "1" {
		t.Fatal("rule_version not stamped")
	}

	// Pass
	rec2 := httptest.NewRecorder()
	h.handleEvaluateRecord(rec2, postJSON(t, "/evaluate-record", map[string]any{
		"bo_name": "order",
		"record":  map[string]any{"TargetQuantity": 500.0},
	}))
	var res2 struct {
		Valid bool `json:"valid"`
	}
	_ = json.Unmarshal(rec2.Body.Bytes(), &res2)
	if !res2.Valid {
		t.Fatal("expected valid")
	}
}

func TestEvaluateRecordContextRequiredIs200(t *testing.T) {
	snap := snapshotWith(t,
		ruleWithAST(t, "ctx-rule", conditionAST("account_status", 1)),
	)
	snap.Rules[0].Severity = "BLOCK"
	h := &ValidationRuleHandler{evaluator: &stubEvalService{snap: snap}}

	rec := httptest.NewRecorder()
	h.handleEvaluateRecord(rec, postJSON(t, "/evaluate-record", map[string]any{
		"bo_name": "order",
		"record":  map[string]any{"TargetQuantity": 5.0},
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("context-required must be 200, got %d", rec.Code)
	}
	var res struct {
		Error   string   `json:"error"`
		Missing []string `json:"missing_context_fields"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Error != "ERR_SERVER_CONTEXT_REQUIRED" || len(res.Missing) != 1 {
		t.Fatalf("unexpected: %+v", res)
	}
}

func TestEvaluateBatchHandler(t *testing.T) {
	h := &ValidationRuleHandler{
		evaluator: &stubEvalService{snap: snapshotWith(t, ruleWithAST(t, "qty-limit", conditionAST("TargetQuantity", 100)))},
	}

	rec := httptest.NewRecorder()
	h.handleEvaluateBatch(rec, postJSON(t, "/evaluate-batch", map[string]any{
		"bo_name": "order",
		"records": []map[string]any{
			{"TargetQuantity": 150.0},
			{"TargetQuantity": 50.0},
		},
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var batchRes analytics.EvaluateBatchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &batchRes); err != nil {
		t.Fatalf("failed to unmarshal batch response: %v", err)
	}

	if batchRes.Summary.TotalRecords != 2 || batchRes.Summary.ValidCount != 1 || batchRes.Summary.InvalidCount != 1 {
		t.Fatalf("unexpected summary: %+v", batchRes.Summary)
	}
	if len(batchRes.Records) != 2 || !batchRes.Records[0].Valid || batchRes.Records[1].Valid {
		t.Fatalf("unexpected records alignment: %+v", batchRes.Records)
	}
}

func TestEvaluatePushdownHandler_MissingParams(t *testing.T) {
	h := &ValidationRuleHandler{}
	rec := httptest.NewRecorder()
	h.handleEvaluatePushdown(rec, postJSON(t, "/evaluate-pushdown", map[string]any{
		"bo_name": "",
	}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty bo_name, got %d", rec.Code)
	}
}

