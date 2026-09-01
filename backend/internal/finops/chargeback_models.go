package finops

import (
	"github.com/google/uuid"
)

type RateCard struct {
	TenantID                uuid.UUID `db:"tenant_id"`
	BackendType             string    `db:"backend_type"`
	ComplexityRatePerUnit   float64   `db:"complexity_rate_per_unit"`
	VolumeRatePerGB         float64   `db:"volume_rate_per_gb"`
	CPUSecondRate           float64   `db:"cpu_second_rate"`
	PDFBaseArtifactRate     float64   `db:"pdf_base_artifact_rate"`
	PDFPageRate             float64   `db:"pdf_page_rate"`
	ExcelBaseArtifactRate   float64   `db:"excel_base_artifact_rate"`
	StorageRatePerGBMonth   float64   `db:"storage_rate_per_gb_month"`
	BackendWeightMultiplier float64   `db:"backend_weight_multiplier"`
}

type ReportRenderMeterPayload struct {
	TenantID           uuid.UUID  `json:"tenantId"`
	BatchID            uuid.UUID  `json:"batchId"`
	ScheduleID         *uuid.UUID `json:"scheduleId,omitempty"`
	ReportDefinitionID uuid.UUID  `json:"reportDefinitionId"`
	ClientSliceID      string     `json:"clientSliceId"`
	ExportFormat       string     `json:"exportFormat"` // "PDF" | "EXCEL"
	PageCount          int        `json:"pageCount"`
	RenderDurationMs   int        `json:"renderDurationMs"`
	FileSizeBytes      int64      `json:"fileSizeBytes"`
	ScannedBytes       int64      `json:"scannedBytes"`
	ASTComplexityScore float64    `json:"astComplexityScore"`
	CPUDurationMs      int        `json:"cpuDurationMs"`
	BackendType        string     `json:"backendType"` // "STARROCKS" | "ICEBERG" | "POSTGRES"
}

type ItemizedCostResult struct {
	QueryCostUSD      float64 `json:"queryCostUsd"`
	VectorMathCostUSD float64 `json:"vectorMathCostUsd"`
	RenderCostUSD     float64 `json:"renderCostUsd"`
	StorageCostUSD    float64 `json:"storageCostUsd"`
	TotalCostUSD      float64 `json:"totalCostUsd"`
}
