package ast

import (
	"fmt"
	. "parser"
	"strconv"
)

// Build AST from CST. build scopes with symbol tables, and perform basic checks.
type Builder struct {
	BaseGolangVisitor
	// Global and local cursor of scope
	global, cur *Scope
	// Point to the program being constructed
	prog *ProgramNode
	// Track receiver of the method next child scope belongs to
	// Since methods and functions share a single function body construction procedure,
	// the procedure itself has no idea whether it has receiver.
	receiver *TableEntry
	// Block stack
	blocks [][]*BlockStmt // [func][block]
}

func NewASTBuilder() *Builder { return &Builder{} }

// Add statement to current function
func (v *Builder) addStmt(stmt IStmtNode) {
	if len(v.blocks) == 0 { // in global function
		v.cur.Func.AddStmt(stmt)
	} else if l := len(v.blocks[len(v.blocks)-1]); l == 0 { // scope of function
		v.cur.Func.AddStmt(stmt)
	} else { // scope of block
		v.blocks[len(v.blocks)-1][l-1].AddStmt(stmt)
	}
}

func (v *Builder) pushFunc() {
	v.blocks = append(v.blocks, make([]*BlockStmt, 0))
}

func (v *Builder) popFunc() {
	v.blocks = v.blocks[:len(v.blocks)-1]
}

func (v *Builder) getBlocksOfCurFunc() []*BlockStmt {
	return v.blocks[len(v.blocks)-1]
}

func (v *Builder) pushBlock(block *BlockStmt) {
	v.blocks[len(v.blocks)-1] = append(v.blocks[len(v.blocks)-1], block)
}

func (v *Builder) popBlock() {
	v.blocks[len(v.blocks)-1] = v.blocks[len(v.blocks)-1][:len(v.blocks[len(v.blocks)-1])-1]
}

func (v *Builder) pushScope(scope *Scope) { v.cur = scope }

func (v *Builder) popScope() { v.cur = v.cur.Parent }

func (v *Builder) VisitSourceFile(ctx *SourceFileContext) interface{} {
	// Get package name
	pkgName := v.VisitPackageClause(ctx.PackageClause().(*PackageClauseContext)).(string)
	// Initialize program node
	v.prog = NewProgramNode(pkgName)
	// Set global and current scope pointer
	v.global = v.prog.Global.Scope
	v.cur = v.global

	// Add declarations to program node
	for _, decl := range ctx.AllTopLevelDecl() {
		v.VisitTopLevelDecl(decl.(*TopLevelDeclContext))
	}

	return v.prog
}

func (v *Builder) VisitPackageClause(ctx *PackageClauseContext) interface{} {
	return ctx.IDENTIFIER().GetText() // string
}

func (v *Builder) VisitTopLevelDecl(ctx *TopLevelDeclContext) interface{} {
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
func (v *Builder) VisitDeclaration(ctx *DeclarationContext) interface{} {
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
func (v *Builder) VisitConstDecl(ctx *ConstDeclContext) interface{} {
	for _, spec := range ctx.AllConstSpec() {
		v.VisitConstSpec(spec.(*ConstSpecContext))
	}
	return nil
}

// One line of constant specification: id_1, id_2, ..., id_n = expr_1, expr_2, ..., expr_n
func (v *Builder) VisitConstSpec(ctx *ConstSpecContext) interface{} {
	// Get expression list on both sides of equation
	rhs := v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)
	lhs := v.VisitIdentifierList(ctx.IdentifierList().(*IdentifierListContext)).([]*IdExpr)

	// Validate expressions
	// Ensure same length
	if len(lhs) != len(rhs) {
		panic(NewSemaError(
			NewLocFromContext(ctx),
			fmt.Sprintf("assignment count mismatch: #%d = #%d", len(lhs), len(rhs)),
		))
	}
	// lhs symbols haven't been defined before
	for _, id := range lhs {
		if v.cur.CheckDefined(id.Name) { // redefined symbol
			panic(NewSemaError(
				NewLocFromContext(ctx),
				fmt.Sprintf("redefined symbol: %s", id.Name),
			))
		}
	}
	// rhs symbols are all constant expressions
	for _, expr := range rhs {
		if _, ok := expr.(*ConstExpr); !ok {
			panic(NewSemaError(
				NewLocFromContext(ctx),
				"not constant expression",
			))
		}
	}

	// Add constant specification
	var specTp IType
	if ctx.Tp() != nil {
		specTp = v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	}
	for i := range lhs {
		tp := rhs[i].GetType() // rhs must be constant expression, its type must be clear then
		val := rhs[i].(*ConstExpr).Val
		loc := lhs[i].GetLoc()

		// Check if need conversion in compile time
		if specTp != nil && !specTp.IsIdentical(tp) {
			convert := constTypeConvert[tp.GetTypeEnum()][specTp.GetTypeEnum()]
			if convert == nil {
				panic(NewSemaError(
					loc,
					fmt.Sprintf("cannot convert from %s to %s",
						TypeToStr[tp.GetTypeEnum()], TypeToStr[specTp.GetTypeEnum()]),
				))
			}
			val = convert(val)
			tp = specTp
		}

		// Add to symbol table of current scope
		v.cur.AddSymbol(NewSymbolEntry(loc, lhs[i].Name, ConstEntry, tp, val))
	}

	return nil
}

func (v *Builder) VisitIdentifierList(ctx *IdentifierListContext) interface{} {
	idList := make([]*IdExpr, 0)
	for _, id := range ctx.AllIDENTIFIER() {
		if IsKeyword[id.GetText()] {
			panic(NewSemaError(
				NewLocFromTerminal(id),
				fmt.Sprintf("cannot use keyword as identifier: %s", id.GetText()),
			))
		}
		idList = append(idList, NewIdExpr(NewLocFromToken(id.GetSymbol()), id.GetText(),
			nil))
	}
	return idList // []*IdExpr
}

func (v *Builder) VisitExpressionList(ctx *ExpressionListContext) interface{} {
	exprList := make([]IExprNode, 0)
	for _, expr := range ctx.AllExpression() {
		exprList = append(exprList, v.VisitExpression(expr.(*ExpressionContext)).(IExprNode))
	}
	return exprList // []IExprNode
}

func (v *Builder) VisitTypeDecl(ctx *TypeDeclContext) interface{} {
	for _, spec := range ctx.AllTypeSpec() {
		v.VisitTypeSpec(spec.(*TypeSpecContext))
	}
	return nil
}

func (v *Builder) VisitTypeSpec(ctx *TypeSpecContext) interface{} {
	name := ctx.IDENTIFIER().GetText()
	tp := v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	alias := NewAliasType(NewLocFromContext(ctx), name, tp)
	v.cur.AddSymbol(NewSymbolEntry(NewLocFromContext(ctx), name, TypeEntry, alias, nil))
	return nil
}

// Top level function declarations
func (v *Builder) VisitFunctionDecl(ctx *FunctionDeclContext) interface{} {
	name := ctx.IDENTIFIER().GetText()
	funcDecl := v.VisitFunction(ctx.Function().(*FunctionContext)).(*FuncDecl)
	funcDecl.Name = name
	v.global.AddSymbol(funcDecl.GenSymbol())
	v.prog.AddFuncDecl(funcDecl)
	return nil
}

func (v *Builder) VisitFunction(ctx *FunctionContext) interface{} {
	// Analyze function signature
	sig := v.VisitSignature(ctx.Signature().(*SignatureContext)).(*FuncSignature)
	paramType := make([]IType, 0)
	resultType := make([]IType, 0)
	namedRet := make([]*TableEntry, 0)
	for _, p := range sig.params {
		paramType = append(paramType, p.Type)
	}
	for _, r := range sig.results {
		resultType = append(resultType, r.Type)
		if r.IsNamed() { // return type has associated name
			namedRet = append(namedRet, r)
		}
	}

	// Panic if mixed named and unnamed parameters
	if len(namedRet) > 0 && len(namedRet) != len(resultType) {
		panic(NewSemaError(
			NewLocFromContext(ctx),
			"mixed named and unnamed parameters",
		))
	}

	// Initialize function declaration node
	decl := NewFuncDecl(NewLocFromContext(ctx), "", // name should be assigned later
		NewFunctionType(NewLocFromContext(ctx), paramType, resultType),
		NewLocalScope(v.cur), namedRet)
	if v.receiver != nil { // add receiver to function type if it is a method
		decl.Type.Receiver = v.receiver.Type
	}
	v.pushFunc()
	v.pushScope(decl.Scope) // move scope cursor deeper

	// Add named parameters and return value symbols to the function scope
	if v.receiver != nil { // add receiver to the method scope
		decl.Scope.AddSymbol(v.receiver)
		v.receiver = nil // receiver should no longer be recorded
	}
	for _, p := range sig.params {
		if p.IsNamed() {
			decl.Scope.AddSymbol(p)
		}
	}

	// Initialize named return values
	if len(namedRet) > 0 {
		lhs := make([]IExprNode, 0, len(namedRet))
		rhs := make([]IExprNode, 0, len(namedRet))
		for _, r := range namedRet {
			decl.Scope.AddSymbol(r)
			lhs = append(lhs, NewIdExpr(r.Loc, r.Name, r))
			rhs = append(rhs, NewZeroValue(r.Loc))
		}
		v.addStmt(NewAssignStmt(NewLocFromContext(ctx), lhs, rhs))
	}

	// Build statement AST nodes
	v.VisitBlock(ctx.Block().(*BlockContext)) // statements are added during visit
	v.popFunc()
	v.popScope() // move cursor back

	return decl // *FuncDecl
}

func (v *Builder) VisitMethodDecl(ctx *MethodDeclContext) interface{} {
	name := ctx.IDENTIFIER().GetText()
	v.receiver = v.VisitReceiver(ctx.Receiver().(*ReceiverContext)).(*TableEntry) // record receiver
	funcDecl := v.VisitFunction(ctx.Function().(*FunctionContext)).(*FuncDecl)
	funcDecl.Name = name
	v.global.AddSymbol(funcDecl.GenSymbol())
	v.prog.AddFuncDecl(funcDecl)
	return nil
}

func (v *Builder) VisitReceiver(ctx *ReceiverContext) interface{} {
	decl := v.VisitParameterDecl(ctx.ParameterDecl().(*ParameterDeclContext)).([]*TableEntry)
	if len(decl) != 1 {
		panic(NewSemaError(
			NewLocFromContext(ctx),
			fmt.Sprintf("expect one paramter in method receiver, have %d", len(decl)),
		))
	}
	return decl[0] // *SymbolEntry
}

func (v *Builder) VisitVarDecl(ctx *VarDeclContext) interface{} {
	for _, decl := range ctx.AllVarSpec() {
		v.VisitVarSpec(decl.(*VarSpecContext))
	}
	return nil
}

func (v *Builder) VisitVarSpec(ctx *VarSpecContext) interface{} {
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
		panic(NewSemaError(
			NewLocFromContext(ctx),
			fmt.Sprintf("assignment count mismatch %d = %d", len(idList), len(exprList)),
		))
	}

	// Add variables to symbol table
	lhs := make([]IExprNode, 0)
	for _, id := range idList {
		// type maybe unknown at this time
		id.Symbol = NewSymbolEntry(id.GetLoc(), id.Name, VarEntry, specType, nil)
		v.cur.AddSymbol(id.Symbol)
		lhs = append(lhs, id)
	}

	// Add statement to current scope
	if exprList == nil { // all declared variables should be assigned zero value
		exprList = make([]IExprNode, len(idList))
		for i := range exprList {
			exprList[i] = NewZeroValue(NewLocFromContext(ctx))
		}
	}
	v.addStmt(NewInitStmt(NewLocFromContext(ctx), lhs, exprList))

	return nil
}

// Scope should be set up before visiting block
// This may be a function definition block, not necessarily a block statement
func (v *Builder) VisitBlock(ctx *BlockContext) interface{} {
	v.VisitStatementList(ctx.StatementList().(*StatementListContext))
	return nil
}

func (v *Builder) VisitStatementList(ctx *StatementListContext) interface{} {
	for _, stmt := range ctx.AllStatement() {
		v.VisitStatement(stmt.(*StatementContext))
	}
	return nil
}

func (v *Builder) VisitStatement(ctx *StatementContext) interface{} {
	if s := ctx.Declaration(); s != nil {
		v.VisitDeclaration(s.(*DeclarationContext))
	} else if s := ctx.SimpleStmt(); s != nil {
		stmt := v.VisitSimpleStmt(s.(*SimpleStmtContext)).(IStmtNode)
		v.addStmt(stmt) // it's impossible that a control statement may need it then
	} else if s := ctx.ReturnStmt(); s != nil {
		v.VisitReturnStmt(s.(*ReturnStmtContext))
	} else if s := ctx.BreakStmt(); s != nil {
		v.VisitBreakStmt(s.(*BreakStmtContext))
	} else if s := ctx.ContinueStmt(); s != nil {
		v.VisitContinueStmt(s.(*ContinueStmtContext))
	} else if s := ctx.Block(); s != nil {
		// Create new scope for block statements
		v.pushScope(NewLocalScope(v.cur))
		// Create block statement and push block onto stack
		block := NewBlockStmt(NewLocFromContext(ctx), v.cur)
		v.addStmt(block)
		v.pushBlock(block)
		// Visit block
		v.VisitBlock(s.(*BlockContext))
		// Restore parent scope and block
		v.popBlock()
		v.popScope()
	} else if s := ctx.IfStmt(); s != nil {
		v.VisitIfStmt(s.(*IfStmtContext))
	} else if s := ctx.ForStmt(); s != nil {
		v.VisitForStmt(s.(*ForStmtContext))
	}
	return nil
}

// Simple statement could appear in if or for clause, so it should be returned
func (v *Builder) VisitSimpleStmt(ctx *SimpleStmtContext) interface{} {
	if s := ctx.ExpressionStmt(); s != nil {
		return v.VisitExpressionStmt(s.(*ExpressionStmtContext)).(IStmtNode)
	} else if s := ctx.IncDecStmt(); s != nil {
		return v.VisitIncDecStmt(s.(*IncDecStmtContext)).(IStmtNode)
	} else if s := ctx.Assignment(); s != nil {
		return v.VisitAssignment(s.(*AssignmentContext)).(IStmtNode)
	} else if s := ctx.ShortVarDecl(); s != nil {
		return v.VisitShortVarDecl(s.(*ShortVarDeclContext)).(IStmtNode)
	}
	return nil
}

func (v *Builder) VisitExpressionStmt(ctx *ExpressionStmtContext) interface{} {
	expr := v.VisitExpression(ctx.Expression().(*ExpressionContext)).(IExprNode)
	return expr
}

func (v *Builder) VisitIncDecStmt(ctx *IncDecStmtContext) interface{} {
	expr := v.VisitExpression(ctx.Expression().(*ExpressionContext)).(IExprNode)
	inc := ctx.GetOp().GetText() == "++"
	return NewIncDecStmt(NewLocFromContext(ctx), expr, inc)
}

func (v *Builder) VisitAssignment(ctx *AssignmentContext) interface{} {
	var stmt IStmtNode
	if op := ctx.GetOp(); op != nil { // assignment after operation
		rhs := v.VisitExpression(ctx.Expression(1).(*ExpressionContext)).(IExprNode)
		lhs := v.VisitExpression(ctx.Expression(0).(*ExpressionContext)).(IExprNode)
		opExpr := NewBinaryExpr(NewLocFromContext(ctx), BinaryOpStrToEnum[op.GetText()],
			lhs, rhs)
		stmt = NewAssignStmt(NewLocFromContext(ctx), []IExprNode{lhs}, []IExprNode{opExpr})

	} else { // only assignment, but can assign multiple values
		rhs := v.VisitExpressionList(ctx.ExpressionList(1).(*ExpressionListContext)).([]IExprNode)
		lhs := v.VisitExpressionList(ctx.ExpressionList(0).(*ExpressionListContext)).([]IExprNode)
		// Check length of expressions on both sides, similar to variable specification
		stmt = NewAssignStmt(NewLocFromContext(ctx), lhs, rhs)
	}
	return stmt
}

func (v *Builder) VisitShortVarDecl(ctx *ShortVarDeclContext) interface{} {
	// Get identifier list (lhs) and expression list (rhs)
	exprList := v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)
	idList := v.VisitIdentifierList(ctx.IdentifierList().(*IdentifierListContext)).([]*IdExpr)

	// Check if numbers of expressions on both sides match
	if len(exprList) > 1 && len(idList) != len(exprList) {
		panic(NewSemaError(NewLocFromContext(ctx),
			fmt.Sprintf("assignment count mismatch %d = %d", len(idList), len(exprList)),
		))
	}

	// Add undefined identifier to symbol table (some can be defined at current scope)
	lhs := make([]IExprNode, 0)
	nNew := 0
	for _, id := range idList {
		if v.cur.CheckDefined(id.Name) { // already defined at current scope
			symbol, _ := v.cur.Lookup(id.Name)
			if symbol.Flag != VarEntry { // report error if this symbol is not a variable
				panic(fmt.Errorf("%s not a variable", id.Loc.ToString()))
			}
			id.Symbol = symbol
		} else {
			id.Symbol = NewSymbolEntry(id.Loc, id.Name, VarEntry, nil, nil)
			v.cur.AddSymbol(id.Symbol)
			nNew++
		}
		lhs = append(lhs, id)
	}

	// Report error if no new variables are declared
	if nNew == 0 {
		panic(NewSemaError(
			NewLocFromContext(ctx),
			fmt.Sprintf("no new variables are declared"),
		))
	}

	// Add statement to current function
	stmt := NewInitStmt(NewLocFromContext(ctx), lhs, exprList)
	return stmt
}

func (v *Builder) VisitReturnStmt(ctx *ReturnStmtContext) interface{} {
	exprList := make([]IExprNode, 0)
	if exprCtx := ctx.ExpressionList(); exprCtx != nil {
		exprList = v.VisitExpressionList(exprCtx.(*ExpressionListContext)).([]IExprNode)
	}
	v.addStmt(NewReturnStmt(NewLocFromContext(ctx), exprList, v.cur.Func))
	return nil
}

func (v *Builder) VisitBreakStmt(ctx *BreakStmtContext) interface{} {
	blocks := v.getBlocksOfCurFunc()
	var target IStmtNode
FindStmt:
	for i := len(blocks) - 1; i >= 0; i-- { // find the innermost target
		switch ctrl := blocks[i].Ctrl; ctrl.(type) {
		case *ForClauseStmt:
			target = ctrl
			break FindStmt
		}
	}
	loc := NewLocFromContext(ctx)
	if target == nil {
		panic(NewSemaError(loc, "cannot find break target"))
	}
	v.addStmt(NewBreakStmt(loc, target))
	return nil
}

func (v *Builder) VisitContinueStmt(ctx *ContinueStmtContext) interface{} {
	blocks := v.getBlocksOfCurFunc()
	var target IStmtNode
FindStmt:
	for i := len(blocks) - 1; i >= 0; i-- { // find the innermost target
		switch ctrl := blocks[i].Ctrl; ctrl.(type) {
		case *ForClauseStmt:
			target = ctrl
			break FindStmt
		}
	}
	loc := NewLocFromContext(ctx)
	if target == nil {
		panic(NewSemaError(loc, "cannot find continue target"))
	}
	v.addStmt(NewContinueStmt(loc, target))
	return nil
}

func (v *Builder) VisitIfStmt(ctx *IfStmtContext) interface{} {
	// Visit if clause
	var init IStmtNode
	v.pushScope(NewLocalScope(v.cur)) // enter if clause scope
	if s := ctx.SimpleStmt(); s != nil {
		init = v.VisitSimpleStmt(s.(*SimpleStmtContext)).(IStmtNode)
	}
	cond := v.VisitExpression(ctx.Expression().(*ExpressionContext)).(IExprNode)

	// Visit if block
	v.pushScope(NewLocalScope(v.cur)) // enter if block scope
	blockCtx := ctx.Block(0).(*BlockContext)
	blockStmt := NewBlockStmt(NewLocFromContext(blockCtx), v.cur)
	v.pushBlock(blockStmt)
	v.VisitBlock(blockCtx)
	v.popBlock()
	v.popScope() // exit block scope

	// Visit else case
	var els IStmtNode
	if s := ctx.IfStmt(); s != nil {
		els = v.VisitIfStmt(s.(*IfStmtContext)).(*IfStmt)
	} else if s := ctx.Block(1); s != nil {
		v.pushScope(NewLocalScope(v.cur)) // enter else block scope
		block := NewBlockStmt(NewLocFromContext(ctx), v.cur)
		v.pushBlock(block)
		els = block
		v.VisitBlock(s.(*BlockContext))
		v.popBlock()
		v.popScope() // exit else block scope
	}

	// Finish if statement
	v.popScope() // exit if clause scope
	v.addStmt(NewIfStmt(NewLocFromContext(ctx), init, cond, blockStmt, els))
	return nil
}

func (v *Builder) VisitForStmt(ctx *ForStmtContext) interface{} {
	// Visit for clause
	var init, post IStmtNode
	var cond IExprNode
	v.pushScope(NewLocalScope(v.cur)) // enter initialization scope
	if c := ctx.Expression(); c != nil {
		cond = v.VisitExpression(c.(*ExpressionContext)).(IExprNode)
	} else if c := ctx.ForClause(); c != nil {
		clause := v.VisitForClause(c.(*ForClauseContext)).(*ForClause)
		init, cond, post = clause.init, clause.cond, clause.post
	}

	// Visit block
	v.pushScope(NewLocalScope(v.cur)) // enter block scope
	blockCtx := ctx.Block().(*BlockContext)
	blockStmt := NewBlockStmt(NewLocFromContext(blockCtx), v.cur)
	// A for block may contain break statement. To ensure this for statement is found,
	// its node must be added before block is visited
	v.addStmt(NewForClauseStmt(NewLocFromContext(ctx), init, cond, post, blockStmt))
	v.pushBlock(blockStmt)
	v.VisitBlock(blockCtx)
	v.popBlock()
	v.popScope() // exit block scope
	v.popScope() // exit initialization scope

	return nil
}

func (v *Builder) VisitForClause(ctx *ForClauseContext) interface{} {
	var init, post IStmtNode
	var cond IExprNode
	if initCtx := ctx.SimpleStmt(0); initCtx != nil {
		init = v.VisitSimpleStmt(initCtx.(*SimpleStmtContext)).(IStmtNode)
	}
	if postCtx := ctx.SimpleStmt(1); postCtx != nil {
		post = v.VisitSimpleStmt(postCtx.(*SimpleStmtContext)).(IStmtNode)
	}
	if condCtx := ctx.Expression(); condCtx != nil {
		cond = v.VisitExpression(condCtx.(*ExpressionContext)).(IExprNode)
	}
	return &ForClause{init: init, cond: cond, post: post}
}

func (v *Builder) VisitTp(ctx *TpContext) interface{} {
	if tp := ctx.TypeName(); tp != nil {
		return v.VisitTypeName(tp.(*TypeNameContext)).(IType)
	} else if tp := ctx.TypeLit(); tp != nil {
		return v.VisitTypeLit(tp.(*TypeLitContext)).(IType)
	} else if tp := ctx.Tp(); tp != nil {
		return v.VisitTp(tp.(*TpContext)).(IType)
	}
	return nil // IType
}

func (v *Builder) VisitTypeName(ctx *TypeNameContext) interface{} {
	// Check if is primitive type
	name := ctx.IDENTIFIER().GetText()
	tp, ok := StrToPrimType[name]
	if ok {
		return NewPrimType(NewLocFromContext(ctx), tp)
	}

	// Try to resolve type symbol
	// It's OK to be unresolved, since it may be defined later in global scope.
	symbol, _ := v.cur.Lookup(name)
	if symbol != nil && symbol.Flag == TypeEntry { // resolved and represents type
		return symbol.Type
	}

	return NewUnresolvedType(NewLocFromContext(ctx), name) // cannot resolve at present
}

func (v *Builder) VisitTypeLit(ctx *TypeLitContext) interface{} {
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

func (v *Builder) VisitArrayType(ctx *ArrayTypeContext) interface{} {
	elem := v.VisitElementType(ctx.ElementType().(*ElementTypeContext)).(IType)
	lenExpr := v.VisitArrayLength(ctx.ArrayLength().(*ArrayLengthContext)).(IExprNode)
	return NewArrayType(NewLocFromContext(ctx), elem, lenExpr)
}

func (v *Builder) VisitArrayLength(ctx *ArrayLengthContext) interface{} {
	return v.VisitExpression(ctx.Expression().(*ExpressionContext)).(*ConstExpr)
}

func (v *Builder) VisitElementType(ctx *ElementTypeContext) interface{} {
	return v.VisitTp(ctx.Tp().(*TpContext)).(IType)
}

func (v *Builder) VisitPointerType(ctx *PointerTypeContext) interface{} {
	return NewPtrType(NewLocFromContext(ctx), v.VisitTp(ctx.Tp().(*TpContext)).(IType))
}

func (v *Builder) VisitSliceType(ctx *SliceTypeContext) interface{} {
	return NewSliceType(NewLocFromContext(ctx),
		v.VisitElementType(ctx.ElementType().(*ElementTypeContext)).(IType))
}

func (v *Builder) VisitMapType(ctx *MapTypeContext) interface{} {
	key := v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	val := v.VisitElementType(ctx.ElementType().(*ElementTypeContext)).(IType)
	return NewMapType(NewLocFromContext(ctx), key, val)
}

func (v *Builder) VisitFunctionType(ctx *FunctionTypeContext) interface{} {
	sig := v.VisitSignature(ctx.Signature().(*SignatureContext)).(*FuncSignature)
	paramType := make([]IType, 0)
	resultType := make([]IType, 0)
	for _, p := range sig.params {
		paramType = append(paramType, p.Type)
	}
	for _, r := range sig.results {
		resultType = append(resultType, r.Type)
	}
	return NewFunctionType(NewLocFromContext(ctx), paramType, resultType)
}

func (v *Builder) VisitSignature(ctx *SignatureContext) interface{} {
	params := v.VisitParameters(ctx.Parameters().(*ParametersContext)).([]*TableEntry)
	results := make([]*TableEntry, 0)
	if r := ctx.Result(); r != nil {
		results = v.VisitResult(r.(*ResultContext)).([]*TableEntry)
	}
	return &FuncSignature{params: params, results: results}
}

func (v *Builder) VisitResult(ctx *ResultContext) interface{} {
	if r := ctx.Tp(); r != nil {
		tp := v.VisitTp(r.(*TpContext)).(IType)
		return []*TableEntry{
			NewSymbolEntry(NewLocFromContext(ctx), "", VarEntry, tp, nil)}
	} else if r := ctx.Parameters(); r != nil {
		return v.VisitParameters(r.(*ParametersContext)).([]*TableEntry)
	}
	return nil // []*SymbolEntry
}

func (v *Builder) VisitParameters(ctx *ParametersContext) interface{} {
	if p := ctx.ParameterList(); p != nil {
		return v.VisitParameterList(p.(*ParameterListContext)).([]*TableEntry)
	} else {
		return make([]*TableEntry, 0)
	} // []*SymbolEntry
}

func (v *Builder) VisitParameterList(ctx *ParameterListContext) interface{} {
	list := make([]*TableEntry, 0)
	for _, d := range ctx.AllParameterDecl() {
		list = append(list, v.VisitParameterDecl(d.(*ParameterDeclContext)).([]*TableEntry)...)
	}
	return list
}

func (v *Builder) VisitParameterDecl(ctx *ParameterDeclContext) interface{} {
	tp := v.VisitTp(ctx.Tp().(*TpContext)).(IType)
	if idListCxt := ctx.IdentifierList(); idListCxt == nil {
		return []*TableEntry{
			NewSymbolEntry(NewLocFromContext(ctx), "", VarEntry, tp, 0)}
	} else {
		idList := v.VisitIdentifierList(idListCxt.(*IdentifierListContext)).([]*IdExpr)
		declList := make([]*TableEntry, 0)
		for _, id := range idList {
			declList = append(declList, NewSymbolEntry(id.Loc, id.Name, VarEntry, tp, 0))
		}
		return declList // []*SymbolEntry
	}
}

func (v *Builder) VisitOperand(ctx *OperandContext) interface{} {
	if o := ctx.Literal(); o != nil {
		return v.VisitLiteral(o.(*LiteralContext)).(IExprNode)
	} else if o := ctx.OperandName(); o != nil {
		return v.VisitOperandName(o.(*OperandNameContext)).(IExprNode)
	} else if o := ctx.Expression(); o != nil {
		return v.VisitExpression(o.(*ExpressionContext)).(IExprNode)
	}
	return nil // IExprNode
}

func (v *Builder) VisitLiteral(ctx *LiteralContext) interface{} {
	if l := ctx.BasicLit(); l != nil {
		return v.VisitBasicLit(l.(*BasicLitContext)).(IExprNode)
	} else if l := ctx.CompositeLit(); l != nil {
		return v.VisitCompositeLit(l.(*CompositeLitContext)).(IExprNode)
	} else if l := ctx.FunctionLit(); l != nil {
		return v.VisitFunctionLit(l.(*FunctionLitContext)).(IExprNode)
	}
	return nil
}

func (v *Builder) VisitBasicLit(ctx *BasicLitContext) interface{} {
	if l := ctx.INT_LIT(); l != nil {
		lit, _ := strconv.ParseInt(l.GetText(), 0, 0)
		return NewIntConst(NewLocFromTerminal(l), int(lit))
	} else if l = ctx.FLOAT_LIT(); l != nil {
		lit, _ := strconv.ParseFloat(l.GetText(), 64)
		return NewFloatConst(NewLocFromTerminal(l), lit)
	}
	return nil
}

func (v *Builder) VisitOperandName(ctx *OperandNameContext) interface{} {
	// Create operand identifier
	name := ctx.IDENTIFIER().GetText()
	if name == "nil" { // return nil value if operand is "nil"
		return NewZeroValue(NewLocFromContext(ctx))
	}
	symbol, scope := v.cur.Lookup(name) // can be nil then
	id := NewIdExpr(NewLocFromContext(ctx), name, symbol)

	// Decide if it should be captured
	id.Captured = scope != nil && !scope.Global && scope.Func != v.cur.Func
	v.cur.AddOperandId(id)

	return id
}

func (v *Builder) VisitCompositeLit(ctx *CompositeLitContext) interface{} {
	litType := v.VisitLiteralType(ctx.LiteralType().(*LiteralTypeContext)).(IType)
	elemList := v.VisitLiteralValue(ctx.LiteralValue().(*LiteralValueContext)).(*ElemList)
	return NewCompLit(NewLocFromContext(ctx), litType, elemList)
}

func (v *Builder) VisitLiteralType(ctx *LiteralTypeContext) interface{} {
	if t := ctx.StructType(); t != nil {
		return v.VisitStructType(t.(*StructTypeContext)).(IType)
	} else if t := ctx.ArrayType(); t != nil {
		return v.VisitArrayType(t.(*ArrayTypeContext)).(IType)
	} else if t := ctx.SliceType(); t != nil {
		return v.VisitSliceType(t.(*SliceTypeContext)).(IType)
	} else if t := ctx.MapType(); t != nil {
		return v.VisitMapType(t.(*MapTypeContext)).(IType)
	} else if t := ctx.TypeName(); t != nil {
		return v.VisitTypeName(t.(*TypeNameContext)).(IType)
	}
	return nil
}

func (v *Builder) VisitStructType(ctx *StructTypeContext) interface{} {
	table := NewSymbolTable()
	for _, f := range ctx.AllFieldDecl() {
		table.Add(v.VisitFieldDecl(f.(*FieldDeclContext)).([]*TableEntry)...)
	}
	return NewStructType(NewLocFromContext(ctx), table) // *StructType
}

func (v *Builder) VisitLiteralValue(ctx *LiteralValueContext) interface{} {
	if e := ctx.ElementList(); e != nil {
		return v.VisitElementList(e.(*ElementListContext)).(*ElemList)
	} else {
		return NewElemList(make([]*LitElem, 0), false)
	}
}

func (v *Builder) VisitElementList(ctx *ElementListContext) interface{} {
	elem := make([]*LitElem, 0)
	for _, e := range ctx.AllKeyedElement() {
		elem = append(elem, v.VisitKeyedElement(e.(*KeyedElementContext)).(*LitElem))
	}
	keyed := elem[0].IsKeyed() // check if there are mixed keyed and unkeyed elements
	for _, v := range elem {
		if (keyed && !v.IsKeyed()) || (!keyed && v.IsKeyed()) {
			panic(NewSemaError(v.Loc, "%s mixed keyed and unkeyed elements"))
		}
	}
	return NewElemList(elem, keyed)
}

func (v *Builder) VisitKeyedElement(ctx *KeyedElementContext) interface{} {
	var key IExprNode
	if k := ctx.Key(); k != nil {
		key = v.VisitKey(k.(*KeyContext)).(IExprNode)
	}
	elem := v.VisitElement(ctx.Element().(*ElementContext)).(IExprNode)
	return NewLitElem(NewLocFromContext(ctx), key, elem)
}

func (v *Builder) VisitKey(ctx *KeyContext) interface{} {
	if k := ctx.IDENTIFIER(); k != nil {
		return NewIdExpr(NewLocFromTerminal(k), k.GetText(), nil)
	} else if k := ctx.Expression(); k != nil {
		return v.VisitExpression(k.(*ExpressionContext)).(IExprNode)
	} else if k := ctx.LiteralValue(); k != nil {
		return v.VisitLiteralValue(k.(*LiteralValueContext)).(IExprNode)
	}
	return nil
}

func (v *Builder) VisitElement(ctx *ElementContext) interface{} {
	if k := ctx.Expression(); k != nil {
		return v.VisitExpression(k.(*ExpressionContext)).(IExprNode)
	} else if k := ctx.LiteralValue(); k != nil {
		return v.VisitLiteralValue(k.(*LiteralValueContext)).(IExprNode)
	}
	return nil
}

func (v *Builder) VisitFieldDecl(ctx *FieldDeclContext) interface{} {
	if l := ctx.IdentifierList(); l != nil {
		idList := v.VisitIdentifierList(l.(*IdentifierListContext)).([]*IdExpr)
		tp := v.VisitTp(ctx.Tp().(*TpContext)).(IType)
		declList := make([]*TableEntry, 0)
		for _, id := range idList {
			declList = append(declList, NewSymbolEntry(id.Loc, id.Name, VarEntry, tp, nil))
		}
		return declList // []*SymbolEntry
	}
	return nil
}

func (v *Builder) VisitFunctionLit(ctx *FunctionLitContext) interface{} {
	// Get declaration from function context
	decl := v.VisitFunction(ctx.Function().(*FunctionContext)).(*FuncDecl)

	// Create lambda capture set (identifiers in different blocks may repeat)
	// Visit scopes of declared function and its nested scopes (excluding the scopes of its nested
	// functions), using DFS
	closureSet := make(map[*TableEntry]bool, 0)
	stack := []*Scope{decl.Scope}
	for len(stack) > 0 {
		// Process operand identifiers in current scope
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1] // pop an element from stack
		for id := range top.operandId {
			if id.Captured {
				closureSet[id.Symbol] = true
			}
		}

		// Push children scopes to stack, within the function
		for _, child := range top.Children {
			if child.Func == decl {
				stack = append(stack, child)
			}
		}
	}

	// Convert set to symbol table (to merge repeated identifiers)
	closureTable := NewSymbolTable()
	for entry := range closureSet {
		closureTable.Add(entry)
	}

	return NewFuncLit(NewLocFromContext(ctx), decl, closureTable)
}

func (v *Builder) VisitPrimaryExpr(ctx *PrimaryExprContext) interface{} {
	if e := ctx.Operand(); e != nil {
		return v.VisitOperand(e.(*OperandContext)).(IExprNode)
	} else if e := ctx.Conversion(); e != nil {
		return v.VisitConversion(e.(*ConversionContext)).(IExprNode)
	}

	// Must be a recursive call then
	prim := v.VisitPrimaryExpr(ctx.PrimaryExpr().(*PrimaryExprContext)).(IExprNode)
	if e := ctx.Arguments(); e != nil {
		args := v.VisitArguments(e.(*ArgumentsContext)).([]IExprNode)
		return NewFuncCallExpr(NewLocFromContext(ctx), prim, args)
	}

	return nil // IExprNode
}

func (v *Builder) VisitArguments(ctx *ArgumentsContext) interface{} {
	return v.VisitExpressionList(ctx.ExpressionList().(*ExpressionListContext)).([]IExprNode)
}

func (v *Builder) VisitExpression(ctx *ExpressionContext) interface{} {
	// Forward unary expression
	if e := ctx.UnaryExpr(); e != nil {
		return v.VisitUnaryExpr(e.(*UnaryExprContext)).(IExprNode)
	}

	// Deal with binary expression
	op := BinaryOpStrToEnum[ctx.GetOp().GetText()]
	left := v.VisitExpression(ctx.Expression(0).(*ExpressionContext)).(IExprNode)
	right := v.VisitExpression(ctx.Expression(1).(*ExpressionContext)).(IExprNode)

	return NewBinaryExpr(NewLocFromContext(ctx), op, left, right) // IExprNode
}

func (v *Builder) VisitUnaryExpr(ctx *UnaryExprContext) interface{} {
	// Forward primary expression
	if e := ctx.PrimaryExpr(); e != nil {
		return v.VisitPrimaryExpr(e.(*PrimaryExprContext)).(IExprNode)
	}

	// Deal with unary expression
	op := UnaryOpStrToEnum[ctx.GetOp().GetText()]
	prim := v.VisitUnaryExpr(ctx.UnaryExpr().(*UnaryExprContext)).(IExprNode)

	return NewUnaryExpr(NewLocFromContext(ctx), op, prim) // IExprNode
}
