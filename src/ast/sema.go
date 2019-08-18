package ast

import (
	"fmt"
)

// A semantic checker performs several tasks:
// 1. Resolve type, variable and constant symbols in global scope;
// 2. Infer type for expressions;
// 3. Check type conformance.
type SemaChecker struct {
	BaseVisitor
	global *Scope
}

func NewSemaChecker() *SemaChecker { return &SemaChecker{} }

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

func (c *SemaChecker) VisitStmt(stmt IStmtNode) interface{} {
	switch stmt.(type) {
	case IExprNode:
		c.VisitExpr(stmt.(IExprNode))
	case *BlockStmt:
		c.VisitBlockStmt(stmt.(*BlockStmt))
	case *AssignStmt:
		c.VisitAssignStmt(stmt.(*AssignStmt))
	}
	return nil
}

func (c *SemaChecker) VisitBlockStmt(stmt *BlockStmt) interface{} {
	for _, s := range stmt.Stmts {
		c.VisitStmt(s)
	}
	return nil
}

func (c *SemaChecker) VisitAssignStmt(stmt *AssignStmt) interface{} {
	// Visit expressions on rhs and lhs
	rhsType := make([]IType, 0)
	for i := range stmt.Rhs {
		c.mayChange(&stmt.Rhs[i])
		rhsType = append(rhsType, stmt.Rhs[i].GetType())
	}
	lhsType := make([]IType, 0)
	for i := range stmt.Lhs {
		c.mayChange(&stmt.Lhs[i])
		lhsType = append(lhsType, stmt.Lhs[i].GetType())
	}

	// Consider multi-valued function
	if len(lhsType) != len(rhsType) { // maybe a multi-valued function
		if len(rhsType) > 1 { // too may value on rhs
			panic(fmt.Errorf("%s assignment count mismatch %d = %d",
				stmt.LocStr(), len(lhsType), len(rhsType)))
		}

		tuple, ok := stmt.Rhs[0].GetType().(*TupleType)
		if !ok { // not function result
			panic(fmt.Errorf("%s assignment count mismatch %d = 1",
				stmt.LocStr(), len(lhsType)))
		}

		if len(lhsType) != len(tuple.Elem) {
			panic(fmt.Errorf("%s function result assignment count mismatch %d = %d",
				stmt.LocStr(), len(lhsType), len(tuple.Elem)))
		}
		rhsType = tuple.Elem
	}

	// Update type in symbol entry or check type conformance
	for i, leftExpr := range stmt.Lhs {
		if stmt.Init {
			id := leftExpr.(*IdExpr)                                  // lhs must be identifier
			if lhsType[i] == nil && rhsType[i].GetTypeEnum() != Nil { // lhs type not determined
				id.Symbol.Type = rhsType[i]
				id.Type = rhsType[i]
				continue
			}
		}
		if !lhsType[i].IsIdentical(rhsType[i]) && rhsType[i].GetTypeEnum() != Nil {
			panic(fmt.Errorf("%s type mismatch %s = %s", stmt.LocStr(),
				lhsType[i].ToString(), rhsType[i].ToString()))
		}
	}

	return nil
}

func (c *SemaChecker) mayChange(expr *IExprNode) {
	newNode, changed := c.VisitExpr(*expr).(IExprNode)
	if changed {
		*expr = newNode
	}
}

func (c *SemaChecker) VisitExpr(expr IExprNode) interface{} {
	switch expr.(type) {
	case ILiteralExpr:
		return c.VisitLiteralExpr(expr.(ILiteralExpr))
	case *IdExpr:
		return c.VisitIdExpr(expr.(*IdExpr))
	case *UnaryExpr:
		return c.VisitUnaryExpr(expr.(*UnaryExpr))
	case *BinaryExpr:
		return c.VisitBinaryExpr(expr.(*BinaryExpr))
	}
	return nil
}

func (c *SemaChecker) VisitIdExpr(expr *IdExpr) interface{} {
	// Resolve symbol in global scope
	if expr.Symbol == nil {
		expr.Symbol, _ = c.global.Lookup(expr.Name)
		if expr.Symbol == nil {
			panic(fmt.Errorf("%s symbol undefined: %s", expr.LocStr(), expr.Name))
		}
	}

	// Use symbol entry type to mark identifier type
	expr.Type = expr.Symbol.Type

	// Replace with constant expression node, if possible
	if expr.Symbol.Flag == ConstEntry {
		switch expr.Symbol.Val.(type) {
		case bool:
			return NewBoolConst(expr.Loc, expr.Symbol.Val.(bool))
		case int:
			return NewIntConst(expr.Loc, expr.Symbol.Val.(int))
		case float64:
			return NewFloatConst(expr.Loc, expr.Symbol.Val.(float64))
		}
	}

	return nil
}

func (c *SemaChecker) VisitUnaryExpr(e *UnaryExpr) interface{} {
	// Visit subexpression
	c.mayChange(&e.Expr)

	// Evaluate constant expression
	if constExpr, ok := e.Expr.(*ConstExpr); ok {
		fun := unaryConstExpr[e.Op][constExpr.Type.GetTypeEnum()]
		if fun != nil {
			return fun(constExpr)
		} else {
			panic(fmt.Errorf("%s cannot evaluate constant expression", e.LocStr()))
		}
	}

	// Check type
	err := fmt.Errorf("%s unary operator %s undefined on type %s", e.LocStr(),
		UnaryOpStr[e.Op], e.Type.ToString())
	switch e.Op {
	case POS, NEG:
		if !exprTypeEnum(e.Expr).Match(PrimitiveType &^ String) {
			panic(err) // not primitive (except string) type
		}
		e.Type = e.Expr.GetType()
	case NOT:
		if !exprTypeEnum(e.Expr).Match(Bool) {
			panic(err) // not bool type
		}
		e.Type = e.Expr.GetType()
	case INV:
		if !exprTypeEnum(e.Expr).Match(IntegerType) {
			panic(err)
		}
		e.Type = e.Expr.GetType()
	case REF:
		if !e.Expr.IsLValue() {
			panic(fmt.Errorf("%s cannot reference rvalue", e.LocStr()))
		}
		e.Type = NewPtrType(e.Loc, e.Expr.GetType())
	case DEREF:
		if !exprTypeEnum(e.Expr).Match(Ptr) {
			panic(fmt.Errorf("%s cannot dereference non-pointer type: %s", e.LocStr(),
				e.Expr.GetType().ToString()))
		}
		e.Type = e.Expr.GetType().(*PtrType).Ref
	}

	return nil
}

func (c *SemaChecker) VisitBinaryExpr(expr *BinaryExpr) interface{} {
	// Visit subexpressions
	c.mayChange(&expr.Left)
	c.mayChange(&expr.Right)

	// Evaluate constant expression
	lConst, lok := expr.Left.(*ConstExpr)
	rConst, rok := expr.Right.(*ConstExpr)
	if lok && rok {
		fun := binaryConstExpr[expr.Op][lConst.Type.GetTypeEnum()][rConst.Type.GetTypeEnum()]
		if fun != nil {
			return fun(lConst, rConst)
		} else {
			panic(fmt.Errorf("%s cannot evaluate constant expression", expr.LocStr()))
		}
	}

	// Check type
	err := fmt.Errorf("%s binary operator %s undefined on type %s and %s",
		expr.LocStr(), BinaryOpStr[expr.Op], expr.Left.GetType().ToString(),
		expr.Right.GetType().ToString())
	if expr.Op&(LSH|RSH) == 0 && !expr.Left.GetType().IsIdentical(expr.Right.GetType()) {
		panic(err) // binary operations (except LSH and RSH), should be on two operands of same type
	}
	lType := expr.Left.GetType()
	lTypeEnum := lType.GetTypeEnum()
	switch expr.Op {
	case ADD:
		if !lTypeEnum.Match(PrimitiveType) {
			panic(err)
		}
		expr.Type = lType
	case SUB, MUL, DIV:
		if !lTypeEnum.Match(PrimitiveType &^ String) {
			panic(err)
		}
		expr.Type = lType
	case MOD, AAND, AOR, XOR:
		if !lTypeEnum.Match(IntegerType) {
			panic(err)
		}
		expr.Type = lType
	case LSH, RSH:
		rTypeEnum := exprTypeEnum(expr.Right)
		if !lTypeEnum.Match(IntegerType) || !rTypeEnum.Match(UnsignedType) {
			panic(err)
		}
		expr.Type = lType
	case EQ, NE, LT, LE, GT, GE:
		if !lTypeEnum.Match(PrimitiveType) {
			panic(err)
		}
		expr.Type = NewPrimType(expr.Loc, Bool)
	case LAND, LOR:
		if !lTypeEnum.Match(Bool) {
			panic(err)
		}
		expr.Type = lType
	}

	return nil
}

func exprTypeEnum(expr IExprNode) TypeEnum {
	return expr.GetType().GetTypeEnum()
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
