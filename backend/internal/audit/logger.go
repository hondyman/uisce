package audit

import (
	"context"
	"log"

	"github.com/google/uuid"
)

// Logger is the operational intelligence logging interface
type Logger interface {
	Log(ctx context.Context, event interface{})
}

// WASMExecution is fired after every WASM call
type WASMExecution struct {
	Function    string
	InputSize   int
	OutputSize  int
	ExecutionMs int64
	TenantID    string
}

// ComplianceEvaluation represents a batch evaluation run
type ComplianceEvaluation struct {
	PortfolioID    uuid.UUID
	ValuationDate  string
	RulesEvaluated int
	BreachesFound  int
	ExecutionMs    int64
	Status         string
}

// WASMError represents a WASM trap or crash
type WASMError struct {
	Function string
	Error    string
	RuleID   string
}

// RuleParseError occurs when DSL config is invalid
type RuleParseError struct {
	RuleID string
	Error  string
}

// RiskComputation covers VaR / Vol runs
type RiskComputation struct {
	PortfolioID   uuid.UUID
	ValuationDate string
	Volatility    float64
	VaR95         float64
	VaR99         float64
	ExecutionMs   int64
}

type SchedulerStarted struct {
	Service string
	Cron    string
}

type SchedulerStopped struct {
	Service string
}

type SchedulerError struct {
	Error string
}

type DailyETLCompleted struct {
	TenantsProcessed int
	Errors           int
	DurationMs       int64
}

// StdLogAudit implements Logger writing to stdout
type StdLogAudit struct{}

func (s *StdLogAudit) Log(ctx context.Context, event interface{}) {
	log.Printf("[AUDIT] %+v\n", event)
}

// MockLogger implements Logger for testing
type MockLogger struct {
	Events []interface{}
}

func (m *MockLogger) Log(ctx context.Context, event interface{}) {
	m.Events = append(m.Events, event)
}
