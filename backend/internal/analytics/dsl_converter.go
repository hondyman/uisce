package analytics

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// DslConverter converts DAX/Excel formulas to internal DSL
type DslConverter struct{}

// NewDslConverter creates a new DSL converter
func NewDslConverter() *DslConverter {
	return &DslConverter{}
}

// ConversionResult contains the converted DSL and any errors
type ConversionResult struct {
	DSL           string   `json:"dsl"`
	Success       bool     `json:"success"`
	Error         string   `json:"error,omitempty"`
	FunctionsUsed []string `json:"functions_used"`
}

// Convert transforms a DAX/Excel formula into internal DSL
func (c *DslConverter) Convert(formula string, formulaType string) ConversionResult {
	result := ConversionResult{
		Success:       true,
		FunctionsUsed: []string{},
	}

	if formula == "" {
		result.Success = false
		result.Error = "empty formula"
		return result
	}

	// Step 1: Normalize
	normalized := c.normalize(formula)

	// Step 2: Extract identifiers (convert {xxx} → xxx)
	normalized = c.extractIdentifiers(normalized)

	// Step 3: Convert to DSL
	dsl, functions, err := c.convertToDSL(normalized)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.DSL = dsl
	result.FunctionsUsed = functions
	return result
}

// normalize cleans up the formula
func (c *DslConverter) normalize(formula string) string {
	// Strip leading = (Excel)
	formula = strings.TrimPrefix(formula, "=")

	// Normalize whitespace
	formula = strings.TrimSpace(formula)

	// Remove newlines
	formula = strings.ReplaceAll(formula, "\n", " ")
	formula = strings.ReplaceAll(formula, "\r", "")

	return formula
}

// extractIdentifiers converts {identifier} → identifier
func (c *DslConverter) extractIdentifiers(formula string) string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	return re.ReplaceAllString(formula, "$1")
}

// convertToDSL converts the formula to LISP-like DSL
func (c *DslConverter) convertToDSL(formula string) (string, []string, error) {
	var functions []string

	// Convert known DAX functions
	daxFunctions := map[string]string{
		"DIVIDE":    "divide",
		"SUMX":      "sumx",
		"MINX":      "minx",
		"MAXX":      "maxx",
		"AVERAGEX":  "averagex",
		"COUNTX":    "countx",
		"FILTER":    "filter",
		"ROUND":     "round",
		"ABS":       "abs",
		"POWER":     "power",
		"SQRT":      "sqrt",
		"LN":        "ln",
		"LOG":       "log",
		"EXP":       "exp",
		"IF":        "if",
		"SWITCH":    "switch",
		"CALCULATE": "calculate",
		"RELATED":   "related",
		"SUM":       "sum",
		"AVERAGE":   "average",
		"MIN":       "min",
		"MAX":       "max",
		"COUNT":     "count",
		"COUNTA":    "counta",
		"BLANK":     "blank",
		"ISBLANK":   "isblank",
	}

	// Convert known Excel functions
	excelFunctions := map[string]string{
		"YIELD":      "yield",
		"PRICE":      "price",
		"DURATION":   "duration",
		"MDURATION":  "mduration",
		"ACCRINT":    "accrint",
		"ACCRINTM":   "accrintm",
		"DISC":       "disc",
		"INTRATE":    "intrate",
		"PRICEDISC":  "pricedisc",
		"PRICEMAT":   "pricemat",
		"YIELDDISC":  "yielddisc",
		"YIELDMAT":   "yieldmat",
		"TBILLPRICE": "tbillprice",
		"TBILLYIELD": "tbillyield",
		"RATE":       "rate",
		"PV":         "pv",
		"FV":         "fv",
		"NPV":        "npv",
		"IRR":        "irr",
		"XIRR":       "xirr",
		"XNPV":       "xnpv",
	}

	result := formula

	// Track functions used
	for dax, dsl := range daxFunctions {
		if strings.Contains(strings.ToUpper(result), dax+"(") {
			functions = append(functions, dax)
			result = c.convertFunctionCall(result, dax, dsl)
		}
	}

	for excel, dsl := range excelFunctions {
		if strings.Contains(strings.ToUpper(result), excel+"(") {
			functions = append(functions, excel)
			result = c.convertFunctionCall(result, excel, dsl)
		}
	}

	// Convert arithmetic operators
	result = c.convertArithmetic(result)

	return result, functions, nil
}

// convertFunctionCall converts FUNC(a, b, c) → (func a b c)
func (c *DslConverter) convertFunctionCall(formula string, funcName string, dslName string) string {
	// Simple regex-based conversion for common patterns
	// This is a simplification - a full parser would be more robust
	pattern := regexp.MustCompile(`(?i)` + funcName + `\s*\(([^)]+)\)`)

	return pattern.ReplaceAllStringFunc(formula, func(match string) string {
		// Extract arguments
		start := strings.Index(match, "(")
		end := strings.LastIndex(match, ")")
		if start == -1 || end == -1 {
			return match
		}

		argsStr := match[start+1 : end]
		args := c.parseArguments(argsStr)

		// Build DSL
		return fmt.Sprintf("(%s %s)", dslName, strings.Join(args, " "))
	})
}

// parseArguments splits comma-separated arguments
func (c *DslConverter) parseArguments(argsStr string) []string {
	var args []string
	var current strings.Builder
	depth := 0

	for _, ch := range argsStr {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				arg := strings.TrimSpace(current.String())
				if arg != "" {
					args = append(args, arg)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	// Don't forget the last argument
	if arg := strings.TrimSpace(current.String()); arg != "" {
		args = append(args, arg)
	}

	return args
}

// convertArithmetic converts a + b → (add a b), etc.
func (c *DslConverter) convertArithmetic(formula string) string {
	// This is a simplified version - a real implementation would use a proper parser
	// For now, we'll handle simple cases

	// Already in DSL format (starts with parenthesis)
	if strings.HasPrefix(formula, "(") {
		return formula
	}

	// Check for simple arithmetic patterns and convert
	// a + b → (add a b)
	// a - b → (subtract a b)
	// a * b → (multiply a b)
	// a / b → (divide a b)

	// Try to identify and convert simple binary operations
	operators := []struct {
		op  string
		dsl string
	}{
		{" + ", "add"},
		{" - ", "subtract"},
		{" * ", "multiply"},
		{" / ", "divide"},
	}

	for _, op := range operators {
		if strings.Contains(formula, op.op) {
			parts := strings.SplitN(formula, op.op, 2)
			if len(parts) == 2 {
				left := strings.TrimSpace(parts[0])
				right := strings.TrimSpace(parts[1])
				return fmt.Sprintf("(%s %s %s)", op.dsl, left, right)
			}
		}
	}

	return formula
}

// DependencyExtractor extracts dependencies from arguments JSON
type DependencyExtractor struct{}

// NewDependencyExtractor creates a new dependency extractor
func NewDependencyExtractor() *DependencyExtractor {
	return &DependencyExtractor{}
}

// Dependency represents a calculation dependency
type Dependency struct {
	Type string `json:"type"` // "term", "calc", "table"
	Ref  string `json:"ref"`  // The referenced entity name
}

// ExtractDependencies extracts dependencies from arguments JSONB
func (e *DependencyExtractor) ExtractDependencies(argumentsJSON json.RawMessage) ([]Dependency, error) {
	var dependencies []Dependency

	if len(argumentsJSON) == 0 {
		return dependencies, nil
	}

	var arguments map[string]string
	if err := json.Unmarshal(argumentsJSON, &arguments); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	for _, value := range arguments {
		dep := e.parseDependencyValue(value)
		if dep != nil {
			dependencies = append(dependencies, *dep)
		}
	}

	return dependencies, nil
}

// parseDependencyValue parses a single dependency value like "field:xxx" or "calc:xxx"
func (e *DependencyExtractor) parseDependencyValue(value string) *Dependency {
	prefixes := map[string]string{
		"field:": "term",
		"calc:":  "calc",
		"table:": "table",
	}

	for prefix, depType := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return &Dependency{
				Type: depType,
				Ref:  strings.TrimPrefix(value, prefix),
			}
		}
	}

	return nil
}
