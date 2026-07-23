package business_process

import (
	"fmt"
)

// CompileProcessTemplate validates a process template and prepares it for execution.
func CompileProcessTemplate(template ProcessTemplate) error {
	if template.ProcessID == "" {
		return fmt.Errorf("process id is required")
	}
	if len(template.Steps) == 0 {
		return fmt.Errorf("process must have at least one step")
	}

	stepMap := make(map[string]bool)
	for _, step := range template.Steps {
		if step.ID == "" {
			return fmt.Errorf("step id is required")
		}
		if stepMap[step.ID] {
			return fmt.Errorf("duplicate step id: %s", step.ID)
		}
		stepMap[step.ID] = true
	}

	// Validate transitions
	for _, transition := range template.Transitions {
		if !stepMap[transition.From] {
			return fmt.Errorf("transition from unknown step: %s", transition.From)
		}
		if !stepMap[transition.To] {
			return fmt.Errorf("transition to unknown step: %s", transition.To)
		}
	}

	// TODO: Check for cycles? Check for reachability?
	// For now, basic connectivity check is sufficient.

	return nil
}

// EvaluateConditions checks if a set of conditions are met for a given business object.
func EvaluateConditions(obj BusinessObject, conditions []string) bool {
	if len(conditions) == 0 {
		return true
	}

	data := obj.GetData()
	for _, cond := range conditions {
		switch cond {
		case "schema_valid":
			if data["valid"] == false {
				return false
			}
		case "pricing_freshness_ok":
			if data["pricing_fresh"] == false {
				return false
			}
		case "maker_checker_passed":
			if data["approved"] == false {
				return false
			}
		default:
			// Unknown condition, fail safe
			return false
		}
	}
	return true
}

// GetNextStep finds the next step ID given the current step ID.
// In a real implementation, this might handle conditional branching.
func GetNextStep(template ProcessTemplate, currentStepID string) (string, error) {
	for _, t := range template.Transitions {
		if t.From == currentStepID {
			return t.To, nil
		}
	}
	return "", nil // End of process
}
