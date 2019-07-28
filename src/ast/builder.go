package ast

import (
	"fmt"
	. "parser"
)

// Build AST from CST. build scopes with symbol tables, and perform basic checks.
type ASTBuilder struct {
	BaseGolangVisitor
	// Global and local cursor of scope
	global, cur *Scope
	// Point to the program being constructed
	prog *ProgramNode
	// Track receiver of the method next child scope belongs to
	// Since methods and functions share a single function body AST construction procedure,
	// the procedure itself has no idea whether it has receiver.
	receiver *SymbolEntry
}

func NewASTBuilder() *ASTBuilder { return &ASTBuilder{} }

// Add statement to current function
func (v *ASTBuilder) addStmt(stmt IStmtNode) {
	v.cur.fun.AddStmt(stmt)
}

func (v *ASTBuilder) VisitSourceFile(ctx *SourceFileContext) interface{} {
	// Get package name
	pkgName := v.VisitPackageClause(ctx.PackageClause().(*PackageClauseContext)).(string)
	// Initialize program node
	v.prog = NewProgramNode(pkgName)
	// Set global and current scope pointer
	v.global = v.prog.global.scope
	v.cur = v.global

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
	return nil
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
	rhs := v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)
	lhs := v.VisitIdentifierList(ctx.IdentifierList().(*IdentifierListContext)).([]*IdExpr)

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
		tp := rhs[i].GetType() // rhs must be constant expression, its type must be clear then
		val := rhs[i].(IConstExpr).GetValue()
		loc := lhs[i].GetLocation()

		// Check if need conversion in compile time
		if specTp != nil && !specTp.IsIdentical(tp) {
			var err error
			val, err = rhs[i].(IConstExpr).ConvertTo(specTp) // try to convert to target type
			if err != nil {
				panic(fmt.Errorf("%s %s", loc.ToString(), err.Error())) // cannot convert
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
		if IsKeyword[id.GetText()] {
			panic(fmt.Errorf("%s cannot use keyword as identifier: %s",
				NewLocationFromTerminal(id).ToString(), id.GetText()))
		}
		idList = append(idList, NewIdExpr(NewLocationFromToken(id.GetSymbol()), id.GetText(),
			nil))
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
		if r.IsNamed() { // return type has associated name
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

	// Add named parameter and return value symbols to the function scope
	if v.receiver != nil { // add receiver to the method scope
		decl.scope.AddSymbol(v.receiver)
		v.receiver = nil // receiver should no longer be recorded
	}
	for _, p := range sig.params {
		if p.IsNamed() {
			decl.scope.AddSymbol(p)
		}
	}

	// Initialize named return values
	if len(namedRet) > 0 {
		lhs := make([]IExprNode, 0, len(namedRet))
		rhs := make([]IExprNode, 0, len(namedRet))
		for _, r := range namedRet {
			decl.scope.AddSymbol(r)
			lhs = append(lhs, NewIdExpr(r.loc, r.name, r))
			rhs = append(rhs, NewZeroValue())
		}
		decl.scope.fun.AddStmt(NewAssignStmt(NewLocationFromContext(ctx), lhs, rhs))
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
	decl := v.VisitParameterDecl(ctx.ParameterDecl().(*ParameterDeclContext)).([]*SymbolEntry)
	if len(decl) != 1 {
		panic(fmt.Errorf("%s expect one paramter in method receiver, have %d",
			NewLocationFromContext(ctx).ToString(), len(decl)))
	}
	return decl[0] // *SymbolEntry
}

func (v *ASTBuilder) VisitVarDecl(ctx *VarDeclContext) interface{} {
	for _, decl := range ctx.AllVarSpec() {
		v.VisitVarSpec(decl.(*VarSpecContext))
	}
	return nil
}

func (v *ASTBuilder) VisitVarSpec(ctx *VarSpecContext) interface{} {
	// Get expression list
	var exprList []IExprNode
	if ctx.ExpressionList() != nil {
		exprList = v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)
	}

	// Get specified variables
	idList := v.VisitIdentifierList(ctx.IdentifierList().(*IdentifierListContext)).([]*IdExpr)

	// Get specified type, if exists
	var specType IType
	if ctx.Tp() != nil {
		specType = v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	}

	// Reject if assignment count mismatch
	// If len(exprList) == 1 and len(idList) > 1, the right hand expression could a call
	// to a function that return multiple values. We cannot tell whether there is any error.
	if len(exprList) > 1 && len(exprList) != len(idList) {
		panic(fmt.Errorf("%s assignment count mismatch %d = %d",
			NewLocationFromContext(ctx).ToString(), len(idList), len(exprList)))
	}

	// Add variables to symbol table
	lhs := make([]IExprNode, 0)
	for _, id := range idList {
		// type maybe unknown at this time
		id.symbol = NewSymbolEntry(id.GetLocation(), id.name, VarEntry, specType, nil)
		v.cur.AddSymbol(id.symbol)
		lhs = append(lhs, id)
	}

	// Add statement to current scope
	if exprList == nil { // all declared variables should be assigned zero value
		exprList = make([]IExprNode, len(idList))
		for i := range exprList {
			exprList[i] = NewZeroValue()
		}
	}
	v.addStmt(NewAssignStmt(NewLocationFromContext(ctx), lhs, exprList))

	return nil
}

// Scope should be set up before visiting block
func (v *ASTBuilder) VisitBlock(ctx *BlockContext) interface{} {
	v.VisitStatementList(ctx.StatementList().(*StatementListContext))
	return nil
}

func (v *ASTBuilder) VisitStatementList(ctx *StatementListContext) interface{} {
	for _, stmt := range ctx.AllStatement() {
		v.VisitStatement(stmt.(*StatementContext))
	}
	return nil
}

func (v *ASTBuilder) VisitStatement(ctx *StatementContext) interface{} {
	if s := ctx.Declaration(); s != nil {
		v.VisitDeclaration(s.(*DeclarationContext))
	} else if s := ctx.SimpleStmt(); s != nil {
		v.VisitSimpleStmt(s.(*SimpleStmtContext))
	} else if s := ctx.Block(); s != nil {
		// Create new scope for block statements
		v.cur = NewLocalScope(v.cur)
		// Visit block
		v.VisitBlock(s.(*BlockContext))
		// Restore parent scope
		v.cur = v.cur.parent
	}
	return nil
}

func (v *ASTBuilder) VisitSimpleStmt(ctx *SimpleStmtContext) interface{} {
	if s := ctx.ExpressionStmt(); s != nil {
		v.VisitExpressionStmt(s.(*ExpressionStmtContext))
	} else if s := ctx.Assignment(); s != nil {
		v.VisitAssignment(s.(*AssignmentContext))
	} else if s := ctx.ShortVarDecl(); s != nil {
		v.VisitShortVarDecl(s.(*ShortVarDeclContext))
	}
	return nil
}

func (v *ASTBuilder) VisitExpressionStmt(ctx *ExpressionStmtContext) interface{} {
	expr := v.VisitExpression(ctx.Expression().(*ExpressionContext)).(IExprNode)
	v.addStmt(expr)
	return nil
}

func (v *ASTBuilder) VisitAssignment(ctx *AssignmentContext) interface{} {
	if op := ctx.GetOp(); op != nil { // assignment after operation
		lhs := v.VisitExpression(ctx.Expression(0).(*ExpressionContext)).(IExprNode)
		rhs := v.VisitExpression(ctx.Expression(1).(*ExpressionContext)).(IExprNode)
		opExpr := NewBinaryExpr(NewLocationFromContext(ctx), BinaryOpStrToEnum[op.GetText()],
			lhs, rhs)
		v.addStmt(NewAssignStmt(NewLocationFromContext(ctx), []IExprNode{lhs},
			[]IExprNode{opExpr}))

	} else { // only assignment, but can assign multiple values
		lhs := v.VisitExpressionList(ctx.ExpressionList(0).(*ExpressionListContext)).([]IExprNode)
		rhs := v.VisitExpressionList(ctx.ExpressionList(1).(*ExpressionListContext)).([]IExprNode)
		// Check length of expressions on both sides, similar to variable specification
		if len(rhs) > 1 && len(lhs) != len(rhs) {
			panic(fmt.Errorf("%s assignment count mismatch %d = %d",
				NewLocationFromContext(ctx).ToString(), len(lhs), len(rhs)))
		}
		v.addStmt(NewAssignStmt(NewLocationFromContext(ctx), lhs, rhs))
	}
	return nil
}

func (v *ASTBuilder) VisitShortVarDecl(ctx *ShortVarDeclContext) interface{} {
	// Get identifier list (lhs) and expression list (rhs)
	exprList := v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)
	idList := v.VisitIdentifierList(ctx.IdentifierList().(*IdentifierListContext)).([]*IdExpr)

	// Check if numbers of expressions on both sides match
	if len(exprList) > 1 && len(idList) != len(exprList) {
		panic(fmt.Errorf("%s assignment count mismatch %d = %d",
			NewLocationFromContext(ctx).ToString(), len(idList), len(exprList)))
	}

	// Add undefined identifier to symbol table (some can be defined at current scope)
	lhs := make([]IExprNode, 0)
	for _, id := range idList {
		if v.cur.CheckDefined(id.name) { // already defined at current scope
			id.symbol, _ = v.cur.Lookup(id.name)
		} else {
			id.symbol = NewSymbolEntry(id.loc, id.name, VarEntry, nil, nil)
			v.cur.AddSymbol(id.symbol)
		}
		lhs = append(lhs, id)
	}

	// Add statement to current function
	v.addStmt(NewAssignStmt(NewLocationFromContext(ctx), lhs, exprList))

	return nil
}

func (v *ASTBuilder) VisitTp(ctx *TpContext) interface{} {
	if tp := ctx.TypeName(); tp != nil {
		return v.VisitTypeName(tp.(*TypeNameContext)).(IType)
	} else if tp := ctx.TypeLit(); tp != nil {
		return v.VisitTypeLit(tp.(*TypeLitContext)).(IType)
	} else if tp := ctx.Tp(); tp != nil {
		return v.VisitTp(tp.(*TpContext)).(IType)
	}
	return nil // IType
}

func (v *ASTBuilder) VisitTypeName(ctx *TypeNameContext) interface{} {
	// Check if is primitive type
	name := ctx.IDENTIFIER().GetText()
	tp, ok := StrToPrimType[name]
	if ok {
		return NewPrimType(tp)
	}

	// Try to resolve type symbol
	// It's OK to be unresolved, since it may be defined later in global scope.
	symbol, _ := v.cur.Lookup(name)
	if symbol != nil && symbol.flag == TypeEntry { // resolved and represents type
		return symbol.tp
	}

	return NewUnresolvedType(name) // cannot resolve at present
}

func (v *ASTBuilder) VisitTypeLit(ctx *TypeLitContext) interface{} {
	if tp := ctx.FunctionType(); tp != nil {
		return v.VisitFunctionType(tp.(*FunctionTypeContext))
	} else if tp := ctx.StructType(); tp != nil {
		return v.VisitStructType(tp.(*StructTypeContext))
	}
	return nil
}

func (v *ASTBuilder) VisitFunctionType(ctx *FunctionTypeContext) interface{} {
	sig := v.VisitSignature(ctx.Signature().(*SignatureContext)).(*FuncSignature)
	paramType := make([]IType, 0)
	resultType := make([]IType, 0)
	for _, p := range sig.params {
		paramType = append(paramType, p.tp)
	}
	for _, r := range sig.results {
		resultType = append(resultType, r.tp)
	}
	return NewFunctionType(paramType, resultType) // *FunctionType
}

func (v *ASTBuilder) VisitSignature(ctx *SignatureContext) interface{} {
	params := v.VisitParameters(ctx.Parameters().(*ParametersContext)).([]*SymbolEntry)
	results := make([]*SymbolEntry, 0)
	if r := ctx.Result(); r != nil {
		results = v.VisitResult(r.(*ResultContext)).([]*SymbolEntry)
	}
	return &FuncSignature{params: params, results: results}
}

func (v *ASTBuilder) VisitResult(ctx *ResultContext) interface{} {
	if r := ctx.Tp(); r != nil {
		tp := v.VisitTp(r.(*TpContext)).(IType)
		return []*SymbolEntry{
			NewSymbolEntry(NewLocationFromContext(ctx), "", VarEntry, tp, nil)}
	} else if r := ctx.Parameters(); r != nil {
		return v.VisitParameters(r.(*ParametersContext)).([]*SymbolEntry)
	}
	return nil // []*SymbolEntry
}

func (v *ASTBuilder) VisitParameters(ctx *ParametersContext) interface{} {
	if p := ctx.ParameterList(); p != nil {
		return v.VisitParameterList(p.(*ParameterListContext)).([]*SymbolEntry)
	} else {
		return make([]*SymbolEntry, 0)
	} // []*SymbolEntry
}

func (v *ASTBuilder) VisitParameterList(ctx *ParameterListContext) interface{} {
	list := make([]*SymbolEntry, 0)
	for _, d := range ctx.AllParameterDecl() {
		list = append(list, v.VisitParameterDecl(d.(*ParameterDeclContext)).([]*SymbolEntry)...)
	}
	return list
}

func (v *ASTBuilder) VisitParameterDecl(ctx *ParameterDeclContext) interface{} {
	tp := v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	idList := v.VisitIdentifierList(ctx.IdentifierList().(*IdentifierListContext)).([]*IdExpr)
	if idList == nil {
		return []*SymbolEntry{
			NewSymbolEntry(NewLocationFromContext(ctx), "", VarEntry, tp, 0)}
	}
	declList := make([]*SymbolEntry, 0)
	for _, id := range idList {
		declList = append(declList, NewSymbolEntry(id.loc, id.name, VarEntry, tp, 0))
	}
	return declList // []*SymbolEntry
}

func (v *ASTBuilder) VisitStructType(ctx *StructTypeContext) interface{} {
	table := NewSymbolTable()
	for _, f := range ctx.AllFieldDecl() {
		table.Add(v.VisitFieldDecl(f.(*FieldDeclContext)).([]*SymbolEntry)...)
	}
	return NewStructType(table) // *StructType
}

func (v *ASTBuilder) VisitFieldDecl(ctx *FieldDeclContext) interface{} {
	if l := ctx.IdentifierList(); l != nil {
		idList := v.VisitIdentifierList(l.(*IdentifierListContext)).([]*IdExpr)
		tp := v.VisitTp(ctx.Tp().(*TpContext)).(IType)
		declList := make([]*SymbolEntry, 0)
		for _, id := range idList {
			declList = append(declList, NewSymbolEntry(id.loc, id.name, VarEntry, tp, nil))
		}
		return declList
	}
	return nil
}
