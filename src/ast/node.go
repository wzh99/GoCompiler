package ast

type IASTNode interface {
	GetLocation() *Location
	LocationStr() string
	ToStringTree() string
}

type BaseASTNode struct {
	loc *Location
}

func NewBaseASTNode(loc *Location) *BaseASTNode {
	return &BaseASTNode{loc}
}

func (n *BaseASTNode) GetLocation() *Location { return n.loc }

func (n *BaseASTNode) LocationStr() string { return n.loc.ToString() }

func (n *BaseASTNode) ToStringTree() string { return "()" }

type ProgramNode struct {
	BaseASTNode
	pkg    string
	global *FuncDecl //
	funcs  []*FuncDecl
}

func NewProgramNode(pkgName string) *ProgramNode {
	n := &ProgramNode{
		BaseASTNode: *NewBaseASTNode(nil), pkg: pkgName,
		global: NewFuncDecl(nil, "_global", NewFunctionType([]IType{}, []IType{}),
			NewGlobalScope(), nil),
		funcs: make([]*FuncDecl, 0),
	}
	return n
}

func (n *ProgramNode) AddFuncDecl(fun *FuncDecl) {
	n.funcs = append(n.funcs, fun)
}
