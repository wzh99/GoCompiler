package ast

import (
	"fmt"
	. "parser"
)

// Build AST from CST. build scopes with symbol tables, and perform basic checks.
type ASTBuilder struct {
	BaseGolangVisitor
	global, cur *Scope
	prog        *ProgramNode
	// Track receiver of the method next child scope belongs to
	// Since methods and functions share the same function body AST construction procedure,
	// the procedure itself has no idea whether it has receiver.
	receiver *SymbolEntry
}

func NewASTBuilder() *ASTBuilder { return &ASTBuilder{} }

func (v *ASTBuilder) VisitSourceFile(ctx *SourceFileContext) interface{} {
	// Get package name
	pkgName := v.VisitPackageClause(ctx.PackageClause().(*PackageClauseContext)).(string)
	// Initialize program node
	v.prog = NewProgramNode(pkgName)
	// Set global and current scope pointer
	v.global = v.prog.global.scope
	v.cur = v.global
	// Point scope back to the function it belongs to
	v.prog.global.scope.fun = v.prog.global

	// Add declarations to program node
	for _, decl := range ctx.AllTopLevelDecl() {
		v.VisitTopLevelDecl(decl.(*TopLevelDeclContext))
	}

	return v.prog
}

func (v *ASTBuilder) VisitPackageClause(ctx *PackageClauseContext) interface{} {
	return ctx.IDENTIFIER().GetText() // string
}

func (v *ASTBuilder) VisitTopLevelDecl(ctx *TopLevelDeclContext) interface{} {
	if d := ctx.Declaration(); d != nil {
		v.VisitDeclaration(d.(*DeclarationContext))
	} else if d := ctx.FunctionDecl(); d != nil {
		v.VisitFunctionDecl(d.(*FunctionDeclContext))
	} else if d := ctx.MethodDecl(); d != nil {
		v.VisitMethodDecl(d.(*MethodDeclContext))
	}
	return nil
}

// Constants, variables and type declarations
func (v *ASTBuilder) VisitDeclaration(ctx *DeclarationContext) interface{} {
	if d := ctx.ConstDecl(); d != nil {
		v.VisitConstDecl(d.(*ConstDeclContext))
	} else if d := ctx.VarDecl(); d != nil {
		v.VisitVarDecl(d.(*VarDeclContext))
	} else if d := ctx.TypeDecl(); d != nil {
		v.VisitTypeDecl(d.(*TypeDeclContext))
	}
	return nil // []*SymbolEntry
}

// A specific constant declaration scope: const (... = ..., ... = ...)
func (v *ASTBuilder) VisitConstDecl(ctx *ConstDeclContext) interface{} {
	for _, spec := range ctx.AllConstSpec() {
		v.VisitConstSpec(spec.(*ConstSpecContext))
	}
	return nil
}

// One line of constant specification: id_1, id_2, ..., id_n = expr_1, expr_2, ..., expr_n
func (v *ASTBuilder) VisitConstSpec(ctx *ConstSpecContext) interface{} {
	// Get expression list on both sides of equation
	lhs := v.VisitIdentifierList(ctx.IdentifierList().(*IdentifierListContext)).([]*IdExpr)
	rhs := v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)

	// Validate expressions
	// Ensure same length
	if len(lhs) != len(rhs) {
		panic(fmt.Errorf("%s assignment count mismatch: #%d = #%d",
			NewLocationFromContext(ctx).ToString(), len(lhs), len(rhs)))
	}
	// lhs symbols haven't been defined before
	for _, id := range lhs {
		if v.cur.CheckDefined(id.name) { // redefined symbol
			panic(fmt.Errorf("%s redefined symbol: %s", id.LocationStr(), id.name))
		}
	}
	// rhs symbols are all constant expressions
	for _, expr := range rhs {
		if _, ok := expr.(IConstExpr); !ok {
			panic(fmt.Errorf("%s not constant expression", expr.LocationStr()))
		}
	}

	// Add constant specification
	var specTp IType
	if ctx.Tp() != nil {
		specTp = v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	}
	for i := range lhs {
		tp := rhs[i].GetType()
		val := rhs[i].(IConstExpr).GetValue()
		loc := lhs[i].GetLocation()

		// Check if need conversion in compile time
		if specTp != nil && !specTp.IsIdentical(tp) {
			var err error
			val, err = rhs[i].(IConstExpr).ConvertTo(specTp) // try to convert to target type
			if err != nil {
				panic(loc.ToString() + " " + err.Error()) // cannot convert
			}
			tp = specTp
		}

		// Add to symbol table of current scope
		v.cur.AddSymbol(NewSymbolEntry(loc, lhs[i].name, ConstEntry, tp, val))
	}

	return nil
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
	for _, spec := range ctx.AllTypeSpec() {
		v.VisitTypeSpec(spec.(*TypeSpecContext))
	}
	return nil
}

func (v *ASTBuilder) VisitTypeSpec(ctx *TypeSpecContext) interface{} {
	name := ctx.IDENTIFIER().GetText()
	tp := v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	alias := NewAliasType(name, tp)
	v.cur.AddSymbol(NewSymbolEntry(NewLocationFromContext(ctx), name, TypeEntry, alias, nil))
	return nil
}

// Top level function declarations
func (v *ASTBuilder) VisitFunctionDecl(ctx *FunctionDeclContext) interface{} {
	name := ctx.IDENTIFIER().GetText()
	funcDecl := v.VisitFunction(ctx.Function().(*FunctionContext)).(*FuncDecl)
	funcDecl.name = name
	v.global.AddSymbol(funcDecl.GenSymbol())
	v.prog.AddFuncDecl(funcDecl)
	return nil
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
		panic(fmt.Errorf("%s function has both named and unnamed parameters",
			NewLocationFromContext(ctx).ToString()))
	}

	// Initialize function declaration node
	decl := NewFuncDecl(NewLocationFromContext(ctx), "", NewFunctionType(paramType, resultType),
		NewLocalScope(v.cur), namedRet)
	decl.scope.fun = decl
	if v.receiver != nil { // add receiver to function type if it is a method
		decl.tp.receiver = v.receiver.tp
	}
	v.cur = decl.scope // move scope cursor deeper

	// Add parameter and named return value symbols to the function scope
	for _, p := range sig.params {
		decl.scope.AddSymbol(p)
	}
	for _, r := range namedRet {
		decl.scope.AddSymbol(r)
	}
	if v.receiver != nil { // add receiver to the method scope
		decl.scope.AddSymbol(v.receiver)
		v.receiver = nil // receiver should no longer be recorded
	}

	// Build statement AST nodes
	v.VisitBlock(ctx.Block().(*BlockContext)) // statements are added during visit
	v.cur = v.cur.parent                      // move cursor back

	return decl // *FuncDecl
}

func (v *ASTBuilder) VisitMethodDecl(ctx *MethodDeclContext) interface{} {
	name := ctx.IDENTIFIER().GetText()
	v.receiver = v.VisitReceiver(ctx.Receiver().(*ReceiverContext)).(*SymbolEntry) // register receiver
	funcDecl := v.VisitFunction(ctx.Function().(*FunctionContext)).(*FuncDecl)
	funcDecl.name = name
	v.global.AddSymbol(funcDecl.GenSymbol())
	v.prog.AddFuncDecl(funcDecl)
	return nil
}

func (v *ASTBuilder) VisitReceiver(ctx *ReceiverContext) interface{} {
	return v.VisitParameterDecl(ctx.ParameterDecl().(*ParameterDeclContext)) // *SymbolEntry
}

func (v *ASTBuilder) VisitVarDecl(ctx *VarDeclContext) interface{} {
	for _, decl := range ctx.AllVarSpec() {
		v.VisitVarSpec(decl.(*VarSpecContext))
	}
	return nil
}

func (v *ASTBuilder) VisitVarSpec(ctx *VarSpecContext) interface{} {
	// Get specified variables
	idList := v.VisitIdentifierList(ctx.IdentifierList().(*IdentifierListContext)).([]*IdExpr)

	// Get specified type, if exists
	var specType IType
	if ctx.Tp() != nil {
		specType = v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	}

	// Add variables to symbol table
	for _, id := range idList {
		// Check if defined in current scope before
		if v.cur.CheckDefined(id.name) {
			panic(fmt.Errorf("%s symbol redefined: %s", id.LocationStr(), id.name))
		}
		// type maybe unknown at this time
		v.cur.AddSymbol(NewSymbolEntry(id.GetLocation(), id.name, VarEntry, specType, nil))
	}

	// Get expression list
	var exprList []IExprNode
	if ctx.ExpressionList() != nil {
		exprList = v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)
	}
	if len(exprList) > 0 && len(exprList) != len(idList) {
		panic(fmt.Errorf("%s assignment count mismatch %d = %d",
			NewLocationFromContext(ctx).ToString(), len(idList), len(exprList)))
	}

	// Generate lhs list for statement node
	lhs := make([]IExprNode, 0)
	for _, id := range idList {
		id.symbol, _ = v.cur.Lookup(id.name) // must be in current scope
		lhs = append(lhs, id)
	}

	// Add statement to current scope
	v.cur.fun.AddStmt(NewAssignStmt(NewLocationFromContext(ctx), lhs, exprList))

	return nil
}
