package datapipeline

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PipelineDefinition represents a saved visual data pipeline DAG
type PipelineDefinition struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	TenantID       uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	Name           string          `db:"name" json:"name"`
	Description    string          `db:"description" json:"description"`
	Mode           string          `db:"mode" json:"mode"`                   // "business_object", "catalog_graph", "hybrid", "external"
	TargetEntity   string          `db:"target_entity" json:"target_entity"` // e.g. "oms.trade_order", "catalog_node", etc.
	DAGJSON        json.RawMessage `db:"dag_json" json:"dag_json"`
	Concurrency    int             `db:"concurrency" json:"concurrency"`   // Worker count (default 8)
	BatchSize      int             `db:"batch_size" json:"batch_size"`     // Chunk size (default 2000)
	ErrorPolicy    string          `db:"error_policy" json:"error_policy"` // "fail_fast", "skip_and_log", "dead_letter"
	IsActive       bool            `db:"is_active" json:"is_active"`
	CreatedBy      string          `db:"created_by" json:"created_by"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	LastModifiedAt time.Time       `db:"last_modified_at" json:"last_modified_at"`
}

// PipelineNode represents a tile in the ReactFlow canvas
type PipelineNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`    // "source", "transform", "validator", "loader", "sink", "graph_synthesizer"
	SubType  string                 `json:"subType"` // specific operator subtype
	Label    string                 `json:"label"`
	Config   map[string]interface{} `json:"config"`
	Position struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"position"`
}

// PipelineEdge represents a connection between tiles
type PipelineEdge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
	Label        string `json:"label,omitempty"`
}

// PipelineDAG is the compiled graph structure
type PipelineDAG struct {
	Nodes []PipelineNode `json:"nodes"`
	Edges []PipelineEdge `json:"edges"`
}

// PipelineRecord is a generic key-value dictionary processed by the engine
type PipelineRecord map[string]interface{}

// StepMetrics captures runtime performance for a single pipeline step
type StepMetrics struct {
	NodeID         string        `json:"node_id"`
	NodeLabel      string        `json:"node_label"`
	NodeType       string        `json:"node_type"`
	RecordsIn      int64         `json:"records_in"`
	RecordsOut     int64         `json:"records_out"`
	RecordsError   int64         `json:"records_error"`
	BytesProcessed int64         `json:"bytes_processed"`
	Duration       time.Duration `json:"duration"`
	RowsPerSec     float64       `json:"rows_per_sec"`
	Status         string        `json:"status"` // "pending", "running", "completed", "failed"
	ErrorMessage   string        `json:"error_message,omitempty"`
}

// PipelineExecutionRun represents a full execution instance of a pipeline
type PipelineExecutionRun struct {
	RunID           uuid.UUID              `json:"run_id"`
	PipelineID      uuid.UUID              `json:"pipeline_id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	Status          string                 `json:"status"` // "queued", "running", "completed", "failed", "simulated"
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	TotalRecordsIn  int64                  `json:"total_records_in"`
	TotalRecordsOut int64                  `json:"total_records_out"`
	TotalErrors     int64                  `json:"total_errors"`
	PeakThroughput  float64                `json:"peak_throughput_rows_sec"`
	StepTelemetry   map[string]StepMetrics `json:"step_telemetry"`
	ErrorDetails    []string               `json:"error_details,omitempty"`
	SampleOutput    []PipelineRecord       `json:"sample_output,omitempty"`
}

// TestStepRequest is sent from the visual canvas to test a single tile with sample data
type TestStepRequest struct {
	NodeType string                 `json:"node_type"`
	SubType  string                 `json:"sub_type"`
	Config   map[string]interface{} `json:"config"`
	Input    []PipelineRecord       `json:"input"`
}

// TestStepResponse returns the transformed sample records and telemetry
type TestStepResponse struct {
	Success      bool             `json:"success"`
	Output       []PipelineRecord `json:"output"`
	Errors       []string         `json:"errors,omitempty"`
	ExecutionMs  int64            `json:"execution_ms"`
	RecordsIn    int              `json:"records_in"`
	RecordsOut   int              `json:"records_out"`
	RecordsError int              `json:"records_error"`
}
