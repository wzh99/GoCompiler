package ast

type IASTNode interface {
	GetLocation() *Location
	LocationStr() string
	ToStringTree() string
}

type BaseASTNode struct {
	Loc *Location
}

func NewBaseASTNode(loc *Location) *BaseASTNode {
	return &BaseASTNode{loc}
}

func (n *BaseASTNode) GetLocation() *Location { return n.Loc }

func (n *BaseASTNode) LocationStr() string { return n.Loc.ToString() }

func (n *BaseASTNode) ToStringTree() string { return "()" }

type ProgramNode struct {
	BaseASTNode
	PkgName string
	Global  *FuncDecl //
	Funcs   []*FuncDecl
}

func NewProgramNode(pkgName string) *ProgramNode {
	n := &ProgramNode{
		BaseASTNode: *NewBaseASTNode(nil), PkgName: pkgName,
		Global: NewFuncDecl(nil, "_global", NewFunctionType([]IType{}, []IType{}),
			NewGlobalScope(), nil),
		Funcs: make([]*FuncDecl, 0),
	}
	return n
}

func (n *ProgramNode) AddFuncDecl(fun *FuncDecl) {
	n.Funcs = append(n.Funcs, fun)
}
