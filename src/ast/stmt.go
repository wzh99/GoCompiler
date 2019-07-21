package ast

type IStmtNode struct {
	IASTNode
}

type BaseStmtNode struct {
	BaseASTNode
}

func newBaseStmtNode(loc *Location) *BaseStmtNode {
	return &BaseStmtNode{*newBaseASTNode(loc)}
}
