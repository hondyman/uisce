package boresolver

import (
	"strings"
)

type ExprType string

const (
	TypeUnknown ExprType = "unknown"
	TypeNumber  ExprType = "number"
	TypeString  ExprType = "string"
	TypeBool    ExprType = "bool"
	TypeDate    ExprType = "date"
)

type TypeEnv interface {
	TermType(termName string) ExprType
}

// InferType attempts to deduce the result type of an expression
func InferType(e Expr, env TypeEnv) ExprType {
	switch v := e.(type) {
	case *TermRef:
		if env != nil {
			return env.TermType(v.Name)
		}
		return TypeUnknown
	case *NumberLiteral:
		return TypeNumber
	case *StringLiteral:
		return TypeString
	case *BinaryExpr:
		lt := InferType(v.Left, env)
		rt := InferType(v.Right, env)
		switch v.Op {
		case "+", "-", "*", "/":
			if lt == TypeNumber && rt == TypeNumber {
				return TypeNumber
			}
			// Could support date arithmetic if one is date?
			if lt == TypeDate || rt == TypeDate {
				return TypeDate
			}
			return TypeNumber // Fallback
		case "=", "!=", ">", "<", ">=", "<=":
			return TypeBool
		case "AND", "OR":
			return TypeBool
		}
	case *UnaryExpr:
		if v.Op == "-" {
			return TypeNumber
		}
		if v.Op == "NOT" {
			return TypeBool
		}
	case *FunctionCall:
		name := strings.ToLower(v.FunctionName)
		switch name {
		case "sum", "avg", "min", "max", "count":
			return TypeNumber
		case "abs", "round", "ceil", "floor":
			return TypeNumber
		case "coalesce":
			// assume type of first arg
			if len(v.Args) > 0 {
				return InferType(v.Args[0], env)
			}
		case "date_add", "date_sub":
			return TypeDate
		case "date_diff":
			return TypeNumber
		case "cast":
			// Simplistic check of 2nd arg if it's a literal
			if len(v.Args) == 2 {
				if sl, ok := v.Args[1].(*StringLiteral); ok {
					t := strings.ToLower(sl.Value)
					if strings.Contains(t, "int") || strings.Contains(t, "float") || strings.Contains(t, "num") {
						return TypeNumber
					}
					if strings.Contains(t, "bool") {
						return TypeBool
					}
					if strings.Contains(t, "date") || strings.Contains(t, "time") {
						return TypeDate
					}
					return TypeString
				}
			}
			return TypeUnknown
		case "case_when":
			// assume type of first result (arg 1)
			if len(v.Args) >= 2 {
				return InferType(v.Args[1], env)
			}
		}
	}
	return TypeUnknown
}
