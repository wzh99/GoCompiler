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
	// Set global and current scope pointer
	v.global = prog.scope
	v.cur = prog.scope

	// Add declarations to program node
	for _, d := range ctx.AllTopLevelDecl() {
		switch decl := v.VisitTopLevelDecl(d.(*TopLevelDeclContext)); decl.(type) {
		case []*SymbolEntry: // several entities may appear in one top level declaration
			// Constants, variables and types are stored only in symbol tables
			for _, symbol := range decl.([]*SymbolEntry) {
				v.cur.AddSymbol(symbol)
			}

		case *FuncDecl: // one function per declaration
			// Functions are stored both in symbol table of global scope and in program node
			fun := decl.(*FuncDecl)
			v.cur.AddSymbol(fun.GenSymbol())
			prog.AddFuncDecl(fun)
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
		return v.VisitFunctionDecl(d.(*FunctionDeclContext)) // *FuncDecl
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

func (v *ASTBuilder) VisitIdentifierList(ctx *IdentifierListContext) interface{} {
	idList := make([]*IdExpr, 0)
	for _, id := range ctx.AllIDENTIFIER() {
		idList = append(idList, NewIdExpr(NewLocationFromToken(id.GetSymbol()), id.GetText(), nil))
	}
	return idList // []*IdExpr
}

func (v *ASTBuilder) VisitExpressionList(ctx *ExpressionListContext) interface{} {
	exprList := make([]IExprNode, 0)
	for _, expr := range ctx.AllExpression() {
		exprList = append(exprList, v.VisitExpression(expr.(*ExpressionContext)).(IExprNode))
	}
	return exprList // []IExprNode
}

func (v *ASTBuilder) VisitTypeDecl(ctx *TypeDeclContext) interface{} {
	typeList := make([]*SymbolEntry, 0)
	for _, spec := range ctx.AllTypeSpec() {
		typeList = append(typeList, v.VisitTypeSpec(spec.(*TypeSpecContext)).(*SymbolEntry))
	}
	return typeList // []*SymbolEntry
}

func (v *ASTBuilder) VisitTypeSpec(ctx *TypeSpecContext) interface{} {
	name := ctx.IDENTIFIER().GetText()
	tp := v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	alias := NewAliasType(name, tp)
	return NewSymbolEntry(NewLocationFromContext(ctx), name, TypeEntry, alias, nil) // *SymbolEntry
}

// Top level function declarations
func (v *ASTBuilder) VisitFunctionDecl(ctx *FunctionDeclContext) interface{} {
	name := ctx.IDENTIFIER().GetText()
	funcDecl := v.VisitFunction(ctx.Function().(*FunctionContext)).(*FuncDecl)
	funcDecl.name = name
	return funcDecl // *FuncDecl
}

func (v *ASTBuilder) VisitFunction(ctx *FunctionContext) interface{} {
	// Analyze function signature
	sig := v.VisitSignature(ctx.Signature().(*SignatureContext)).(*FuncSignature)
	paramType := make([]IType, 0)
	resultType := make([]IType, 0)
	namedRet := make([]*SymbolEntry, 0)
	for _, p := range sig.params {
		paramType = append(paramType, p.tp)
	}
	for _, r := range sig.results {
		resultType = append(resultType, r.tp)
		if len(r.name) != 0 {
			namedRet = append(namedRet, r)
		}
	}

	// Panic if mixed named and unnamed parameters
	if len(namedRet) > 0 && len(namedRet) != len(resultType) {
		panic(fmt.Sprintf("%s Function has both named and unnamed parameters.",
			NewLocationFromContext(ctx).ToString()))
	}

	// Initialize function declaration node
	decl := NewFuncDecl(NewLocationFromContext(ctx), "", NewFunctionType(paramType, resultType),
		NewLocalScope(v.cur), namedRet)
	v.cur = decl.scope // move scope cursor deeper

	// Add parameter and named return value symbols to the function scope
	for _, p := range sig.params {
		decl.scope.AddSymbol(p)
	}
	for _, r := range namedRet {
		decl.scope.AddSymbol(r)
	}

	// Build statement AST nodes
	decl.stmts = v.VisitBlock(ctx.Block().(*BlockContext)).([]IStmtNode)
	v.cur = v.cur.parent // move cursor back

	return decl // *FuncDecl
}
