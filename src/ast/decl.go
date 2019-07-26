package ast

type IDeclNode interface {
	IASTNode
	GetType() IType
}

// Function declarations are only allowed in global scope
// In local scope, functions are defined as lambda literals
type FuncDecl struct {
	BaseASTNode
	name  string
	tp    *FuncType
	scope *Scope
	stmts []IStmtNode
}

func NewFuncDecl(loc *Location, name string, tp *FuncType, scope *Scope, stmt []IStmtNode) *FuncDecl {
	return &FuncDecl{BaseASTNode: *NewBaseASTNode(loc), name: name, tp: tp, scope: scope, stmts: stmt}
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
