package ir

import "fmt"

type IInstr interface {
	GetPrev() IInstr
	SetPrev(instr IInstr)
	GetNext() IInstr
	SetNext(instr IInstr)
	GetBasicBlock() *BasicBlock
}

// Instruction iterator
type InstrIter struct {
	Cur IInstr
	BB  *BasicBlock
}

// Constructed from a existing instruction
func NewInstrIter(instr IInstr) *InstrIter {
	if instr == nil {
		panic(NewIRError("cannot construct iterator from a nil instruction"))
	}
	return &InstrIter{
		Cur: instr,
		BB:  instr.GetBasicBlock(),
	}
}

// Iterate from the first instruction of basic block
func NewInstrIterFromBlock(bb *BasicBlock) *InstrIter {
	return &InstrIter{
		Cur: bb.Head,
		BB:  bb,
	}
}

func (i *InstrIter) IsValid() bool { return i.Cur != nil }

func (i *InstrIter) Next() { i.Cur = i.Cur.GetNext() }

func (i *InstrIter) Prev() { i.Cur = i.Cur.GetPrev() }

func (i *InstrIter) HasNext() bool { return i.Cur.GetNext() != nil }

func (i *InstrIter) HasPrev() bool { return i.Cur.GetPrev() != nil }

// Insert instruction before current instruction, and point to that one.
func (i *InstrIter) InsertBefore(instr IInstr) {
	prev := i.Cur.GetPrev()
	cur := i.Cur
	if cur == nil { // empty block
		i.Cur = instr
		i.BB.Head = instr
		i.BB.Tail = instr
		return
	}
	if prev != nil { // cur is not the first node
		// prev -> instr - cur
		prev.SetNext(instr)
	} else {
		// head -> instr
		i.BB.Head = instr
	}
	// prev <-> instr - cur
	instr.SetPrev(prev)
	// instr <- cur
	cur.SetPrev(instr)
	// instr <-> cur
	instr.SetNext(cur)
	// point to the added instruction
	i.Cur = instr
}

// // Insert instruction after current instruction, and point to that one.
func (i *InstrIter) InsertAfter(instr IInstr) {
	next := i.Cur.GetNext()
	cur := i.Cur
	if cur == nil { // empty block
		i.Cur = instr
		i.BB.Head = instr
		i.BB.Tail = instr
		return
	}
	if next != nil { // cur is not the last node
		// cur - instr <- next
		next.SetPrev(instr)
	} else {
		// instr <- tail
		i.BB.Tail = instr
	}
	// cur - instr <-> next
	instr.SetNext(next)
	// cur -> instr
	cur.SetNext(instr)
	// cur <-> instr
	instr.SetPrev(cur)
	// point to added instruction
	i.Cur = instr
}

func (i *InstrIter) Remove() {
	if i.Cur == nil {
		return // no instruction to remove
	}
	prev, next := i.Cur.GetPrev(), i.Cur.GetNext()
	if prev != nil { // not the first node
		prev.SetNext(next)
	} else {
		i.BB.Head = next
	}
	if next != nil { // not the last node
		next.SetPrev(prev)
	} else {
		i.BB.Tail = prev
	}
	i.Cur = next
}

type BaseInstr struct {
	// an instruction also serves as a node in the linked list of a basic block
	Prev, Next IInstr
	// the basic block that this instruction lies in
	BB *BasicBlock
}

func NewBaseInstr(bb *BasicBlock) *BaseInstr {
	return &BaseInstr{BB: bb}
}

func (i *BaseInstr) GetPrev() IInstr { return i.Prev }

func (i *BaseInstr) SetPrev(instr IInstr) { i.Prev = instr }

func (i *BaseInstr) GetNext() IInstr { return i.Next }

func (i *BaseInstr) SetNext(instr IInstr) { i.Next = instr }

func (i *BaseInstr) GetBasicBlock() *BasicBlock { return i.BB }

// Move data from one operand to another
type Move struct {
	BaseInstr
	Src, Dst IValue
	Type     IType
}

func NewMove(bb *BasicBlock, src, dst IValue) *Move {
	if _, ok := dst.(*ImmValue); ok {
		panic(NewIRError("destination operand cannot be an immediate"))
	}
	if !src.GetType().IsIdentical(dst.GetType()) {
		panic(NewIRError(
			fmt.Sprintf("source and destination operands are not of same type"),
		))
	}
	return &Move{
		BaseInstr: *NewBaseInstr(bb),
		Src:       src,
		Dst:       dst,
		Type:      src.GetType(),
	}
}

// Load value from pointer to an operand
type Load struct {
	BaseInstr
	Src, Dst IValue
	Type     IType
}

func NewLoad(bb *BasicBlock, src, dst IValue) *Load {
	if src.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("source operand is not pointer type"))
	}
	baseType := src.GetType().(*PtrType).Base
	if !baseType.IsIdentical(dst.GetType()) && baseType.GetTypeEnum() != Void {
		panic(NewIRError("base type of source is not identical to destination type"))
	}
	return &Load{
		BaseInstr: *NewBaseInstr(bb),
		Src:       src,
		Dst:       dst,
		Type:      dst.GetType(),
	}
}

// Store the value in an operand to a pointer
type Store struct {
	BaseInstr
	Src, Dst IValue
	Type     IType
}

func NewStore(bb *BasicBlock, src, dst IValue) *Store {
	if dst.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("destination operand is not pointer type"))
	}
	baseType := dst.GetType().(*PtrType).Base
	if !baseType.IsIdentical(src.GetType()) && baseType.GetTypeEnum() != Void {
		panic(NewIRError("base type of destination is not identical to source type"))
	}
	return &Store{
		BaseInstr: *NewBaseInstr(bb),
		Src:       src,
		Dst:       dst,
		Type:      src.GetType(),
	}
}

// Allocate memory in heap space
type Malloc struct {
	BaseInstr
	Return IValue
	Size   int // decided by the type the base type of pointer
}

func NewMalloc(bb *BasicBlock, ret IValue) *Malloc {
	if ret.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("source operand is not pointer"))
	}
	baseType := ret.GetType().(*PtrType).Base
	if baseType.GetTypeEnum() == Void {
		panic(NewIRError("source operand is void pointer"))
	}
	return &Malloc{
		BaseInstr: *NewBaseInstr(bb),
		Return:    ret,
		Size:      baseType.GetSize(),
	}
}

// Get pointer to elements in data aggregate (array or struct)
type GetPtr struct {
	BaseInstr
	Base    IValue // base operand, must be data aggregate
	Result  IValue // result pointer
	Indices []IValue
	Offset  []int // offset for struct, width for array
}

func NewGetPtr(bb *BasicBlock, base, result IValue, indices []IValue) *GetPtr {
	// Check aggregate type and build offset list
	curType := base.GetType()
	offset := make([]int, len(indices))

	for dim := 0; dim < len(indices); dim++ {
		// Check index operand type
		if indices[dim].GetType().GetTypeEnum() != I64 {
			panic(NewIRError("index is not an integer"))
		}

		// Compute offset according to type of aggregate
		switch curType.GetTypeEnum() {
		case Struct:
			structType := curType.(*StructType)
			immIdx, ok := indices[dim].(*ImmValue)
			if !ok {
				panic(NewIRError("struct index is not an immediate"))
			}
			index := immIdx.Value.(int)
			offset[dim] = structType.Field[index].Offset
			curType = structType.Field[index].Type

		case Array:
			arrayType := curType.(*ArrayType)
			offset[dim] = arrayType.Elem.GetSize()
			curType = arrayType.Elem

		default:
			panic(NewIRError("not aggregate type"))
		}
	}

	// Check result operand
	ptrType, ok := result.GetType().(*PtrType)
	if !ok {
		panic(NewIRError("result is not a pointer"))
	}
	if !curType.IsIdentical(ptrType.Base) {
		panic(NewIRError("invalid result type"))
	}

	return &GetPtr{
		BaseInstr: *NewBaseInstr(bb),
		Base:      base,
		Result:    result,
		Indices:   indices,
		Offset:    offset,
	}
}

func (p *GetPtr) AppendIndex(index IValue, result IValue) *GetPtr {
	return NewGetPtr(p.BB, p.Base, result, append(p.Indices, index))
}

// Add offset to a pointer
type PtrOffset struct {
	BaseInstr
	Src, Dst IValue // must be pointer type
	Offset   int    // evaluated at compile time.
}

func NewPtrOffset(bb *BasicBlock, src, dst IValue, offset int) *PtrOffset {
	if src.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("source operand is not pointer"))
	}
	if dst.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("source operand is not pointer"))
	}
	return &PtrOffset{
		BaseInstr: *NewBaseInstr(bb),
		Src:       src,
		Dst:       dst,
		Offset:    offset,
	}
}

// Set specified memory space to all zero
type Clear struct {
	BaseInstr
	Value IValue
}

func NewClear(bb *BasicBlock, value IValue) *Clear {
	return &Clear{
		BaseInstr: *NewBaseInstr(bb),
		Value:     value,
	}
}

type UnaryOp int

const (
	NEG UnaryOp = iota // integer, float
	NOT                // integer
)

type Unary struct {
	BaseInstr
	Op              UnaryOp
	Operand, Result IValue
}

func NewUnary(bb *BasicBlock, op UnaryOp, operand, result IValue) *Unary {
	if !operand.GetType().IsIdentical(result.GetType()) {
		panic(NewIRError("result type incompatible with operand type"))
	}

	switch op {
	case NEG:
		if !operand.GetType().GetTypeEnum().Match(Integer | Float) {
			panic(NewIRError("invalid operand type"))
		}
	case NOT:
		if !operand.GetType().GetTypeEnum().Match(Integer) {
			panic(NewIRError("invalid operand type"))
		}
	}

	return &Unary{
		BaseInstr: *NewBaseInstr(bb),
		Op:        op,
		Operand:   operand,
		Result:    result,
	}
}

type BinaryOp int

const (
	ADD BinaryOp = iota
	SUB
	MUL
	DIV
	MOD
	AND
	OR
	XOR
	SHL
	SHR
	EQ
	NE
	LT
	LE
	GT
	GE
)

const (
	ArithmeticOp = ADD | SUB | MUL | DIV | AND | OR | XOR | SHL | SHR
	CompareOp    = EQ | NE | LT | LE | GT | GE
)

type Binary struct {
	BaseInstr
	Op                  BinaryOp
	Left, Right, Result IValue
}

func NewBinary(bb *BasicBlock, op BinaryOp, left, right, result IValue) *Binary {
	// Check type equivalence of operands
	if !left.GetType().IsIdentical(right.GetType()) {
		panic(NewIRError("two operands are not of same type"))
	}

	// Check type enum of operands
	switch op {
	case ADD, SUB, MUL, DIV:
		if !left.GetType().GetTypeEnum().Match(Integer | Float) {
			panic(NewIRError("invalid operand type"))
		}
	case MOD, AND, OR, XOR:
		if !left.GetType().GetTypeEnum().Match(Integer) {
			panic(NewIRError("invalid operand type"))
		}
	case SHL, SHR, EQ, NE, LT, LE, GT, GE:
		if !left.GetType().GetTypeEnum().Match(Integer &^ I1) {
			panic(NewIRError("invalid operand type"))
		}
	}

	// Check result type
	switch op {
	case ADD, SUB, MUL, DIV, AND, OR, XOR, SHL, SHR:
		if !result.GetType().IsIdentical(left.GetType()) {
			panic(NewIRError("invalid result type"))
		}
	case EQ, NE, LT, LE, GT, GE:
		if !result.GetType().GetTypeEnum().Match(I1) {
			panic(NewIRError("invalid result type"))
		}
	}

	return &Binary{
		BaseInstr: *NewBaseInstr(bb),
		Op:        op,
		Left:      left,
		Right:     right,
		Result:    result,
	}
}

type Jump struct {
	BaseInstr
	Target *BasicBlock
}

func NewJump(cur, target *BasicBlock) *Jump {
	return &Jump{
		BaseInstr: *NewBaseInstr(cur),
		Target:    target,
	}
}

type Branch struct {
	BaseInstr
	Cond        IValue
	True, False *BasicBlock
}

func NewBranch(cur *BasicBlock, cond IValue, bTrue, bFalse *BasicBlock) *Branch {
	if cond.GetType().GetTypeEnum() != I1 {
		panic(NewIRError("wrong condition value type"))
	}
	return &Branch{
		BaseInstr: *NewBaseInstr(cur),
		Cond:      cond,
		True:      bTrue,
		False:     bFalse,
	}
}

type Call struct {
	BaseInstr
	Func IValue
	Args []IValue
	Ret  IValue // struct that accept return value
}

func NewCall(bb *BasicBlock, fun IValue, args []IValue, ret IValue) *Call {
	// Check parameter type
	funcType, ok := fun.GetType().(*FuncType)
	if !ok {
		panic(NewIRError("cannot call a non-function value"))
	}
	if len(funcType.Param) != len(args) {
		panic(NewIRError(
			fmt.Sprintf("wrong argument number, want %d, have %d",
				len(funcType.Param), len(args)),
		))
	}
	for i := range funcType.Param {
		if !funcType.Param[i].IsIdentical(args[i].GetType()) {
			panic(NewIRError("invalid argument type"))
		}
	}

	// Check return type
	if len(funcType.Return.Field) == 0 { // no sense in accessing return operands
		goto Construct
	} else if ret == nil {
		panic(NewIRError("return operand not provided"))
	}
	if !funcType.Return.IsIdentical(ret.GetType()) {
		panic(NewIRError("invalid operand type"))
	}

Construct:
	return &Call{
		BaseInstr: *NewBaseInstr(bb),
		Func:      fun,
		Args:      args,
		Ret:       ret,
	}
}

type Return struct {
	BaseInstr
	Func   *Func
	Values []IValue
}

func NewReturn(bb *BasicBlock, fun *Func, values []IValue) *Return {
	paramType := fun.Type.(*FuncType).Param
	if len(paramType) != len(values) {
		panic(NewIRError(
			fmt.Sprintf("wrong return number, want %d, have %d", len(paramType),
				len(values)),
		))
	}
	for i := range paramType {
		if !paramType[i].IsIdentical(values[i].GetType()) {
			panic(NewIRError("invalid return type"))
		}
	}
	return &Return{
		BaseInstr: *NewBaseInstr(bb),
		Func:      fun,
		Values:    values,
	}
}
