package ast

type IVisitor interface {
	VisitProgram(program *ProgramNode) interface{}

	VisitFuncDecl(decl *FuncDecl) interface{}
	VisitScope(scope *Scope) interface{}

	VisitStmt(stmt IStmtNode) interface{}
	VisitBlockStmt(stmt *BlockStmt) interface{}
	VisitAssignStmt(stmt *AssignStmt) interface{}
	VisitIncDecStmt(stmt *IncDecStmt) interface{}
	VisitReturnStmt(stmt *ReturnStmt) interface{}
	VisitForClauseStmt(stmt *ForClauseStmt) interface{}
	VisitBreakStmt(stmt *BreakStmt) interface{}
	VisitContinueStmt(stmt *ContinueStmt) interface{}
	VisitIfStmt(stmt *IfStmt) interface{}

	VisitExpr(expr IExprNode) interface{}
	VisitLiteralExpr(expr ILiteralExpr) interface{}
	VisitFuncLit(expr *FuncLit) interface{}
	VisitCompLit(expr *CompLit) interface{}
	VisitConstExpr(expr *ConstExpr) interface{}
	VisitIdExpr(expr *IdExpr) interface{}
	VisitFuncCallExpr(expr *FuncCallExpr) interface{}
	VisitUnaryExpr(expr *UnaryExpr) interface{}
	VisitBinaryExpr(expr *BinaryExpr) interface{}

	VisitType(tp IType) interface{}
	VisitUnresolvedType(tp *UnresolvedType) interface{}
	VisitAliasType(tp *AliasType) interface{}
	VisitPrimType(tp *PrimType) interface{}
	VisitNilType(tp *NilType) interface{}
	VisitPtrType(tp *PtrType) interface{}
	VisitArrayType(tp *ArrayType) interface{}
	VisitSliceType(tp *SliceType) interface{}
	VisitMapType(tp *MapType) interface{}
	VisitStructType(tp *StructType) interface{}
	VisitFuncType(tp *FuncType) interface{}
}
