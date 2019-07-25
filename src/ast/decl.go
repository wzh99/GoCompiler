package ast

type IDeclNode interface {
	IASTNode
	GetType() IType
}

// Only function declarations need to be created as nodes, others are in symbol table.
type FuncDecl struct {
	BaseASTNode
	tp    *FuncType
	scope *Scope
	stmts []IStmtNode
}

func NewFuncDecl(loc *Location, tp *FuncType, scope *Scope, stmt []IStmtNode) *FuncDecl {
	return &FuncDecl{BaseASTNode: *NewBaseASTNode(loc), tp: tp, scope: scope, stmts: stmt}
}

func (d *FuncDecl) GetType() IType { return d.tp }

func (d *FuncDecl) AddStmt(stmt IStmtNode) {
	d.stmts = append(d.stmts, stmt)
}

func (d *FuncDecl) ToStringTree() string {
	str := "(funcDecl"
	for _, s := range d.stmts {
		str += " " + s.ToStringTree()
	}
	return str + ")"
}
