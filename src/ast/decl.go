package ast

type IDeclNode interface {
	IASTNode
	GetType() IType
}

// This struct is for both top level function declaration and part of function literals
// In global scope, Only function declarations are translated to AST nodes.
type FuncDecl struct {
	BaseASTNode
	name     string
	tp       *FuncType
	scope    *Scope
	stmts    []IStmtNode
	namedRet []*SymbolEntry // for named return values
}

func NewFuncDecl(loc *Location, name string, tp *FuncType, scope *Scope,
	namedRet []*SymbolEntry) *FuncDecl {
	return &FuncDecl{BaseASTNode: *NewBaseASTNode(loc), name: name, tp: tp, scope: scope,
		stmts: make([]IStmtNode, 0), namedRet: namedRet}
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

func (d *FuncDecl) GenSymbol() *SymbolEntry {
	return NewSymbolEntry(d.GetLocation(), d.name, FuncEntry, d.tp, nil)
}

// An intermediate structure when building AST
type FuncSignature struct {
	params, results []*SymbolEntry
}
