package vm

// ParseExpression turns calc-term/rule expression text - what a user
// types into an editor, e.g. "SUM(ExecQuantity * ExecPrice)" or
// "XIRR(cash_flows, dates) / 2" - into the same *Expression AST every
// other consumer in this package already works with (EvaluateNumeric,
// CompileToSQL, the WASM build). Before this file, the only way to
// produce an Expression was to build the Go struct literal by hand (see
// every *_test.go in this package, and cmd/verify_calc_measure's
// hand-written JSON) - there was no text surface at all, which is why
// no editor could offer one.
//
// Deliberately scoped to arithmetic expressions only (+ - * / and
// function calls) - not RuleCondition/RuleGroup's boolean vocabulary,
// which already has a real authoring surface (AdvancedConditionBuilder's
// structured field/operator/value UI) that this isn't replacing.
// Deliberately no string-literal support yet: Literal only holds a
// float64 (ast.go) - a function like YEARFRAC that takes a literal
// basis string ("ACT/365") can't be fully authored as text until that's
// added; it's fine referenced via a FieldRef column instead. Documented
// as an open item in the handoff rather than silently worked around.

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ParseExpression parses src into an Expression AST, or returns a
// ParseError describing exactly where and why parsing failed - the
// editor needs a position to underline, not just a message. Top-level
// grammar allows one optional comparison (==, !=, >, <, >=, <=) wrapping
// an arithmetic expression on each side, e.g. "XIRR(cash_flows, dates) >
// 0.15" - not general boolean combinators (no &&/||/!): grouping
// multiple conditions is already the structured condition builder's job
// (RuleGroup/AND/OR), and this parser's whole reason to exist is
// authoring the FuncCall/Expression trees that builder can't produce,
// not duplicating what it already does well.
func ParseExpression(src string) (*Expression, error) {
	toks, err := lexExpression(src)
	if err != nil {
		return nil, err
	}
	p := &exprParser{toks: toks, src: src}
	root, err := p.parseCompare()
	if err != nil {
		return nil, err
	}
	if !p.atEnd() {
		return nil, p.errorf(p.peek(), "unexpected %s", p.peek().describe())
	}
	return &Expression{Root: root}, nil
}

// ParseError reports a parse failure with a byte offset into the
// original source, so a caller (an editor's diagnostics, or an API
// error response) can point at the exact spot rather than just quoting
// a message.
type ParseError struct {
	Message string
	Pos     int // byte offset into the source
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("expression syntax error at position %d: %s", e.Pos, e.Message)
}

// --- Lexer ---

type tokenKind int

const (
	tokNumber tokenKind = iota
	tokIdent
	tokLParen
	tokRParen
	tokComma
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokEq
	tokNe
	tokGt
	tokLt
	tokGe
	tokLe
	tokEOF
)

type token struct {
	kind tokenKind
	text string
	pos  int
}

func (t token) describe() string {
	if t.kind == tokEOF {
		return "end of expression"
	}
	return fmt.Sprintf("%q", t.text)
}

func lexExpression(src string) ([]token, error) {
	var toks []token
	runes := []rune(src)
	i := 0
	for i < len(runes) {
		c := runes[i]
		switch {
		case unicode.IsSpace(c):
			i++
		case c == '(':
			toks = append(toks, token{tokLParen, "(", i})
			i++
		case c == ')':
			toks = append(toks, token{tokRParen, ")", i})
			i++
		case c == ',':
			toks = append(toks, token{tokComma, ",", i})
			i++
		case c == '+':
			toks = append(toks, token{tokPlus, "+", i})
			i++
		case c == '-':
			toks = append(toks, token{tokMinus, "-", i})
			i++
		case c == '*':
			toks = append(toks, token{tokStar, "*", i})
			i++
		case c == '/':
			toks = append(toks, token{tokSlash, "/", i})
			i++
		case c == '=' && i+1 < len(runes) && runes[i+1] == '=':
			toks = append(toks, token{tokEq, "==", i})
			i += 2
		case c == '!' && i+1 < len(runes) && runes[i+1] == '=':
			toks = append(toks, token{tokNe, "!=", i})
			i += 2
		case c == '>' && i+1 < len(runes) && runes[i+1] == '=':
			toks = append(toks, token{tokGe, ">=", i})
			i += 2
		case c == '<' && i+1 < len(runes) && runes[i+1] == '=':
			toks = append(toks, token{tokLe, "<=", i})
			i += 2
		case c == '>':
			toks = append(toks, token{tokGt, ">", i})
			i++
		case c == '<':
			toks = append(toks, token{tokLt, "<", i})
			i++
		case unicode.IsDigit(c) || (c == '.' && i+1 < len(runes) && unicode.IsDigit(runes[i+1])):
			start := i
			seenDot := false
			for i < len(runes) && (unicode.IsDigit(runes[i]) || (runes[i] == '.' && !seenDot)) {
				if runes[i] == '.' {
					seenDot = true
				}
				i++
			}
			toks = append(toks, token{tokNumber, string(runes[start:i]), start})
		case isIdentStart(c):
			start := i
			for i < len(runes) && isIdentPart(runes[i]) {
				i++
			}
			toks = append(toks, token{tokIdent, string(runes[start:i]), start})
		default:
			return nil, &ParseError{Message: fmt.Sprintf("unexpected character %q", string(c)), Pos: i}
		}
	}
	toks = append(toks, token{tokEOF, "", len(runes)})
	return toks, nil
}

func isIdentStart(c rune) bool {
	return unicode.IsLetter(c) || c == '_'
}

// isIdentPart allows '.' inside an identifier so dotted field paths
// (e.g. "client.risk_score") lex as one token, matching FieldRef.Path's
// existing dotted-path convention (evalFieldRef, unresolvedFieldRefs).
func isIdentPart(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '.'
}

// --- Parser ---

// Grammar (lowest to highest precedence):
//
//	addsub := muldiv (('+' | '-') muldiv)*
//	muldiv := unary (('*' | '/') unary)*
//	unary  := '-' unary | primary
//	primary := NUMBER | IDENT '(' (addsub (',' addsub)*)? ')' | IDENT | '(' addsub ')'
type exprParser struct {
	toks []token
	pos  int
	src  string
}

func (p *exprParser) peek() token  { return p.toks[p.pos] }
func (p *exprParser) atEnd() bool  { return p.peek().kind == tokEOF }
func (p *exprParser) advance() token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *exprParser) errorf(t token, format string, args ...any) error {
	return &ParseError{Message: fmt.Sprintf(format, args...), Pos: t.pos}
}

var compareOps = map[tokenKind]string{
	tokEq: "==", tokNe: "!=", tokGt: ">", tokLt: "<", tokGe: ">=", tokLe: "<=",
}

func (p *exprParser) parseCompare() (ExprNode, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}
	opStr, isCompare := compareOps[p.peek().kind]
	if !isCompare {
		return left, nil
	}
	p.advance()
	right, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}
	return &BinaryExpr{Op: opStr, Left: left, Right: right}, nil
}

func (p *exprParser) parseAddSub() (ExprNode, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokPlus || p.peek().kind == tokMinus {
		op := p.advance()
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		opStr := "+"
		if op.kind == tokMinus {
			opStr = "-"
		}
		left = &BinaryExpr{Op: opStr, Left: left, Right: right}
	}
	return left, nil
}

func (p *exprParser) parseMulDiv() (ExprNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokStar || p.peek().kind == tokSlash {
		op := p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		opStr := "*"
		if op.kind == tokSlash {
			opStr = "/"
		}
		left = &BinaryExpr{Op: opStr, Left: left, Right: right}
	}
	return left, nil
}

func (p *exprParser) parseUnary() (ExprNode, error) {
	if p.peek().kind == tokMinus {
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		// No dedicated unary-minus AST node - "-x" compiles to "0 - x",
		// which every consumer (VM compiler, SQL compiler, evaluator)
		// already handles via the existing binary '-' path.
		return &BinaryExpr{Op: "-", Left: &Literal{Value: 0}, Right: operand}, nil
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (ExprNode, error) {
	t := p.peek()
	switch t.kind {
	case tokNumber:
		p.advance()
		v, err := strconv.ParseFloat(t.text, 64)
		if err != nil {
			return nil, p.errorf(t, "invalid number %q", t.text)
		}
		return &Literal{Value: v}, nil

	case tokLParen:
		p.advance()
		inner, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRParen {
			return nil, p.errorf(p.peek(), "expected ')', got %s", p.peek().describe())
		}
		p.advance()
		return inner, nil

	case tokIdent:
		p.advance()
		if p.peek().kind == tokLParen {
			return p.parseFuncCallArgs(t.text)
		}
		if strings.TrimSpace(t.text) == "" {
			return nil, p.errorf(t, "empty identifier")
		}
		return &FieldRef{Path: t.text}, nil

	default:
		return nil, p.errorf(t, "expected a number, field name, or function call, got %s", t.describe())
	}
}

func (p *exprParser) parseFuncCallArgs(name string) (ExprNode, error) {
	p.advance() // consume '('
	var args []ExprNode
	if p.peek().kind != tokRParen {
		for {
			arg, err := p.parseAddSub()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.peek().kind == tokComma {
				p.advance()
				continue
			}
			break
		}
	}
	if p.peek().kind != tokRParen {
		return nil, p.errorf(p.peek(), "expected ')' or ',' in %s(...) arguments, got %s", name, p.peek().describe())
	}
	p.advance()
	return &FuncCall{Name: name, Args: args}, nil
}
