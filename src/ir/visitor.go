package ir

type IVisitor interface {
	VisitProgram(prg *Program) interface{}
	VisitScope(scope *Scope) interface{}
	VisitFunc(fun *Func) interface{}
	VisitBasicBlock(bb *BasicBlock) interface{}

	VisitInstr(instr IInstr) interface{}
	VisitMove(instr *Move) interface{}
	VisitLoad(instr *Load) interface{}
	VisitStore(instr *Store) interface{}
	VisitMalloc(instr *Malloc) interface{}
	VisitGetPtr(instr *GetPtr) interface{}
	VisitPtrOffset(instr *PtrOffset) interface{}
	VisitClear(instr *Clear) interface{}
	VisitUnary(instr *Unary) interface{}
	VisitBinary(instr *Binary) interface{}
	VisitJump(instr *Jump) interface{}
	VisitBranch(instr *Branch) interface{}
	VisitCall(instr *Call) interface{}
	VisitReturn(instr *Return) interface{}
	VisitPhi(instr *Phi) interface{}

	VisitValue(val IValue) interface{}
	VisitVariable(val *Variable) interface{}
	VisitImmValue(val *Immediate) interface{}

	VisitType(tp IType) interface{}
	VisitBaseType(tp *BaseType) interface{}
	VisitStructType(tp *StructType) interface{}
	VisitArrayType(tp *ArrayType) interface{}
	VisitPtrType(tp *PtrType) interface{}
	VisitFuncType(tp *FuncType) interface{}
}
