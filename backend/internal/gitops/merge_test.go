package gitops_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/gitops"
)

func TestThreeWayMerge_CleanMerge(t *testing.T) {
	tenantID := uuid.New()
	engine := gitops.NewOverlayMergeEngine()

	ancestor := &gitops.BusinessObjectOverlay{
		BOKey: "Portfolio", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "portfolio_id", DataType: "string", IsRequired: true},
		},
	}

	upstream := &gitops.BusinessObjectOverlay{
		BOKey: "Portfolio", TenantID: tenantID, Version: "v1.1.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "portfolio_id", DataType: "string", IsRequired: true},
			{FieldKey: "esg_rating", DataType: "float", IsRequired: false},
		},
	}

	tenant := &gitops.BusinessObjectOverlay{
		BOKey: "Portfolio", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "portfolio_id", DataType: "string", IsRequired: true},
			{FieldKey: "custom_cost_center", DataType: "string", IsRequired: false},
		},
	}

	res, err := engine.ThreeWayMerge(ancestor, upstream, tenant)
	if err != nil {
		t.Fatalf("unexpected error during 3-way merge: %v", err)
	}

	if res.HasConflict {
		t.Fatalf("expected clean merge with 0 conflicts, got %d", len(res.Conflicts))
	}

	if len(res.MergedBO.Fields) != 3 {
		t.Errorf("expected 3 fields in merged BO, got %d", len(res.MergedBO.Fields))
	}
}

func TestThreeWayMerge_ConflictSameKeyType(t *testing.T) {
	tenantID := uuid.New()
	engine := gitops.NewOverlayMergeEngine()

	ancestor := &gitops.BusinessObjectOverlay{
		BOKey: "Portfolio", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "portfolio_id", DataType: "string", IsRequired: true},
		},
	}

	upstream := &gitops.BusinessObjectOverlay{
		BOKey: "Portfolio", TenantID: tenantID, Version: "v1.1.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "portfolio_id", DataType: "float", IsRequired: true},
		},
	}

	tenant := &gitops.BusinessObjectOverlay{
		BOKey: "Portfolio", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "portfolio_id", DataType: "int", IsRequired: true},
		},
	}

	res, err := engine.ThreeWayMerge(ancestor, upstream, tenant)
	if err != nil {
		t.Fatalf("unexpected error during 3-way merge: %v", err)
	}

	if !res.HasConflict {
		t.Fatal("expected conflict since upstream=float and tenant=int")
	}

	if len(res.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(res.Conflicts))
	}

	conflict := res.Conflicts[0]
	if conflict.FieldKey != "portfolio_id" {
		t.Errorf("expected conflict on 'portfolio_id', got '%s'", conflict.FieldKey)
	}
	if conflict.BaseValue != "string" {
		t.Errorf("expected base value 'string', got '%s'", conflict.BaseValue)
	}
	if conflict.UpstreamValue != "float" {
		t.Errorf("expected upstream value 'float', got '%s'", conflict.UpstreamValue)
	}
	if conflict.TenantValue != "int" {
		t.Errorf("expected tenant value 'int', got '%s'", conflict.TenantValue)
	}
}

func TestThreeWayMerge_TenantOnlyAddition(t *testing.T) {
	tenantID := uuid.New()
	engine := gitops.NewOverlayMergeEngine()

	ancestor := &gitops.BusinessObjectOverlay{
		BOKey: "Trade", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "trade_id", DataType: "string", IsRequired: true},
			{FieldKey: "quantity", DataType: "int", IsRequired: true},
		},
	}

	upstream := &gitops.BusinessObjectOverlay{
		BOKey: "Trade", TenantID: tenantID, Version: "v1.1.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "trade_id", DataType: "string", IsRequired: true},
			{FieldKey: "quantity", DataType: "int", IsRequired: true},
			{FieldKey: "execution_timestamp", DataType: "timestamp", IsRequired: false},
		},
	}

	tenant := &gitops.BusinessObjectOverlay{
		BOKey: "Trade", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "trade_id", DataType: "string", IsRequired: true},
			{FieldKey: "quantity", DataType: "int", IsRequired: true},
			{FieldKey: "desk_code", DataType: "string", IsRequired: false},
		},
	}

	res, err := engine.ThreeWayMerge(ancestor, upstream, tenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.HasConflict {
		t.Errorf("expected no conflicts, got %d", len(res.Conflicts))
	}

	if len(res.MergedBO.Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(res.MergedBO.Fields))
	}

	fieldKeys := make(map[string]bool)
	for _, f := range res.MergedBO.Fields {
		fieldKeys[f.FieldKey] = true
	}

	for _, k := range []string{"trade_id", "quantity", "execution_timestamp", "desk_code"} {
		if !fieldKeys[k] {
			t.Errorf("expected merged BO to contain field '%s'", k)
		}
	}
}

func TestThreeWayMerge_UpstreamOnlyModification(t *testing.T) {
	tenantID := uuid.New()
	engine := gitops.NewOverlayMergeEngine()

	ancestor := &gitops.BusinessObjectOverlay{
		BOKey: "Position", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "position_id", DataType: "string", IsRequired: true},
		},
	}

	upstream := &gitops.BusinessObjectOverlay{
		BOKey: "Position", TenantID: tenantID, Version: "v1.1.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "position_id", DataType: "uuid", IsRequired: true},
		},
	}

	tenant := &gitops.BusinessObjectOverlay{
		BOKey: "Position", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "position_id", DataType: "string", IsRequired: true},
		},
	}

	res, err := engine.ThreeWayMerge(ancestor, upstream, tenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.HasConflict {
		t.Errorf("expected no conflict (upstream modified, tenant kept ancestor value)")
	}

	if len(res.MergedBO.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(res.MergedBO.Fields))
	}

	if res.MergedBO.Fields[0].DataType != "uuid" {
		t.Errorf("expected merged field to take upstream value 'uuid', got '%s'", res.MergedBO.Fields[0].DataType)
	}
}

func TestThreeWayMerge_BothModifySameWay(t *testing.T) {
	tenantID := uuid.New()
	engine := gitops.NewOverlayMergeEngine()

	ancestor := &gitops.BusinessObjectOverlay{
		BOKey: "Order", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "order_ref", DataType: "string", IsRequired: true},
		},
	}

	upstream := &gitops.BusinessObjectOverlay{
		BOKey: "Order", TenantID: tenantID, Version: "v1.1.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "order_ref", DataType: "string", IsRequired: true},
			{FieldKey: "synthetic_priority", DataType: "int", IsRequired: false},
		},
	}

	tenant := &gitops.BusinessObjectOverlay{
		BOKey: "Order", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "order_ref", DataType: "string", IsRequired: true},
			{FieldKey: "synthetic_priority", DataType: "int", IsRequired: false},
		},
	}

	res, err := engine.ThreeWayMerge(ancestor, upstream, tenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.HasConflict {
		t.Errorf("expected no conflict (both agree on int)")
	}

	if len(res.MergedBO.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(res.MergedBO.Fields))
	}
}

func TestThreeWayMerge_EmptyAncestor(t *testing.T) {
	tenantID := uuid.New()
	engine := gitops.NewOverlayMergeEngine()

	ancestor := &gitops.BusinessObjectOverlay{
		BOKey: "NewBO", TenantID: tenantID, Version: "v0.0.0",
		Fields:   []gitops.FieldDefinition{},
	}

	upstream := &gitops.BusinessObjectOverlay{
		BOKey: "NewBO", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "id", DataType: "uuid", IsRequired: true},
			{FieldKey: "name", DataType: "string", IsRequired: true},
		},
	}

	tenant := &gitops.BusinessObjectOverlay{
		BOKey: "NewBO", TenantID: tenantID, Version: "v0.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "id", DataType: "uuid", IsRequired: true},
			{FieldKey: "region_code", DataType: "string", IsRequired: false},
		},
	}

	res, err := engine.ThreeWayMerge(ancestor, upstream, tenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.HasConflict {
		t.Errorf("expected no conflict for empty ancestor case")
	}

	if len(res.MergedBO.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(res.MergedBO.Fields))
	}
}

func TestThreeWayMerge_TenantDeletesField(t *testing.T) {
	tenantID := uuid.New()
	engine := gitops.NewOverlayMergeEngine()

	ancestor := &gitops.BusinessObjectOverlay{
		BOKey: "Account", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "account_id", DataType: "string", IsRequired: true},
			{FieldKey: "legacy_ref", DataType: "string", IsRequired: false},
		},
	}

	upstream := &gitops.BusinessObjectOverlay{
		BOKey: "Account", TenantID: tenantID, Version: "v1.1.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "account_id", DataType: "string", IsRequired: true},
			{FieldKey: "legacy_ref", DataType: "string", IsRequired: false},
		},
	}

	tenant := &gitops.BusinessObjectOverlay{
		BOKey: "Account", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "account_id", DataType: "string", IsRequired: true},
		},
	}

	res, err := engine.ThreeWayMerge(ancestor, upstream, tenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.HasConflict {
		t.Errorf("expected no conflict for tenant removing field")
	}

	if len(res.MergedBO.Fields) != 2 {
		t.Errorf("expected 2 fields (tenant removal should not auto-delete upstream-preserved field), got %d", len(res.MergedBO.Fields))
	}
}

func TestThreeWayMerge_UpstreamDeletesField(t *testing.T) {
	tenantID := uuid.New()
	engine := gitops.NewOverlayMergeEngine()

	ancestor := &gitops.BusinessObjectOverlay{
		BOKey: "Account", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "account_id", DataType: "string", IsRequired: true},
			{FieldKey: "deprecated_field", DataType: "string", IsRequired: false},
		},
	}

	upstream := &gitops.BusinessObjectOverlay{
		BOKey: "Account", TenantID: tenantID, Version: "v1.1.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "account_id", DataType: "string", IsRequired: true},
		},
	}

	tenant := &gitops.BusinessObjectOverlay{
		BOKey: "Account", TenantID: tenantID, Version: "v1.0.0",
		Fields: []gitops.FieldDefinition{
			{FieldKey: "account_id", DataType: "string", IsRequired: true},
			{FieldKey: "deprecated_field", DataType: "string", IsRequired: false},
		},
	}

	res, err := engine.ThreeWayMerge(ancestor, upstream, tenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.HasConflict {
		t.Errorf("expected no conflict for upstream removing deprecated field")
	}

	if len(res.MergedBO.Fields) != 1 {
		t.Errorf("expected 1 field (upstream removal should take precedence), got %d", len(res.MergedBO.Fields))
	}
}
