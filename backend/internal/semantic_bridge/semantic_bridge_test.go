package semantic_bridge_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/semantic_bridge"
	"gopkg.in/yaml.v3"
)

func TestSemanticBridge_ModelsAndStructures(t *testing.T) {
	tenantID := uuid.New()
	targetID := uuid.New()

	target := semantic_bridge.BridgeTarget{
		ID:         targetID,
		TenantID:   tenantID,
		VendorType: semantic_bridge.VendorSnowflakeCortex,
		TargetName: "prod_snowflake_cortex",
		IsActive:   true,
	}

	if target.VendorType != semantic_bridge.VendorSnowflakeCortex {
		t.Fatalf("expected vendor type %s, got %s", semantic_bridge.VendorSnowflakeCortex, target.VendorType)
	}

	cortexSpec := semantic_bridge.CortexModelSpec{
		Name:        "uisce_semantic_model",
		Description: "Multi-tenant semantic model",
		Tables: []semantic_bridge.CortexTableSpec{
			{
				Name: "account",
				BaseTable: semantic_bridge.CortexBaseTable{
					Database: "ANALYTICS_DB",
					Schema:   "OMS",
					Table:    "ACCOUNT",
				},
				Dimensions: []semantic_bridge.CortexDimension{
					{Name: "account_number", Expr: "account_number", DataType: "VARCHAR"},
				},
				Measures: []semantic_bridge.CortexMeasure{
					{Name: "total_aum", Expr: "total_aum", DataType: "DECIMAL", DefaultAggregation: "SUM"},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(cortexSpec)
	if err != nil {
		t.Fatalf("failed marshaling cortex spec: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Fatalf("expected non-empty yaml bytes")
	}
}

func TestSemanticBridge_NilTenantGuard(t *testing.T) {
	exporter := semantic_bridge.NewCortexExporter(nil)
	_, err := exporter.CompileFullCortexModel(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatalf("expected Rule 7 violation error when tenant_id is nil")
	}

	genieExp := semantic_bridge.NewDatabricksExporter(nil)
	_, err = genieExp.CompileGenieModel(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatalf("expected Rule 7 violation error when tenant_id is nil")
	}

	mcpProv := semantic_bridge.NewMCPCatalogProvider(nil)
	_, err = mcpProv.GetSemanticCatalog(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatalf("expected Rule 7 violation error when tenant_id is nil")
	}
}
