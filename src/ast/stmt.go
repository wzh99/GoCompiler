package ast

type IStmtNode interface {
	IASTNode
}

type BaseStmtNode struct {
	BaseASTNode
}

func newBaseStmtNode(loc *Location) *BaseStmtNode {
	return &BaseStmtNode{*newBaseASTNode(loc)}
}
