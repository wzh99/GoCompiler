package ast

import "fmt"

// Abstract expression interface
type IExprNode interface {
	IStmtNode
	GetType() IType
	SetType(tp IType)
}

type BaseExprNode struct {
	BaseStmtNode
	// tp is nil: type of current node is unknown, to be determined in a later pass.
	// tp is Unresolved: type name of current node is known, but its validity remains to be verified.
	tp IType
}

func newBaseExprNode(loc *Location, tp IType) *BaseExprNode {
	return &BaseExprNode{BaseStmtNode: *newBaseStmtNode(loc), tp: tp}
}

func (n *BaseExprNode) GetType() IType { return n.tp }

func (n *BaseExprNode) SetType(tp IType) { n.tp = tp }

// Literal expression
type IntLiteral struct {
	BaseExprNode
	val int
}

func NewIntLiteral(loc *Location, val int) *IntLiteral {
	return &IntLiteral{BaseExprNode: *newBaseExprNode(loc, NewPrimType(Int)), val: val}
}

func (l *IntLiteral) ToStringTree() string {
	return fmt.Sprint(l.val)
}

type FloatLiteral struct {
	BaseExprNode
	val float64
}

func NewFloatLiteral(loc *Location, val float64) *FloatLiteral {
	return &FloatLiteral{BaseExprNode: *newBaseExprNode(loc, NewPrimType(Float64)), val: val}
}

func (l *FloatLiteral) ToStringTree() string {
	return fmt.Sprintf("%.4f", l.val)
}

type BinaryOp int

const (
	MUL BinaryOp = iota
	DIV
	MOD
	LSH // left shift
	RSH
	AAND // arithmetic AND
	ADD
	SUB
	AOR
	XOR
	EQ
	NEQ
	LT
	LEQ
	GT
	GEQ
	LAND // logical AND
	LOR
)

var BinaryOpStr = [...]string{
	"*", "/", "%", "<<", ">>", "&", "+", "-", "|", "^", "==", "!=",
	"<", "<=", ">", ">=", "&&", "||",
}

var BinaryOpStrToEnum map[string]BinaryOp

func init() {
	for op := MUL; op <= LOR; op++ {
		BinaryOpStrToEnum[BinaryOpStr[op]] = op
	}
}

type BinaryExpr struct {
	BaseExprNode
	op          BinaryOp
	left, right IExprNode
}

func NewBinaryExpr(loc *Location, op BinaryOp, left, right IExprNode) *BinaryExpr {
	// result type of binary expression is determined during semantic analysis
	return &BinaryExpr{BaseExprNode: *newBaseExprNode(loc, nil), op: op, left: left, right: right}
}

func (e *BinaryExpr) ToStringTree() string {
	return fmt.Sprintf("(%s %s %s)", BinaryOpStr[e.op], e.left.ToStringTree(),
		e.right.ToStringTree())
}

type UnaryExpr struct {
	BaseExprNode
	op   UnaryOp
	expr IExprNode
}

type UnaryOp int

const (
	POS UnaryOp = iota
	NEG
	NOT
	INV
	DEREF
	REF
)

var UnaryOpStr = [...]string{
	"+", "-", "!", "^", "*", "&",
}

var UnaryOpStrToEnum map[string]UnaryOp

func init() {
	for op := POS; op <= REF; op++ {
		UnaryOpStrToEnum[UnaryOpStr[op]] = op
	}
}

func NewUnaryExpr(loc *Location, op UnaryOp, expr IExprNode) *UnaryExpr {
	return &UnaryExpr{BaseExprNode: *newBaseExprNode(loc, nil), op: op, expr: expr}
}
