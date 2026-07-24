package appslos

import (
	"context"

	"github.com/google/uuid"
)

type AppSLOType string

const (
	SLOTypeWorkflowTime  AppSLOType = "workflow_completion_time"
	SLOTypeCrossPageLat  AppSLOType = "cross_page_latency"
	SLOTypeMutationRate  AppSLOType = "mutation_success_rate"
	SLOTypeDataFreshness AppSLOType = "data_freshness"
)

type AppSLO struct {
	ID          uuid.UUID  `json:"id"`
	AppID       uuid.UUID  `json:"app_id"`
	Type        AppSLOType `json:"type"`
	Target      float64    `json:"target"`
	TargetUnit  string     `json:"target_unit"` // ms, %, minutes
	Percentile  float64    `json:"percentile,omitempty"`
	Window      string     `json:"window"`
	MetricQuery string     `json:"metric_query"`
}

type AppSLOStatus struct {
	SLOID      uuid.UUID `json:"slo_id"`
	Status     string    `json:"status"` // passing, failing
	CurrentVal float64   `json:"current_val"`
	TargetVal  float64   `json:"target_val"`
	Gap        float64   `json:"gap"`
}

type AppSLOEvaluator struct{}

func NewAppSLOEvaluator() *AppSLOEvaluator {
	return &AppSLOEvaluator{}
}

func (e *AppSLOEvaluator) EvaluateSLOs(ctx context.Context, appID uuid.UUID) ([]AppSLOStatus, error) {
	// Mock: Evaluate app-level SLOs
	// Real: Query metrics DB for workflow times, cross-page latencies, etc.
	return []AppSLOStatus{
		{
			SLOID:      uuid.New(),
			Status:     "passing",
			CurrentVal: 95000,  // 95 seconds
			TargetVal:  120000, // 2 minutes
			Gap:        -25000,
		},
		{
			SLOID:      uuid.New(),
			Status:     "failing",
			CurrentVal: 6200, // 6.2 seconds
			TargetVal:  5000, // 5 seconds
			Gap:        1200,
		},
	}, nil
}

func (e *AppSLOEvaluator) CreateSLO(ctx context.Context, slo *AppSLO) error {
	// Mock: Save SLO
	slo.ID = uuid.New()
	return nil
}
