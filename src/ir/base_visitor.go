package ir

type BaseVisitor struct{}

func (v *BaseVisitor) VisitProgram(prg *Program) interface{} { return nil }

func (v *BaseVisitor) VisitFunc(fun *Func) interface{} { return nil }

func (v *BaseVisitor) VisitScope(scope *Scope) interface{} { return nil }

func (v *BaseVisitor) VisitBasicBlock(bb *BasicBlock) interface{} { return nil }

func (v *BaseVisitor) VisitInstr(instr IInstr) interface{} { return nil }

func (v *BaseVisitor) VisitMove(instr *Move) interface{} { return nil }

func (v *BaseVisitor) VisitLoad(instr *Load) interface{} { return nil }

func (v *BaseVisitor) VisitStore(instr *Store) interface{} { return nil }

func (v *BaseVisitor) VisitMalloc(instr *Malloc) interface{} { return nil }

func (v *BaseVisitor) VisitGetPtr(instr *GetPtr) interface{} { return nil }

func (v *BaseVisitor) VisitPtrOffset(instr *PtrOffset) interface{} { return nil }

func (v *BaseVisitor) VisitClear(instr *Clear) interface{} { return nil }

func (v *BaseVisitor) VisitUnary(instr *Unary) interface{} { return nil }

func (v *BaseVisitor) VisitBinary(instr *Binary) interface{} { return nil }

func (v *BaseVisitor) VisitJump(instr *Jump) interface{} { return nil }

func (v *BaseVisitor) VisitBranch(instr *Branch) interface{} { return nil }

func (v *BaseVisitor) VisitCall(instr *Call) interface{} { return nil }

func (v *BaseVisitor) VisitReturn(instr *Return) interface{} { return nil }

func (v *BaseVisitor) VisitPhi(instr *Phi) interface{} { return nil }

func (v *BaseVisitor) VisitValue(val IValue) interface{} { return nil }

func (v *BaseVisitor) VisitVariable(val *Variable) interface{} { return nil }

func (v *BaseVisitor) VisitImmValue(val *ImmValue) interface{} { return nil }

func (v *BaseVisitor) VisitType(tp IType) interface{} { return nil }

func (v *BaseVisitor) VisitBaseType(tp *BaseType) interface{} { return nil }

func (v *BaseVisitor) VisitStructType(tp *StructType) interface{} { return nil }

func (v *BaseVisitor) VisitArrayType(tp *ArrayType) interface{} { return nil }

func (v *BaseVisitor) VisitPtrType(tp *PtrType) interface{} { return nil }

func (v *BaseVisitor) VisitFuncType(tp *FuncType) interface{} { return nil }
