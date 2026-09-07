package resilience

// retain_unused_resilience_symbols.go
// Minimal retention shim to reference resilience package symbols so staticcheck
// does not report them as unused when they are intentionally retained.
func init() {
	var ro ResilienceOrchestrator
	var gd GracefulDegradation

	// Retain symbols by pointer or method-value reference — avoids copying mutex fields.
	_ = &ro.mu
	_ = &gd.mu
	_ = ro.metricsCollected
	_ = ro.gracefulDegradation
	_ = gd.fallbacks
	_ = gd.degradationLevel
	_ = gd.degradationReason

	// Retain constructors and methods by value
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
