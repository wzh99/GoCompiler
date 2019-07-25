package ast

import (
	"fmt"
)

type IStmtNode interface {
	IASTNode
}

type BaseStmtNode struct {
	BaseASTNode
}

func NewBaseStmtNode(loc *Location) *BaseStmtNode {
	return &BaseStmtNode{BaseASTNode: *NewBaseASTNode(loc)}
}

type BlockStmt struct {
	BaseASTNode
	scope *Scope
	stmts []IStmtNode
}

func NewBlockStmt(loc *Location, scope *Scope, stmts []IStmtNode) *BlockStmt {
	return &BlockStmt{BaseASTNode: *NewBaseASTNode(loc), scope: scope, stmts: stmts}
}

func (s *BlockStmt) ToStringTree() string {
	str := "(block"
	for _, s := range s.stmts {
		str += s.ToStringTree()
	}
	return str + ")"
}

type ExprStmt struct {
	BaseASTNode
	expr IExprNode
}

func NewExprStmt(loc *Location, expr IExprNode) *ExprStmt {
	return &ExprStmt{BaseASTNode: *NewBaseASTNode(loc), expr: expr}
}

func (s *ExprStmt) ToStringTree() string {
	return fmt.Sprintf("(exprStmt %s)", s.expr.ToStringTree())
}

type AssignStmt struct {
	BaseASTNode
	lhs, rhs []IExprNode
}

func NewAssignStmt(loc *Location, lhs, rhs []IExprNode) *AssignStmt {
	return &AssignStmt{BaseASTNode: *NewBaseASTNode(loc), lhs: lhs, rhs: rhs}
}

func (s *AssignStmt) ToStringTree() string {
	str := "(= ("
	for i, e := range s.lhs {
		if i != 0 {
			str += " "
		}
		str += e.ToStringTree()
	}
	str += ") ("
	for i, e := range s.rhs {
		if i != 0 {
			str += " "
		}
		str += e.ToStringTree()
	}
	return str + ")"
}

type ShortVarDeclStmt struct {
	BaseASTNode
	lhs, rhs []IExprNode
}

func NewShortVarDeclStmt(loc *Location, lhs, rhs []IExprNode) *ShortVarDeclStmt {
	return &ShortVarDeclStmt{BaseASTNode: *NewBaseASTNode(loc), lhs: lhs, rhs: rhs}
}

func (s *ShortVarDeclStmt) ToStringTree() string {
	str := "(:= ("
	for i, e := range s.lhs {
		if i != 0 {
			str += " "
		}
		str += e.ToStringTree()
	}
	str += ") ("
	for i, e := range s.rhs {
		if i != 0 {
			str += " "
		}
		str += e.ToStringTree()
	}
	return str + ")"
}
