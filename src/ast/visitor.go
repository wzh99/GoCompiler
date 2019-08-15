package ast

type ASTVisitor interface {
	VisitProgram(program *ProgramNode) interface{}

	VisitFuncDecl(decl *FuncDecl) interface{}

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
	VisitFuncLiteral(expr *FuncLiteral) interface{}
	VisitConstExpr(expr *ConstExpr) interface{}
	VisitNilValue(expr *NilValue) interface{}
	VisitIdExpr(expr *IdExpr) interface{}
	VisitFuncCallExpr(expr *FuncCallExpr) interface{}
	VisitUnaryExpr(expr *UnaryExpr) interface{}
	VisitBinaryExpr(expr *BinaryExpr) interface{}
}
