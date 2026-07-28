package domain

import (
	"context"
	"database/sql"
)

// ExecutionRequest is the output from BOSQLGenerator passed into the Executor
type ExecutionRequest struct {
	BOID          string        `json:"boId"`
	GeneratedSQL  string        `json:"generatedSql"`
	Args          []interface{} `json:"args,omitempty"`
	TargetEngine  string        `json:"targetEngine"` // e.g., "POSTGRES_HOT", "STARROCKS_FEDERATED", "TRINO_COLD"
	EffectiveTime string        `json:"effectiveTime,omitempty"`
}

// QueryExecutor defines the boundary for physical execution
type QueryExecutor interface {
	Execute(ctx context.Context, req ExecutionRequest) (*sql.Rows, error)
	Explain(ctx context.Context, req ExecutionRequest) (string, error)
}

// EventPublisher defines the non-blocking event streaming abstraction (e.g. Redpanda / Kafka producer)
type EventPublisher interface {
	Publish(topic string, key string, payload []byte) error
}
