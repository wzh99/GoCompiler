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

func NewProgramNode() *ProgramNode {
	return &ProgramNode{*newBaseASTNode(nil), make([]IDeclNode, 0)}
}

func (n *ProgramNode) ToStringTree() string {
	str := "(program"
	for _, decl := range n.decl {
		str += " " + decl.ToStringTree()
	}
	return str + ")"
}

func (n *ProgramNode) AddDecl(node IDeclNode) {
	n.decl = append(n.decl, node)
}

func (n *ProgramNode) GetDecl() []IDeclNode { return n.decl }
