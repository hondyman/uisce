package finops_test

import (
	"context"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/finops"
)

func TestChargebackMeterService_MultiTierAttribution(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	meter := finops.NewChargebackMeterService(nil)

	// Single 2-Page Institutional PDF Render Payload
	payload := finops.ReportRenderMeterPayload{
		TenantID:           tenantID,
		BatchID:            uuid.New(),
		ReportDefinitionID: uuid.New(),
		ClientSliceID:      "CLIENT_ACC_8819",
		ExportFormat:       "PDF",
		PageCount:          2,
		RenderDurationMs:   250,        // 0.25 sec
		FileSizeBytes:      524288,     // 0.5 MB
		ScannedBytes:       1073741824, // 1.0 GB
		ASTComplexityScore: 35.0,
		CPUDurationMs:      100,
		BackendType:        "ICEBERG",
	}

	result, err := meter.CalculateAndRecordRenderChargeback(ctx, payload)
	if err != nil {
		t.Fatalf("chargeback calculation failed: %v", err)
	}

	// 1. Verify Query Cost: (35.0 * 0.0005) + (1.0 GB * 0.02 * 1.0) = 0.0175 + 0.0214748...
	scannedGB := float64(payload.ScannedBytes) / 1e9
	expectedQueryCost := (35.0 * 0.0005) + (scannedGB * 0.02 * 1.0)
	if math.Abs(result.QueryCostUSD-expectedQueryCost) > 1e-6 {
		t.Errorf("expected query cost = %f, got %f", expectedQueryCost, result.QueryCostUSD)
	}

	// 2. Verify Render Cost: Base ($0.01) + 2 pages * $0.002 + 0.25 sec * $0.005 = 0.01 + 0.004 + 0.00125 = $0.01525
	expectedRenderCost := 0.01 + (2.0 * 0.002) + (0.25 * 0.005)
	if math.Abs(result.RenderCostUSD-expectedRenderCost) > 1e-6 {
		t.Errorf("expected render cost = %f, got %f", expectedRenderCost, result.RenderCostUSD)
	}

	// 3. Verify Total Calculation Sum
	expectedTotal := result.QueryCostUSD + result.VectorMathCostUSD + result.RenderCostUSD + result.StorageCostUSD
	if math.Abs(result.TotalCostUSD-expectedTotal) > 1e-6 {
		t.Errorf("total mismatch: %f != sum(%f)", result.TotalCostUSD, expectedTotal)
	}

	// 4. Security Check: Nil Tenant Violation
	payload.TenantID = uuid.Nil
	_, errNil := meter.CalculateAndRecordRenderChargeback(ctx, payload)
	if errNil == nil {
		t.Fatalf("expected Rule 7 violation error on nil tenant_id")
	}
}
