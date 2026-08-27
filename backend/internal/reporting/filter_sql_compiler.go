package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type FilterSQLCompiler struct {
	macroResolver *CalendarMacroResolver
}

func NewFilterSQLCompiler(resolver *CalendarMacroResolver) *FilterSQLCompiler {
	return &FilterSQLCompiler{macroResolver: resolver}
}

type CompiledFilterResult struct {
	SQLWhereClause string        `json:"sqlWhereClause"`
	BindArguments  []interface{} `json:"bindArguments"`
}

// CompileFilterAST transforms a nested expression tree into a sanitized SQL predicate with bind variables
func (c *FilterSQLCompiler) CompileFilterAST(
	ctx context.Context,
	rootNode FilterExpressionNode,
	execCtx FilterExecutionContext,
) (*CompiledFilterResult, error) {
	if execCtx.TenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var binds []interface{}
	sql, err := c.compileNode(ctx, rootNode, execCtx, &binds)
	if err != nil {
		return nil, err
	}

	return &CompiledFilterResult{
		SQLWhereClause: sql,
		BindArguments:  binds,
	}, nil
}

func (c *FilterSQLCompiler) compileNode(
	ctx context.Context,
	node FilterExpressionNode,
	execCtx FilterExecutionContext,
	binds *[]interface{},
) (string, error) {
	if (node.Type == NodeComparison || node.Type == NodeGroup) && !node.IsEnabled {
		return "1=1", nil
	}

	switch node.Type {
	case NodeGroup:
		if len(node.Children) == 0 {
			return "1=1", nil
		}
		var childClauses []string
		for _, child := range node.Children {
			if !child.IsEnabled && child.Type != NodeGroup {
				continue
			}
			clause, err := c.compileNode(ctx, child, execCtx, binds)
			if err != nil {
				return "", err
			}
			if clause != "" && clause != "1=1" {
				childClauses = append(childClauses, clause)
			}
		}
		if len(childClauses) == 0 {
			return "1=1", nil
		}
		combinator := string(node.Combinator)
		if combinator == "" {
			combinator = "AND"
		}
		return fmt.Sprintf("(%s)", strings.Join(childClauses, fmt.Sprintf(" %s ", combinator))), nil

	case NodeComparison:
		leftSQL, err := c.compileNode(ctx, *node.LeftExpression, execCtx, binds)
		if err != nil {
			return "", err
		}

		// Handle unary operators (IS NULL / IS NOT NULL)
		switch node.Operator {
		case CmpOpIsNull:
			return fmt.Sprintf("%s IS NULL", leftSQL), nil
		case CmpOpIsNotNull:
			return fmt.Sprintf("%s IS NOT NULL", leftSQL), nil
		}

		// Handle BETWEEN
		if node.Operator == CmpOpBetween {
			if len(node.RightValues) < 2 {
				return "", fmt.Errorf("BETWEEN operator requires 2 values")
			}
			val1SQL, err := c.compileNode(ctx, node.RightValues[0], execCtx, binds)
			if err != nil {
				return "", err
			}
			val2SQL, err := c.compileNode(ctx, node.RightValues[1], execCtx, binds)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s BETWEEN %s AND %s", leftSQL, val1SQL, val2SQL), nil
		}

		// Handle IN / NOT IN
		if node.Operator == CmpOpIn || node.Operator == CmpOpNotIn {
			var valClauses []string
			for _, rv := range node.RightValues {
				vSQL, err := c.compileNode(ctx, rv, execCtx, binds)
				if err != nil {
					return "", err
				}
				valClauses = append(valClauses, vSQL)
			}
			opStr := "IN"
			if node.Operator == CmpOpNotIn {
				opStr = "NOT IN"
			}
			return fmt.Sprintf("%s %s (%s)", leftSQL, opStr, strings.Join(valClauses, ", ")), nil
		}

		// Handle binary operators
		rightSQL, err := c.compileNode(ctx, *node.RightExpression, execCtx, binds)
		if err != nil {
			return "", err
		}

		switch node.Operator {
		case CmpOpEquals:
			return fmt.Sprintf("%s = %s", leftSQL, rightSQL), nil
		case CmpOpNotEquals:
			return fmt.Sprintf("%s != %s", leftSQL, rightSQL), nil
		case CmpOpGreaterThan:
			return fmt.Sprintf("%s > %s", leftSQL, rightSQL), nil
		case CmpOpGreaterThanOrEquals:
			return fmt.Sprintf("%s >= %s", leftSQL, rightSQL), nil
		case CmpOpLessThan:
			return fmt.Sprintf("%s < %s", leftSQL, rightSQL), nil
		case CmpOpLessThanOrEquals:
			return fmt.Sprintf("%s <= %s", leftSQL, rightSQL), nil
		case CmpOpContains:
			return fmt.Sprintf("%s LIKE '%%' || %s || '%%'", leftSQL, rightSQL), nil
		case CmpOpStartsWith:
			return fmt.Sprintf("%s LIKE %s || '%%'", leftSQL, rightSQL), nil
		case CmpOpEndsWith:
			return fmt.Sprintf("%s LIKE '%%' || %s", leftSQL, rightSQL), nil
		case CmpOpMatchesRegex:
			return fmt.Sprintf("REGEXP_LIKE(%s, %s)", leftSQL, rightSQL), nil
		default:
			return fmt.Sprintf("%s = %s", leftSQL, rightSQL), nil
		}

	case NodeField:
		return sanitizeFieldIdentifier(node.FieldKey), nil

	case NodeLiteral:
		*binds = append(*binds, node.LiteralValue)
		return fmt.Sprintf("$%d", len(*binds)), nil

	case NodeParameter:
		// Resolve Parameter from Execution Context
		paramName := strings.TrimPrefix(node.ParamKey, "@")

		// Check Session ABAC Variables first (@Session.TenantID, etc.)
		if strings.HasPrefix(paramName, "Session.") {
			sessionKey := strings.TrimPrefix(paramName, "Session.")
			val, exists := execCtx.SessionVariables[sessionKey]
			if !exists {
				if sessionKey == "TenantID" {
					val = execCtx.TenantID.String()
				} else {
					return "", fmt.Errorf("unresolved session variable: %s", node.ParamKey)
				}
			}
			*binds = append(*binds, val)
			return fmt.Sprintf("$%d", len(*binds)), nil
		}

		// User/Report Parameters
		val, exists := execCtx.Parameters[paramName]
		if !exists {
			return "", fmt.Errorf("missing required filter parameter: %s", node.ParamKey)
		}
		*binds = append(*binds, val)
		return fmt.Sprintf("$%d", len(*binds)), nil

	case NodeFunction:
		// Special handling for Relative Date Macros
		if isDateMacro(node.FunctionName) {
			start, _, err := c.macroResolver.ResolveDateMacro(ctx, execCtx.TenantID, node.FunctionName, execCtx.ExchangeCalendar, execCtx.ExecutionTime, 0)
			if err != nil {
				return "", err
			}
			*binds = append(*binds, start.Format("2006-01-02"))
			return fmt.Sprintf("$%d", len(*binds)), nil
		}

		// General SQL Functions: SUBSTR(f, 1, 3), UPPER(f), COALESCE(a, b)
		var argSQLs []string
		for _, arg := range node.FunctionArgs {
			argSQL, err := c.compileNode(ctx, arg, execCtx, binds)
			if err != nil {
				return "", err
			}
			argSQLs = append(argSQLs, argSQL)
		}
		return fmt.Sprintf("%s(%s)", strings.ToUpper(node.FunctionName), strings.Join(argSQLs, ", ")), nil
	}

	return "1=1", nil
}

func isDateMacro(fnName string) bool {
	switch strings.ToUpper(fnName) {
	case "TODAY", "YESTERDAY", "THIS_WEEK", "MTD", "QTD", "YTD", "PREVIOUS_BUSINESS_DAY", "T-1", "T-2":
		return true
	default:
		return false
	}
}

func sanitizeFieldIdentifier(ident string) string {
	cleaned := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			return r
		}
		return -1
	}, ident)
	if cleaned == "" {
		return "unknown_col"
	}
	return cleaned
}
