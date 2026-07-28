package boresolver

import (
	"fmt"
	"regexp"
	"strings"
)

var formulaRegex = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\}`)

// CompileDeepCalculations generates the nested CTE plan
// baseQuery is the pre-compiled String containing the Hot/Cold joins and Rule 7 Tenant Security
func (g *BOSQLGenerator) CompileDeepCalculations(layers [][]*CalcNode, baseQuery string, requestedFields []string) (string, error) {
	if len(layers) == 0 {
		return baseQuery, nil
	}

	var sb strings.Builder

	// Layer 0: Base Data Extraction (Security & Isolation boundary)
	sb.WriteString("WITH layer_0 AS (\n  ")
	sb.WriteString(baseQuery)
	sb.WriteString("\n)")

	// Layers 1..N: Derived Math Execution
	for depth := 1; depth < len(layers); depth++ {
		sb.WriteString(fmt.Sprintf(",\nlayer_%d AS (\n", depth))
		sb.WriteString("  SELECT *") // Carry forward lower-level variables

		for _, calc := range layers[depth] {
			// Convert "${gross_return} - ${management_fee}" to "gross_return - management_fee AS net_fund_yield"
			safeFormula := formulaRegex.ReplaceAllString(calc.Formula, "$1")
			sb.WriteString(fmt.Sprintf(",\n    (%s) AS %s", safeFormula, calc.TermKey))
		}

		sb.WriteString(fmt.Sprintf("\n  FROM layer_%d\n)", depth-1))
	}

	// Final Select Projection
	finalLayerIdx := len(layers) - 1
	if len(requestedFields) == 0 {
		sb.WriteString(fmt.Sprintf("\nSELECT * FROM layer_%d;", finalLayerIdx))
	} else {
		sb.WriteString(fmt.Sprintf("\nSELECT\n  %s\nFROM layer_%d;", strings.Join(requestedFields, ",\n  "), finalLayerIdx))
	}

	return sb.String(), nil
}
