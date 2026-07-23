package boresolver

type Expr interface {
	exprNode()
}

type TermRef struct {
	Name string
}

func (*TermRef) exprNode() {}

type NumberLiteral struct {
	Value string
}

func (*NumberLiteral) exprNode() {}

type StringLiteral struct {
	Value string
}

func (*StringLiteral) exprNode() {}

type BinaryExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

func (*BinaryExpr) exprNode() {}

type UnaryExpr struct {
	Op   string
	Expr Expr
}

func (*UnaryExpr) exprNode() {}

type FunctionCall struct {
	FunctionName string
	Args         []Expr
}

func (*FunctionCall) exprNode() {}
