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

// Type of expressions will be left empty when initialized, except literals.
func NewBaseExprNode(loc *Location) *BaseExprNode {
	return &BaseExprNode{BaseStmtNode: *NewBaseStmtNode(loc)}
}

func (n *BaseExprNode) GetType() IType { return n.tp }

func (n *BaseExprNode) SetType(tp IType) { n.tp = tp }

// Constant expression
type IConstExpr interface {
	IExprNode
	GetValue() interface{}
}

type IntConst struct {
	BaseExprNode
	val int
}

func NewIntConst(loc *Location, val int) *IntConst {
	c := &IntConst{BaseExprNode: *NewBaseExprNode(loc), val: val}
	c.SetType(NewPrimType(Int))
	return c
}

func (c *IntConst) ToStringTree() string {
	return fmt.Sprint(c.val)
}

func (c *IntConst) GetValue() interface{} { return c.val }

type FloatConst struct {
	BaseExprNode
	val float64
}

func NewFloatConst(loc *Location, val float64) *FloatConst {
	c := &FloatConst{BaseExprNode: *NewBaseExprNode(loc), val: val}
	c.SetType(NewPrimType(Float64))
	return c
}

func (c *FloatConst) ToStringTree() string {
	return fmt.Sprintf("%.4f", c.val)
}

func (c *FloatConst) GetValue() interface{} { return c.val }

// Identifier expression
type IdExpr struct {
	BaseExprNode
	name   string // should keep name for lookup in global scope
	symbol *SymbolEntry
}

func NewIdExpr(loc *Location, name string, symbol *SymbolEntry) *IdExpr {
	return &IdExpr{BaseExprNode: *NewBaseExprNode(loc), name: name, symbol: symbol}
}

func (i *IdExpr) ToStringTree() string { return i.name }

// Function calling expression
type FuncCallExpr struct {
	BaseExprNode
	fun     IExprNode
	args    []IExprNode
	argType TupleType
}

func NewFuncCallExpr(loc *Location, fun IExprNode, args []IExprNode) *FuncCallExpr {
	return &FuncCallExpr{BaseExprNode: *NewBaseExprNode(loc), args: args}
}

func (e *FuncCallExpr) ToStringTree() string {
	str := fmt.Sprintf("(call %s (", e.fun.ToStringTree())
	for i, arg := range e.args {
		if i != 0 {
			str += " "
		}
		str += arg.ToStringTree()
	}
	return str + "))"
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

var UnaryOpStr = map[UnaryOp]string{
	POS: "+", NEG: "-", NOT: "!", INV: "^", DEREF: "*", REF: "&",
}

var UnaryOpStrToEnum map[string]UnaryOp

func init() {
	for op := POS; op <= REF; op++ {
		UnaryOpStrToEnum[UnaryOpStr[op]] = op
	}
}

func NewUnaryExpr(loc *Location, op UnaryOp, expr IExprNode) *UnaryExpr {
	return &UnaryExpr{BaseExprNode: *NewBaseExprNode(loc), op: op, expr: expr}
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

var BinaryOpStr = map[BinaryOp]string{
	MUL: "*", DIV: "/", MOD: "%", LSH: "<<", RSH: ">>", AAND: "&",
	ADD: "+", SUB: "-", AOR: "|", XOR: "^", EQ: "==", NEQ: "!=",
	LT: "<", LEQ: "<=", GT: ">", GEQ: ">=", LAND: "&&", LOR: "||",
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
	return &BinaryExpr{BaseExprNode: *NewBaseExprNode(loc), op: op, left: left, right: right}
}

func (e *BinaryExpr) ToStringTree() string {
	return fmt.Sprintf("(%s %s %s)", BinaryOpStr[e.op], e.left.ToStringTree(),
		e.right.ToStringTree())
}
