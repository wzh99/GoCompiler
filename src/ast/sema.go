package ast

import "fmt"

// A semantic checker performs several tasks:
// 1. Resolve type, variable and constant symbols in global scope;
// 2. Infer type for expressions;
// 3. Check type match.
type SemaChecker struct {
	BaseVisitor
	global, cur *Scope
}

func NewSemaChecker() *SemaChecker { return &SemaChecker{} }

func (c *SemaChecker) pushScope(scope *Scope) { c.cur = scope }

func (c *SemaChecker) popScope() { c.cur = c.cur.Parent }

func (c *SemaChecker) VisitProgram(program *ProgramNode) interface{} {
	c.global = program.Global.Scope // set global scope cursor
	c.VisitFuncDecl(program.Global)
	for _, f := range program.Funcs {
		c.VisitFuncDecl(f)
	}
	return nil
}

func (c *SemaChecker) VisitFuncDecl(decl *FuncDecl) interface{} {
	c.VisitFuncType(decl.Type)
	c.VisitScope(decl.Scope)
	for _, stmt := range decl.Stmts {
		c.VisitStmt(stmt)
	}
	return nil
}

func (c *SemaChecker) VisitScope(scope *Scope) interface{} {
	for _, symbol := range scope.Symbols.Entries {
		c.resolveType(&symbol.Type)
	}
	return nil
}

func (c *SemaChecker) VisitType(tp IType) interface{} {
	switch tp.(type) {
	case *AliasType:
		c.VisitAliasType(tp.(*AliasType))
	case *PtrType:
		c.VisitPtrType(tp.(*PtrType))
	case *ArrayType:
		c.VisitArrayType(tp.(*ArrayType))
	case *SliceType:
		c.VisitSliceType(tp.(*SliceType))
	case *MapType:
		c.VisitMapType(tp.(*MapType))
	case *StructType:
		c.VisitStructType(tp.(*StructType))
	}
	return nil
}

func (c *SemaChecker) VisitAliasType(tp *AliasType) interface{} {
	c.resolveType(&tp.Under)
	return nil
}

func (c *SemaChecker) VisitPtrType(tp *PtrType) interface{} {
	c.resolveType(&tp.Ref)
	return nil
}

func (c *SemaChecker) VisitArrayType(tp *ArrayType) interface{} {
	c.resolveType(&tp.Elem)
	return nil
}

func (c *SemaChecker) VisitSliceType(tp *SliceType) interface{} {
	c.resolveType(&tp.Elem)
	return nil
}

func (c *SemaChecker) VisitMapType(tp *MapType) interface{} {
	c.resolveType(&tp.Key)
	c.resolveType(&tp.Val)
	return nil
}

func (c *SemaChecker) VisitStructType(tp *StructType) interface{} {
	for _, symbol := range tp.Field.Entries {
		c.resolveType(&symbol.Type)
	}
	return nil
}

func (c *SemaChecker) VisitFuncType(tp *FuncType) interface{} {
	for i := range tp.Param.Elem {
		c.resolveType(&tp.Param.Elem[i])
	}
	if tp.Receiver != nil {
		c.resolveType(&tp.Receiver)
	}
	for i := range tp.Result.Elem {
		c.resolveType(&tp.Result.Elem[i])
	}

	return nil
}

// Resolve type in the global scope, and further resolve its component type
func (c *SemaChecker) resolveType(tp *IType) {
	if _, unres := (*tp).(*UnresolvedType); unres {
		name := (*tp).(*UnresolvedType).Name
		entry, _ := c.global.Lookup(name)
		if entry == nil {
			panic(fmt.Errorf("undefined symbol: %s", name))
		} else if entry.Flag != TypeEntry {
			panic(fmt.Errorf("not a type: %s", name))
		}
		*tp = entry.Type
	}
	c.VisitType(*tp)
}
