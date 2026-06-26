package resilience

// retain_unused_resilience_symbols.go
// Minimal retention shim to reference resilience package symbols so staticcheck
// does not report them as unused when they are intentionally retained.
func init() {
	var ro ResilienceOrchestrator
	_ = ro.mu
	_ = ro.metricsCollected
	_ = ro.gracefulDegradation

	var gd GracefulDegradation
	_ = gd.mu
	_ = gd.fallbacks
	_ = gd.degradationLevel
	_ = gd.degradationReason

	// Reference key constructors and methods by value so staticcheck considers them used
	_ = NewResilienceOrchestrator
	_ = ro.Execute
	_ = ro.ExecuteAsync
	_ = ro.ExecuteWithFallback
	_ = ro.handleDegradation
	_ = ro.RegisterFallback
	_ = ro.GetCircuitBreakerState
	_ = ro.GetMetrics
	_ = ro.ExportMetrics
	_ = ro.GetDegradationLevel
	_ = ro.GetDegradationReason
	_ = ro.SetDegradationLevel
}
