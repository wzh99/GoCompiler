package ast

type BaseVisitor struct{}

func (v *BaseVisitor) VisitProgram(program *ProgramNode) interface{} { return nil }

func (v *BaseVisitor) VisitFuncDecl(decl *FuncDecl) interface{} { return nil }

func (v *BaseVisitor) VisitScope(scope *Scope) interface{} { return nil }

func (v *BaseVisitor) VisitStmt(stmt IStmtNode) interface{} { return nil }

func (v *BaseVisitor) VisitBlockStmt(stmt *BlockStmt) interface{} { return nil }

func (v *BaseVisitor) VisitAssignStmt(stmt *AssignStmt) interface{} { return nil }

func (v *BaseVisitor) VisitIncDecStmt(stmt *IncDecStmt) interface{} { return nil }

func (v *BaseVisitor) VisitReturnStmt(stmt *ReturnStmt) interface{} { return nil }

func (v *BaseVisitor) VisitForClauseStmt(stmt *ForClauseStmt) interface{} { return nil }

func (v *BaseVisitor) VisitBreakStmt(stmt *BreakStmt) interface{} { return nil }

func (v *BaseVisitor) VisitContinueStmt(stmt *ContinueStmt) interface{} { return nil }

func (v *BaseVisitor) VisitIfStmt(stmt *IfStmt) interface{} { return nil }

func (v *BaseVisitor) VisitExpr(expr IExprNode) interface{} { return nil }

func (v *BaseVisitor) VisitLiteralExpr(expr ILiteralExpr) interface{} { return nil }

func (v *BaseVisitor) VisitFuncLit(expr *FuncLit) interface{} { return nil }

func (v *BaseVisitor) VisitCompLit(lit *CompLit) interface{} { return nil }

func (v *BaseVisitor) VisitConstExpr(expr *ConstExpr) interface{} { return nil }

func (v *BaseVisitor) VisitIdExpr(expr *IdExpr) interface{} { return nil }

func (v *BaseVisitor) VisitFuncCallExpr(expr *FuncCallExpr) interface{} { return nil }

func (v *BaseVisitor) VisitSelectExpr(expr *SelectExpr) interface{} { return nil }

func (v *BaseVisitor) VisitIndexExpr(expr *IndexExpr) interface{} { return nil }

func (v *BaseVisitor) VisitUnaryExpr(expr *UnaryExpr) interface{} { return nil }

func (v *BaseVisitor) VisitBinaryExpr(expr *BinaryExpr) interface{} { return nil }

func (v *BaseVisitor) VisitType(tp IType) interface{} { return nil }

func (v *BaseVisitor) VisitUnresolvedType(tp *UnresolvedType) interface{} { return nil }

func (v *BaseVisitor) VisitAliasType(tp *AliasType) interface{} { return nil }

func (v *BaseVisitor) VisitPrimType(tp *PrimType) interface{} { return nil }

func (v *BaseVisitor) VisitNilType(tp *NilType) interface{} { return nil }

func (v *BaseVisitor) VisitPtrType(tp *PtrType) interface{} { return nil }

func (v *BaseVisitor) VisitArrayType(tp *ArrayType) interface{} { return nil }

func (v *BaseVisitor) VisitSliceType(tp *SliceType) interface{} { return nil }

func (v *BaseVisitor) VisitMapType(tp *MapType) interface{} { return nil }

func (v *BaseVisitor) VisitStructType(tp *StructType) interface{} { return nil }

func (v *BaseVisitor) VisitFuncType(tp *FuncType) interface{} { return nil }
