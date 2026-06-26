package domain

import (
	"context"
	"fmt"
	"time"
)

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string)
	RecordHistogram(name string, value float64, labels map[string]string)
	RecordGauge(name string, value float64, labels map[string]string)
}

// TelemetryService provides observability for the governance system
type TelemetryService struct {
	Metrics MetricsCollector
}

// DecisionMetrics tracks access control decision metrics
type DecisionMetrics struct {
	Service *TelemetryService
}

func (dm *DecisionMetrics) RecordEvaluation(ctx context.Context, req EvaluationRequest, allow bool, duration time.Duration) {
	labels := map[string]string{
		"tenant_id": req.TenantID,
		"action":    string(req.Action),
		"result":    map[bool]string{true: "allow", false: "deny"}[allow],
	}

	dm.Service.Metrics.IncrementCounter("governance_evaluations_total", labels)
	dm.Service.Metrics.RecordHistogram("governance_evaluation_duration_seconds", duration.Seconds(), labels)
}

func (dm *DecisionMetrics) RecordPolicyCheck(ctx context.Context, req EvaluationRequest, allow bool, duration time.Duration, policyCount int) {
	labels := map[string]string{
		"tenant_id":    req.TenantID,
		"action":       string(req.Action),
		"result":       map[bool]string{true: "allow", false: "deny"}[allow],
		"policy_count": fmt.Sprintf("%d", policyCount),
	}

	dm.Service.Metrics.IncrementCounter("governance_policy_checks_total", labels)
	dm.Service.Metrics.RecordHistogram("governance_policy_check_duration_seconds", duration.Seconds(), labels)
}

func (dm *DecisionMetrics) RecordCacheHit(ctx context.Context, cacheType string) {
	labels := map[string]string{
		"cache_type": cacheType,
	}
	dm.Service.Metrics.IncrementCounter("governance_cache_hits_total", labels)
}

func (dm *DecisionMetrics) RecordCacheMiss(ctx context.Context, cacheType string) {
	labels := map[string]string{
		"cache_type": cacheType,
	}
	dm.Service.Metrics.IncrementCounter("governance_cache_misses_total", labels)
}

// InstrumentedEvaluator wraps an evaluator with metrics collection
type InstrumentedEvaluator struct {
	Evaluator Evaluator
	Metrics   *DecisionMetrics
}

func (ie *InstrumentedEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (bool, string, []EffectiveClaim, error) {
	start := time.Now()
	allow, reason, claims, err := ie.Evaluator.Evaluate(ctx, req)
	duration := time.Since(start)

	ie.Metrics.RecordEvaluation(ctx, req, allow, duration)

	return allow, reason, claims, err
}

// InstrumentedPolicyChecker wraps a policy checker with metrics
type InstrumentedPolicyChecker struct {
	PolicyChecker PolicyChecker
	Metrics       *DecisionMetrics
}

func (ipc *InstrumentedPolicyChecker) Check(ctx context.Context, req EvaluationRequest, claims []EffectiveClaim) (bool, string, []map[string]any, []string, error) {
	start := time.Now()
	allow, reason, matched, scopes, err := ipc.PolicyChecker.Check(ctx, req, claims)
	duration := time.Since(start)

	ipc.Metrics.RecordPolicyCheck(ctx, req, allow, duration, len(matched))

	return allow, reason, matched, scopes, err
}
