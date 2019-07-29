package ast

import (
	"fmt"
)

// Abstract expression interface
type IExprNode interface {
	IStmtNode
	GetType() IType
	SetType(tp IType)
}

type BaseExprNode struct {
	BaseStmtNode
	// tp is nil: type of current node is unknown, to be determined in a later pass.
	// tp is Unresolved: type name of current node is known, but its validity remains to be checked.
	tp IType
}

// Type of expressions will be left empty when initialized, except literals.
func NewBaseExprNode(loc *Location) *BaseExprNode {
	return &BaseExprNode{BaseStmtNode: *NewBaseStmtNode(loc)}
}

func (n *BaseExprNode) GetType() IType { return n.tp }

func (n *BaseExprNode) SetType(tp IType) { n.tp = tp }

// Literal expressions
type ILiteralExpr interface {
	IExprNode
	GetValue() interface{}
}

type BaseLiteralExpr struct {
	BaseExprNode
}

func NewBaseLiteralExpr(loc *Location) *BaseLiteralExpr {
	return &BaseLiteralExpr{BaseExprNode: *NewBaseExprNode(loc)}
}

func (n *BaseLiteralExpr) IsLValue() bool { return true }

// Constant expressions (can be assigned to constant, special case of literal expression)
type ConstExpr struct {
	BaseLiteralExpr
	val interface{}
}

func (e *ConstExpr) GetValue() interface{} { return e.val }

func NewIntConst(loc *Location, val int) *ConstExpr {
	e := &ConstExpr{BaseLiteralExpr: *NewBaseLiteralExpr(loc), val: val}
	e.tp = NewPrimType(Int)
	return e
}

func NewFloatConst(loc *Location, val float64) *ConstExpr {
	e := &ConstExpr{BaseLiteralExpr: *NewBaseLiteralExpr(loc), val: val}
	e.tp = NewPrimType(Float64)
	return e
}

func NewBoolConst(loc *Location, val bool) *ConstExpr {
	e := &ConstExpr{BaseLiteralExpr: *NewBaseLiteralExpr(loc), val: val}
	e.tp = NewPrimType(Bool)
	return e
}

// Zero value is the internal representation of initial value of any type.
// It cannot be declared as a literal like nil.
type ZeroValue struct {
	BaseExprNode
}

func NewZeroValue() *ZeroValue { return &ZeroValue{} }

func (c *ZeroValue) ToStringTree() string { return "0" }

func (c *ZeroValue) GetValue() interface{} { return nil }

// nil is a predeclared identifier representing the zero value for a pointer, channel, func,
// interface, map, or slice type. It can be declared as a literal
type NilValue struct {
	BaseExprNode
}

func NewNilValue(loc *Location) *NilValue {
	n := &NilValue{BaseExprNode: *NewBaseExprNode(loc)}
	n.tp = NewNilType()
	return n
}

func (c *NilValue) ToStringTree() string { return "nil" }

func (c *NilValue) GetValue() interface{} { return nil }

// Identifier expression
type IdExpr struct {
	BaseExprNode
	name   string // should keep name for lookup in global scope
	symbol *SymbolEntry
}

var IsKeyword = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
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

var UnaryOpStrToEnum = map[string]UnaryOp{}

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
	NE
	LT
	LE
	GT
	GE
	LAND // logical AND
	LOR
)

var BinaryOpStr = map[BinaryOp]string{
	MUL: "*", DIV: "/", MOD: "%", LSH: "<<", RSH: ">>", AAND: "&",
	ADD: "+", SUB: "-", AOR: "|", XOR: "^", EQ: "==", NE: "!=",
	LT: "<", LE: "<=", GT: ">", GE: ">=", LAND: "&&", LOR: "||",
}

var BinaryOpStrToEnum = map[string]BinaryOp{}

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

var constTypeConvert = map[TypeEnum]map[TypeEnum]func(interface{}) interface{}{
	Int: {
		Int:     func(v interface{}) interface{} { return v },
		Float64: func(v interface{}) interface{} { return float64(v.(int)) },
	},
	Float64: {
		Float64: func(v interface{}) interface{} { return v },
		Int:     func(v interface{}) interface{} { return int(v.(float64)) },
	},
}

var unaryConstExpr = map[UnaryOp]map[TypeEnum]func(*ConstExpr) *ConstExpr{
	POS: {
		Int:     func(e *ConstExpr) *ConstExpr { return NewIntConst(e.loc, e.val.(int)) },
		Float64: func(e *ConstExpr) *ConstExpr { return NewFloatConst(e.loc, e.val.(float64)) },
	},
	NEG: {
		Int:     func(e *ConstExpr) *ConstExpr { return NewIntConst(e.loc, -e.val.(int)) },
		Float64: func(e *ConstExpr) *ConstExpr { return NewFloatConst(e.loc, -e.val.(float64)) },
	},
	NOT: {
		Bool: func(e *ConstExpr) *ConstExpr { return NewBoolConst(e.loc, !e.val.(bool)) },
	},
	INV: {
		Int: func(e *ConstExpr) *ConstExpr { return NewIntConst(e.loc, ^e.val.(int)) },
	},
}

var binaryConstExpr = map[BinaryOp]map[TypeEnum]map[TypeEnum]func(l, r *ConstExpr) *ConstExpr{
	ADD: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)+r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, float64(l.val.(int))+r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, l.val.(float64)+float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, l.val.(float64)+r.val.(float64))
			},
		},
	},
	SUB: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)-r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, float64(l.val.(int))-r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, l.val.(float64)-float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, l.val.(float64)-r.val.(float64))
			},
		},
	},
	MUL: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)*r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, float64(l.val.(int))*r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, l.val.(float64)*float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, l.val.(float64)*r.val.(float64))
			},
		},
	},
	DIV: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)/r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, float64(l.val.(int))/r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, l.val.(float64)/float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.loc, l.val.(float64)/r.val.(float64))
			},
		},
	},
	MOD: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)%r.val.(int))
			},
		},
	},
	LSH: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)<<uint(r.val.(int)))
			},
		},
	},
	RSH: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)>>uint(r.val.(int)))
			},
		},
	},
	AAND: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)&r.val.(int))
			},
		},
	},
	AOR: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)|r.val.(int))
			},
		},
	},
	XOR: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.loc, l.val.(int)^r.val.(int))
			},
		},
	},
	EQ: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(int) == r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, float64(l.val.(int)) == r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) == float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) == r.val.(float64))
			},
		},
		Bool: {
			Bool: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(bool) == r.val.(bool))
			},
		},
	},
	NE: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(int) != r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, float64(l.val.(int)) != r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) != float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) != r.val.(float64))
			},
		},
		Bool: {
			Bool: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(bool) != r.val.(bool))
			},
		},
	},
	LT: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(int) < r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, float64(l.val.(int)) < r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) < float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) < r.val.(float64))
			},
		},
	},
	LE: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(int) <= r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, float64(l.val.(int)) <= r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) <= float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) <= r.val.(float64))
			},
		},
	},
	GT: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(int) > r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, float64(l.val.(int)) > r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) > float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) > r.val.(float64))
			},
		},
	},
	GE: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(int) >= r.val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, float64(l.val.(int)) >= r.val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) >= float64(r.val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(float64) >= r.val.(float64))
			},
		},
	},
	LAND: {
		Bool: {
			Bool: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(bool) && r.val.(bool))
			},
		},
	},
	LOR: {
		Bool: {
			Bool: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.loc, l.val.(bool) || r.val.(bool))
			},
		},
	},
}
