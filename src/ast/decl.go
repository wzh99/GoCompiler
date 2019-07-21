package ast

type IDeclNode interface {
	IASTNode
	GetName() string
}

type BaseDeclNode struct {
	BaseASTNode
	name string
}

func newBaseDeclNode(loc *Location, name string) *BaseDeclNode {
	return &BaseDeclNode{*newBaseASTNode(loc), name}
}

func (n *BaseDeclNode) GetName() string { return n.name }
