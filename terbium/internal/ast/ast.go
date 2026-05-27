package ast

type Program struct {
	Statements []Stmt
}

type Stmt interface {
	stmtNode()
}

type PrintStmt struct {
	Value Expr
}

func (*PrintStmt) stmtNode() {}

type Expr interface {
	exprNode()
}

type IntLiteral struct {
	Value int64
}

type StringLiteral struct {
	Value string
}

func (*IntLiteral) exprNode()    {}
func (*StringLiteral) exprNode() {}

type BinaryExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

func (*BinaryExpr) exprNode() {}
