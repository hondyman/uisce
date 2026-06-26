package dsl

import (
	"fmt"
	"regexp"
	"strings"
)

// Parser parses row filter DSL expressions into an AST.
type Parser struct {
	// Token patterns
	operatorRe *regexp.Regexp
	fieldRe    *regexp.Regexp
	valueRe    *regexp.Regexp
}

// NewParser creates a new DSL parser.
func NewParser() *Parser {
	return &Parser{
		operatorRe: regexp.MustCompile(`(?i)\b(AND|OR|NOT|IN|LIKE|IS NULL|IS NOT NULL)\b`),
		fieldRe:    regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_\.]*`),
		valueRe:    regexp.MustCompile(`'[^']*'|\d+`),
	}
}

// ASTNode represents a node in the abstract syntax tree.
type ASTNode struct {
	Type     string     // "binary_op", "unary_op", "comparison", "field", "literal"
	Operator string     // "AND", "OR", "=", "!=", "IN", etc.
	Left     *ASTNode   // Left operand
	Right    *ASTNode   // Right operand
	Value    string     // For field/literal nodes
	Children []*ASTNode // For IN operator
}

// Parse parses a DSL expression into an AST.
func (p *Parser) Parse(expr string) (*ASTNode, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// Simple recursive descent parser
	// For production, use ANTLR or a proper parser generator

	// Handle OR (lowest precedence)
	if parts := p.splitByOperator(expr, "OR"); len(parts) > 1 {
		left, err := p.Parse(parts[0])
		if err != nil {
			return nil, err
		}
		right, err := p.Parse(strings.Join(parts[1:], " OR "))
		if err != nil {
			return nil, err
		}
		return &ASTNode{
			Type:     "binary_op",
			Operator: "OR",
			Left:     left,
			Right:    right,
		}, nil
	}

	// Handle AND
	if parts := p.splitByOperator(expr, "AND"); len(parts) > 1 {
		left, err := p.Parse(parts[0])
		if err != nil {
			return nil, err
		}
		right, err := p.Parse(strings.Join(parts[1:], " AND "))
		if err != nil {
			return nil, err
		}
		return &ASTNode{
			Type:     "binary_op",
			Operator: "AND",
			Left:     left,
			Right:    right,
		}, nil
	}

	// Handle NOT
	if strings.HasPrefix(strings.ToUpper(expr), "NOT ") {
		inner, err := p.Parse(expr[4:])
		if err != nil {
			return nil, err
		}
		return &ASTNode{
			Type:     "unary_op",
			Operator: "NOT",
			Left:     inner,
		}, nil
	}

	// Handle parentheses
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return p.Parse(expr[1 : len(expr)-1])
	}

	// Handle comparison operators
	for _, op := range []string{">=", "<=", "!=", "=", "LIKE", "IN", "IS NULL", "IS NOT NULL", ">", "<"} {
		if idx := p.indexOfOperator(expr, op); idx != -1 {
			field := strings.TrimSpace(expr[:idx])
			valueExpr := strings.TrimSpace(expr[idx+len(op):])

			node := &ASTNode{
				Type:     "comparison",
				Operator: op,
				Left: &ASTNode{
					Type:  "field",
					Value: field,
				},
			}

			if op == "IS NULL" || op == "IS NOT NULL" {
				// No right operand
				return node, nil
			}

			if op == "IN" {
				// Parse list: (val1, val2, val3)
				valueExpr = strings.TrimSpace(valueExpr)
				if !strings.HasPrefix(valueExpr, "(") || !strings.HasSuffix(valueExpr, ")") {
					return nil, fmt.Errorf("IN operator requires parentheses: %s", valueExpr)
				}
				items := strings.Split(valueExpr[1:len(valueExpr)-1], ",")
				for _, item := range items {
					node.Children = append(node.Children, &ASTNode{
						Type:  "literal",
						Value: strings.TrimSpace(item),
					})
				}
			} else {
				node.Right = &ASTNode{
					Type:  "literal",
					Value: valueExpr,
				}
			}

			return node, nil
		}
	}

	return nil, fmt.Errorf("unable to parse expression: %s", expr)
}

// splitByOperator splits an expression by a logical operator (respecting parentheses).
func (p *Parser) splitByOperator(expr, operator string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	operatorLen := len(operator)

	for i := 0; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			depth--
		}

		if depth == 0 && i+operatorLen <= len(expr) &&
			strings.EqualFold(expr[i:i+operatorLen], operator) &&
			(i == 0 || !isAlphaNum(expr[i-1])) &&
			(i+operatorLen >= len(expr) || !isAlphaNum(expr[i+operatorLen])) {
			parts = append(parts, current.String())
			current.Reset()
			i += operatorLen - 1
			continue
		}

		current.WriteByte(expr[i])
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// indexOfOperator finds the index of an operator in the expression.
func (p *Parser) indexOfOperator(expr, operator string) int {
	operatorUpper := strings.ToUpper(operator)
	exprUpper := strings.ToUpper(expr)
	idx := strings.Index(exprUpper, operatorUpper)
	if idx == -1 {
		return -1
	}
	// Ensure it's not part of a larger word
	if idx > 0 && isAlphaNum(expr[idx-1]) {
		return -1
	}
	if idx+len(operator) < len(expr) && isAlphaNum(expr[idx+len(operator)]) {
		return -1
	}
	return idx
}

// isAlphaNum checks if a character is alphanumeric or underscore.
func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// GetReferencedFields returns all field names referenced in the AST.
func (n *ASTNode) GetReferencedFields() []string {
	var fields []string
	seen := make(map[string]struct{})
	n.collectFields(&fields, seen)
	return fields
}

func (n *ASTNode) collectFields(fields *[]string, seen map[string]struct{}) {
	if n == nil {
		return
	}
	if n.Type == "field" {
		if _, exists := seen[n.Value]; !exists {
			seen[n.Value] = struct{}{}
			*fields = append(*fields, n.Value)
		}
	}
	n.Left.collectFields(fields, seen)
	n.Right.collectFields(fields, seen)
	for _, child := range n.Children {
		child.collectFields(fields, seen)
	}
}

// ToSQL converts the AST to a SQL WHERE clause predicate.
func (n *ASTNode) ToSQL() string {
	if n == nil {
		return ""
	}

	switch n.Type {
	case "binary_op":
		left := n.Left.ToSQL()
		right := n.Right.ToSQL()
		return fmt.Sprintf("(%s %s %s)", left, n.Operator, right)
	case "unary_op":
		inner := n.Left.ToSQL()
		return fmt.Sprintf("%s (%s)", n.Operator, inner)
	case "comparison":
		left := n.Left.ToSQL()
		if n.Operator == "IS NULL" || n.Operator == "IS NOT NULL" {
			return fmt.Sprintf("%s %s", left, n.Operator)
		}
		if n.Operator == "IN" {
			var values []string
			for _, child := range n.Children {
				values = append(values, child.Value)
			}
			return fmt.Sprintf("%s IN (%s)", left, strings.Join(values, ", "))
		}
		right := n.Right.ToSQL()
		return fmt.Sprintf("%s %s %s", left, n.Operator, right)
	case "field":
		return n.Value
	case "literal":
		return n.Value
	}

	return ""
}
