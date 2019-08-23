package ast

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
	Loc    *Loc
	Key    IExprNode
	Val    IExprNode
	Symbol *Symbol // record corresponding struct symbol table entry
}

func NewLitElem(loc *Loc, key IExprNode, val IExprNode) *LitElem {
	return &LitElem{
		Loc: loc,
		Key: key,
		Val: val,
	}
}

func (e *LitElem) IsKeyed() bool { return e.Key != nil }

type ElemList struct {
	Elem  []*LitElem
	Keyed bool
}

func NewElemList(list []*LitElem, keyed bool) *ElemList {
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
	Symbol   *Symbol
	Captured bool
}

var IsKeyword = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

func NewIdExpr(loc *Loc, name string, symbol *Symbol) *IdExpr {
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
	return &FuncCallExpr{
		BaseExprNode: *NewBaseExprNode(loc),
		Func:         fun,
		Args:         args,
	}
}

func (e *FuncCallExpr) IsLValue() bool { return false }

type SelectExpr struct {
	BaseExprNode
	Target IExprNode
	Member *IdExpr
}

func NewSelectExpr(loc *Loc, target IExprNode, member *IdExpr) *SelectExpr {
	return &SelectExpr{
		BaseExprNode: *NewBaseExprNode(loc),
		Target:       target,
		Member:       member,
	}
}

func (e *SelectExpr) IsLValue() bool { return true }

type IndexExpr struct {
	BaseExprNode
	Array IExprNode // array type and slice type are all acceptable
	Index IExprNode
}

func NewIndexExpr(loc *Loc, array IExprNode, index IExprNode) *IndexExpr {
	return &IndexExpr{
		BaseExprNode: *NewBaseExprNode(loc),
		Array:        array,
		Index:        index,
	}
}

func (e *IndexExpr) IsLValue() bool { return true } // a[0] = c

type UnaryExpr struct {
	BaseExprNode
	Op   UnaryOp
	Expr IExprNode
}

type UnaryOp int

const (
	POS UnaryOp = 1 << iota
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

func (e *UnaryExpr) IsLValue() bool {
	if e.Op == DEREF {
		return true // *a = v
	} else {
		return false
	}
}

type BinaryOp int

const (
	MUL BinaryOp = 1 << iota
	DIV
	MOD
	SHL // left shift
	SHR
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
	MUL: "*", DIV: "/", MOD: "%", SHL: "<<", SHR: ">>", AAND: "&",
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
	SHL: {
		Int: {
			Int: func(l, r *ConstExpr) *ConstExpr {
				return NewIntConst(l.Loc, l.Val.(int)<<uint(r.Val.(int)))
			},
		},
	},
	SHR: {
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
