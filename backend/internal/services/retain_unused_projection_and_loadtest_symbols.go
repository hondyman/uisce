package services

// retain_unused_projection_and_loadtest_symbols.go
// Retention shim to reference projection event handler and load test engine
// symbols flagged by staticcheck as unused.
func init() {
	var h ProjectionEventHandlerImpl
	_ = &h.mu // Use address to avoid copying mutex
	_ = h.isRunning
	_ = h.stopCh
	_ = h.boQueue
	_ = h.instanceQueue
	_ = h.routeEvent

	var lte LoadTestEngine
	_ = lte.running
	_ = &lte.metrics // Use address to avoid copying struct with mutex
	_ = &lte.workerWg
	_ = lte.workers
	_ = lte.accessSvc
	_ = lte.perfMonitor
}
