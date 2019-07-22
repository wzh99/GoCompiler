package ast

type IExprNode interface {
	IStmtNode
	GetType() IType
}

type BaseExprNode struct {
	BaseStmtNode
	tp IType
}

func newBaseExprNode(loc *Location) *BaseExprNode {
	return &BaseExprNode{BaseStmtNode: *newBaseStmtNode(loc)}
}
