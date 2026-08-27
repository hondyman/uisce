package finops

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var GoldCopyTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

type ChargebackMeterService struct {
	db *sqlx.DB
}

func NewChargebackMeterService(db *sqlx.DB) *ChargebackMeterService {
	return &ChargebackMeterService{db: db}
}

// ResolveEffectiveRate fetches tenant-specific rate with Gold Copy fallback (Rule 1 & Rule 9)
func (s *ChargebackMeterService) ResolveEffectiveRate(
	ctx context.Context,
	tenantID uuid.UUID,
	backendType string,
) (*RateCard, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var rate RateCard
	if s.db == nil {
		return &RateCard{
			TenantID:                tenantID,
			BackendType:             backendType,
			ComplexityRatePerUnit:   0.0005,
			VolumeRatePerGB:         0.02,
			CPUSecondRate:           0.005,
			PDFBaseArtifactRate:     0.01,
			PDFPageRate:             0.002,
			ExcelBaseArtifactRate:   0.005,
			StorageRatePerGBMonth:   0.023,
			BackendWeightMultiplier: 1.0,
		}, nil
	}

	query := `
		SELECT tenant_id, backend_type, complexity_rate_per_unit, volume_rate_per_gb,
		       cpu_second_rate, pdf_base_artifact_rate, pdf_page_rate,
		       excel_base_artifact_rate, storage_rate_per_gb_month, backend_weight_multiplier
		FROM finops.bo_charge_rates
		WHERE (tenant_id = $1 OR tenant_id = $2)
		  AND (backend_type = $3 OR backend_type = 'DEFAULT')
		  AND is_active = TRUE
		ORDER BY (tenant_id = $1) DESC, (backend_type = $3) DESC, active_from DESC
		LIMIT 1;
	`
	err := s.db.GetContext(ctx, &rate, query, tenantID, GoldCopyTenantID, backendType)
	if err != nil {
		return nil, fmt.Errorf("failed resolving rate card for tenant %s: %w", tenantID, err)
	}

	return &rate, nil
}

// CalculateAndRecordRenderChargeback meters and persists itemized cost per client document slice
func (s *ChargebackMeterService) CalculateAndRecordRenderChargeback(
	ctx context.Context,
	payload ReportRenderMeterPayload,
) (*ItemizedCostResult, error) {
	if payload.TenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	rate, err := s.ResolveEffectiveRate(ctx, payload.TenantID, payload.BackendType)
	if err != nil {
		return nil, err
	}

	// 1. Query Cost (Complexity + Scanned Volume * Backend Multiplier)
	scannedGB := float64(payload.ScannedBytes) / 1e9
	multiplier := rate.BackendWeightMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}
	queryCost := (payload.ASTComplexityScore * rate.ComplexityRatePerUnit) + (scannedGB * rate.VolumeRatePerGB * multiplier)

	// 2. Vector Math Compute Cost
	cpuSeconds := float64(payload.CPUDurationMs) / 1000.0
	vectorCost := cpuSeconds * rate.CPUSecondRate

	// 3. Document Rendering Cost (Base Artifact + Per-Page + Render CPU)
	var renderCost float64
	renderCPUSeconds := float64(payload.RenderDurationMs) / 1000.0

	if payload.ExportFormat == "PDF" {
		renderCost = rate.PDFBaseArtifactRate + (float64(payload.PageCount) * rate.PDFPageRate) + (renderCPUSeconds * rate.CPUSecondRate)
	} else {
		renderCost = rate.ExcelBaseArtifactRate + (renderCPUSeconds * rate.CPUSecondRate)
	}

	// 4. Monthly Vault Storage Cost
	fileSizeGB := float64(payload.FileSizeBytes) / 1e9
	storageCost := fileSizeGB * rate.StorageRatePerGBMonth

	totalCost := queryCost + vectorCost + renderCost + storageCost
	period := time.Now().UTC().Format("2006-01")

	result := &ItemizedCostResult{
		QueryCostUSD:      queryCost,
		VectorMathCostUSD: vectorCost,
		RenderCostUSD:     renderCost,
		StorageCostUSD:    storageCost,
		TotalCostUSD:      totalCost,
	}

	// 5. Persist to Chargeback Ledger
	if s.db != nil {
		insertSQL := `
			INSERT INTO finops.report_render_chargeback_ledger (
				tenant_id, batch_id, schedule_id, report_definition_id,
				client_slice_id, export_format, page_count, render_duration_ms,
				file_size_bytes, scanned_bytes, ast_complexity_score,
				query_cost_usd, vector_math_cost_usd, render_cost_usd, storage_cost_usd,
				total_cost_usd, billing_period
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17);
		`
		_, err = s.db.ExecContext(ctx, insertSQL,
			payload.TenantID, payload.BatchID, payload.ScheduleID, payload.ReportDefinitionID,
			payload.ClientSliceID, payload.ExportFormat, payload.PageCount, payload.RenderDurationMs,
			payload.FileSizeBytes, payload.ScannedBytes, payload.ASTComplexityScore,
			result.QueryCostUSD, result.VectorMathCostUSD, result.RenderCostUSD, result.StorageCostUSD,
			result.TotalCostUSD, period,
		)
		if err != nil {
			return nil, fmt.Errorf("failed recording chargeback entry: %w", err)
		}
	}

	return result, nil
}
