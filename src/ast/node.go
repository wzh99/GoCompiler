package ast

type IASTNode interface {
	GetLocation() *Location
	ToStringTree() string
}

type BaseASTNode struct {
	loc *Location
}

func newBaseASTNode(loc *Location) *BaseASTNode {
	return &BaseASTNode{loc}
}

func (n *BaseASTNode) GetLocation() *Location { return n.loc }

func (n *BaseASTNode) ToStringTree() string { return "()" }

type ProgramNode struct {
	BaseASTNode
	decl []IDeclNode
}

func NewProgramNode(decl []IDeclNode) *ProgramNode {
	return &ProgramNode{BaseASTNode: *newBaseASTNode(nil), decl: decl}
}

func (n *ProgramNode) ToStringTree() string {
	str := "(program"
	for _, decl := range n.decl {
		str += " " + decl.ToStringTree()
	}
	return str + ")"
}

func (n *ProgramNode) GetDecl() []IDeclNode { return n.decl }
