package ast

type A int

type IDeclNode interface {
	IStmtNode
	GetName() string
	GetType() IType
}

type BaseDeclNode struct {
	BaseStmtNode
	name string
	tp   IType
}

func newBaseDeclNode(loc *Location, name string, tp IType) *BaseDeclNode {
	return &BaseDeclNode{BaseStmtNode: *newBaseStmtNode(loc), name: name, tp: tp}
}

func (n *BaseDeclNode) GetName() string { return n.name }

func (n *BaseDeclNode) GetType() IType { return n.tp }
