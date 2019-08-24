package ast

type IASTNode interface {
	GetLoc() *Loc
}

type BaseASTNode struct {
	Loc *Loc
}

func NewBaseASTNode(loc *Loc) *BaseASTNode {
	return &BaseASTNode{loc}
}

func (n *BaseASTNode) GetLoc() *Loc { return n.Loc }

type ProgramNode struct {
	BaseASTNode
	PkgName string
	Global  *FuncDecl
	Funcs   []*FuncDecl
}

func NewProgramNode(pkgName string) *ProgramNode {
	n := &ProgramNode{
		BaseASTNode: *NewBaseASTNode(nil), PkgName: pkgName,
		Global: NewFuncDecl(nil, "_global",
			NewFunctionType(&Loc{line: 0, col: 0}, []IType{}, []IType{}),
			NewGlobalScope(), nil),
		Funcs: make([]*FuncDecl, 0),
	}
	return n
}

func (n *ProgramNode) AddFuncDecl(fun *FuncDecl) {
	n.Funcs = append(n.Funcs, fun)
}
