package ast

import (
	"fmt"
	. "parser"
	"strconv"
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
	// Block stack
	blocks [][]*BlockStmt // [func][block]
}

func NewASTBuilder() *ASTBuilder { return &ASTBuilder{} }

// Add statement to current function
func (v *ASTBuilder) addStmt(stmt IStmtNode) {
	if len(v.blocks) == 0 { // in global function
		v.cur.fun.AddStmt(stmt)
	} else if l := len(v.blocks[len(v.blocks)-1]); l == 0 { // scope of function
		v.cur.fun.AddStmt(stmt)
	} else { // scope of block
		v.blocks[len(v.blocks)-1][l-1].AddStmt(stmt)
	}
}

func (v *ASTBuilder) pushFuncBlock() {
	v.blocks = append(v.blocks, make([]*BlockStmt, 0))
}

func (v *ASTBuilder) popFuncBlock() {
	v.blocks = v.blocks[:len(v.blocks)-1]
}

func (v *ASTBuilder) getBlocksOfCurFunc() []*BlockStmt {
	return v.blocks[len(v.blocks)-1]
}

func (v *ASTBuilder) pushBlockStmt(block *BlockStmt) {
	v.addStmt(block)
	v.blocks[len(v.blocks)-1] = append(v.blocks[len(v.blocks)-1], block)
}

func (v *ASTBuilder) popBlockStmt() {
	v.blocks[len(v.blocks)-1] = v.blocks[len(v.blocks)-1][:len(v.blocks[len(v.blocks)-1])-1]
}

func (v *ASTBuilder) pushScope(scope *Scope) { v.cur = scope }

func (v *ASTBuilder) popScope() { v.cur = v.cur.parent }

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
		if _, ok := expr.(*ConstExpr); !ok {
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
		val := rhs[i].(*ConstExpr).val
		loc := lhs[i].GetLocation()

		// Check if need conversion in compile time
		if specTp != nil && !specTp.IsIdentical(tp) {
			convert := constTypeConvert[tp.GetTypeEnum()][specTp.GetTypeEnum()]
			if convert == nil {
				panic(fmt.Errorf("%s cannot convert from %s to %s", loc.ToString(),
					TypeToStr[tp.GetTypeEnum()], TypeToStr[specTp.GetTypeEnum()]))
			}
			val = convert(val)
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
	v.pushFuncBlock()
	v.pushScope(decl.scope) // move scope cursor deeper

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
		v.addStmt(NewAssignStmt(NewLocationFromContext(ctx), lhs, rhs))
	}

	// Build statement AST nodes
	v.VisitBlock(ctx.Block().(*BlockContext)) // statements are added during visit
	v.popFuncBlock()
	v.popScope() // move cursor back

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
	// If len(exprList) == 1 and len(idList) > 1, the right hand expression could be a call
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
	v.addStmt(NewInitStmt(NewLocationFromContext(ctx), lhs, exprList))

	return nil
}

// Scope should be set up before visiting block
// This may be a function definition block, not necessarily a block statement
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
		v.pushScope(NewLocalScope(v.cur))
		// Create block statement and push block onto stack
		block := NewBlockStmt(NewLocationFromContext(ctx), v.cur)
		v.pushBlockStmt(block)
		// Visit block
		v.VisitBlock(s.(*BlockContext))
		// Restore parent scope and block
		v.popScope()
		v.popBlockStmt()
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

func (v *ASTBuilder) VisitIncDecStmt(ctx *IncDecStmtContext) interface{} {
	expr := v.VisitExpression(ctx.Expression().(*ExpressionContext)).(IExprNode)
	inc := ctx.GetOp().GetText() == "++"
	return NewIncDecStmt(NewLocationFromContext(ctx), expr, inc)
}

func (v *ASTBuilder) VisitAssignment(ctx *AssignmentContext) interface{} {
	if op := ctx.GetOp(); op != nil { // assignment after operation
		rhs := v.VisitExpression(ctx.Expression(1).(*ExpressionContext)).(IExprNode)
		lhs := v.VisitExpression(ctx.Expression(0).(*ExpressionContext)).(IExprNode)
		opExpr := NewBinaryExpr(NewLocationFromContext(ctx), BinaryOpStrToEnum[op.GetText()],
			lhs, rhs)
		v.addStmt(NewAssignStmt(NewLocationFromContext(ctx), []IExprNode{lhs},
			[]IExprNode{opExpr}))

	} else { // only assignment, but can assign multiple values
		rhs := v.VisitExpressionList(ctx.ExpressionList(1).(*ExpressionListContext)).([]IExprNode)
		lhs := v.VisitExpressionList(ctx.ExpressionList(0).(*ExpressionListContext)).([]IExprNode)
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
	nNew := 0
	for _, id := range idList {
		if v.cur.CheckDefined(id.name) { // already defined at current scope
			symbol, _ := v.cur.Lookup(id.name)
			if symbol.flag != VarEntry { // report error if this symbol is not a variable
				panic(fmt.Errorf("%s not a variable", id.loc.ToString()))
			}
			id.symbol = symbol
		} else {
			id.symbol = NewSymbolEntry(id.loc, id.name, VarEntry, nil, nil)
			v.cur.AddSymbol(id.symbol)
			nNew++
		}
		lhs = append(lhs, id)
	}

	// Report error if no new variables are declared
	if nNew == 0 {
		panic(fmt.Errorf("%s no new variables are declared",
			NewLocationFromContext(ctx).ToString()))
	}

	// Add statement to current function
	v.addStmt(NewInitStmt(NewLocationFromContext(ctx), lhs, exprList))

	return nil
}

func (v *ASTBuilder) VisitReturnStmt(ctx *ReturnStmtContext) interface{} {
	exprList := make([]IExprNode, 0)
	if exprCtx := ctx.ExpressionList(); exprCtx != nil {
		exprList = v.VisitExpressionList(exprCtx.(*ExpressionListContext)).([]IExprNode)
	}
	return NewReturnStmt(NewLocationFromContext(ctx), exprList)
}

func (v *ASTBuilder) VisitBreakStmt(ctx *BreakStmtContext) interface{} {
	blocks := v.getBlocksOfCurFunc()
	var target IStmtNode
	for i := len(blocks) - 1; i >= 0; i-- { // find the innermost breakable target
		if stmt := blocks[i].breakable; stmt != nil {
			target = stmt
			break
		}
	}
	loc := NewLocationFromContext(ctx)
	if target == nil {
		panic(fmt.Errorf("%s cannot find break target", loc.ToString()))
	}
	return NewBreakStmt(loc, target)
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
	if tp := ctx.ArrayType(); tp != nil {
		return v.VisitArrayType(tp.(*ArrayTypeContext)).(IType)
	} else if tp := ctx.SliceType(); tp != nil {
		return v.VisitSliceType(tp.(*SliceTypeContext)).(IType)
	} else if tp := ctx.MapType(); tp != nil {
		return v.VisitMapType(tp.(*MapTypeContext)).(IType)
	} else if tp := ctx.FunctionType(); tp != nil {
		return v.VisitFunctionType(tp.(*FunctionTypeContext)).(IType)
	} else if tp := ctx.StructType(); tp != nil {
		return v.VisitStructType(tp.(*StructTypeContext)).(IType)
	}
	return nil
}

func (v *ASTBuilder) VisitArrayType(ctx *ArrayTypeContext) interface{} {
	elem := v.VisitElementType(ctx.ElementType().(*ElementTypeContext)).(IType)
	length := v.VisitArrayLength(ctx.ArrayLength().(*ArrayLengthContext)).(int)
	return NewArrayType(elem, length)
}

func (v *ASTBuilder) VisitArrayLength(ctx *ArrayLengthContext) interface{} {
	expr, ok := v.VisitExpression(ctx.Expression().(*ExpressionContext)).(*ConstExpr)
	if !ok {
		panic(fmt.Errorf("%s array length should be constant expression",
			NewLocationFromContext(ctx).ToString()))
	}
	val, ok := expr.val.(int)
	if !ok {
		panic(fmt.Errorf("%s array length should be an constant integer",
			NewLocationFromContext(ctx).ToString()))
	}
	return val
}

func (v *ASTBuilder) VisitElementType(ctx *ElementTypeContext) interface{} {
	return v.VisitTp(ctx.Tp().(*TpContext)).(IType)
}

func (v *ASTBuilder) VisitPointerType(ctx *PointerTypeContext) interface{} {
	return NewPtrType(v.VisitTp(ctx.Tp().(*TpContext)).(IType))
}

func (v *ASTBuilder) VisitSliceType(ctx *SliceTypeContext) interface{} {
	return NewSliceType(v.VisitElementType(ctx.ElementType().(*ElementTypeContext)).(IType))
}

func (v *ASTBuilder) VisitMapType(ctx *MapTypeContext) interface{} {
	key := v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	val := v.VisitElementType(ctx.ElementType().(*ElementTypeContext)).(IType)
	return NewMapType(key, val)
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
	if idListCxt := ctx.IdentifierList(); idListCxt == nil {
		return []*SymbolEntry{
			NewSymbolEntry(NewLocationFromContext(ctx), "", VarEntry, tp, 0)}
	} else {
		idList := v.VisitIdentifierList(idListCxt.(*IdentifierListContext)).([]*IdExpr)
		declList := make([]*SymbolEntry, 0)
		for _, id := range idList {
			declList = append(declList, NewSymbolEntry(id.loc, id.name, VarEntry, tp, 0))
		}
		return declList // []*SymbolEntry
	}
}

func (v *ASTBuilder) VisitOperand(ctx *OperandContext) interface{} {
	if o := ctx.Literal(); o != nil {
		return v.VisitLiteral(o.(*LiteralContext)).(IExprNode)
	} else if o := ctx.OperandName(); o != nil {
		return v.VisitOperandName(o.(*OperandNameContext)).(IExprNode)
	} else if o := ctx.Expression(); o != nil {
		return v.VisitExpression(o.(*ExpressionContext)).(IExprNode)
	}
	return nil // IExprNode
}

func (v *ASTBuilder) VisitLiteral(ctx *LiteralContext) interface{} {
	if l := ctx.BasicLit(); l != nil {
		return v.VisitBasicLit(l.(*BasicLitContext)).(IExprNode)
	} else if l := ctx.FunctionLit(); l != nil {
		return v.VisitFunctionLit(l.(*FunctionLitContext)).(IExprNode)
	}
	return nil
}

func (v *ASTBuilder) VisitBasicLit(ctx *BasicLitContext) interface{} {
	if l := ctx.INT_LIT(); l != nil {
		lit, _ := strconv.ParseInt(l.GetText(), 0, 0)
		return NewIntConst(NewLocationFromTerminal(l), int(lit))
	} else if l = ctx.FLOAT_LIT(); l != nil {
		lit, _ := strconv.ParseFloat(l.GetText(), 64)
		return NewFloatConst(NewLocationFromTerminal(l), lit)
	}
	return nil
}

func (v *ASTBuilder) VisitOperandName(ctx *OperandNameContext) interface{} {
	// Create operand identifier
	name := ctx.IDENTIFIER().GetText()
	symbol, scope := v.cur.Lookup(name)
	id := NewIdExpr(NewLocationFromContext(ctx), name, symbol)

	// Decide if it should be captured
	id.captured = scope != nil && !scope.global && scope.fun != v.cur.fun
	v.cur.AddOperandId(id)

	return id
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
		return declList // []*SymbolEntry
	}
	return nil
}

func (v *ASTBuilder) VisitFunctionLit(ctx *FunctionLitContext) interface{} {
	// Get declaration from function context
	decl := v.VisitFunction(ctx.Function().(*FunctionContext)).(*FuncDecl)

	// Create lambda capture set (identifiers in different blocks may repeat)
	// Visit scopes of declared function and its nested scopes (excluding the scopes of its nested
	// functions), using DFS
	closureSet := make(map[*SymbolEntry]bool, 0)
	stack := []*Scope{decl.scope}
	for len(stack) > 0 {
		// Process operand identifiers in current scope
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1] // pop an element from stack
		for id := range top.operandId {
			if id.captured {
				closureSet[id.symbol] = true
			}
		}

		// Push children scopes to stack, within the function
		for _, child := range top.children {
			if child.fun == decl {
				stack = append(stack, child)
			}
		}
	}

	// Convert set to symbol table (to merge repeated identifiers)
	closureTable := NewSymbolTable()
	for entry := range closureSet {
		closureTable.Add(entry)
	}

	return NewFuncLiteral(NewLocationFromContext(ctx), decl, closureTable)
}

func (v *ASTBuilder) VisitPrimaryExpr(ctx *PrimaryExprContext) interface{} {
	if e := ctx.Operand(); e != nil {
		return v.VisitOperand(e.(*OperandContext)).(IExprNode)
	} else if e := ctx.Conversion(); e != nil {
		return v.VisitConversion(e.(*ConversionContext)).(IExprNode)
	}

	// Must be a recursive call then
	prim := v.VisitPrimaryExpr(ctx.PrimaryExpr().(*PrimaryExprContext)).(IExprNode)
	if e := ctx.Arguments(); e != nil {
		args := v.VisitArguments(e.(*ArgumentsContext)).([]IExprNode)
		return NewFuncCallExpr(NewLocationFromContext(ctx), prim, args)
	}

	return nil // IExprNode
}

func (v *ASTBuilder) VisitArguments(ctx *ArgumentsContext) interface{} {
	return v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)
}

func (v *ASTBuilder) VisitExpression(ctx *ExpressionContext) interface{} {
	// Forward unary expression
	if e := ctx.UnaryExpr(); e != nil {
		return v.VisitUnaryExpr(e.(*UnaryExprContext)).(IExprNode)
	}

	// Deal with binary expression
	op := BinaryOpStrToEnum[ctx.GetOp().GetText()]
	left := v.VisitExpression(ctx.Expression(0).(*ExpressionContext)).(IExprNode)
	right := v.VisitExpression(ctx.Expression(1).(*ExpressionContext)).(IExprNode)

	// Evaluate constant expression
	lConst, lok := left.(*ConstExpr)
	rConst, rok := right.(*ConstExpr)
	if lok && rok {
		fun := binaryConstExpr[op][lConst.tp.GetTypeEnum()][rConst.tp.GetTypeEnum()]
		if fun != nil {
			return fun(lConst, rConst)
		} else {
			panic(fmt.Errorf("%s cannot evaluate constant expression",
				NewLocationFromContext(ctx).ToString()))
		}
	}

	return NewBinaryExpr(NewLocationFromContext(ctx), op, left, right) // IExprNode
}

func (v *ASTBuilder) VisitUnaryExpr(ctx *UnaryExprContext) interface{} {
	// Forward primary expression
	if e := ctx.PrimaryExpr(); e != nil {
		return v.VisitPrimaryExpr(e.(*PrimaryExprContext)).(IExprNode)
	}

	// Deal with unary expression
	op := UnaryOpStrToEnum[ctx.GetOp().GetText()]
	prim := v.VisitUnaryExpr(ctx.UnaryExpr().(*UnaryExprContext)).(IExprNode)

	// Evaluate constant expression
	if pConst, ok := prim.(*ConstExpr); ok {
		fun := unaryConstExpr[op][pConst.tp.GetTypeEnum()]
		if fun != nil {
			return fun(pConst)
		} else {
			panic(fmt.Errorf("%s cannot evaluate constant expression",
				NewLocationFromContext(ctx).ToString()))
		}
	}

	return NewUnaryExpr(NewLocationFromContext(ctx), op, prim) // IExprNode
}
