package ir

import (
	"ast"
	"fmt"
)

type Program struct {
	Name string
	// Scope of global variables
	// Note that global function can also has its local scope. Global scope is not the
	// same as the scope of global function.
	Global *Scope
	// List of all compiled functions
	// The first function is global function, which should be treated specially.
	Funcs []*Func
	// Basic block counter (to distinguish labels)
	nBlock int
	// Temporary operand counter (to distinguish temporaries)
	nTmp int
}

type ClosureEnv struct {
	// Environment item specification
	symbols []*Symbol
	// The struct type descriptor of capture list
	tp *StructType
}

type LoopInfo struct {
	Continue, Break *BasicBlock
}

type Builder struct {
	ast.BaseVisitor
	// Target program
	prg *Program
	// Current function being visited
	fun *Func
	// Current basic block being built
	bb *BasicBlock
	// Look up table for functions
	funcTable map[*ast.FuncDecl]*Func
	// Function literal counter
	nFuncLit int
	// AST scope counter (to distinguish symbols in different scope). It is not for IR scope.
	nScope int
	// Continue and break targets for AST loop statement
	loopInfo map[ast.IStmtNode]LoopInfo
}

func NewBuilder() *Builder {
	return &Builder{
		funcTable: make(map[*ast.FuncDecl]*Func), // the first function is global
		loopInfo:  make(map[ast.IStmtNode]LoopInfo),
	}
}

func (b *Builder) VisitProgram(node *ast.ProgramNode) interface{} {
	// Generate signature of all top level functions, in case they are referenced.
	b.prg = &Program{
		Name:   node.PkgName,
		Funcs:  []*Func{b.genSignature(node.Global)},
		nBlock: 0,
		nTmp:   0,
	}
	for _, decl := range node.Funcs {
		sig := b.genSignature(decl)
		b.prg.Funcs = append(b.prg.Funcs, sig)
		b.funcTable[decl] = sig
	}

	// Visit global function
	b.prg.Global = NewGlobalScope() // set global scope pointer of program
	b.VisitScope(node.Global.Scope) // add symbols to global scope
	b.fun = b.prg.Funcs[0]
	b.VisitFuncDecl(node.Global) // visit global function

	// Visit other top level functions
	for i, decl := range node.Funcs {
		b.fun = b.prg.Funcs[i+1] // skip global function
		b.VisitFuncDecl(decl)
	}

	return b.prg
}

func (b *Builder) VisitFuncDecl(decl *ast.FuncDecl) interface{} {
	b.fun.Scope = NewLocalScope() // build function scope
	if !decl.Scope.Global {
		b.VisitScope(decl.Scope) // global scope has already been visited
	}
	b.fun.Enter = b.newBasicBlock("Start") // create entrance block
	b.bb = b.fun.Enter
	for _, stmt := range decl.Stmts {
		b.VisitStmt(stmt)
	}
	return nil
}

func (b *Builder) requestBlockName() string {
	str := fmt.Sprintf("_B%d", b.prg.nBlock)
	b.prg.nBlock++
	return str
}

func (b *Builder) newBasicBlock(tag string) *BasicBlock {
	return NewBasicBlock(fmt.Sprintf("%s_%s", b.requestBlockName(), tag), b.fun)
}

func (b *Builder) VisitScope(astScope *ast.Scope) interface{} {
	idx := 0 // index in AST function scope
	fun := astScope.Func
	entries := astScope.Symbols.Entries
	nested := astScope.Func.Scope != astScope
	var scope *Scope
	if astScope.Global {
		scope = b.prg.Global
	} else {
		scope = b.fun.Scope
	}

	if nested { // nested scope
		goto LocalSymbols
	}

	// Add receiver to IR scope and its parameter list
	if fun.Type.Receiver != nil {
		symbol := entries[idx]
		scope.AddFromAST(symbol, b.genSymbolName(symbol.Name),
			b.VisitType(symbol.Type).(IType), true)
		idx++
	}

	// Add normal parameters to IR scope and its parameter list
	for k := 0; k < len(fun.Type.Param.Elem); k++ {
		symbol := entries[idx]
		scope.AddFromAST(symbol, b.genSymbolName(symbol.Name),
			b.VisitType(symbol.Type).(IType), true)
		idx++
	}

	// Add function environment to IR scope and parameter list
	if !astScope.Global {
		scope.AddSymbol("_env", NewPtrType(NewBaseType(Void)), true)
	}

	// Add local symbols to IR scope
LocalSymbols:
	for ; idx < len(astScope.Symbols.Entries); idx++ {
		symbol := entries[idx]
		if symbol.Flag == ast.VarEntry {
			scope.AddFromAST(symbol, b.genSymbolName(symbol.Name),
				b.VisitType(symbol.Type).(IType), false)
		}
	}
	if nested {
		goto EndScope
	}

	// Add captured symbols to IR scope, if this scope is from a closure
	if astScope.Func.Lit != nil {
		lit := astScope.Func.Lit
		for _, symbol := range lit.Closure.Entries {
			scope.AddFromAST(symbol, b.genSymbolName(symbol.Name),
				b.VisitType(symbol.Type).(IType), false)
		}
	}

EndScope:
	b.nScope++
	return nil
}

func (b *Builder) genSymbolName(name string) string {
	return fmt.Sprintf("_s%d_%s", b.nScope, name)
}

func (b *Builder) genSignature(decl *ast.FuncDecl) *Func {
	var funcType *FuncType
	if decl.Scope.Global { // treat global function specially
		funcType = NewFuncType([]IType{}, NewStructType([]IType{}))
	} else {
		funcType = b.VisitFuncType(decl.Type).(*StructType).Field[0].Type.(*FuncType)
	}
	return NewFunc(funcType, decl.Name, nil)
}

func (b *Builder) VisitStmt(stmt ast.IStmtNode) interface{} {
	switch stmt.(type) {
	case ast.IExprNode:
		b.VisitExpr(stmt.(ast.IExprNode))
	case *ast.BlockStmt:
		b.VisitBlockStmt(stmt.(*ast.BlockStmt))
	case *ast.AssignStmt:
		b.VisitAssignStmt(stmt.(*ast.AssignStmt))
	case *ast.IncDecStmt:
		b.VisitIncDecStmt(stmt.(*ast.IncDecStmt))
	case *ast.ReturnStmt:
		b.VisitReturnStmt(stmt.(*ast.ReturnStmt))
	case *ast.IfStmt:
		b.VisitIfStmt(stmt.(*ast.IfStmt))
	case *ast.ForClauseStmt:
		b.VisitForClauseStmt(stmt.(*ast.ForClauseStmt))
	case *ast.BreakStmt:
		b.VisitBreakStmt(stmt.(*ast.BreakStmt))
	case *ast.ContinueStmt:
		b.VisitContinueStmt(stmt.(*ast.ContinueStmt))
	}
	return nil
}

// AST blocks just set up new scope, they have nothing to do with basic blocks in IR
func (b *Builder) VisitBlockStmt(stmt *ast.BlockStmt) interface{} {
	b.VisitScope(stmt.Scope)
	for _, s := range stmt.Stmts {
		b.VisitStmt(s)
	}
	return nil
}

func (b *Builder) emit(instr IInstr) { b.bb.Append(instr) }

func (b *Builder) VisitAssignStmt(stmt *ast.AssignStmt) interface{} {
	// Deal with function returns
	if stmt.Rhs[0].GetType().GetTypeEnum() == ast.Tuple {
		ret := b.VisitFuncCallExpr(stmt.Rhs[0].(*ast.FuncCallExpr)).(*SymbolValue)
		for i := range ret.GetType().(*StructType).Field {
			fieldVal := b.loadField(ret, i)
			b.moveToDst(stmt.Lhs[i], fieldVal) // move to destination
		}
		return nil
	}

	// Process assignment with possibly multiple values
	interList := make([]IValue, 0)
	for _, r := range stmt.Rhs { // move from source to intermediates
		if _, ok := r.(*ast.ZeroValue); ok {
			interList = append(interList, nil)
			continue
		}
		inter := b.newTempSymbol(b.VisitType(r.GetType()).(IType))
		interList = append(interList, inter)
		b.moveFromSrc(r, inter)
	}
	for i, l := range stmt.Lhs {
		b.moveToDst(l, interList[i])
	}

	return nil
}

// Temporary IR values serves as an intermediate. The emitted instruction depends on the
// expression type of destination.
func (b *Builder) moveFromSrc(srcNode ast.IExprNode, inter IValue) {
	srcRet := b.VisitExpr(srcNode)

	switch srcRet.(type) {
	case IValue:
		b.emit(NewMove(srcRet.(IValue), inter))

	case *GetPtr:
		srcPtr := srcRet.(*GetPtr).Result
		b.emit(srcRet.(*GetPtr))       // emit that getptr instruction
		b.emit(NewLoad(srcPtr, inter)) // load value from pointer
	}
}

func (b *Builder) moveToDst(dstNode ast.IExprNode, inter IValue) {
	// DEREF has different semantic on different sides of assignment
	if unary, ok := dstNode.(*ast.UnaryExpr); ok {
		// DEREF is the only possible unary operation on left hand side
		dstRet := b.VisitExpr(unary.Expr)

		switch dstRet.(type) {
		case IValue: // an symbol value, must be a pointer type (*p = s)
			b.emit(NewStore(inter, dstRet.(IValue))) // store to the pointer

		case *GetPtr: // an get pointer instruction (*p.f = s)
			getPtr := dstRet.(*GetPtr)
			b.emit(getPtr)            // emit that instruction
			fieldPtr := getPtr.Result // get pointer to that pointer field
			field := b.newTempSymbol(fieldPtr.GetType().(*PtrType).Base)
			b.emit(NewLoad(fieldPtr, field)) // get pointer field
			b.emit(NewStore(inter, field))   // store to that field
		}

	} else {
		dstRet := b.VisitExpr(dstNode)

		switch dstRet.(type) {
		case IValue: // d = s
			dstVal := dstRet.(IValue)
			if inter == nil {
				b.emit(NewClear(dstVal))
			} else {
				b.emit(NewMove(inter, dstVal)) // move between symbols
			}

		case *GetPtr: // d.f = s
			instr := dstRet.(*GetPtr)
			b.emit(instr)
			b.emit(NewStore(inter, instr.Result)) // store to field pointer
		}
	}
}

func (b *Builder) newTempSymbol(tp IType) *SymbolValue {
	name := fmt.Sprintf("_t%d", b.prg.nTmp)
	b.prg.nTmp++
	sym := b.fun.Scope.AddTemp(name, tp)
	return NewSymbolValue(sym)
}

func (b *Builder) VisitIncDecStmt(stmt *ast.IncDecStmt) interface{} {
	exprRet := b.VisitExpr(stmt.Expr)
	var op BinaryOp
	if stmt.Inc {
		op = ADD
	} else {
		op = SUB
	}
	switch exprRet.(type) {
	case *SymbolValue:
		val := exprRet.(*SymbolValue)
		b.emit(NewBinary(op, val, NewI64Imm(1), val))
	case *GetPtr:
		instr := exprRet.(*GetPtr)
		b.emit(instr)
		valType := instr.Result.GetType().(*PtrType).Base
		val := b.newTempSymbol(valType)
		b.emit(NewLoad(instr.Result, val))
		b.emit(NewBinary(op, val, NewI64Imm(1), val))
		b.emit(NewStore(val, instr.Result))
	}
	return nil
}

func (b *Builder) VisitReturnStmt(stmt *ast.ReturnStmt) interface{} {
	ret := make([]IValue, 0)
	for _, r := range stmt.Expr {
		ret = append(ret, b.retrieveValue(b.VisitExpr(r)))
	}
	b.emit(NewReturn(b.fun, ret))
	return nil
}

func (b *Builder) VisitIfStmt(stmt *ast.IfStmt) interface{} {
	// Initialize basic blocks
	b.VisitScope(stmt.Scope)
	if stmt.Init != nil {
		b.VisitStmt(stmt.Init)
	}
	trueBB := b.newBasicBlock("IfThen")
	nextBB := b.newBasicBlock("IfNext")
	var falseBB *BasicBlock
	hasElse := stmt.Else != nil
	if hasElse {
		falseBB = b.newBasicBlock("IfFalse")
	} else {
		falseBB = nextBB
	}

	// Use short circuit evaluation
	b.shortCircuitCtrl(stmt.Cond, trueBB, falseBB)

	// Build IR in then and else clause
	b.bb = trueBB
	b.VisitBlockStmt(stmt.Then)
	b.bb.JumpTo(nextBB)
	if hasElse {
		b.bb = falseBB
		b.VisitStmt(stmt.Else)
		b.bb.JumpTo(nextBB)
	}
	b.bb = nextBB

	return nil
}

func (b *Builder) VisitForClauseStmt(stmt *ast.ForClauseStmt) interface{} {
	// Initialize basic blocks
	b.VisitScope(stmt.Scope)
	if stmt.Init != nil {
		b.VisitStmt(stmt.Init)
	}
	initBB := b.bb
	bodyBB := b.newBasicBlock("ForBody")
	postBB := b.newBasicBlock("ForPost") // instructions to be executed after main body
	nextBB := b.newBasicBlock("ForNext") // the basic block after for statement
	hasCond := stmt.Cond != nil
	var newIteBB *BasicBlock
	if hasCond {
		condBB := b.newBasicBlock("ForCond")
		newIteBB = condBB
		initBB.JumpTo(condBB)                         // init -> cond
		b.bb = condBB                                 // instructions must all be in a new block
		b.shortCircuitCtrl(stmt.Cond, bodyBB, nextBB) // cond ? body : next
	} else {
		newIteBB = bodyBB
		initBB.JumpTo(bodyBB) // init -> body
	}

	// Visit main body
	b.loopInfo[stmt] = LoopInfo{
		Continue: postBB,
		Break:    nextBB,
	}
	b.bb = bodyBB
	b.VisitStmt(stmt.Block)
	b.bb.JumpTo(postBB) // current basic block may not be the body defined before
	b.bb = postBB
	if stmt.Post != nil {
		b.VisitStmt(stmt.Post)
	}
	postBB.JumpTo(newIteBB) // post -> newIte (cond/body)
	b.bb = nextBB

	return nil
}

func (b *Builder) VisitBreakStmt(stmt *ast.BreakStmt) interface{} {
	target := b.loopInfo[stmt.Target].Break
	b.bb.JumpTo(target)
	b.bb = b.newBasicBlock("BreakNext") // just in case there are following statements
	return nil
}

func (b *Builder) VisitContinueStmt(stmt *ast.ContinueStmt) interface{} {
	target := b.loopInfo[stmt.Target].Continue
	b.bb.JumpTo(target)
	b.bb = b.newBasicBlock("ContinueNext")
	return nil
}

// All expression visiting methods return result operand
func (b *Builder) VisitExpr(expr ast.IExprNode) interface{} {
	if canUseShortCircuit(expr) {
		return b.shortCircuitEval(expr)
	}
	switch expr.(type) {
	case ast.ILiteralExpr:
		return b.VisitLiteralExpr(expr.(ast.ILiteralExpr))
	case *ast.IdExpr:
		return b.VisitIdExpr(expr.(*ast.IdExpr))
	case *ast.FuncCallExpr:
		return b.VisitFuncCallExpr(expr.(*ast.FuncCallExpr))
	case *ast.SelectExpr:
		return b.VisitSelectExpr(expr.(*ast.SelectExpr))
	case *ast.IndexExpr:
		return b.VisitIndexExpr(expr.(*ast.IndexExpr))
	case *ast.UnaryExpr:
		return b.VisitUnaryExpr(expr.(*ast.UnaryExpr))
	case *ast.BinaryExpr:
		return b.VisitBinaryExpr(expr.(*ast.BinaryExpr))
	}
	return nil
}

func (b *Builder) VisitLiteralExpr(expr ast.ILiteralExpr) interface{} {
	switch expr.(type) {
	case *ast.ConstExpr:
		return b.VisitConstExpr(expr.(*ast.ConstExpr))
	case *ast.FuncLit:
		return b.VisitFuncLit(expr.(*ast.FuncLit))
	}
	return nil
}

func (b *Builder) VisitFuncLit(expr *ast.FuncLit) interface{} {
	// Backup outer function context
	prevFunc := b.fun
	prevBB := b.bb

	// Build function scope
	scope := NewLocalScope()
	closureType := b.VisitType(expr.Type).(*StructType)
	funcLit := NewFunc(closureType.Field[0].Type.(*FuncType), b.requestFuncLitName(), scope)
	b.fun = funcLit
	b.prg.Funcs = append(b.prg.Funcs, b.fun) // add to program function list
	b.funcTable[expr.Decl] = funcLit
	b.VisitScope(expr.Decl.Scope)

	// Build capture list
	params := b.fun.Scope.Params
	capList := &ClosureEnv{
		symbols: make([]*Symbol, 0),
		tp:      nil,
	}
	field := make([]IType, 0)
	for _, s := range expr.Closure.Entries {
		irSymbol := prevFunc.Scope.astToIr[s]
		capList.symbols = append(capList.symbols, irSymbol)
		field = append(field, irSymbol.Type)
	}
	capList.tp = NewStructType(field)

	// Load captured values to local variables
	// This make captured values behave like normal local variables.
	listPtr := NewSymbolValue(params[len(params)-1])
	for i := range capList.symbols {
		valPtr := b.newTempSymbol(NewPtrType(NewBaseType(Void)))
		b.emit(NewPtrOffset(listPtr, valPtr, capList.tp.Field[i].Offset))
		b.emit(NewLoad(valPtr, NewSymbolValue(capList.symbols[i])))
	}

	// Visit statements and build IR
	b.fun.Enter = NewBasicBlock(b.requestBlockName(), b.fun) // create entrance block
	b.bb = b.fun.Enter
	for _, stmt := range expr.Decl.Stmts {
		b.VisitStmt(stmt)
	}

	// Restore outer function context
	b.fun = prevFunc
	b.bb = prevBB

	// Setup closure in the outer function
	litVal := b.newTempSymbol(closureType)
	// Store function as the first field of closure
	funcPtr := b.newTempSymbol(NewPtrType(closureType.Field[0].Type))
	b.emit(NewGetPtr(litVal, funcPtr, []IValue{NewI64Imm(0)}))
	b.emit(NewStore(funcLit, funcPtr))
	// Allocate heap space for capture list
	mallocRet := b.newTempSymbol(NewPtrType(capList.tp)) // provide size to instruction
	b.emit(NewMalloc(mallocRet))
	// Store captured operands to list
	for i, s := range capList.symbols {
		elemPtr := b.newTempSymbol(NewPtrType(s.Type))
		b.emit(NewPtrOffset(mallocRet, elemPtr, capList.tp.Field[i].Offset))
		b.emit(NewStore(NewSymbolValue(s), elemPtr))
	}
	// Store environment pointer as the second field of closure
	envPtrPtr := b.newTempSymbol(NewPtrType(closureType.Field[1].Type))
	b.emit(NewGetPtr(litVal, envPtrPtr, []IValue{NewI64Imm(1)}))
	b.emit(NewStore(mallocRet, envPtrPtr))

	return litVal
}

func (b *Builder) requestFuncLitName() string {
	name := fmt.Sprintf("_F%d", b.nFuncLit)
	b.nFuncLit++
	return name
}

func (b *Builder) VisitConstExpr(expr *ast.ConstExpr) interface{} {
	switch expr.Type.GetTypeEnum() {
	case ast.Bool:
		return NewI1Imm(expr.Val.(bool))
	case ast.Int:
		return NewI64Imm(expr.Val.(int))
	case ast.Float64:
		return NewF64Imm(expr.Val.(float64))
	}
	return nil
}

func (b *Builder) VisitIdExpr(expr *ast.IdExpr) interface{} {
	switch expr.Symbol.Flag {
	case ast.VarEntry:
		symbol := b.fun.Scope.astToIr[expr.Symbol]
		if symbol == nil {
			symbol = b.prg.Global.astToIr[expr.Symbol]
		}
		return NewSymbolValue(symbol)
	case ast.FuncEntry:
		return b.funcTable[expr.Symbol.Val.(*ast.FuncDecl)]
	}
	return nil
}

func (b *Builder) VisitFuncCallExpr(expr *ast.FuncCallExpr) interface{} {
	// Evaluate function and argument expressions
	fun := b.VisitExpr(expr.Func).(IValue)
	args := make([]IValue, 0)
	for _, e := range expr.Args {
		args = append(args, b.VisitExpr(e).(IValue))
	}

	// Emit instructions depending on function type
	ret := b.newTempSymbol(fun.GetType().(*FuncType).Return)
	if topFunc, ok := fun.(*Func); ok { // top level function
		args = append(args, NewNullPtr()) // capture list is nil
		b.emit(NewCall(topFunc, args, ret))
	} else { // closure (constructed from literals)
		funcAddr := b.loadField(fun, 0)
		capList := b.loadField(fun, 1)
		args = append(args, capList)
		b.emit(NewCall(funcAddr, args, ret))
	}

	return ret
}

func (b *Builder) loadField(value IValue, index int) *SymbolValue {
	structType := value.GetType().(*StructType)
	ptr := b.newTempSymbol(NewPtrType(structType.Field[index].Type))
	b.emit(NewGetPtr(value, ptr, []IValue{NewI64Imm(index)}))
	field := b.newTempSymbol(structType.Field[index].Type)
	b.emit(NewLoad(ptr, field))
	return field
}

func (b *Builder) VisitSelectExpr(expr *ast.SelectExpr) interface{} {
	// Find out the index of member in the struct field
	index := -1
	for i, s := range asAstStructType(expr.Target.GetType()).Field.Entries {
		if s.Name == expr.Member.Name {
			index = i // must be found, otherwise there's logic error in program
			break
		}
	}

	// Visit target expression
	retVal := b.VisitExpr(expr.Target)
	exprType := b.VisitType(expr.Type).(IType)
	switch retVal.(type) {
	case IValue:
		return NewGetPtr(retVal.(IValue), b.newTempSymbol(NewPtrType(exprType)),
			[]IValue{NewI64Imm(index)})
	case *GetPtr:
		return b.appendIndex(retVal.(*GetPtr), NewI64Imm(index), exprType)
	}

	return nil
}

func asAstStructType(tp ast.IType) *ast.StructType {
	if alias, ok := tp.(*ast.AliasType); ok {
		return alias.Under.(*ast.StructType)
	}
	return tp.(*ast.StructType)
}

func (b *Builder) appendIndex(instr *GetPtr, index IValue, elemType IType) *GetPtr {
	prevRes := instr.Result
	switch prevRes.(type) {
	case *SymbolValue:
		symVal := prevRes.(*SymbolValue)
		ptrType := NewPtrType(elemType)
		symVal.Type = ptrType
		symVal.Symbol.Type = ptrType
	default:
		panic(NewIRError("type cannot be changed for non-symbol value"))
	}
	return instr.AppendIndex(index, prevRes)
}

func (b *Builder) VisitIndexExpr(expr *ast.IndexExpr) interface{} {
	arrRet := b.VisitExpr(expr.Array)
	index := b.retrieveValue(b.VisitExpr(expr.Index))
	switch arrRet.(type) {
	case IValue:
		array := arrRet.(IValue)
		return NewGetPtr(
			array, b.newTempSymbol(NewPtrType(array.GetType().(*ArrayType).Elem)),
			[]IValue{index},
		)
	case *GetPtr:
		return b.appendIndex(arrRet.(*GetPtr), index, b.VisitType(expr.Type).(IType))
	}
	return nil
}

func (b *Builder) retrieveValue(obj interface{}) IValue {
	switch obj.(type) {
	case IValue:
		return obj.(IValue)
	case *GetPtr:
		instr := obj.(*GetPtr)
		val := b.newTempSymbol(instr.Result.GetType().(*PtrType).Base)
		b.emit(instr)
		b.emit(NewLoad(instr.Result, val))
		return val
	}
	return nil
}

// Short circuit evaluation for boolean expressions
func (b *Builder) shortCircuitEval(expr ast.IExprNode) interface{} {
	// Build initial basic blocks.
	//      Start
	//    T /   \ F
	//   True   False
	//      \   /
	//      Next
	result := b.newTempSymbol(NewBaseType(I1))
	nextBB := b.newBasicBlock("ShortCircuitNext")
	setTrue := b.newBasicBlock("SetTrue")
	setTrue.Append(NewMove(NewI1Imm(true), result))
	setTrue.JumpTo(nextBB)
	setFalse := b.newBasicBlock("SetFalse")
	setFalse.Append(NewMove(NewI1Imm(false), result))
	setFalse.JumpTo(nextBB)

	// Main recursion
	b.shortCircuitCtrl(expr, setTrue, setFalse)
	b.bb = nextBB

	return result
}

func (b *Builder) shortCircuitCtrl(expr ast.IExprNode, trueBB, falseBB *BasicBlock) {
	// Decide whether short circuit control flow transform should be used on this expression
	if !canUseShortCircuit(expr) {
		// When the expression cannot be divided, branch to the blocks provided in
		// the control info.
		cond := b.retrieveValue(b.VisitExpr(expr))
		b.bb.BranchTo(cond, trueBB, falseBB)
		return
	}

	switch expr.(type) {
	case *ast.UnaryExpr: // NOT operator
		// Invert the control blocks, stay in current basic block.
		b.shortCircuitCtrl(expr.(*ast.UnaryExpr).Expr, falseBB, trueBB)

	case *ast.BinaryExpr:
		binExpr := expr.(*ast.BinaryExpr)
		rightBB := b.newBasicBlock("ShortCircuitRight")
		switch binExpr.Op {
		case ast.LAND:
			b.shortCircuitCtrl(binExpr.Left, rightBB, falseBB) // stays in current basic block
		case ast.LOR:
			b.shortCircuitCtrl(binExpr.Left, trueBB, rightBB)
		}
		b.bb = rightBB
		b.shortCircuitCtrl(binExpr.Right, trueBB, falseBB)
		return
	}
}

func canUseShortCircuit(expr ast.IExprNode) bool {
	if expr.GetType().GetTypeEnum() != ast.Bool {
		return false
	}
	switch expr.(type) {
	case *ast.UnaryExpr:
		return expr.(*ast.UnaryExpr).Op == ast.NOT
	case *ast.BinaryExpr:
		return expr.(*ast.BinaryExpr).Op&(ast.LAND|ast.LOR) != 0
	default:
		return false
	}
}

func (b *Builder) VisitUnaryExpr(expr *ast.UnaryExpr) interface{} {
	val := b.retrieveValue(b.VisitExpr(expr.Expr))
	var res IValue
	switch expr.Op {
	case ast.POS:
		return val
	case ast.NEG:
		res = b.newTempSymbol(val.GetType())
		b.emit(NewUnary(NEG, val, res))
	case ast.NOT, ast.INV:
		res = b.newTempSymbol(val.GetType())
		b.emit(NewUnary(NOT, val, res))
	case ast.DEREF:
		ptrType := val.GetType().(*PtrType)
		res = b.newTempSymbol(ptrType.Base)
		b.emit(NewLoad(val, res))
	case ast.REF:
		res = b.newTempSymbol(NewPtrType(val.GetType()))
		b.emit(NewGetPtr(val, res, []IValue{}))
	}
	return res
}

var astBinOpToIr = map[ast.BinaryOp]BinaryOp{
	ast.ADD: ADD, ast.SUB: SUB, ast.MUL: MUL, ast.DIV: DIV, ast.MOD: MOD,
	ast.SHL: SHL, ast.SHR: SHR, ast.AAND: AND, ast.AOR: OR, ast.XOR: XOR,
	ast.LAND: AND, ast.LOR: OR, ast.EQ: EQ, ast.NE: NE, ast.LT: LT, ast.LE: LE,
	ast.GT: GT, ast.GE: GE,
}

func (b *Builder) VisitBinaryExpr(expr *ast.BinaryExpr) interface{} {
	left := b.retrieveValue(b.VisitExpr(expr.Left))
	right := b.retrieveValue(b.VisitExpr(expr.Right))
	op := astBinOpToIr[expr.Op]
	var resType IType
	switch op {
	case ADD, SUB, MUL, DIV, MOD, SHL, SHR, AND, OR, XOR:
		resType = left.GetType()
	case EQ, NE, LT, LE, GT, GE:
		resType = NewBaseType(I1)
	}
	res := b.newTempSymbol(resType)
	b.emit(NewBinary(op, left, right, res))
	return res
}

func (b *Builder) VisitType(tp ast.IType) interface{} {
	switch tp.(type) {
	case *ast.AliasType:
		return b.VisitAliasType(tp.(*ast.AliasType))
	case *ast.PrimType:
		return b.VisitPrimType(tp.(*ast.PrimType))
	case *ast.NilType:
		return NewBaseType(Void)
	case *ast.PtrType:
		return b.VisitPtrType(tp.(*ast.PtrType))
	case *ast.ArrayType:
		return b.VisitArrayType(tp.(*ast.ArrayType))
	case *ast.StructType:
		return b.VisitStructType(tp.(*ast.StructType))
	case *ast.FuncType:
		return b.VisitFuncType(tp.(*ast.FuncType))
	}
	return nil
}

func (b *Builder) VisitAliasType(tp *ast.AliasType) interface{} {
	return b.VisitType(tp.Under).(IType) // no alias type in IR
}

var astPrimToIR = map[ast.TypeEnum]TypeEnum{
	ast.Bool: I1, ast.Int: I64, ast.Int64: I64, ast.Float64: F64,
}

func (b *Builder) VisitPrimType(tp *ast.PrimType) interface{} {
	return NewBaseType(astPrimToIR[tp.Enum])
}

func (b *Builder) VisitPtrType(tp *ast.PtrType) interface{} {
	return NewPtrType(b.VisitType(tp.Base).(IType))
}

func (b *Builder) VisitArrayType(tp *ast.ArrayType) interface{} {
	return NewArrayType(b.VisitType(tp.Elem).(IType), tp.Len)
}

func (b *Builder) VisitStructType(tp *ast.StructType) interface{} {
	field := make([]IType, 0)
	for _, f := range tp.Field.Entries {
		field = append(field, b.VisitType(f.Type).(IType))
	}
	return NewStructType(field)
}

// The function type in AST is actually a struct type with two members (function and capture
// list pointer). This support the closure feature in the language.
func (b *Builder) VisitFuncType(tp *ast.FuncType) interface{} {
	// Build parameter type
	params := make([]IType, 0)
	if tp.Receiver != nil { // receiver should be the first param in IR function, if exists.
		params = append(params, b.VisitType(tp.Receiver).(IType))
	}
	for _, p := range tp.Param.Elem { // normal parameters
		params = append(params, b.VisitType(p).(IType))
	}
	// detailed capture list (struct) type is unknown
	capPtrType := NewPtrType(NewBaseType(Void)) // capture list pointer
	params = append(params, capPtrType)

	// Build result type
	retList := make([]IType, 0)
	for _, r := range tp.Result.Elem {
		retList = append(retList, b.VisitType(r).(IType))
	}
	// return type is always a struct, no matter how many values are returned.
	retType := NewStructType(retList)

	// Build closure
	funcType := NewFuncType(params, retType)
	return NewStructType([]IType{funcType, capPtrType})
}
