package ir

import "ast"

type Program struct {
	Global    *Scope           // scope of global variables
	Funcs     []*Func          // list of all compiled functions
	funcTable map[string]*Func // look up table for functions
	nBlock    int              // basic block counter
	nTmp      int              // temporary operand counter
}

func NewProgram(global *Scope, funcs []*Func, table map[string]*Func) *Program {
	p := &Program{
		Global:    global,
		Funcs:     funcs,
		funcTable: table,
		nBlock:    0,
		nTmp:      0,
	}
	return p
}

type Builder struct {
	ast.IVisitor
	program *Program    // target program
	fun     *Func       // current function being visited
	bb      *BasicBlock // current basic block being built
}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) VisitProgram(node *ast.ProgramNode) interface{} {
	// Generate signature of all top level functions, in case they are referenced.
	funcs := []*Func{b.genSignature(node.Global)}
	table := make(map[string]*Func)
	for _, decl := range node.Funcs {
		sig := b.genSignature(decl)
		funcs = append(funcs, sig)
		table[sig.Name] = sig
	}

	// Build global scope

	// Visit functions and build IR

	return NewProgram(nil, funcs, table)
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
	return b.VisitType(tp).(IType) // no alias type in IR
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
