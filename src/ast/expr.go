package ast

import (
	"fmt"
	"strconv"
)

// Abstract expression interface
type IExprNode interface {
	IStmtNode
	GetType() IType
	IsLValue() bool
}

type BaseExprNode struct {
	BaseStmtNode
	// tp is nil: type of current node is unknown, to be determined in a later pass.
	// tp is Unresolved: type name of current node is known, but its validity remains to be checked.
	Type IType
}

// Type of expressions will be left empty when initialized, except literals.
func NewBaseExprNode(loc *Loc) *BaseExprNode {
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

func NewBaseLiteralExpr(loc *Loc) *BaseLiteralExpr {
	return &BaseLiteralExpr{BaseExprNode: *NewBaseExprNode(loc)}
}

func (n *BaseLiteralExpr) IsLValue() bool { return true }

// Anonymous function (lambda)
type FuncLit struct {
	BaseLiteralExpr
	Decl    *FuncDecl
	Closure *SymbolTable
}

func NewFuncLit(loc *Loc, decl *FuncDecl, closure *SymbolTable) *FuncLit {
	return &FuncLit{BaseLiteralExpr: *NewBaseLiteralExpr(loc), Decl: decl, Closure: closure}
}

func (l *FuncLit) GetValue() interface{} { return l.Decl }

type LitElem struct {
	Loc  *Loc
	Key  IExprNode
	Type IType
	Elem IExprNode
}

func NewLitElem(loc *Loc, key IExprNode, elem IExprNode) *LitElem {
	return &LitElem{
		Loc:  loc,
		Key:  key,
		Elem: elem,
	}
}

func (e *LitElem) IsKeyed() bool { return e.Key != nil }

type ElemList struct {
	Elem  []*LitElem
	Keyed bool
}

func NewElemList(list []*LitElem) *ElemList {
	if len(list) == 0 { // handle empty list
		return &ElemList{
			Elem:  list,
			Keyed: false,
		}
	}
	keyed := list[0].IsKeyed() // check if there are mixed keyed and unkeyed elements
	for _, v := range list {
		if (keyed && !v.IsKeyed()) || (!keyed && v.IsKeyed()) {
			panic(fmt.Errorf("%s mixed keyed and unkeyed elements", v.Loc.ToString()))
		}
	}
	return &ElemList{
		Elem:  list,
		Keyed: keyed,
	}
}

type CompLit struct {
	BaseLiteralExpr
	Elem  []*LitElem
	Keyed bool
}

func NewCompLit(loc *Loc, tp IType, elem *ElemList) *CompLit {
	l := &CompLit{
		BaseLiteralExpr: *NewBaseLiteralExpr(loc),
		Elem:            elem.Elem,
		Keyed:           elem.Keyed,
	}
	l.Type = tp // type is assigned when constructed, but its validity needs check
	return l
}

func (l *CompLit) GetValue() interface{} { return l.Elem }

// Constant expressions (can be assigned to constant, special case of literal expression)
type ConstExpr struct {
	BaseLiteralExpr
	Val interface{}
}

func (e *ConstExpr) GetValue() interface{} { return e.Val }

func NewIntConst(loc *Loc, val int) *ConstExpr {
	e := &ConstExpr{BaseLiteralExpr: *NewBaseLiteralExpr(loc), Val: val}
	e.Type = NewPrimType(loc, Int)
	return e
}

func NewFloatConst(loc *Loc, val float64) *ConstExpr {
	e := &ConstExpr{BaseLiteralExpr: *NewBaseLiteralExpr(loc), Val: val}
	e.Type = NewPrimType(loc, Float64)
	return e
}

func NewBoolConst(loc *Loc, val bool) *ConstExpr {
	e := &ConstExpr{BaseLiteralExpr: *NewBaseLiteralExpr(loc), Val: val}
	e.Type = NewPrimType(loc, Bool)
	return e
}

func (e *ConstExpr) ToStringTree() string {
	switch e.Type.GetTypeEnum() {
	case Int:
		return strconv.FormatInt(int64(e.Val.(int)), 10)
	case Float64:
		return strconv.FormatFloat(e.Val.(float64), 'g', -1, 64)
	case Bool:
		return strconv.FormatBool(e.Val.(bool))
	default:
		return ""
	}
}

// Zero value is the internal representation of initial value of any type.
// It cannot be declared as a literal like nil.
type ZeroValue struct {
	BaseExprNode
}

func NewZeroValue(loc *Loc) *ZeroValue {
	z := &ZeroValue{}
	z.Type = NewNilType(loc)
	return z
}

func (c *ZeroValue) ToStringTree() string { return "0" }

func (c *ZeroValue) GetValue() interface{} { return nil }

func (c *ZeroValue) IsLValue() bool { return false }

// Identifier expression
type IdExpr struct {
	BaseExprNode
	Name     string // should keep name for lookup in global scope
	Symbol   *TableEntry
	Captured bool
}

var IsKeyword = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

func NewIdExpr(loc *Loc, name string, symbol *TableEntry) *IdExpr {
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

func NewFuncCallExpr(loc *Loc, fun IExprNode, args []IExprNode) *FuncCallExpr {
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

func NewUnaryExpr(loc *Loc, op UnaryOp, expr IExprNode) *UnaryExpr {
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

func NewBinaryExpr(loc *Loc, op BinaryOp, left, right IExprNode) *BinaryExpr {
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
				if r.Val.(int) < 0 {
					panic(fmt.Errorf("%s shift amount cannot be negative", r.LocStr()))
				}
				return NewIntConst(l.Loc, l.Val.(int)<<uint(r.Val.(int)))
			},
		},
	},
	RSH: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				if r.Val.(int) < 0 {
					panic(fmt.Errorf("%s shift amount cannot be negative", r.LocStr()))
				}
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
