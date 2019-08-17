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
	// Optional, track the breakable statement that creates this block
	Ctrl IStmtNode
}

func NewBlockStmt(loc *Loc, scope *Scope) *BlockStmt {
	return &BlockStmt{BaseASTNode: *NewBaseASTNode(loc), Scope: scope}
}

func (s *BlockStmt) AddStmt(stmt IStmtNode) {
	s.Stmts = append(s.Stmts, stmt)
}

func (s *BlockStmt) ToStringTree() string {
	str := "(block"
	for _, s := range s.Stmts {
		str += s.ToStringTree()
	}
	return str + ")"
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

func (s *AssignStmt) ToStringTree() string {
	str := "(= ("
	for i, e := range s.Lhs {
		if i != 0 {
			str += " "
		}
		str += e.ToStringTree()
	}
	str += ") ("
	for i, e := range s.Rhs {
		if i != 0 {
			str += " "
		}
		str += e.ToStringTree()
	}
	return str + ")"
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

func (s *IncDecStmt) ToStringTree() string {
	var op string
	if s.Inc {
		op = "++"
	} else {
		op = "--"
	}
	return fmt.Sprintf("(%s %s)", op, s.Expr.ToStringTree())
}

type ReturnStmt struct {
	BaseASTNode
	Expr []IExprNode
}

func NewReturnStmt(loc *Loc, expr []IExprNode) *ReturnStmt {
	return &ReturnStmt{
		BaseASTNode: *NewBaseASTNode(loc),
		Expr:        expr,
	}
}

func (s *ReturnStmt) ToStringTree() string {
	str := "(return"
	for _, expr := range s.Expr {
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

func (s *ForClauseStmt) ToStringTree() string {
	init := ""
	if s.Init != nil {
		init = s.Init.ToStringTree()
	}
	post := ""
	if s.Post != nil {
		post = s.Post.ToStringTree()
	}
	return fmt.Sprintf("(for %s %s %s %s)", init, s.Cond.ToStringTree(), post,
		s.Block.ToStringTree())
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
	return fmt.Sprintf("(break %s)", s.Target.LocStr())
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
	return fmt.Sprintf("(continue %s)", s.Target.LocStr())
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

func (s *IfStmt) ToStringTree() string {
	init := ""
	if s.Init != nil {
		init = s.Init.ToStringTree()
	}
	els := ""
	if s.Else != nil {
		els = s.Else.ToStringTree()
	}
	return fmt.Sprintf("(if %s %s %s)", init, s.Cond.ToStringTree(), els)
}
