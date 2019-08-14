package ast

import "fmt"

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
	// Optional, track the breakable statement that creates this block
	breakable IStmtNode
}

func NewBlockStmt(loc *Location, scope *Scope) *BlockStmt {
	return &BlockStmt{BaseASTNode: *NewBaseASTNode(loc), scope: scope}
}

func (s *BlockStmt) AddStmt(stmt IStmtNode) {
	s.stmts = append(s.stmts, stmt)
}

func (s *BlockStmt) ToStringTree() string {
	str := "(block"
	for _, s := range s.stmts {
		str += s.ToStringTree()
	}
	return str + ")"
}

type AssignStmt struct {
	BaseASTNode
	lhs, rhs []IExprNode
	// Set this to true if it initialize variables
	init bool
}

func NewAssignStmt(loc *Location, lhs, rhs []IExprNode) *AssignStmt {
	return &AssignStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		lhs:         lhs,
		rhs:         rhs,
		init:        false,
	}
}

func NewInitStmt(loc *Location, lhs, rhs []IExprNode) *AssignStmt {
	return &AssignStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		lhs:         lhs,
		rhs:         rhs,
		init:        true,
	}
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

type IncDecStmt struct {
	BaseASTNode
	expr IExprNode
	inc  bool
}

func NewIncDecStmt(loc *Location, expr IExprNode, inc bool) *IncDecStmt {
	return &IncDecStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		expr:        expr,
		inc:         inc,
	}
}

func (s *IncDecStmt) ToStringTree() string {
	var op string
	if s.inc {
		op = "++"
	} else {
		op = "--"
	}
	return fmt.Sprintf("(%s %s)", op, s.expr.ToStringTree())
}

type ReturnStmt struct {
	BaseASTNode
	expr []IExprNode
}

func NewReturnStmt(loc *Location, expr []IExprNode) *ReturnStmt {
	return &ReturnStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		expr:        expr,
	}
}

func (s *ReturnStmt) ToStringTree() string {
	str := "(return"
	for _, expr := range s.expr {
		str += " " + expr.ToStringTree()
	}
	return str + ")"
}

type ForClause struct {
	init IStmtNode
	cond IExprNode
	post IStmtNode
}

type ForClauseStmt struct {
	BaseASTNode
	init  IStmtNode
	cond  IExprNode
	post  IStmtNode
	block *BlockStmt
}

func NewForClauseStmt(loc *Location, init IStmtNode, cond IExprNode, post IStmtNode,
	block *BlockStmt) *ForClauseStmt {
	s := &ForClauseStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		init:        init,
		cond:        cond,
		post:        post,
		block:       block,
	}
	block.breakable = s
	return s
}

func (s *ForClauseStmt) ToStringTree() string {
	init := ""
	if s.init != nil {
		init = s.init.ToStringTree()
	}
	post := ""
	if s.post != nil {
		post = s.post.ToStringTree()
	}
	return fmt.Sprintf("(for %s %s %s %s)", init, s.cond.ToStringTree(), post,
		s.block.ToStringTree())
}

type BreakStmt struct {
	BaseASTNode
	target IStmtNode
}

func NewBreakStmt(loc *Location, target IStmtNode) *BreakStmt {
	return &BreakStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		target:      target,
	}
}

func (s *BreakStmt) ToStringTree() string {
	return fmt.Sprintf("(break %s)", s.target.LocationStr())
}
