package boresolver

import (
	"fmt"
)

// ToSQL converts an AST expression into a SQL string and required joins
func (r *Resolver) ToSQL(expr Expr) (string, []JoinStep, error) {
	switch e := expr.(type) {

	case *NumberLiteral:
		return e.Value, nil, nil

	case *StringLiteral:
		return r.Dialect.QuoteLiteral(e.Value), nil, nil

	case *TermRef:
		mapping, joins, err := r.ResolveTerm(e.Name)
		if err != nil {
			return "", nil, err
		}
		// Assuming term resolution is just table.column
		// If term maps to an expression, that would be recursively resolved, but
		// "PhysicalMapping" struct implies simpleness.
		sql := fmt.Sprintf("%s.%s",
			r.Dialect.QuoteIdent(mapping.Table),
			r.Dialect.QuoteIdent(mapping.Column),
		)
		return sql, joins, nil

	case *UnaryExpr:
		rightSQL, joins, err := r.ToSQL(e.Expr)
		if err != nil {
			return "", nil, err
		}

		switch e.Op {
		case "-":
			return fmt.Sprintf("(-%s)", rightSQL), joins, nil
		case "NOT":
			return fmt.Sprintf("(NOT %s)", rightSQL), joins, nil
		default:
			return "", nil, fmt.Errorf("unknown unary operator: %s", e.Op)
		}

	case *BinaryExpr:
		leftSQL, leftJoins, err := r.ToSQL(e.Left)
		if err != nil {
			return "", nil, err
		}
		rightSQL, rightJoins, err := r.ToSQL(e.Right)
		if err != nil {
			return "", nil, err
		}

		// Handle safe division
		if e.Op == "/" {
			return r.Dialect.SafeDiv(leftSQL, rightSQL), append(leftJoins, rightJoins...), nil
		}

		opStr := e.Op
		switch e.Op {
		case "+":
			opStr = r.Dialect.OpAdd()
		case "-":
			opStr = r.Dialect.OpSub()
		case "*":
			opStr = r.Dialect.OpMul()
		case "/":
			opStr = r.Dialect.OpDiv()
		case "=":
			opStr = "="
		case "!=", "<>":
			opStr = "<>"
		case ">":
			opStr = ">"
		case "<":
			opStr = "<"
		case ">=":
			opStr = ">="
		case "<=":
			opStr = "<="
		case "AND":
			opStr = "AND"
		case "OR":
			opStr = "OR"
		}

		sql := fmt.Sprintf("(%s %s %s)", leftSQL, opStr, rightSQL)
		return sql, append(leftJoins, rightJoins...), nil

	case *FunctionCall:
		argSQLs := []string{}
		allJoins := []JoinStep{}

		for _, arg := range e.Args {
			s, j, err := r.ToSQL(arg)
			if err != nil {
				return "", nil, err
			}
			argSQLs = append(argSQLs, s)
			allJoins = append(allJoins, j...)
		}

		// Delegate to Dialect
		// Some arguments for functions like cast might need unquoting if passed as StringLiteral.
		// Detailed handling inside Dialect.Func is better.
		sql := r.Dialect.Func(e.FunctionName, argSQLs...)
		return sql, allJoins, nil
	}

	return "", nil, fmt.Errorf("unknown expression type: %T", expr)
}
