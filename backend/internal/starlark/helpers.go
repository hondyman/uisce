package starlark

// ============================================================================
// SHARED HELPER FUNCTIONS
// ============================================================================
//
// These helpers were previously defined in services/business_process_service.go
// and are used by starlark_engine.go. They were migrated here to avoid pulling
// in the entire business_process_service.go into internal/starlark.

// getString extracts a string from a generic command data map.
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getBool extracts a bool from a generic command data map.
func getBool(data map[string]interface{}, key string) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return false
}

// getInt extracts an int from a generic command data map.
// JSON unmarshaling produces float64 for numbers, so we handle both.
func getInt(data map[string]interface{}, key string) int {
	if v, ok := data[key].(float64); ok {
		return int(v)
	}
	if v, ok := data[key].(int); ok {
		return v
	}
	return 0
}