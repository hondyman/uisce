package services

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// DAXFunction represents a DAX function definition
type DAXFunction struct {
	Name        string
	Category    string
	Description string
	MinArgs     int
	MaxArgs     int
	ReturnType  string
	Handler     DAXFunctionHandler
}

// DAXFunctionHandler is the function signature for DAX function implementations
type DAXFunctionHandler func(args []interface{}, context *DAXContext) (interface{}, error)

// DAXContext holds execution context for DAX functions
type DAXContext struct {
	DB         *sqlx.DB
	TableName  string
	ColumnName string
	Filters    map[string]interface{}
	DateColumn string
}

// DAXEngine manages DAX function execution
type DAXEngine struct {
	functions map[string]*DAXFunction
}

// NewDAXEngine creates a new DAX engine with all supported functions
func NewDAXEngine() *DAXEngine {
	engine := &DAXEngine{
		functions: make(map[string]*DAXFunction),
	}
	engine.registerFunctions()
	return engine
}

// registerFunctions registers all supported DAX functions
func (e *DAXEngine) registerFunctions() {
	// Time Intelligence Functions
	e.functions["DATESYTD"] = &DAXFunction{
		Name:        "DATESYTD",
		Category:    "time_intelligence",
		Description: "Returns dates for year-to-date period",
		MinArgs:     1,
		MaxArgs:     2,
		ReturnType:  "table",
		Handler:     e.handleDatesYTD,
	}

	e.functions["TOTALYTD"] = &DAXFunction{
		Name:        "TOTALYTD",
		Category:    "time_intelligence",
		Description: "Evaluates year-to-date value of expression",
		MinArgs:     2,
		MaxArgs:     4,
		ReturnType:  "scalar",
		Handler:     e.handleTotalYTD,
	}

	e.functions["DATEADD"] = &DAXFunction{
		Name:        "DATEADD",
		Category:    "time_intelligence",
		Description: "Returns dates shifted by specified intervals",
		MinArgs:     3,
		MaxArgs:     3,
		ReturnType:  "table",
		Handler:     e.handleDateAdd,
	}

	e.functions["SAMEPERIODLASTYEAR"] = &DAXFunction{
		Name:        "SAMEPERIODLASTYEAR",
		Category:    "time_intelligence",
		Description: "Returns dates shifted by one year",
		MinArgs:     1,
		MaxArgs:     1,
		ReturnType:  "table",
		Handler:     e.handleSamePeriodLastYear,
	}

	// Financial Functions
	e.functions["XIRR"] = &DAXFunction{
		Name:        "XIRR",
		Category:    "financial",
		Description: "Returns internal rate of return for cash flows",
		MinArgs:     2,
		MaxArgs:     3,
		ReturnType:  "scalar",
		Handler:     e.handleXIRR,
	}

	e.functions["XNPV"] = &DAXFunction{
		Name:        "XNPV",
		Category:    "financial",
		Description: "Returns net present value for cash flows",
		MinArgs:     3,
		MaxArgs:     3,
		ReturnType:  "scalar",
		Handler:     e.handleXNPV,
	}

	// Statistical Functions
	e.functions["STDEVX.P"] = &DAXFunction{
		Name:        "STDEVX.P",
		Category:    "statistical",
		Description: "Returns standard deviation of expression (population)",
		MinArgs:     2,
		MaxArgs:     2,
		ReturnType:  "scalar",
		Handler:     e.handleStdevXP,
	}

	e.functions["VARX.P"] = &DAXFunction{
		Name:        "VARX.P",
		Category:    "statistical",
		Description: "Returns variance of expression (population)",
		MinArgs:     2,
		MaxArgs:     2,
		ReturnType:  "scalar",
		Handler:     e.handleVarXP,
	}

	e.functions["COVARIANCE.P"] = &DAXFunction{
		Name:        "COVARIANCE.P",
		Category:    "statistical",
		Description: "Returns covariance of two expressions (population)",
		MinArgs:     3,
		MaxArgs:     3,
		ReturnType:  "scalar",
		Handler:     e.handleCovarianceP,
	}

	e.functions["CORR"] = &DAXFunction{
		Name:        "CORR",
		Category:    "statistical",
		Description: "Returns correlation coefficient",
		MinArgs:     3,
		MaxArgs:     3,
		ReturnType:  "scalar",
		Handler:     e.handleCorrelation,
	}

	e.functions["RANKX"] = &DAXFunction{
		Name:        "RANKX",
		Category:    "statistical",
		Description: "Returns ranking of a number in a list",
		MinArgs:     2,
		MaxArgs:     5,
		ReturnType:  "scalar",
		Handler:     e.handleRankX,
	}

	e.functions["PERCENTILEX.INC"] = &DAXFunction{
		Name:        "PERCENTILEX.INC",
		Category:    "statistical",
		Description: "Returns k-th percentile of expression",
		MinArgs:     3,
		MaxArgs:     3,
		ReturnType:  "scalar",
		Handler:     e.handlePercentileXInc,
	}

	// Iterator Functions
	e.functions["SUMX"] = &DAXFunction{
		Name:        "SUMX",
		Category:    "iterator",
		Description: "Returns sum of expression evaluated for each row",
		MinArgs:     2,
		MaxArgs:     2,
		ReturnType:  "scalar",
		Handler:     e.handleSumX,
	}

	e.functions["AVERAGEX"] = &DAXFunction{
		Name:        "AVERAGEX",
		Category:    "iterator",
		Description: "Returns average of expression evaluated for each row",
		MinArgs:     2,
		MaxArgs:     2,
		ReturnType:  "scalar",
		Handler:     e.handleAverageX,
	}

	e.functions["FILTER"] = &DAXFunction{
		Name:        "FILTER",
		Category:    "iterator",
		Description: "Returns table with rows meeting condition",
		MinArgs:     2,
		MaxArgs:     2,
		ReturnType:  "table",
		Handler:     e.handleFilter,
	}

	e.functions["TOPN"] = &DAXFunction{
		Name:        "TOPN",
		Category:    "iterator",
		Description: "Returns top N rows based on expression",
		MinArgs:     3,
		MaxArgs:     4,
		ReturnType:  "table",
		Handler:     e.handleTopN,
	}

	// Additional Iterator Functions
	e.functions["MINX"] = &DAXFunction{
		Name:        "MINX",
		Category:    "iterator",
		Description: "Returns minimum of expression evaluated for each row",
		MinArgs:     2,
		MaxArgs:     2,
		ReturnType:  "scalar",
		Handler:     e.handleMinX,
	}

	e.functions["MAXX"] = &DAXFunction{
		Name:        "MAXX",
		Category:    "iterator",
		Description: "Returns maximum of expression evaluated for each row",
		MinArgs:     2,
		MaxArgs:     2,
		ReturnType:  "scalar",
		Handler:     e.handleMaxX,
	}

	// Logical Functions
	e.functions["DIVIDE"] = &DAXFunction{
		Name:        "DIVIDE",
		Category:    "logical",
		Description: "Performs division and returns alternate result or BLANK() on division by 0",
		MinArgs:     2,
		MaxArgs:     3,
		ReturnType:  "scalar",
		Handler:     e.handleDivide,
	}
	e.functions["SWITCH"] = &DAXFunction{
		Name:        "SWITCH",
		Category:    "logical",
		Description: "Evaluates expression and returns different values",
		MinArgs:     3,
		MaxArgs:     -1, // Variable arguments
		ReturnType:  "scalar",
		Handler:     e.handleSwitch,
	}

	e.functions["IF"] = &DAXFunction{
		Name:        "IF",
		Category:    "logical",
		Description: "Checks condition and returns one value if TRUE, another if FALSE",
		MinArgs:     2,
		MaxArgs:     3,
		ReturnType:  "scalar",
		Handler:     e.handleIf,
	}

	e.functions["BLANK"] = &DAXFunction{
		Name:        "BLANK",
		Category:    "logical",
		Description: "Returns a blank value",
		MinArgs:     0,
		MaxArgs:     0,
		ReturnType:  "scalar",
		Handler:     e.handleBlank,
	}

	e.functions["COALESCE"] = &DAXFunction{
		Name:        "COALESCE",
		Category:    "logical",
		Description: "Returns first non-blank value from list",
		MinArgs:     1,
		MaxArgs:     -1, // Variable arguments
		ReturnType:  "scalar",
		Handler:     e.handleCoalesce,
	}

	// Information Functions
	e.functions["ISFILTERED"] = &DAXFunction{
		Name:        "ISFILTERED",
		Category:    "information",
		Description: "Returns TRUE if column is filtered directly",
		MinArgs:     1,
		MaxArgs:     1,
		ReturnType:  "boolean",
		Handler:     e.handleIsFiltered,
	}

	e.functions["HASONEVALUE"] = &DAXFunction{
		Name:        "HASONEVALUE",
		Category:    "information",
		Description: "Returns TRUE if column has only one distinct value",
		MinArgs:     1,
		MaxArgs:     1,
		ReturnType:  "boolean",
		Handler:     e.handleHasOneValue,
	}

	e.functions["SELECTEDVALUE"] = &DAXFunction{
		Name:        "SELECTEDVALUE",
		Category:    "information",
		Description: "Returns value when context has one distinct value",
		MinArgs:     1,
		MaxArgs:     2,
		ReturnType:  "scalar",
		Handler:     e.handleSelectedValue,
	}

	e.functions["ISINSCOPE"] = &DAXFunction{
		Name:        "ISINSCOPE",
		Category:    "information",
		Description: "Returns TRUE if column is in current scope",
		MinArgs:     1,
		MaxArgs:     1,
		ReturnType:  "boolean",
		Handler:     e.handleIsInScope,
	}
}

// ExecuteFunction executes a DAX function with given arguments and context
func (e *DAXEngine) ExecuteFunction(functionName string, args []interface{}, context *DAXContext) (interface{}, error) {
	function, exists := e.functions[strings.ToUpper(functionName)]
	if !exists {
		return nil, fmt.Errorf("unknown DAX function: %s", functionName)
	}

	// Validate argument count
	if len(args) < function.MinArgs || (function.MaxArgs >= 0 && len(args) > function.MaxArgs) {
		return nil, fmt.Errorf("function %s expects %d-%d arguments, got %d",
			functionName, function.MinArgs, function.MaxArgs, len(args))
	}

	return function.Handler(args, context)
}

// GetFunctionInfo returns information about a DAX function
func (e *DAXEngine) GetFunctionInfo(functionName string) (*DAXFunction, bool) {
	function, exists := e.functions[strings.ToUpper(functionName)]
	return function, exists
}

// ListFunctions returns all available DAX functions
func (e *DAXEngine) ListFunctions() map[string]*DAXFunction {
	return e.functions
}

// ListFunctionsByCategory returns functions grouped by category
func (e *DAXEngine) ListFunctionsByCategory() map[string][]*DAXFunction {
	categories := make(map[string][]*DAXFunction)
	for _, function := range e.functions {
		categories[function.Category] = append(categories[function.Category], function)
	}
	return categories
}

// Function implementations

func (e *DAXEngine) handleDatesYTD(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for DATESYTD function
	// This would generate a table of dates for the year-to-date period
	return nil, fmt.Errorf("DATESYTD not yet implemented")
}

func (e *DAXEngine) handleTotalYTD(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for TOTALYTD function
	return nil, fmt.Errorf("TOTALYTD not yet implemented")
}

func (e *DAXEngine) handleDateAdd(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for DATEADD function
	return nil, fmt.Errorf("DATEADD not yet implemented")
}

func (e *DAXEngine) handleSamePeriodLastYear(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for SAMEPERIODLASTYEAR function
	return nil, fmt.Errorf("SAMEPERIODLASTYEAR not yet implemented")
}

func (e *DAXEngine) handleXIRR(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for XIRR function
	// This would calculate the internal rate of return for irregular cash flows
	return nil, fmt.Errorf("XIRR not yet implemented")
}

func (e *DAXEngine) handleXNPV(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for XNPV function
	return nil, fmt.Errorf("XNPV not yet implemented")
}

func (e *DAXEngine) handleStdevXP(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for STDEVX.P function
	return nil, fmt.Errorf("STDEVX.P not yet implemented")
}

func (e *DAXEngine) handleVarXP(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for VARX.P function
	return nil, fmt.Errorf("VARX.P not yet implemented")
}

func (e *DAXEngine) handleCovarianceP(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for COVARIANCE.P function
	return nil, fmt.Errorf("COVARIANCE.P not yet implemented")
}

func (e *DAXEngine) handleCorrelation(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for CORR function
	return nil, fmt.Errorf("CORR not yet implemented")
}

func (e *DAXEngine) handleRankX(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for RANKX function
	return nil, fmt.Errorf("RANKX not yet implemented")
}

func (e *DAXEngine) handlePercentileXInc(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for PERCENTILEX.INC function
	return nil, fmt.Errorf("PERCENTILEX.INC not yet implemented")
}

func (e *DAXEngine) handleSumX(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for SUMX function
	return nil, fmt.Errorf("SUMX not yet implemented")
}

func (e *DAXEngine) handleAverageX(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for AVERAGEX function
	return nil, fmt.Errorf("AVERAGEX not yet implemented")
}

func (e *DAXEngine) handleFilter(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for FILTER function
	return nil, fmt.Errorf("FILTER not yet implemented")
}

func (e *DAXEngine) handleTopN(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for TOPN function
	return nil, fmt.Errorf("TOPN not yet implemented")
}

func (e *DAXEngine) handleSwitch(args []interface{}, context *DAXContext) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("SWITCH requires at least 3 arguments")
	}

	expression := args[0]

	// Check each value-result pair
	for i := 1; i < len(args)-1; i += 2 {
		if expression == args[i] {
			return args[i+1], nil
		}
	}

	// Check for else clause
	if len(args)%2 == 0 {
		return args[len(args)-1], nil
	}

	return nil, nil // Return blank if no match and no else
}

func (e *DAXEngine) handleIf(args []interface{}, context *DAXContext) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("IF requires at least 2 arguments")
	}

	condition := args[0]
	if condition == nil {
		condition = false
	}

	// Convert condition to boolean
	var conditionBool bool
	switch v := condition.(type) {
	case bool:
		conditionBool = v
	case int, int32, int64, float32, float64:
		conditionBool = v != 0
	case string:
		conditionBool = v != ""
	default:
		conditionBool = true // Non-nil values are truthy
	}

	if conditionBool {
		return args[1], nil
	}

	if len(args) > 2 {
		return args[2], nil
	}

	return nil, nil // Return blank
}

func (e *DAXEngine) handleBlank(args []interface{}, context *DAXContext) (interface{}, error) {
	return nil, nil
}

func (e *DAXEngine) handleCoalesce(args []interface{}, context *DAXContext) (interface{}, error) {
	for _, arg := range args {
		if arg != nil {
			return arg, nil
		}
	}
	return nil, nil
}

func (e *DAXEngine) handleIsFiltered(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for ISFILTERED function
	return false, fmt.Errorf("ISFILTERED not yet implemented")
}

func (e *DAXEngine) handleHasOneValue(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for HASONEVALUE function
	return false, fmt.Errorf("HASONEVALUE not yet implemented")
}

func (e *DAXEngine) handleSelectedValue(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for SELECTEDVALUE function
	return nil, fmt.Errorf("SELECTEDVALUE not yet implemented")
}

func (e *DAXEngine) handleIsInScope(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for ISINSCOPE function
	return false, fmt.Errorf("ISINSCOPE not yet implemented")
}

func (e *DAXEngine) handleMinX(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for MINX function
	return nil, fmt.Errorf("MINX not yet implemented")
}

func (e *DAXEngine) handleMaxX(args []interface{}, context *DAXContext) (interface{}, error) {
	// Implementation for MAXX function
	return nil, fmt.Errorf("MAXX not yet implemented")
}

func (e *DAXEngine) handleDivide(args []interface{}, context *DAXContext) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("DIVIDE requires at least 2 arguments")
	}

	numerator := args[0]
	denominator := args[1]

	// Handle nil values
	if numerator == nil || denominator == nil {
		if len(args) > 2 {
			return args[2], nil // Return alternate result
		}
		return nil, nil // Return blank
	}

	// Convert to float64 for division
	var numFloat, denFloat float64
	switch v := numerator.(type) {
	case float64:
		numFloat = v
	case float32:
		numFloat = float64(v)
	case int:
		numFloat = float64(v)
	case int32:
		numFloat = float64(v)
	case int64:
		numFloat = float64(v)
	default:
		return nil, fmt.Errorf("cannot convert numerator to number")
	}

	switch v := denominator.(type) {
	case float64:
		denFloat = v
	case float32:
		denFloat = float64(v)
	case int:
		denFloat = float64(v)
	case int32:
		denFloat = float64(v)
	case int64:
		denFloat = float64(v)
	default:
		return nil, fmt.Errorf("cannot convert denominator to number")
	}

	// Check for division by zero
	if denFloat == 0 {
		if len(args) > 2 {
			return args[2], nil // Return alternate result
		}
		return nil, nil // Return blank
	}

	return numFloat / denFloat, nil
}
