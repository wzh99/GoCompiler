package ast

import (
	"fmt"
)

// Abstract expression interface
type IExprNode interface {
	IStmtNode
	GetType() IType
	SetType(tp IType)
	IsLValue() bool
}

type BaseExprNode struct {
	BaseStmtNode
	// tp is nil: type of current node is unknown, to be determined in a later pass.
	// tp is Unresolved: type name of current node is known, but its validity remains to be checked.
	Type IType
}

// Type of expressions will be left empty when initialized, except literals.
func NewBaseExprNode(loc *Location) *BaseExprNode {
	return &BaseExprNode{BaseStmtNode: *NewBaseStmtNode(loc)}
}

func (n *BaseExprNode) GetType() IType { return n.Type }

func (n *BaseExprNode) SetType(tp IType) { n.Type = tp }

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

// Anonymous function (lambda)
type FuncLiteral struct {
	BaseLiteralExpr
	Decl    *FuncDecl
	Closure *SymbolTable
}

func NewFuncLiteral(loc *Location, decl *FuncDecl, closure *SymbolTable) *FuncLiteral {
	return &FuncLiteral{BaseLiteralExpr: *NewBaseLiteralExpr(loc), Decl: decl, Closure: closure}
}

// Constant expressions (can be assigned to constant, special case of literal expression)
type ConstExpr struct {
	BaseLiteralExpr
	Val interface{}
}

func (e *ConstExpr) GetValue() interface{} { return e.Val }

func NewIntConst(loc *Location, val int) *ConstExpr {
	e := &ConstExpr{BaseLiteralExpr: *NewBaseLiteralExpr(loc), Val: val}
	e.Type = NewPrimType(Int)
	return e
}

func NewFloatConst(loc *Location, val float64) *ConstExpr {
	e := &ConstExpr{BaseLiteralExpr: *NewBaseLiteralExpr(loc), Val: val}
	e.Type = NewPrimType(Float64)
	return e
}

func NewBoolConst(loc *Location, val bool) *ConstExpr {
	e := &ConstExpr{BaseLiteralExpr: *NewBaseLiteralExpr(loc), Val: val}
	e.Type = NewPrimType(Bool)
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

func (c *ZeroValue) IsLValue() bool { return false }

// nil is a predeclared identifier representing the zero value for a pointer, channel, func,
// interface, map, or slice type. It can be declared as a literal
type NilValue struct {
	BaseLiteralExpr
}

func NewNilValue(loc *Location) *NilValue {
	n := &NilValue{BaseLiteralExpr: *NewBaseLiteralExpr(loc)}
	n.Type = NewNilType()
	return n
}

func (c *NilValue) ToStringTree() string { return "nil" }

func (c *NilValue) GetValue() interface{} { return nil }

func (c *NilValue) IsLValue() bool { return false }

// Identifier expression
type IdExpr struct {
	BaseExprNode
	Name     string // should keep name for lookup in global scope
	Symbol   *SymbolEntry
	Captured bool
}

var IsKeyword = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

func NewIdExpr(loc *Location, name string, symbol *SymbolEntry) *IdExpr {
	return &IdExpr{BaseExprNode: *NewBaseExprNode(loc), Name: name, Symbol: symbol}
}

func (i *IdExpr) ToStringTree() string { return i.Name }

func (i *IdExpr) IsLValue() bool { return true }

// Function calling expression
type FuncCallExpr struct {
	BaseExprNode
	Func IExprNode
	Args []IExprNode
}

func NewFuncCallExpr(loc *Location, fun IExprNode, args []IExprNode) *FuncCallExpr {
	return &FuncCallExpr{BaseExprNode: *NewBaseExprNode(loc), Func: fun, Args: args}
}

func (e *FuncCallExpr) ToStringTree() string {
	str := fmt.Sprintf("(call %s (", e.Func.ToStringTree())
	for i, arg := range e.Args {
		if i != 0 {
			str += " "
		}
		str += arg.ToStringTree()
	}
	return str + "))"
}

func (e *FuncCallExpr) IsLValue() bool { return false }

type UnaryExpr struct {
	BaseExprNode
	Op   UnaryOp
	Expr IExprNode
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
	return &UnaryExpr{BaseExprNode: *NewBaseExprNode(loc), Op: op, Expr: expr}
}

func (e *UnaryExpr) IsLValue() bool { return false }

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
	Op          BinaryOp
	Left, Right IExprNode
}

func NewBinaryExpr(loc *Location, op BinaryOp, left, right IExprNode) *BinaryExpr {
	// result type of binary expression is determined during semantic analysis
	return &BinaryExpr{BaseExprNode: *NewBaseExprNode(loc), Op: op, Left: left, Right: right}
}

func (e *BinaryExpr) ToStringTree() string {
	return fmt.Sprintf("(%s %s %s)", BinaryOpStr[e.Op], e.Left.ToStringTree(),
		e.Right.ToStringTree())
}

func (e *BinaryExpr) IsLValue() bool { return false }

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
		Int:     func(e *ConstExpr) *ConstExpr { return NewIntConst(e.Loc, e.Val.(int)) },
		Float64: func(e *ConstExpr) *ConstExpr { return NewFloatConst(e.Loc, e.Val.(float64)) },
	},
	NEG: {
		Int:     func(e *ConstExpr) *ConstExpr { return NewIntConst(e.Loc, -e.Val.(int)) },
		Float64: func(e *ConstExpr) *ConstExpr { return NewFloatConst(e.Loc, -e.Val.(float64)) },
	},
	NOT: {
		Bool: func(e *ConstExpr) *ConstExpr { return NewBoolConst(e.Loc, !e.Val.(bool)) },
	},
	INV: {
		Int: func(e *ConstExpr) *ConstExpr { return NewIntConst(e.Loc, ^e.Val.(int)) },
	},
}

var binaryConstExpr = map[BinaryOp]map[TypeEnum]map[TypeEnum]func(l, r *ConstExpr) *ConstExpr{
	ADD: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)+r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, float64(l.Val.(int))+r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, l.Val.(float64)+float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, l.Val.(float64)+r.Val.(float64))
			},
		},
	},
	SUB: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)-r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, float64(l.Val.(int))-r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, l.Val.(float64)-float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, l.Val.(float64)-r.Val.(float64))
			},
		},
	},
	MUL: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)*r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, float64(l.Val.(int))*r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, l.Val.(float64)*float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, l.Val.(float64)*r.Val.(float64))
			},
		},
	},
	DIV: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)/r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, float64(l.Val.(int))/r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, l.Val.(float64)/float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewFloatConst(l.Loc, l.Val.(float64)/r.Val.(float64))
			},
		},
	},
	MOD: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)%r.Val.(int))
			},
		},
	},
	LSH: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)<<uint(r.Val.(int)))
			},
		},
	},
	RSH: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)>>uint(r.Val.(int)))
			},
		},
	},
	AAND: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)&r.Val.(int))
			},
		},
	},
	AOR: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)|r.Val.(int))
			},
		},
	},
	XOR: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)^r.Val.(int))
			},
		},
	},
	EQ: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(int) == r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, float64(l.Val.(int)) == r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) == float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) == r.Val.(float64))
			},
		},
		Bool: {
			Bool: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(bool) == r.Val.(bool))
			},
		},
	},
	NE: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(int) != r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, float64(l.Val.(int)) != r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) != float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) != r.Val.(float64))
			},
		},
		Bool: {
			Bool: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(bool) != r.Val.(bool))
			},
		},
	},
	LT: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(int) < r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, float64(l.Val.(int)) < r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) < float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) < r.Val.(float64))
			},
		},
	},
	LE: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(int) <= r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, float64(l.Val.(int)) <= r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) <= float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) <= r.Val.(float64))
			},
		},
	},
	GT: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(int) > r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, float64(l.Val.(int)) > r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) > float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) > r.Val.(float64))
			},
		},
	},
	GE: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(int) >= r.Val.(int))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, float64(l.Val.(int)) >= r.Val.(float64))
			},
		},
		Float64: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) >= float64(r.Val.(int)))
			},
			Float64: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(float64) >= r.Val.(float64))
			},
		},
	},
	LAND: {
		Bool: {
			Bool: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(bool) && r.Val.(bool))
			},
		},
	},
	LOR: {
		Bool: {
			Bool: func(l, r *ConstExpr) *ConstExpr {
				return NewBoolConst(l.Loc, l.Val.(bool) || r.Val.(bool))
			},
		},
	},
}
