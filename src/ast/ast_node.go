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
