package uxslo

import (
	"time"

	"github.com/google/uuid"
)

type SLOType string

const (
	SLOTypeMaxClicks        SLOType = "max_clicks"
	SLOTypeModalLatency     SLOType = "modal_latency"
	SLOTypeInteractionLat   SLOType = "interaction_latency"
	SLOTypeWorkflowDuration SLOType = "workflow_completion_time"
)

type UXContract struct {
	ID          uuid.UUID `json:"id" db:"id"`
	PageID      uuid.UUID `json:"page_id" db:"page_id"`
	TenantID    string    `json:"tenant_id,omitempty" db:"tenant_id"` // Optional if global
	Type        SLOType   `json:"type" db:"type"`
	Target      float64   `json:"target" db:"target"`                   // e.g. 3 (clicks), 300 (ms)
	TargetUnit  string    `json:"target_unit" db:"target_unit"`         // clicks, ms, s
	Percentile  float64   `json:"percentile,omitempty" db:"percentile"` // e.g. 95 for p95
	Window      string    `json:"window" db:"window"`                   // e.g. "7d"
	MetricQuery string    `json:"metric_query" db:"metric_query"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type SLOStatus struct {
	ContractID uuid.UUID `json:"contract_id"`
	Status     string    `json:"status"` // passing, failing, warning
	Current    float64   `json:"current_val"`
	Target     float64   `json:"target_val"`
	Gap        float64   `json:"gap"`
}
