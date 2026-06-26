package boresolver

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Message string
}

// IsAggregateExpr checks if the expression contains any aggregation functions
func IsAggregateExpr(e Expr) bool {
	switch v := e.(type) {
	case *FunctionCall:
		name := strings.ToLower(v.FunctionName)
		if name == "sum" || name == "avg" || name == "min" || name == "max" || name == "count" {
			return true
		}
		for _, a := range v.Args {
			if IsAggregateExpr(a) {
				return true
			}
		}
		return false
	case *BinaryExpr:
		return IsAggregateExpr(v.Left) || IsAggregateExpr(v.Right)
	case *UnaryExpr:
		return IsAggregateExpr(v.Expr)
	default:
		return false
	}
}

// ValidateExpression performs semantic checks on the expression
func ValidateExpression(expr Expr, env TypeEnv) []ValidationError {
	var errs []ValidationError

	var walk func(e Expr)
	walk = func(e Expr) {
		switch v := e.(type) {
		case *BinaryExpr:
			if v.Op == "AND" || v.Op == "OR" {
				lt := InferType(v.Left, env)
				rt := InferType(v.Right, env)
				if (lt != TypeBool && lt != TypeUnknown) || (rt != TypeBool && rt != TypeUnknown) {
					errs = append(errs, ValidationError{
						Message: fmt.Sprintf("Operator '%s' requires boolean operands, got %s and %s", v.Op, lt, rt),
					})
				}
			}
			// Could add numeric checks for math ops
			if v.Op == "+" || v.Op == "-" || v.Op == "*" || v.Op == "/" {
				lt := InferType(v.Left, env)
				rt := InferType(v.Right, env)
				// Allow date arithmetic loosely, but prevent String math if possible
				if lt == TypeString || rt == TypeString {
					// unless it's string concat? '+' might be concat in some dialects?
					// But we strictly defined OpAdd as math usually.
					// Let's warn only if both are definitely not numbers/dates?
					// Strict check for now:
					if lt == TypeString {
						errs = append(errs, ValidationError{
							Message: fmt.Sprintf("Mathematical operator '%s' cannot be applied to String type", v.Op),
						})
					}
				}
			}

			walk(v.Left)
			walk(v.Right)

		case *UnaryExpr:
			if v.Op == "NOT" {
				t := InferType(v.Expr, env)
				if t != TypeBool && t != TypeUnknown {
					errs = append(errs, ValidationError{
						Message: fmt.Sprintf("NOT requires boolean operand, got %s", t),
					})
				}
			}
			walk(v.Expr)

		case *FunctionCall:
			for _, a := range v.Args {
				walk(a)
			}
			// TODO add arg count checks per function if desired
		}
	}

	walk(expr)
	return errs
}
