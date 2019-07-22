package ast

type IExprNode interface {
	IStmtNode
	GetType() IType
}

type BaseExprNode struct {
	BaseStmtNode
	// tp is nil: type of current node is unknown, to be determined in a later pass.
	// tp is Unresolved: type name of current node is known, but its validity remains to be verified.
	tp IType
}

func newBaseExprNode(loc *Location, tp IType) *BaseExprNode {
	return &BaseExprNode{BaseStmtNode: *newBaseStmtNode(loc), tp: tp}
}
