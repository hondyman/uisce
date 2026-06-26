package api

// This small helper intentionally references a few symbols that are
// currently not wired into the live router but are kept as reference
// implementations for future use. Assigning them to the blank identifier
// prevents staticcheck from reporting U1000 while keeping the code in
// place for later re-integration.
func _retainUnusedAPIHelpers() {
	// Route registration helpers
	_ = (*Server).registerFabricRoutes
	_ = (*Server).registerViewsRoutes

	// WebSocketHub method value (non-exported duplicate kept for historical reasons)
	_ = (*WebSocketHub).broadcastToUser
}

func init() {
	// Ensure the helper is referenced so staticcheck won't report it as unused.
	_retainUnusedAPIHelpers()
}
