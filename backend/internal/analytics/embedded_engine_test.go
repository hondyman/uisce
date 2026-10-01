package analytics

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"
)

func TestEmbeddedEngineColdCacheFailClosed(t *testing.T) {
	// No DB available: a cold-cache ValidateRecord must error, never evaluate
	// against an empty snapshot (empty snapshot = silently no rules = incident).
	e := NewEmbeddedEngine(nil, "tenant-x", nil, WithoutNotify())
	_, err := e.ValidateRecord(context.Background(), "order", "", "", map[string]any{})
	if err == nil {
		t.Fatal("cold cache with no DB must fail closed")
	}
}

func TestEmbeddedEngineWarmStaleServesOnRefreshFailure(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	ast := vm.RuleNode{
		Type: vm.NodeTypeCondition,
		Condition: &vm.RuleCondition{
			ID:       "c1",
			Field:    "Quantity",
			Operator: "greater_than",
			Value:    0.0,
		},
	}
	snap := &RuleSnapshot{
		TenantID:   "tenant-x",
		BOName:     "order",
		SnapshotID: "snap-1",
		Rules: []EvaluableRule{
			{
				RuleID:   "r1",
				RuleKey:  "qty_pos",
				RuleName: "Quantity Positive",
				Severity: models.ValidationRuleSeverityBlock,
				AST:      &ast,
			},
		},
	}

	e := NewEmbeddedEngine(sqlxDB, "tenant-x", nil, WithoutNotify())
	key := snapshotKey{bo: "order", domain: "", timing: ""}
	ptr := &atomic.Pointer[RuleSnapshot]{}
	ptr.Store(snap)
	e.snapshot[key] = ptr
	e.versions[key] = "v1"

	// ValidateRecord should serve the warm snapshot even though svc is nil (refresh fails)
	res, err := e.ValidateRecord(context.Background(), "order", "", "", map[string]any{
		"Quantity": 10.0,
	})
	if err != nil {
		t.Fatalf("expected warm snapshot to serve on refresh failure, got err: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected record to be valid, got: %v", res)
	}
	if atomic.LoadInt64(&e.Metrics.StaleServes) == 0 {
		t.Fatal("expected StaleServes to be incremented")
	}
}

func TestEmbeddedEngineInvalidateForcesReprobe(t *testing.T) {
	e := NewEmbeddedEngine(nil, "tenant-x", nil, WithoutNotify())
	key := snapshotKey{bo: "order", domain: "", timing: ""}
	ptr := &atomic.Pointer[RuleSnapshot]{}
	ptr.Store(&RuleSnapshot{SnapshotID: "s1"})
	e.snapshot[key] = ptr
	e.versions[key] = "v1"

	if len(e.snapshot) == 0 {
		t.Fatal("expected snapshot to be present")
	}

	e.Invalidate()

	if len(e.snapshot) != 0 {
		t.Fatalf("expected snapshot map to be empty after Invalidate, got len %d", len(e.snapshot))
	}
}
