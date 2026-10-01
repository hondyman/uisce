package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestLoadRuleSnapshotAsOf_BitemporalQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	svc := NewValidationRuleService(sqlxDB)

	ruleID := uuid.New()
	asOfTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// Mock gold copy tenant
	mock.ExpectQuery(`SELECT public\.uisce_gold_copy_tenant_id\(\)::text`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(uuid.New().String()))

	// Mock bitemporal versions query
	mock.ExpectQuery(`SELECT v\.rule_node_id, v\.version, v\.rule_ast, v\.properties FROM validation_rule_versions`).
		WithArgs(sqlmock.AnyArg(), "order", asOfTime).
		WillReturnRows(sqlmock.NewRows([]string{"rule_node_id", "version", "rule_ast", "properties"}).
			AddRow(
				ruleID,
				3,
				[]byte(`{"type":"condition","id":"c1","field":"Qty","operator":"greater_than","value":10.0}`),
				[]byte(`{"rule_key":"qty_check","bo_name":"order","severity":"BLOCK","domain":"default","governance_status":"published"}`),
			))

	// Mock column mappings
	mock.ExpectQuery(`SELECT bf\.field_name, COALESCE`).
		WithArgs("order", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"field_name", "column_name"}).AddRow("Qty", "quantity"))

	snap, err := svc.LoadRuleSnapshotAsOf(context.Background(), uuid.New().String(), "order", "", "", asOfTime)
	if err != nil {
		t.Fatalf("LoadRuleSnapshotAsOf failed: %v", err)
	}

	if len(snap.Rules) != 1 {
		t.Fatalf("expected 1 rule in as-of snapshot, got %d", len(snap.Rules))
	}
	if snap.Rules[0].RuleVersion != "3" {
		t.Fatalf("expected rule version 3, got %s", snap.Rules[0].RuleVersion)
	}
	if snap.Rules[0].RuleKey != "qty_check" {
		t.Fatalf("expected rule key qty_check, got %s", snap.Rules[0].RuleKey)
	}
	if snap.LoadedAt != asOfTime {
		t.Fatalf("expected LoadedAt %v, got %v", asOfTime, snap.LoadedAt)
	}
}

func TestLoadRuleSnapshot_ChokePointFiltersDraftAndReview(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	svc := NewValidationRuleService(sqlxDB)

	goldID := uuid.New().String()

	// Mock gold copy tenant (called in LoadRuleSnapshotAsOf)
	mock.ExpectQuery(`SELECT public\.uisce_gold_copy_tenant_id\(\)::text`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(goldID))

	// Mock gold copy tenant (called in ListByBO)
	mock.ExpectQuery(`SELECT public\.uisce_gold_copy_tenant_id\(\)::text`).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(goldID))

	// Mock catalog_node query returning 1 published, 1 draft, 1 submitted_for_review, 1 legacy empty status
	pubID := uuid.New()
	draftID := uuid.New()
	reviewID := uuid.New()
	legacyEmptyID := uuid.New()

	mock.ExpectQuery(`SELECT n\.id, n\.node_name, COALESCE\(n\.description, ''\) as description, n\.properties, n\.config, n\.is_active, n\.tenant_id::text AS tenant_id FROM catalog_node n`).
		WithArgs(sqlmock.AnyArg(), "order", "", "validation", "survivorship").
		WillReturnRows(sqlmock.NewRows([]string{"id", "node_name", "description", "properties", "config", "is_active", "tenant_id"}).
			AddRow(
				pubID,
				"Published Rule",
				"",
				[]byte(`{"bo_name":"order","severity":"BLOCK","domain":"default","governance_status":"published"}`),
				[]byte(`{"rule_ast":{"type":"condition","id":"c1","field":"Qty","operator":"greater_than","value":1.0}}`),
				true,
				goldID,
			).
			AddRow(
				draftID,
				"Draft Rule",
				"",
				[]byte(`{"bo_name":"order","severity":"BLOCK","domain":"default","governance_status":"draft"}`),
				[]byte(`{"rule_ast":{"type":"condition","id":"c2","field":"Qty","operator":"greater_than","value":2.0}}`),
				true,
				goldID,
			).
			AddRow(
				reviewID,
				"Review Rule",
				"",
				[]byte(`{"bo_name":"order","severity":"BLOCK","domain":"default","governance_status":"submitted_for_review"}`),
				[]byte(`{"rule_ast":{"type":"condition","id":"c3","field":"Qty","operator":"greater_than","value":3.0}}`),
				true,
				goldID,
			).
			AddRow(
				legacyEmptyID,
				"Legacy Unbackfilled Rule",
				"",
				[]byte(`{"bo_name":"order","severity":"BLOCK","domain":"default"}`),
				[]byte(`{"rule_ast":{"type":"condition","id":"c4","field":"Qty","operator":"greater_than","value":4.0}}`),
				true,
				goldID,
			))

	// Mock column mappings
	mock.ExpectQuery(`SELECT bf\.field_name, COALESCE`).
		WithArgs("order", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"field_name", "column_name"}).AddRow("Qty", "quantity"))

	snap, err := svc.LoadRuleSnapshot(context.Background(), uuid.New().String(), "order", "", "")
	if err != nil {
		t.Fatalf("LoadRuleSnapshot failed: %v", err)
	}

	// Choke point verification: only the 1 published rule should be loaded into the snapshot
	if len(snap.Rules) != 1 {
		t.Fatalf("choke point failed: expected exactly 1 published rule, got %d", len(snap.Rules))
	}
	if snap.Rules[0].RuleID != pubID.String() {
		t.Fatalf("expected published rule ID %s, got %s", pubID.String(), snap.Rules[0].RuleID)
	}
}
