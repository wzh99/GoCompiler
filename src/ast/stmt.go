package ast

import "fmt"

type IStmtNode interface {
	IASTNode
}

type BaseStmtNode struct {
	BaseASTNode
}

func NewBaseStmtNode(loc *Loc) *BaseStmtNode {
	return &BaseStmtNode{BaseASTNode: *NewBaseASTNode(loc)}
}

type BlockStmt struct {
	BaseASTNode
	Scope *Scope
	Stmts []IStmtNode
	// Optional, track the statement that creates this block
	Ctrl IStmtNode
}

func NewBlockStmt(loc *Loc, scope *Scope) *BlockStmt {
	return &BlockStmt{BaseASTNode: *NewBaseASTNode(loc), Scope: scope}
}

func (s *BlockStmt) AddStmt(stmt IStmtNode) {
	s.Stmts = append(s.Stmts, stmt)
}

type AssignStmt struct {
	BaseASTNode
	Lhs, Rhs []IExprNode
	// Set this to true if it initialize variables
	Init bool
}

func NewAssignStmt(loc *Loc, lhs, rhs []IExprNode) *AssignStmt {
	return &AssignStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		Lhs:         lhs,
		Rhs:         rhs,
		Init:        false,
	}
}

func NewInitStmt(loc *Loc, lhs, rhs []IExprNode) *AssignStmt {
	return &AssignStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		Lhs:         lhs,
		Rhs:         rhs,
		Init:        true,
	}
}

type IncDecStmt struct {
	BaseASTNode
	Expr IExprNode
	Inc  bool // true: ++, false: --
}

func NewIncDecStmt(loc *Loc, expr IExprNode, inc bool) *IncDecStmt {
	return &IncDecStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		Expr:        expr,
		Inc:         inc,
	}
}

type ReturnStmt struct {
	BaseASTNode
	Expr []IExprNode
	Func *FuncDecl
}

func NewReturnStmt(loc *Loc, expr []IExprNode, decl *FuncDecl) *ReturnStmt {
	return &ReturnStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		Expr:        expr,
		Func:        decl,
	}
}

type ForClause struct {
	init IStmtNode
	cond IExprNode
	post IStmtNode
}

type ForClauseStmt struct {
	BaseASTNode
	Init  IStmtNode
	Cond  IExprNode
	Post  IStmtNode
	Block *BlockStmt
}

func NewForClauseStmt(loc *Loc, init IStmtNode, cond IExprNode, post IStmtNode,
	block *BlockStmt) *ForClauseStmt {
	s := &ForClauseStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		Init:        init,
		Cond:        cond,
		Post:        post,
		Block:       block,
	}
	block.Ctrl = s
	return s
}

type BreakStmt struct {
	BaseASTNode
	Target IStmtNode
}

func NewBreakStmt(loc *Loc, target IStmtNode) *BreakStmt {
	return &BreakStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		Target:      target,
	}
}

func (s *BreakStmt) ToStringTree() string {
	return fmt.Sprintf("(break %s)", s.Target.GetLoc().ToString())
}

type ContinueStmt struct {
	BaseASTNode
	Target IStmtNode
}

func NewContinueStmt(loc *Loc, target IStmtNode) *ContinueStmt {
	return &ContinueStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		Target:      target,
	}
}

func (s *ContinueStmt) ToStringTree() string {
	return fmt.Sprintf("(continue %s)", s.Target.GetLoc().ToString())
}

type IfStmt struct {
	BaseASTNode
	Init  IStmtNode // optional
	Cond  IExprNode
	Block *BlockStmt
	Else  IStmtNode // optional
}

func NewIfStmt(loc *Loc, init IStmtNode, cond IExprNode, block *BlockStmt,
	els IStmtNode) *IfStmt {
	return &IfStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		Init:        init,
		Cond:        cond,
		Block:       block,
		Else:        els,
	}
}
