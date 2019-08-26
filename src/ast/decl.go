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
	NamedRet []*Symbol // for named return values
	Lit      *FuncLit  // point back to the literal that defines this function, if exists
}

func NewFuncDecl(loc *Loc, name string, tp *FuncType, scope *Scope,
	namedRet []*Symbol) *FuncDecl {
	d := &FuncDecl{BaseASTNode: *NewBaseASTNode(loc), Name: name, Type: tp, Scope: scope,
		Stmts: make([]IStmtNode, 0), NamedRet: namedRet}
	d.Scope.Func = d
	return d
}

func (d *FuncDecl) GetType() IType { return d.Type }

func (d *FuncDecl) AddStmt(stmt IStmtNode) {
	d.Stmts = append(d.Stmts, stmt)
}

func (d *FuncDecl) GenSymbol() *Symbol {
	return NewSymbol(d.GetLoc(), d.Name, FuncEntry, d.Type, nil)
}

// An intermediate structure when building AST
type FuncSignature struct {
	params, results []*Symbol
}
