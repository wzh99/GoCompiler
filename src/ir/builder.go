package ir

import (
	"ast"
	"fmt"
)

type Program struct {
	Name string
	// Scope of global variables
	// Note that in AST, the global member of program node is a function, while in IR
	// this member is a scope. This is because in IR, the hierarchy of scope is lost.
	// Access global scope through global function is not straightforward.
	Global *Scope
	// List of all compiled functions
	// The first function is global function, which should be treated specially.
	Funcs []*Func
	// Basic block counter (to distinguish labels)
	nBlock int
	// Temporary operand counter (to distinguish temporaries)
	nTmp int
}

type CaptureList struct {
	// Captured item specification
	symbols []*Symbol
	// The struct type descriptor of capture list
	tp *StructType
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
}

func NewBuilder() *Builder {
	return &Builder{
		funcTable: make(map[*ast.FuncDecl]*Func), // the first function is global
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
	b.prg.Global = NewGlobalScope()     // set global scope pointer of program
	b.prg.Funcs[0].Scope = b.prg.Global // give global scope to global function
	b.fun = b.prg.Funcs[0]              // set builder function pointer to global
	b.VisitFuncDecl(node.Global)        // add symbols to scope

	// Visit other top level functions
	for i, decl := range node.Funcs {
		b.fun = b.prg.Funcs[i+1] // skip global function
		b.VisitFuncDecl(decl)
	}

	return b.prg
}

func (b *Builder) VisitFuncDecl(decl *ast.FuncDecl) interface{} {
	if b.fun.Scope == nil { // global scope is constructed outside this method
		b.fun.Scope = NewLocalScope() // build function scope
	}
	b.VisitScope(decl.Scope)
	b.fun.Enter = NewBasicBlock(b.requestBlockName(), b.fun) // create entrance block
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

func (b *Builder) VisitScope(astScope *ast.Scope) interface{} {
	idx := 0 // index in AST function scope
	fun := astScope.Func
	entries := astScope.Symbols.Entries

	// Add receiver to IR scope and its parameter list
	if fun.Type.Receiver != nil {
		symbol := entries[idx]
		b.fun.Scope.AddSymbolFromAST(symbol, b.genSymbolName(symbol.Name),
			b.VisitType(symbol.Type).(IType), true)
		idx++
	}

	// Add normal parameters to IR scope and its parameter list
	for k := 0; k < len(fun.Type.Param.Elem); k++ {
		symbol := entries[idx]
		b.fun.Scope.AddSymbolFromAST(symbol, b.genSymbolName(symbol.Name),
			b.VisitType(symbol.Type).(IType), true)
		idx++
	}

	// Add function capture list pointer to IR scope and parameter list
	if !astScope.Global {
		b.fun.Scope.AddSymbol("_cap", NewPtrType(NewBaseType(Void)), true)
	}

	// Add local symbols to IR scope
	for ; idx < len(astScope.Symbols.Entries); idx++ {
		symbol := entries[idx]
		if symbol.Flag == ast.VarEntry {
			b.fun.Scope.AddSymbolFromAST(symbol, b.genSymbolName(symbol.Name),
				b.VisitType(symbol.Type).(IType), false)
		}
	}

	// Add captured symbols to IR scope, if this scope is from a literal function
	if astScope.Func.Lit != nil {
		lit := astScope.Func.Lit
		for _, symbol := range lit.Closure.Entries {
			b.fun.Scope.AddSymbolFromAST(symbol, b.genSymbolName(symbol.Name),
				b.VisitType(symbol.Type).(IType), false)
		}
	}

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
	case *ast.AssignStmt:
		b.VisitAssignStmt(stmt.(*ast.AssignStmt))
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
		b.emit(NewMove(b.bb, srcRet.(IValue), inter))

	case *GetPtr:
		srcPtr := srcRet.(*GetPtr).Result
		b.emit(srcRet.(*GetPtr))             // emit that getptr instruction
		b.emit(NewLoad(b.bb, srcPtr, inter)) // load value from pointer
	}
}

func (b *Builder) moveToDst(dstNode ast.IExprNode, inter IValue) {
	// DEREF has different semantic on different sides of assignment
	if unary, ok := dstNode.(*ast.UnaryExpr); ok {
		// DEREF is the only possible unary operation on left hand side
		dstRet := b.VisitExpr(unary.Expr)

		switch dstRet.(type) {
		case IValue: // an symbol value, must be a pointer type (*p = s)
			b.emit(NewStore(b.bb, inter, dstRet.(IValue))) // store to the pointer

		case *GetPtr: // an get pointer instruction (*p.f = s)
			getPtr := dstRet.(*GetPtr)
			b.emit(getPtr)            // emit that instruction
			fieldPtr := getPtr.Result // get pointer to that pointer field
			field := b.newTempSymbol(fieldPtr.GetType().(*PtrType).Base)
			b.emit(NewLoad(b.bb, fieldPtr, field)) // get pointer field
			b.emit(NewStore(b.bb, inter, field))   // store to that field
		}

	} else {
		dstRet := b.VisitExpr(dstNode)

		switch dstRet.(type) {
		case IValue: // d = s
			dstVal := dstRet.(IValue)
			if inter == nil {
				b.emit(NewClear(b.bb, dstVal))
			} else {
				b.emit(NewMove(b.bb, inter, dstVal)) // move between symbols
			}

		case *GetPtr: // d.f = s
			instr := dstRet.(*GetPtr)
			b.emit(instr)
			b.emit(NewStore(b.bb, inter, instr.Result)) // store to field pointer
		}
	}
}

func (b *Builder) newTempSymbol(tp IType) *SymbolValue {
	name := fmt.Sprintf("_t%d", b.prg.nTmp)
	b.prg.nTmp++
	sym := b.fun.Scope.AddTempSymbol(name, tp)
	return NewSymbolValue(sym)
}

// All expression visiting methods return result operand
func (b *Builder) VisitExpr(expr ast.IExprNode) interface{} {
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
	capList := &CaptureList{
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
		b.emit(NewPtrOffset(b.bb, listPtr, valPtr, capList.tp.Field[i].Offset))
		b.emit(NewLoad(b.bb, valPtr, NewSymbolValue(capList.symbols[i])))
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
	b.emit(NewGetPtr(b.bb, litVal, funcPtr, []IValue{NewI64Imm(0)}))
	b.emit(NewStore(b.bb, funcLit, funcPtr))
	// Allocate heap space for capture list
	mallocRet := b.newTempSymbol(NewPtrType(capList.tp)) // provide size to instruction
	b.emit(NewMalloc(b.bb, mallocRet))
	// Store captured operands to list
	for i, s := range capList.symbols {
		elemPtr := b.newTempSymbol(NewPtrType(s.Type))
		b.emit(NewPtrOffset(b.bb, mallocRet, elemPtr, capList.tp.Field[i].Offset))
		b.emit(NewStore(b.bb, NewSymbolValue(s), elemPtr))
	}
	// Store capture list pointer as the second field of closure
	listPtrPtr := b.newTempSymbol(NewPtrType(closureType.Field[1].Type))
	b.emit(NewGetPtr(b.bb, litVal, listPtrPtr, []IValue{NewI64Imm(1)}))
	b.emit(NewStore(b.bb, mallocRet, listPtrPtr))

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
		return NewSymbolValue(b.fun.Scope.astToIr[expr.Symbol])
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
		b.emit(NewCall(b.bb, topFunc, args, ret))
	} else { // closure (constructed from literals)
		funcAddr := b.loadField(fun, 0)
		capList := b.loadField(fun, 1)
		args = append(args, capList)
		b.emit(NewCall(b.bb, funcAddr, args, ret))
	}

	return ret
}

func (b *Builder) loadField(value IValue, index int) *SymbolValue {
	structType := value.GetType().(*StructType)
	ptr := b.newTempSymbol(NewPtrType(structType.Field[index].Type))
	b.emit(NewGetPtr(b.bb, value, ptr, []IValue{NewI64Imm(index)}))
	field := b.newTempSymbol(structType.Field[index].Type)
	b.emit(NewLoad(b.bb, ptr, field))
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
		return NewGetPtr(b.bb, retVal.(IValue), b.newTempSymbol(NewPtrType(exprType)),
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
			b.bb, array, b.newTempSymbol(NewPtrType(array.GetType().(*ArrayType).Elem)),
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
		b.emit(NewLoad(b.bb, instr.Result, val))
		return val
	}
	return nil
}

func (b *Builder) VisitUnaryExpr(expr *ast.UnaryExpr) interface{} {
	val := b.retrieveValue(b.VisitExpr(expr.Expr))
	var res IValue
	switch expr.Op {
	case ast.POS:
		return val
	case ast.NEG:
		res = b.newTempSymbol(val.GetType())
		b.emit(NewUnary(b.bb, NEG, val, res))
	case ast.NOT, ast.INV:
		res = b.newTempSymbol(val.GetType())
		b.emit(NewUnary(b.bb, NOT, val, res))
	case ast.DEREF:
		ptrType := val.GetType().(*PtrType)
		res = b.newTempSymbol(ptrType.Base)
		b.emit(NewLoad(b.bb, val, res))
	case ast.REF:
		res = b.newTempSymbol(NewPtrType(val.GetType()))
		b.emit(NewGetPtr(b.bb, val, res, []IValue{}))
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
	b.emit(NewBinary(b.bb, op, left, right, res))
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
