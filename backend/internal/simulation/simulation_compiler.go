package simulation

import (
	"fmt"
	"strings"
)

type ShockRule struct {
	Field    string  `json:"field"`
	Operator string  `json:"operator"` // MULTIPLY, ADD, OVERRIDE
	Value    float64 `json:"value"`
}

// ScenarioDefinition defines a What-If scenario with AST shock rules
type ScenarioDefinition struct {
	ScenarioID  string      `json:"scenario_id" db:"scenario_id"`
	TenantID    string      `json:"tenant_id" db:"tenant_id"`
	Name        string      `json:"scenario_name" db:"scenario_name"`
	Description string      `json:"description" db:"description"`
	TargetBOID  string      `json:"target_bo_id" db:"target_bo_id"`
	Rules       []ShockRule `json:"shock_rules"`
	IsGlobal    bool        `json:"is_global" db:"is_global"`
	CreatedBy   string      `json:"created_by" db:"created_by"`
}

// ApplySimulationTransform mutates field projections in the AST to reflect simulation shocks
func ApplySimulationTransform(fieldProjections []string, scenario *ScenarioDefinition) []string {
	if scenario == nil || len(scenario.Rules) == 0 {
		return fieldProjections
	}

	var transformed []string
	for _, proj := range fieldProjections {
		applied := proj
		for _, rule := range scenario.Rules {
			// Check if projection matches the target field of the shock rule
			if strings.Contains(strings.ToLower(proj), strings.ToLower(rule.Field)) {
				switch rule.Operator {
				case "MULTIPLY":
					applied = fmt.Sprintf("(%s * %f) AS %s_simulated", proj, rule.Value, rule.Field)
				case "ADD":
					applied = fmt.Sprintf("(%s + %f) AS %s_simulated", proj, rule.Value, rule.Field)
				case "OVERRIDE":
					applied = fmt.Sprintf("%f AS %s_simulated", rule.Value, rule.Field)
				}
			}
		}
		transformed = append(transformed, applied)
	}

	return transformed
}
