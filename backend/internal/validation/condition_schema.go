package validation

import "fmt"

// ValidateConditionTree checks if a rule condition JSON map adheres to the schema.
func ValidateConditionTree(m map[string]interface{}) error {
	// minimal: check "type", "operator", "field" or "conditions" exist in the right combinations
	t, ok := m["type"].(string)
	if !ok {
		return fmt.Errorf("condition missing type")
	}
	switch t {
	case "condition":
		if _, ok := m["field"].(string); !ok {
			return fmt.Errorf("condition missing field")
		}
		if _, ok := m["operator"].(string); !ok {
			return fmt.Errorf("condition missing operator")
		}
	case "group":
		if _, ok := m["operator"].(string); !ok {
			return fmt.Errorf("group missing operator")
		}
		if _, ok := m["conditions"].([]interface{}); !ok {
			return fmt.Errorf("group missing conditions")
		}
	default:
		return fmt.Errorf("unknown condition type %s", t)
	}
	// TODO: Recursively validate children for groups
	return nil
}
