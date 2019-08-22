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
	case *IncDecStmt:
		c.VisitIncDecStmt(stmt.(*IncDecStmt))
	case *ReturnStmt:
		c.VisitReturnStmt(stmt.(*ReturnStmt))
	case *ForClauseStmt:
		c.VisitForClauseStmt(stmt.(*ForClauseStmt))
	case *IfStmt:
		c.VisitIfStmt(stmt.(*IfStmt))
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
	if tuple, ok := stmt.Rhs[0].GetType().(*TupleType); ok {
		if len(rhsType) != 1 { // other expressions than function call appears
			panic(NewSemaError(stmt.Lhs[0].GetLoc(),
				fmt.Sprintf("other expressions than function call appear on the right hand side"),
			))
		}
		rhsType = tuple.Elem
	}
	if len(lhsType) != len(rhsType) {
		panic(NewSemaError(stmt.Loc,
			fmt.Sprintf("assignment count mismatch: %d = %d", len(lhsType), len(rhsType)),
		))
	}

	// Update type in symbol entry or check type conformance
	for i, leftExpr := range stmt.Lhs {
		if rhsType[i].GetTypeEnum() == Nil {
			continue // initialize with zero value, cannot do check or inference
		}
		if stmt.Init {
			id := leftExpr.(*IdExpr) // lhs must be identifier
			if lhsType[i] == nil {   // lhs type not determined
				id.Symbol.Type = rhsType[i]
				id.Type = rhsType[i]
				continue
			}
		}
		if !lhsType[i].IsIdentical(rhsType[i]) {
			panic(NewSemaError(stmt.Loc,
				fmt.Sprintf("type mismatch: %s = %s", lhsType[i].ToString(),
					rhsType[i].ToString()),
			))
		}
	}

	return nil
}

func (c *SemaChecker) VisitIncDecStmt(stmt *IncDecStmt) interface{} {
	c.VisitExpr(stmt.Expr)
	if !exprTypeEnum(stmt.Expr).Match(IntegerType) {
		panic(NewSemaError(stmt.Loc,
			"cannot increment or decrement non-integer type",
		))
	}
	if !stmt.Expr.IsLValue() {
		panic(NewSemaError(stmt.Loc,
			"cannot increment or decrement rvalue",
		))
	}
	return nil
}

func (c *SemaChecker) VisitReturnStmt(stmt *ReturnStmt) interface{} {
	// Deal with return statements with no value
	if len(stmt.Expr) == 0 {
		if len(stmt.Func.Type.Result.Elem) == 0 { // this function has no returns
			return nil // no need to check
		}
		if len(stmt.Func.NamedRet) > 0 { // return values are named
			return nil // ok to have no return expressions
		}
		panic(NewSemaError(stmt.Loc, "empty return value"))
	}

	// Check number of return values
	exprType := make([]IType, 0)
	for i := range stmt.Expr {
		c.mayChange(&stmt.Expr[i])
		exprType = append(exprType, stmt.Expr[i].GetType())
	}
	retType := stmt.Func.Type.Result.Elem

	if tuple, ok := exprType[0].(*TupleType); ok {
		if len(exprType) != 1 { // other expressions than function call appears
			panic(NewSemaError(stmt.Expr[0].GetLoc(),
				fmt.Sprintf("other expressions than function call appear on the right hand side"),
			))
		}
		exprType = tuple.Elem
	}
	if len(exprType) != len(retType) {
		panic(NewSemaError(stmt.Loc,
			fmt.Sprintf("assignment count mismatch, need %d, have %d", len(retType),
				len(exprType)),
		))
	}

	// Check type conformance
	for i := range retType {
		if !retType[i].IsIdentical(exprType[i]) {
			panic(NewSemaError(stmt.Expr[i].GetLoc(),
				fmt.Sprintf("type mismatch, need %s, have %s", retType[i].ToString(),
					exprType[i].ToString()),
			))
		}
	}

	return nil
}

func (c *SemaChecker) VisitForClauseStmt(stmt *ForClauseStmt) interface{} {
	if stmt.Init != nil {
		c.VisitStmt(stmt.Init)
	}
	if stmt.Cond != nil {
		c.mayChange(&stmt.Cond)
		if exprTypeEnum(stmt.Cond) != Bool {
			panic(NewSemaError(stmt.Cond.GetLoc(),
				fmt.Sprintf("invalid type of for clause condition expression"),
			))
		}
	}
	if stmt.Post != nil {
		c.VisitStmt(stmt.Post)
	}
	c.VisitBlockStmt(stmt.Block)
	return nil
}

func (c *SemaChecker) VisitIfStmt(stmt *IfStmt) interface{} {
	if stmt.Init != nil {
		c.VisitStmt(stmt.Init)
	}
	c.mayChange(&stmt.Cond)
	if exprTypeEnum(stmt.Cond) != Bool {
		panic(NewSemaError(stmt.Cond.GetLoc(),
			fmt.Sprintf("invalid type of if clause condition expression"),
		))
	}
	c.VisitBlockStmt(stmt.Block)
	if stmt.Else != nil {
		c.VisitStmt(stmt.Else)
	}
	return nil
}

func (c *SemaChecker) VisitExpr(expr IExprNode) interface{} {
	switch expr.(type) {
	case ILiteralExpr:
		return c.VisitLiteralExpr(expr.(ILiteralExpr))
	case *IdExpr:
		return c.VisitIdExpr(expr.(*IdExpr))
	case *FuncCallExpr:
		return c.VisitFuncCallExpr(expr.(*FuncCallExpr))
	case *UnaryExpr:
		return c.VisitUnaryExpr(expr.(*UnaryExpr))
	case *BinaryExpr:
		return c.VisitBinaryExpr(expr.(*BinaryExpr))
	}
	return nil
}

func (c *SemaChecker) VisitLiteralExpr(expr ILiteralExpr) interface{} {
	switch expr.(type) {
	case *FuncLit:
		c.VisitFuncLit(expr.(*FuncLit))
	case *CompLit:
		c.VisitCompLit(expr.(*CompLit))
	}
	return nil
}

func (c *SemaChecker) VisitFuncLit(expr *FuncLit) interface{} {
	c.VisitFuncDecl(expr.Decl)
	return nil
}

func (c *SemaChecker) VisitCompLit(expr *CompLit) interface{} {
	c.resolveType(&expr.Type)
	litType := expr.Type
	if alias, ok := expr.Type.(*AliasType); ok {
		litType = alias.Under
	}
	switch litType.GetTypeEnum() {
	case Struct:
		structType := litType.(*StructType)
		for i := range expr.Elem {
			// Update value expression, key should not be visited
			c.mayChange(&expr.Elem[i].Val)
			elem := expr.Elem[i]

			// Check semantic according to whether elements are keyed
			if expr.Keyed {
				// Check if element key is an identifier
				id, ok := elem.Key.(*IdExpr)
				if !ok {
					panic(NewSemaError(elem.Key.GetLoc(), "key is not identifier"))
				}
				// Check if this identifier has corresponding member in struct
				entry := structType.Field.Lookup(id.Name)
				if entry == nil {
					panic(NewSemaError(elem.Key.GetLoc(),
						fmt.Sprintf("member not found for key %s", id.Name),
					))
				}
				// Check if element value is valid
				if !entry.Type.IsIdentical(elem.Val.GetType()) {
					panic(NewSemaError(elem.Val.GetLoc(),
						fmt.Sprintf("invalid value type for key %s", entry.Name),
					))
				}
				elem.Symbol = entry

			} else {
				// Check if more values are needed are provided
				if i >= len(structType.Field.Entries) {
					panic(NewSemaError(elem.Key.GetLoc(), "too many values"))
				}
				// Check type conformance
				target := structType.Field.Entries[i]
				if !elem.Val.GetType().IsIdentical(target.Type) {
					panic(NewSemaError(elem.Val.GetLoc(),
						fmt.Sprintf("invalid type for key %s", target.Name),
					))
				}
				elem.Symbol = target
			}
		}

	case Array:
		arrayType := litType.(*ArrayType)
		for i := range expr.Elem {
			c.mayChange(&expr.Elem[i].Key)
			c.mayChange(&expr.Elem[i].Val)
			elem := expr.Elem[i]

			if expr.Keyed {
				// Check if element key is a constant integer
				constExpr, ok := elem.Key.(*ConstExpr)
				if !ok {
					panic(NewSemaError(elem.Key.GetLoc(),
						"array index is not a constant expression",
					))
				}
				index, ok := constExpr.Val.(int)
				if !ok {
					panic(NewSemaError(constExpr.Loc, "array index is not an integer"))
				}
				// Check if index is out of bound
				if index >= arrayType.Len {
					panic(NewSemaError(constExpr.Loc, "array index out of bound"))
				}

			} else {
				if i >= arrayType.Len {
					panic(NewSemaError(elem.Key.GetLoc(), "too many values"))
				}
			}

			// Check value type
			if !elem.Val.GetType().IsIdentical(arrayType.Elem) {
				panic(NewSemaError(elem.Val.GetLoc(), "invalid type"))
			}
		}

	default:
		panic(NewSemaError(litType.GetLoc(), "cannot construct with composite literal"))

	}

	return nil
}

func (c *SemaChecker) VisitIdExpr(expr *IdExpr) interface{} {
	// Resolve symbol in global scope
	if expr.Symbol == nil {
		expr.Symbol, _ = c.global.Lookup(expr.Name)
		if expr.Symbol == nil {
			panic(NewSemaError(expr.Loc,
				fmt.Sprintf("symbol undefined: %s", expr.Name),
			))
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

func (c *SemaChecker) VisitFuncCallExpr(expr *FuncCallExpr) interface{} {
	// Check if called expression is really a callable
	c.VisitExpr(expr.Func)
	if exprTypeEnum(expr.Func) != Func {
		panic(NewSemaError(expr.Loc, "expression is not callable"))
	}

	// Check if argument length match parameter length
	funcType := expr.Func.GetType().(*FuncType)
	paramType := funcType.Param.Elem
	if len(paramType) != len(expr.Args) {
		panic(NewSemaError(expr.Loc,
			fmt.Sprintf("argument count mismatch, want %d, have %d", len(paramType),
				len(expr.Args)),
		))
	}

	// Check if every argument type match parameter type
	for i := range expr.Args {
		c.mayChange(&expr.Args[i])
		arg := expr.Args[i]
		if !arg.GetType().IsIdentical(paramType[i]) {
			panic(NewSemaError(arg.GetLoc(), "invalid argument type"))
		}
	}

	// Mark return type
	expr.Type = funcType.Result

	return nil
}

func (c *SemaChecker) VisitSelectExpr(expr *SelectExpr) interface{} {
	c.VisitExpr(expr.Target)
	switch exprTypeEnum(expr.Target) {
	case Struct:
		// Get struct field
		structType := expr.Type
		if alias, ok := structType.(*AliasType); ok {
			structType = alias.Under
		}
		field := structType.(*StructType).Field
		name := expr.Member.Name

		// Look up identifier name in struct field
		entry := field.Lookup(name)
		if entry == nil {
			panic(NewSemaError(expr.Member.Loc,
				fmt.Sprintf("member %s not found in %s", name, structType.ToString()),
			))
		}
		expr.Type = entry.Type

	default:
		panic(NewSemaError(expr.Target.GetLoc(), "type is not selectable"))
	}

	return nil
}

func (c *SemaChecker) VisitIndexExpr(expr *IndexExpr) interface{} {
	if !exprTypeEnum(expr.Array).Match(Array | Slice) {
		panic(NewSemaError(expr.Array.GetLoc(), "cannot be indexed"))
	}
	if !exprTypeEnum(expr.Index).Match(IntegerType) {
		panic(NewSemaError(expr.Index.GetLoc(), "index is not an integer"))
	}
	switch exprTypeEnum(expr.Array) {
	case Array:
		expr.Type = expr.Array.GetType().(*ArrayType).Elem
	case Slice:
		expr.Type = expr.Array.GetType().(*SliceType).Elem
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
			panic(NewSemaError(e.Loc,
				"cannot evaluate constant expression",
			))
		}
	}

	// Check type
	err := NewSemaError(e.Loc, fmt.Sprintf(" unary operator %s undefined on type %s",
		UnaryOpStr[e.Op], e.Type.ToString()))
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
			panic(NewSemaError(e.Loc,
				"cannot reference rvalue",
			))
		}
		e.Type = NewPtrType(e.Loc, e.Expr.GetType())
	case DEREF:
		if !exprTypeEnum(e.Expr).Match(Ptr) {
			panic(NewSemaError(e.Loc,
				fmt.Sprintf("cannot dereference non-pointer type: %s",
					e.Expr.GetType().ToString()),
			))
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
			panic(NewSemaError(expr.Loc, "cannot evaluate constant expression"))
		}
	}

	// Check type
	err := NewSemaError(expr.Loc,
		fmt.Sprintf("binary operator %s undefined on type %s and %s",
			BinaryOpStr[expr.Op], expr.Left.GetType().ToString(),
			expr.Right.GetType().ToString(),
		))
	if !expr.Left.GetType().IsIdentical(expr.Right.GetType()) {
		panic(err) // binary operations should be on two operands of same type
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
	case MOD, AAND, AOR, XOR, LSH, RSH:
		if !lTypeEnum.Match(IntegerType) {
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

func (c *SemaChecker) mayChange(expr *IExprNode) {
	newNode, changed := c.VisitExpr(*expr).(IExprNode)
	if changed {
		*expr = newNode
	}
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
	// Check element type
	c.resolveType(&tp.Elem)

	// Check array length
	c.mayChange(&tp.lenExpr)
	constExpr, ok := tp.lenExpr.(*ConstExpr)
	if !ok {
		panic(NewSemaError(tp.lenExpr.GetLoc(), "array length should be a constant expression"))
	}
	length, ok := constExpr.Val.(int)
	if !ok {
		panic(NewSemaError(tp.lenExpr.GetLoc(), "array length should be an integer"))
	}
	tp.Len = length

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
	if _, ok := (*tp).(*UnresolvedType); ok {
		name := (*tp).(*UnresolvedType).Name
		entry, _ := c.global.Lookup(name)
		if entry == nil {
			panic(NewSemaError((*tp).GetLoc(), fmt.Sprintf("undefined symbol: %s", name)))
		} else if entry.Flag != TypeEntry {
			panic(NewSemaError((*tp).GetLoc(), fmt.Sprintf("not a type: %s", name)))
		}
		*tp = entry.Type
	}
	c.VisitType(*tp)
}
