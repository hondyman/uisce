package handlers

// retain_unused_timeout_triggers_symbols.go
// Minimal retention shim to reference handler helper methods used in manual
// testing or example flows so staticcheck does not flag them as unused.
func init() {
	var h TimeoutTriggersHandler
	_ = h.getUser
}
