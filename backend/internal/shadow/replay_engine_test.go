package shadow_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/rules"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/hondyman/uisce/backend/internal/shadow"
)

func TestShadowReplayEngine_ProcessOrder_NoJobsRegistered(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	syms.Intern("order.quantity")
	syms.Intern("order.price")

	engine := shadow.NewReplayEngine(nil, syms, enums)

	tenantID := uuid.New()
	tradeData := map[string]any{
		"order.quantity": 5000.0,
		"order.price":    185.0,
	}

	engine.ProcessShadowOrder(context.Background(), tenantID, "ORD-001", true, tradeData)
}

func TestShadowReplayEngine_StartShadowJob_CompilesRule(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	syms.Intern("order.quantity")
	syms.Intern("order.price")

	engine := shadow.NewReplayEngine(nil, syms, enums)

	draftNode := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			Field:    "order.quantity",
			Operator: ">",
			Value:    1000.0,
			ValueType: "number",
		},
	}

	tenantID := uuid.New()
	draftRuleID := uuid.New()

	job, err := engine.StartShadowJob(
		context.Background(),
		tenantID,
		draftRuleID,
		"Shadow Concentration Rule",
		draftNode,
		"admin@uisce.internal",
	)
	if err != nil {
		t.Fatalf("unexpected error starting shadow job: %v", err)
	}

	if job == nil {
		t.Fatal("expected non-nil job")
	}

	if job.Status != "RUNNING" {
		t.Errorf("expected status 'RUNNING', got '%s'", job.Status)
	}

	if job.CompiledProgram == nil {
		t.Error("expected non-nil CompiledProgram")
	}
}

func TestShadowReplayEngine_ProcessShadowOrder_DiscrepancyDetected(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	syms.Intern("order.quantity")
	syms.Intern("order.price")

	engine := shadow.NewReplayEngine(nil, syms, enums)

	draftNode := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			Field:    "order.quantity",
			Operator: ">",
			Value:    1000.0,
			ValueType: "number",
		},
	}

	tenantID := uuid.New()
	draftRuleID := uuid.New()

	job, err := engine.StartShadowJob(
		context.Background(),
		tenantID,
		draftRuleID,
		"Shadow Concentration Rule",
		draftNode,
		"admin@uisce.internal",
	)
	if err != nil {
		t.Fatalf("unexpected error starting shadow job: %v", err)
	}

	tradeData := map[string]any{
		"order.quantity": 500.0,
		"order.price":    185.0,
	}

	engine.ProcessShadowOrder(context.Background(), tenantID, "ORD-991", true, tradeData)

	time.Sleep(100 * time.Millisecond)

	if job.TotalEvaluated != 1 {
		t.Errorf("expected 1 evaluated trade, got %d", job.TotalEvaluated)
	}

	if job.DiscrepancyCount != 1 {
		t.Errorf("expected 1 discrepancy (prod passed but shadow blocked on qty=500<1000), got %d", job.DiscrepancyCount)
	}

	if job.ProdPassedCount != 1 {
		t.Errorf("expected 1 prod pass, got %d", job.ProdPassedCount)
	}
}

func TestShadowReplayEngine_ProcessShadowOrder_NoDiscrepancy(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	syms.Intern("order.quantity")
	syms.Intern("order.price")

	engine := shadow.NewReplayEngine(nil, syms, enums)

	draftNode := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			Field:    "order.quantity",
			Operator: ">",
			Value:    1000.0,
			ValueType: "number",
		},
	}

	tenantID := uuid.New()
	draftRuleID := uuid.New()

	job, err := engine.StartShadowJob(
		context.Background(),
		tenantID,
		draftRuleID,
		"Shadow Concentration Rule",
		draftNode,
		"admin@uisce.internal",
	)
	if err != nil {
		t.Fatalf("unexpected error starting shadow job: %v", err)
	}

	tradeData := map[string]any{
		"order.quantity": 5000.0,
		"order.price":    185.0,
	}

	engine.ProcessShadowOrder(context.Background(), tenantID, "ORD-002", true, tradeData)

	time.Sleep(100 * time.Millisecond)

	if job.DiscrepancyCount != 0 {
		t.Errorf("expected 0 discrepancies (trade 5000 > 1000, both pass), got %d", job.DiscrepancyCount)
	}
}

func TestShadowReplayEngine_GetImpactReport_NoDB(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	engine := shadow.NewReplayEngine(nil, syms, enums)

	_, err := engine.GetImpactReport(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error when database is nil")
	}
}

func TestShadowReplayEngine_MultipleJobsPerTenant(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	syms.Intern("order.quantity")
	syms.Intern("order.price")

	engine := shadow.NewReplayEngine(nil, syms, enums)

	tenantID := uuid.New()

	rule1 := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			Field:    "order.quantity",
			Operator: ">",
			Value:    1000.0,
			ValueType: "number",
		},
	}

	rule2 := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			Field:    "order.price",
			Operator: "<",
			Value:    200.0,
			ValueType: "number",
		},
	}

	job1, err := engine.StartShadowJob(context.Background(), tenantID, uuid.New(), "Rule1", rule1, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job2, err := engine.StartShadowJob(context.Background(), tenantID, uuid.New(), "Rule2", rule2, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tradeData := map[string]any{
		"order.quantity": 500.0,
		"order.price":    185.0,
	}

	engine.ProcessShadowOrder(context.Background(), tenantID, "ORD-100", true, tradeData)

	time.Sleep(100 * time.Millisecond)

	if job1.TotalEvaluated != 1 {
		t.Errorf("job1: expected 1 evaluation, got %d", job1.TotalEvaluated)
	}
	if job2.TotalEvaluated != 1 {
		t.Errorf("job2: expected 1 evaluation, got %d", job2.TotalEvaluated)
	}

	if job1.DiscrepancyCount != 1 {
		t.Errorf("job1: expected 1 discrepancy (qty=500 < 1000: shadow blocked, prod passed), got %d", job1.DiscrepancyCount)
	}
	if job2.DiscrepancyCount != 0 {
		t.Errorf("job2: expected 0 discrepancies (price 185 < 200: both pass), got %d", job2.DiscrepancyCount)
	}
}

func TestShadowReplayEngine_CancelJob(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	engine := shadow.NewReplayEngine(nil, syms, enums)

	draftNode := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			Field:    "order.quantity",
			Operator: ">",
			Value:    1000.0,
			ValueType: "number",
		},
	}

	tenantID := uuid.New()
	job, err := engine.StartShadowJob(context.Background(), tenantID, uuid.New(), "TestJob", draftNode, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = engine.CancelJob(context.Background(), job.JobID)
	if err != nil {
		t.Errorf("unexpected error cancelling job: %v", err)
	}

	if job.Status != "CANCELLED" {
		t.Errorf("expected status 'CANCELLED', got '%s'", job.Status)
	}
}

func TestShadowReplayEngine_CompleteJob(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	engine := shadow.NewReplayEngine(nil, syms, enums)

	draftNode := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			Field:    "order.quantity",
			Operator: ">",
			Value:    1000.0,
			ValueType: "number",
		},
	}

	tenantID := uuid.New()
	job, err := engine.StartShadowJob(context.Background(), tenantID, uuid.New(), "TestJob", draftNode, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = engine.CompleteJob(context.Background(), job.JobID)
	if err != nil {
		t.Errorf("unexpected error completing job: %v", err)
	}

	if job.Status != "COMPLETED" {
		t.Errorf("expected status 'COMPLETED', got '%s'", job.Status)
	}
}

func TestShadowReplayEngine_ConcurrentEvaluations(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	syms.Intern("order.quantity")
	syms.Intern("order.price")

	engine := shadow.NewReplayEngine(nil, syms, enums)

	draftNode := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			Field:    "order.quantity",
			Operator: ">",
			Value:    1000.0,
			ValueType: "number",
		},
	}

	tenantID := uuid.New()
	job, err := engine.StartShadowJob(context.Background(), tenantID, uuid.New(), "TestJob", draftNode, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tradeData := map[string]any{
				"order.quantity": float64(idx * 100),
				"order.price":    185.0,
			}
			prodPassed := idx*100 > 1000
			engine.ProcessShadowOrder(context.Background(), tenantID, "ORD-"+string(rune('A'+idx)), prodPassed, tradeData)
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	if job.TotalEvaluated != 100 {
		t.Errorf("expected 100 evaluations, got %d", job.TotalEvaluated)
	}
}
