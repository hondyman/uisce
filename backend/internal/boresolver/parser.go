package boresolver

import (
	"fmt"
	"strings"
	"unicode"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenNumber
	TokenString
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenLParen
	TokenRParen
	TokenComma

	// Comparison
	TokenEQ  // =
	TokenNEQ // !=
	TokenGT  // >
	TokenGTE // >=
	TokenLT  // <
	TokenLTE // <=

	// Boolean
	TokenAND
	TokenOR
	TokenNOT
)

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	input []rune
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: []rune(input), pos: 0}
}

func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	ch := l.input[l.pos]
	l.pos++
	return ch
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) NextToken() Token {
	// skip whitespace
	for unicode.IsSpace(l.peek()) {
		l.next()
	}

	ch := l.peek()
	if ch == 0 {
		return Token{Type: TokenEOF}
	}

	// identifiers and keywords
	if unicode.IsLetter(ch) || ch == '_' {
		start := l.pos
		for {
			ch = l.peek()
			if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '.' {
				l.next()
			} else {
				break
			}
		}
		val := string(l.input[start:l.pos])

		// Check keywords
		upperVal := strings.ToUpper(val)
		if upperVal == "AND" {
			return Token{Type: TokenAND, Value: "AND"}
		}
		if upperVal == "OR" {
			return Token{Type: TokenOR, Value: "OR"}
		}
		if upperVal == "NOT" {
			return Token{Type: TokenNOT, Value: "NOT"}
		}

		return Token{Type: TokenIdent, Value: val}
	}

	// numbers
	if unicode.IsDigit(ch) {
		start := l.pos
		hasDot := false
		for {
			ch = l.peek()
			if unicode.IsDigit(ch) {
				l.next()
			} else if ch == '.' && !hasDot {
				hasDot = true
				l.next()
			} else {
				break
			}
		}
		return Token{Type: TokenNumber, Value: string(l.input[start:l.pos])}
	}

	// strings
	if ch == '\'' || ch == '"' {
		quote := l.next()
		start := l.pos
		for {
			ch = l.peek()
			if ch == 0 {
				break
			}
			if ch == quote {
				break
			}
			l.next()
		}
		val := string(l.input[start:l.pos])
		l.next()
		return Token{Type: TokenString, Value: val}
	}

	// symbol tokens
	switch ch {
	case '+':
		l.next()
		return Token{Type: TokenPlus, Value: "+"}
	case '-':
		l.next()
		return Token{Type: TokenMinus, Value: "-"}
	case '*':
		l.next()
		return Token{Type: TokenStar, Value: "*"}
	case '/':
		l.next()
		return Token{Type: TokenSlash, Value: "/"}
	case '(':
		l.next()
		return Token{Type: TokenLParen, Value: "("}
	case ')':
		l.next()
		return Token{Type: TokenRParen, Value: ")"}
	case ',':
		l.next()
		return Token{Type: TokenComma, Value: ","}
	case '=':
		l.next()
		return Token{Type: TokenEQ, Value: "="}
	case '!':
		if l.peek() == '=' {
			l.next() // consume '!'
			l.next() // consume '='
			return Token{Type: TokenNEQ, Value: "!="}
		}
		l.next()
		return Token{Type: TokenEOF} // or error
	case '>':
		l.next()
		if l.peek() == '=' {
			l.next()
			return Token{Type: TokenGTE, Value: ">="}
		}
		return Token{Type: TokenGT, Value: ">"}
	case '<':
		l.next()
		if l.peek() == '=' {
			l.next()
			return Token{Type: TokenLTE, Value: "<="}
		}
		// check for <> as NEQ?
		if l.peek() == '>' {
			l.next()
			return Token{Type: TokenNEQ, Value: "<>"}
		}
		return Token{Type: TokenLT, Value: "<"}
	default:
		l.next()
		return Token{Type: TokenEOF}
	}
}

type Parser struct {
	lexer  *Lexer
	cur    Token
	peeked bool
}

func NewParser(input string) *Parser {
	return &Parser{
		lexer: NewLexer(input),
	}
}

func (p *Parser) nextToken() Token {
	if p.peeked {
		p.peeked = false
		return p.cur
	}
	p.cur = p.lexer.NextToken()
	return p.cur
}

func (p *Parser) peekToken() Token {
	if !p.peeked {
		p.cur = p.lexer.NextToken()
		p.peeked = true
	}
	return p.cur
}

// precedence levels
const (
	precLowest  = iota
	precOr      // OR
	precAnd     // AND
	precCompare // = != > < >= <=
	precSum     // + -
	precProduct // * /
	precPrefix  // unary - NOT
	precCall    // func(
)

func precedence(tok Token) int {
	switch tok.Type {
	case TokenOR:
		return precOr
	case TokenAND:
		return precAnd
	case TokenEQ, TokenNEQ, TokenGT, TokenGTE, TokenLT, TokenLTE:
		return precCompare
	case TokenPlus, TokenMinus:
		return precSum
	case TokenStar, TokenSlash:
		return precProduct
	case TokenLParen:
		return precCall
	default:
		return precLowest
	}
}

func ParseExpression(input string) (Expr, error) {
	p := NewParser(input)
	p.nextToken()

	expr, err := p.parseExpr(precLowest)
	if err != nil {
		return nil, err
	}
	return expr, nil
}

func (p *Parser) parseExpr(minPrec int) (Expr, error) {
	// prefix
	tok := p.cur
	var left Expr
	var err error

	switch tok.Type {
	case TokenIdent:
		val := tok.Value
		if p.peekToken().Type == TokenLParen {
			p.nextToken()
			args, err := p.parseCallArguments()
			if err != nil {
				return nil, err
			}
			left = &FunctionCall{FunctionName: val, Args: args}
		} else {
			left = &TermRef{Name: val}
		}

	case TokenNumber:
		left = &NumberLiteral{Value: tok.Value}
	case TokenString:
		left = &StringLiteral{Value: tok.Value}
	case TokenMinus:
		p.nextToken()
		right, err := p.parseExpr(precPrefix)
		if err != nil {
			return nil, err
		}
		left = &UnaryExpr{Op: "-", Expr: right}
	case TokenNOT:
		p.nextToken()
		right, err := p.parseExpr(precPrefix)
		if err != nil {
			return nil, err
		}
		left = &UnaryExpr{Op: "NOT", Expr: right}
	case TokenLParen:
		p.nextToken()
		left, err = p.parseExpr(precLowest)
		if err != nil {
			return nil, err
		}
		if p.peekToken().Type == TokenRParen {
			p.nextToken()
		} else if p.cur.Type == TokenRParen {
			if p.peekToken().Type != TokenRParen {
				return nil, fmt.Errorf("expected ')'")
			}
			p.nextToken()
		} else {
			return nil, fmt.Errorf("expected ')'")
		}
	default:
		return nil, fmt.Errorf("unexpected token: %v", tok)
	}

	// infix
	for {
		next := p.peekToken()
		if next.Type == TokenEOF || next.Type == TokenRParen || next.Type == TokenComma {
			break
		}

		opPrec := precedence(next)
		if opPrec <= minPrec {
			break
		}

		p.nextToken() // consume operator
		opTok := p.cur

		p.nextToken() // move to right operand
		right, err := p.parseExpr(opPrec)
		if err != nil {
			return nil, err
		}

		left = &BinaryExpr{
			Op:    opTok.Value,
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

func (p *Parser) parseCallArguments() ([]Expr, error) {
	args := []Expr{}

	// Check for empty call "()"
	if p.peekToken().Type == TokenRParen {
		p.nextToken()
		return args, nil
	}

	p.nextToken()

	arg, err := p.parseExpr(precLowest)
	if err != nil {
		return nil, err
	}
	args = append(args, arg)

	for p.peekToken().Type == TokenComma {
		p.nextToken()
		p.nextToken()
		arg, err := p.parseExpr(precLowest)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	if p.peekToken().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after arguments")
	}
	p.nextToken()
	return args, nil
}
