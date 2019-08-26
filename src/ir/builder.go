package ir

import (
	"ast"
	"fmt"
)

type Program struct {
	// Scope of global variables
	// Note that in AST, the Global member of program node is a function, while in IR
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
	for _, symbol := range astScope.Symbols.Entries {
		if symbol.Flag == ast.VarEntry { // only add variable symbol
			irType := b.VisitType(symbol.Type).(IType)
			name := fmt.Sprintf("_s%d_%s", b.nScope, symbol.Name)
			b.fun.Scope.AddSymbolFromAST(symbol, name, irType)
		}
	}
	b.nScope++
	return nil
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
	}

	return nil
}

func (b *Builder) newTempSymbol(tp IType) *SymbolValue {
	sym := b.fun.Scope.AddTempSymbol(b.requestTempName(), tp)
	return NewSymbolValue(sym)
}

// Source operand is always an IR value. The emitted instruction depends on the expression
// type of destination.
func (b *Builder) moveToDst(dstNode ast.IExprNode, src IValue) {
	unary, ok := dstNode.(*ast.UnaryExpr)
	if ok && unary.Op == ast.DEREF {
		dstPtr := b.VisitExpr(unary.Expr).(IValue)
		b.emit(NewStore(b.bb, src, dstPtr)) // store to the pointer
	} else {
		dstVal := b.VisitExpr(dstNode).(IValue)
		b.emit(NewMove(b.bb, src, dstVal))
	}
}

func (b *Builder) requestTempName() string {
	str := fmt.Sprintf("_t%d", b.prg.nTmp)
	b.prg.nTmp++
	return str
}

// All expression visiting methods return result operand
func (b *Builder) VisitExpr(expr ast.IExprNode) interface{} {
	switch expr.(type) {
	case ast.ILiteralExpr:
		return b.VisitLiteralExpr(expr.(ast.ILiteralExpr))
	case *ast.FuncCallExpr:
		return b.VisitFuncCallExpr(expr.(*ast.FuncCallExpr))
	case *ast.IdExpr:
		return b.VisitIdExpr(expr.(*ast.IdExpr))
	}
	return nil
}

func (b *Builder) VisitLiteralExpr(expr ast.ILiteralExpr) interface{} {
	switch expr.(type) {
	case *ast.ConstExpr:
		return b.VisitConstExpr(expr.(*ast.ConstExpr))
	}
	return nil
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

func (b *Builder) VisitFuncCallExpr(expr *ast.FuncCallExpr) interface{} {
	// Evaluate function and argument expressions
	fun := b.VisitExpr(expr.Func).(IValue)
	args := make([]IValue, 0)
	for _, e := range expr.Args {
		args = append(args, b.VisitExpr(e).(IValue))
	}

	// Emit instructions depending on function type
	ret := NewSymbolValue(&Symbol{
		Name: b.requestTempName(),
		Type: fun.GetType().(*FuncType).Return,
	})
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

func (b *Builder) VisitIdExpr(expr *ast.IdExpr) interface{} {
	switch expr.Symbol.Flag {
	case ast.VarEntry:
		return NewSymbolValue(b.fun.Scope.astToIr[expr.Symbol])
	case ast.FuncEntry:
		return b.funcTable[expr.Symbol.Val.(*ast.FuncDecl)]
	}
	return nil
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
