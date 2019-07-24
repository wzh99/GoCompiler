package ast

type IStmtNode interface {
	IASTNode
}

type BaseStmtNode struct {
	BaseASTNode
}

func NewBaseStmtNode(loc *Location) *BaseStmtNode {
	return &BaseStmtNode{*NewBaseASTNode(loc)}
}
