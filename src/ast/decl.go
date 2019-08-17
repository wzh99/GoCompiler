package ast

type IDeclNode interface {
	IASTNode
	GetType() IType
}

// This struct is for both top level function declaration and part of function literals
// In global scope, Only function declarations are translated to AST nodes.
type FuncDecl struct {
	BaseASTNode
	Name     string
	Type     *FuncType
	Scope    *Scope
	Stmts    []IStmtNode
	NamedRet []*TableEntry // for named return values
}

func NewFuncDecl(loc *Loc, name string, tp *FuncType, scope *Scope,
	namedRet []*TableEntry) *FuncDecl {
	d := &FuncDecl{BaseASTNode: *NewBaseASTNode(loc), Name: name, Type: tp, Scope: scope,
		Stmts: make([]IStmtNode, 0), NamedRet: namedRet}
	d.Scope.Func = d
	return d
}

func (d *FuncDecl) GetType() IType { return d.Type }

func (d *FuncDecl) AddStmt(stmt IStmtNode) {
	d.Stmts = append(d.Stmts, stmt)
}

func (d *FuncDecl) ToStringTree() string {
	str := "(" + d.Name
	for _, s := range d.Stmts {
		str += " " + s.ToStringTree()
	}
	return str + ")"
}

func (d *FuncDecl) GenSymbol() *TableEntry {
	return NewSymbolEntry(d.GetLoc(), d.Name, FuncEntry, d.Type, nil)
}

// An intermediate structure when building AST
type FuncSignature struct {
	params, results []*TableEntry
}
