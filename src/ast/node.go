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
	pkg   string
	scope *Scope // root of the scope tree
	funcs []*FuncDecl
}

func NewProgramNode(pkgName string) *ProgramNode {
	return &ProgramNode{BaseASTNode: *NewBaseASTNode(nil), pkg: pkgName, scope: NewGlobalScope(),
		funcs: make([]*FuncDecl, 0)}
}
