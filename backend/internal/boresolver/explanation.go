package boresolver

type ExplanationNode struct {
	Kind     string            `json:"kind"`                // "binary", "unary", "function", "term", "literal"
	Op       string            `json:"op,omitempty"`        // for binary/unary
	Name     string            `json:"name,omitempty"`      // for term/function
	Value    string            `json:"value,omitempty"`     // for literal
	DataType ExprType          `json:"data_type,omitempty"` // inferred type
	Left     *ExplanationNode  `json:"left,omitempty"`
	Right    *ExplanationNode  `json:"right,omitempty"`
	Expr     *ExplanationNode  `json:"expr,omitempty"`     // unary
	Args     []ExplanationNode `json:"args,omitempty"`     // function
	Physical string            `json:"physical,omitempty"` // physical mapping for term
}

// ExplainExpression builds a structured explanation of the expression AST
func ExplainExpression(expr Expr, env TypeEnv) ExplanationNode {
	switch e := expr.(type) {
	case *TermRef:
		// Attempt to resolve physical mapping via env if possible?
		// env is TypeEnv, which just gives Type.
		// If we want physical mapping, we need Resolver context.
		// But ExplainExpression signature only takes TypeEnv.
		// We can add "Physical" field if we have access to it,
		// but standard TypeEnv doesn't provide it.
		// For now, we'll leave Physical empty or require a richer Env interface if needed.
		// The user example showed "physical": "trades.execution_price".
		// We might need to extend TypeEnv or pass Resolver.

		return ExplanationNode{
			Kind:     "term",
			Name:     e.Name,
			DataType: InferType(e, env),
		}

	case *NumberLiteral:
		return ExplanationNode{
			Kind:     "literal",
			Value:    e.Value,
			DataType: TypeNumber,
		}

	case *StringLiteral:
		return ExplanationNode{
			Kind:     "literal",
			Value:    e.Value,
			DataType: TypeString,
		}

	case *BinaryExpr:
		left := ExplainExpression(e.Left, env)
		right := ExplainExpression(e.Right, env)
		return ExplanationNode{
			Kind:     "binary",
			Op:       e.Op,
			DataType: InferType(e, env),
			Left:     &left,
			Right:    &right,
		}

	case *UnaryExpr:
		sub := ExplainExpression(e.Expr, env)
		return ExplanationNode{
			Kind:     "unary",
			Op:       e.Op,
			DataType: InferType(e, env),
			Expr:     &sub,
		}

	case *FunctionCall:
		args := make([]ExplanationNode, len(e.Args))
		for i, arg := range e.Args {
			args[i] = ExplainExpression(arg, env)
		}
		return ExplanationNode{
			Kind:     "function",
			Name:     e.FunctionName,
			DataType: InferType(e, env),
			Args:     args,
		}
	}

	return ExplanationNode{Kind: "unknown"}
}
