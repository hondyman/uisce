package services

// ============================================================================
// SHARED HELPER FUNCTIONS FOR COMMAND HANDLERS
// ============================================================================
//
// These helpers were previously defined in bo_command_handler.go which
// was extracted to internal/bo. They remain in services/ for now to
// support the legacy instance_command_handler.go and other consumers
// that haven't migrated to internal/bo yet.
//
// Future migration: instance_command_handler.go should also be extracted
// to internal/bo, and these helpers will be deleted alongside it.

// getStringField extracts a string from a generic command data map.
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getMapField extracts a nested map from a generic command data map.
// Returns an empty map if the key is missing or not a map - never nil.
func getMapField(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key]; ok {
		if mapVal, ok := val.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return make(map[string]interface{})
}