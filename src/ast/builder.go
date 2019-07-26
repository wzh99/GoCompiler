package ast

import (
	"fmt"
	. "parser"
)

// Build AST from CST. build scopes with symbol tables, and perform basic checks.
type ASTBuilder struct {
	BaseGolangVisitor
	global, cur *Scope
}

func NewASTBuilder() *ASTBuilder { return &ASTBuilder{} }

func (v *ASTBuilder) VisitSourceFile(ctx *SourceFileContext) interface{} {
	// Get package name
	pkgName := v.VisitPackageClause(ctx.PackageClause().(*PackageClauseContext)).(string)
	// Initialize program node
	prog := NewProgramNode(pkgName)
	// Set global scope pointer
	v.global = prog.scope

	// Add declarations to program node
	for _, d := range ctx.AllTopLevelDecl() {
		switch decl := v.VisitTopLevelDecl(d.(*TopLevelDeclContext)); decl.(type) {
		// Constants, variables and types are stored only in symbol tables
		case []*SymbolEntry:

		// Functions are stored both in symbol table of global scope and in program node
		case []*FuncDecl:

		}
	}

	return prog
}

func (v *ASTBuilder) VisitPackageClause(ctx *PackageClauseContext) interface{} {
	return ctx.IDENTIFIER().GetText() // string
}

func (v *ASTBuilder) VisitTopLevelDecl(ctx *TopLevelDeclContext) interface{} {
	if d := ctx.Declaration(); d != nil {
		return v.VisitDeclaration(d.(*DeclarationContext)) // []*SymbolEntry
	} else if d := ctx.FunctionDecl(); d != nil {
		return v.VisitFunctionDecl(d.(*FunctionDeclContext)) // []*FuncDecl
	}
	return nil
}

// Constants, variables and type declarations
func (v *ASTBuilder) VisitDeclaration(ctx *DeclarationContext) interface{} /* []*SymbolEntry */ {
	if d := ctx.ConstDecl(); d != nil {
		return v.VisitConstDecl(d.(*ConstDeclContext))
	} else if d := ctx.VarDecl(); d != nil {
		return v.VisitVarDecl(d.(*VarDeclContext))
	} else if d := ctx.TypeDecl(); d != nil {
		return v.VisitTypeDecl(d.(*TypeDeclContext))
	}
	return nil
}

// A specific constant declaration scope: const (... = ..., ... = ...)
func (v *ASTBuilder) VisitConstDecl(ctx *ConstDeclContext) interface{} {
	declList := make([]*SymbolEntry, 0)
	for _, spec := range ctx.AllConstSpec() {
		declList = append(declList, v.VisitConstSpec(spec.(*ConstSpecContext)).([]*SymbolEntry)...)
	}
	return declList // []*SymbolEntry
}

// One line of constant specification: id_1, id_2, ..., id_n = expr_1, expr_2, ..., expr_n
func (v *ASTBuilder) VisitConstSpec(ctx *ConstSpecContext) interface{} {
	// Get expression list on both sides of equation
	lhs := v.VisitIdentifierList(ctx.IdentifierList().(*IdentifierListContext)).([]*IdExpr)
	rhs := v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)

	// Validate expressions
	// Ensure same length
	if len(lhs) != len(rhs) {
		panic(fmt.Sprintf("%s Assignment count mismatch: #%d = #%d",
			NewLocationFromContext(ctx).ToString(), len(lhs), len(rhs)))
	}
	// lhs symbols haven't been defined before
	for _, id := range lhs {
		if v.cur.CheckRedefined(id.name) { // redefined symbol
			panic(fmt.Sprintf("%s Redefined symbol: %s", id.LocationStr(), id.name))
		}
	}
	// rhs symbols are all constant expressions
	for _, expr := range rhs {
		if _, ok := expr.(IConstExpr); !ok {
			panic(fmt.Sprintf("%s Not constant expression", expr.LocationStr()))
		}
	}

	// Build constant specification list
	specList := make([]*SymbolEntry, 0)
	for i := range lhs {
		specList = append(specList, NewSymbolEntry(lhs[i].GetLocation(), lhs[i].name,
			ConstEntry, rhs[i].GetType(), rhs[i].(IConstExpr).GetValue()))
	}

	return specList // []*SymbolEntry
}
