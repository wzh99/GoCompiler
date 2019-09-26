package ir

import "fmt"

type IInstr interface {
	GetPrev() IInstr
	SetPrev(instr IInstr)
	GetNext() IInstr
	SetNext(instr IInstr)
	GetBasicBlock() *BasicBlock
	SetBasicBlock(bb *BasicBlock)
	// Values may be changed by other functions, pointers should be returned
	GetDef() *IValue
	GetOpd() []*IValue
}

// Instruction iterator
type InstrIter struct {
	cur IInstr
	bb  *BasicBlock
}

// Iterate from the first instruction of basic block
func NewIterFromBlock(bb *BasicBlock) *InstrIter {
	return &InstrIter{
		cur: bb.Head,
		bb:  bb,
	}
}

func NewIterFromInstr(instr IInstr) *InstrIter {
	return &InstrIter{
		cur: instr,
		bb:  instr.GetBasicBlock(),
	}
}

func (i *InstrIter) Get() IInstr { return i.cur }

func (i *InstrIter) Valid() bool { return i.cur != nil }

func (i *InstrIter) MoveNext() { i.cur = i.cur.GetNext() }

func (i *InstrIter) MovePrev() { i.cur = i.cur.GetPrev() }

func (i *InstrIter) HasNext() bool { return i.cur.GetNext() != nil }

func (i *InstrIter) HasPrev() bool { return i.cur.GetPrev() != nil }

// Insert instruction before current instruction, and point to that one.
func (i *InstrIter) InsertBefore(instr IInstr) {
	prev := i.cur.GetPrev()
	cur := i.cur
	if cur == nil { // empty block
		i.cur = instr
		i.bb.Head = instr
		i.bb.Tail = instr
		return
	}
	if prev != nil { // cur is not the first node
		// prev -> instr - cur
		prev.SetNext(instr)
	} else {
		// head -> instr
		i.bb.Head = instr
	}
	// prev <-> instr - cur
	instr.SetPrev(prev)
	// instr <- cur
	cur.SetPrev(instr)
	// instr <-> cur
	instr.SetNext(cur)
}

// // Insert instruction after current instruction, and point to that one.
func (i *InstrIter) InsertAfter(instr IInstr) {
	next := i.cur.GetNext()
	cur := i.cur
	if cur == nil { // empty block
		i.cur = instr
		i.bb.Head = instr
		i.bb.Tail = instr
		return
	}
	if next != nil { // cur is not the last node
		// cur - instr <- next
		next.SetPrev(instr)
	} else {
		// instr <- tail
		i.bb.Tail = instr
	}
	// cur - instr <-> next
	instr.SetNext(next)
	// cur -> instr
	cur.SetNext(instr)
	// cur <-> instr
	instr.SetPrev(cur)
}

func (i *InstrIter) Remove() {
	if i.cur == nil {
		return // no instruction to remove
	}
	prev, next := i.cur.GetPrev(), i.cur.GetNext()
	if prev != nil { // not the first node
		prev.SetNext(next)
	} else {
		i.bb.Head = next
	}
	if next != nil { // not the last node
		next.SetPrev(prev)
	} else {
		i.bb.Tail = prev
	}
	i.cur.SetPrev(nil)
	i.cur.SetNext(nil)
	i.cur = next
}

func (i *InstrIter) Replace(instr IInstr) {
	if i.cur == nil {
		return // no instruction to remove
	}
	prev, next := i.cur.GetPrev(), i.cur.GetNext()
	if prev != nil { // not the first node
		prev.SetNext(instr)
	} else {
		i.bb.Head = instr
	}
	if next != nil { // not the last node
		next.SetPrev(instr)
	} else {
		i.bb.Tail = instr
	}
	instr.SetPrev(prev)
	instr.SetNext(next)
	i.cur = instr
}

type BaseInstr struct {
	// an instruction also serves as a node in the linked list of a basic block
	Prev, Next IInstr
	// the basic block that this instruction lies in
	BB *BasicBlock
}

func (i *BaseInstr) GetPrev() IInstr { return i.Prev }

func (i *BaseInstr) SetPrev(instr IInstr) { i.Prev = instr }

func (i *BaseInstr) GetNext() IInstr { return i.Next }

func (i *BaseInstr) SetNext(instr IInstr) { i.Next = instr }

func (i *BaseInstr) GetBasicBlock() *BasicBlock { return i.BB }

func (i *BaseInstr) SetBasicBlock(bb *BasicBlock) { i.BB = bb }

func (i *BaseInstr) GetDef() *IValue { return nil }

func (i *BaseInstr) GetOpd() []*IValue { return nil }

// Move data from one operand to another
type Move struct {
	BaseInstr
	Src, Dst IValue
	Type     IType
}

func NewMove(src, dst IValue) *Move {
	if _, ok := dst.(*Constant); ok {
		panic(NewIRError("destination operand cannot be an immediate"))
	}
	if !src.GetType().IsIdentical(dst.GetType()) {
		panic(NewIRError(
			fmt.Sprintf("source and destination operands are not of same type"),
		))
	}
	return &Move{
		Src:  src,
		Dst:  dst,
		Type: src.GetType(),
	}
}

func (m *Move) GetDef() *IValue { return &m.Dst }

func (m *Move) GetOpd() []*IValue { return []*IValue{&m.Src} }

// Load value from pointer to an operand
type Load struct {
	BaseInstr
	Src, Dst IValue
	Type     IType
}

func NewLoad(src, dst IValue) *Load {
	if src.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("source operand is not pointer type"))
	}
	baseType := src.GetType().(*PtrType).Base
	if !baseType.IsIdentical(dst.GetType()) && baseType.GetTypeEnum() != Void {
		panic(NewIRError("base type of source is not identical to destination type"))
	}
	return &Load{
		Src:  src,
		Dst:  dst,
		Type: dst.GetType(),
	}
}

func (l *Load) GetDef() *IValue { return &l.Dst }

func (l *Load) GetOpd() []*IValue { return []*IValue{&l.Src} }

// Store the value in an operand to a pointer
type Store struct {
	BaseInstr
	Src, Dst IValue
	Type     IType
}

func NewStore(src, dst IValue) *Store {
	if dst.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("destination operand is not pointer type"))
	}
	baseType := dst.GetType().(*PtrType).Base
	if !baseType.IsIdentical(src.GetType()) && baseType.GetTypeEnum() != Void {
		panic(NewIRError("base type of destination is not identical to source type"))
	}
	return &Store{
		Src:  src,
		Dst:  dst,
		Type: src.GetType(),
	}
}

// Only focus the value destination pointer, not the memory content it points to.
func (s *Store) GetOpd() []*IValue { return []*IValue{&s.Src, &s.Dst} }

// Allocate memory in heap space
type Malloc struct {
	BaseInstr
	Result IValue
}

func NewMalloc(ret IValue) *Malloc {
	if ret.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("source operand is not pointer"))
	}
	baseType := ret.GetType().(*PtrType).Base
	if baseType.GetTypeEnum() == Void {
		panic(NewIRError("source operand is void pointer"))
	}
	return &Malloc{
		Result: ret,
	}
}

func (m *Malloc) GetDef() *IValue { return &m.Result }

// Get pointer to elements in data aggregate (array or struct)
type GetPtr struct {
	BaseInstr
	Base    IValue // base operand, must be data aggregate
	Result  IValue // result pointer
	Indices []IValue
	Offset  []int // offset for struct, width for array
}

func NewGetPtr(base, result IValue, indices []IValue) *GetPtr {
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
			immIdx, ok := indices[dim].(*Constant)
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
		Base:    base,
		Result:  result,
		Indices: indices,
		Offset:  offset,
	}
}

func (p *GetPtr) AppendIndex(index IValue, result IValue) *GetPtr {
	return NewGetPtr(p.Base, result, append(p.Indices, index))
}

func (p *GetPtr) GetDef() *IValue { return &p.Result }

func (p *GetPtr) GetOpd() []*IValue {
	use := []*IValue{&p.Base}
	for i := range p.Indices {
		use = append(use, &p.Indices[i])
	}
	return use
}

// Add offset to a pointer
type PtrOffset struct {
	BaseInstr
	Src, Dst IValue // must be pointer type
	Offset   int    // evaluated at compile time.
}

func NewPtrOffset(src, dst IValue, offset int) *PtrOffset {
	if src.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("source operand is not pointer"))
	}
	if dst.GetType().GetTypeEnum() != Pointer {
		panic(NewIRError("source operand is not pointer"))
	}
	return &PtrOffset{
		Src:    src,
		Dst:    dst,
		Offset: offset,
	}
}

func (p *PtrOffset) GetDef() *IValue { return &p.Dst }

func (p *PtrOffset) GetOpd() []*IValue { return []*IValue{&p.Src} }

// Set memory content of specified value to all zero
type Clear struct {
	BaseInstr
	Value IValue
}

func NewClear(value IValue) *Clear {
	return &Clear{
		Value: value,
	}
}

func (c *Clear) GetDef() *IValue { return &c.Value }

type UnaryOp int

const (
	NEG UnaryOp = iota // integer, float
	NOT                // integer
)

var unaryOpStr = map[UnaryOp]string{
	NEG: "neg", NOT: "not",
}

var unaryStrToOp = map[string]UnaryOp{
	"neg": NEG, "not": NOT,
}

type Unary struct {
	BaseInstr
	Op              UnaryOp
	Operand, Result IValue
}

func NewUnary(op UnaryOp, operand, result IValue) *Unary {
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
		Op:      op,
		Operand: operand,
		Result:  result,
	}
}

func (u *Unary) GetDef() *IValue { return &u.Result }

func (u *Unary) GetOpd() []*IValue { return []*IValue{&u.Operand} }

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

var binaryOpStr = map[BinaryOp]string{
	ADD: "add", SUB: "sub", MUL: "mul", DIV: "div", MOD: "mod", AND: "and", OR: "or", XOR: "xor",
	SHL: "shl", SHR: "shr", EQ: "eq", NE: "ne", LT: "lt", LE: "le", GT: "gt", GE: "ge",
}

var binaryStrToOp map[string]BinaryOp

func init() {
	binaryStrToOp = make(map[string]BinaryOp)
	for op, str := range binaryOpStr {
		binaryStrToOp[str] = op
	}
}

var commutative = map[BinaryOp]bool{
	ADD: true, MUL: true, AND: true, OR: true, XOR: true, EQ: true, NE: true,
}

type Binary struct {
	BaseInstr
	Op                  BinaryOp
	Left, Right, Result IValue
}

func NewBinary(op BinaryOp, left, right, result IValue) *Binary {
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
		Op:     op,
		Left:   left,
		Right:  right,
		Result: result,
	}
}

func (b *Binary) GetDef() *IValue { return &b.Result }

func (b *Binary) GetOpd() []*IValue { return []*IValue{&b.Left, &b.Right} }

type Jump struct {
	BaseInstr
	Target *BasicBlock
}

func NewJump(target *BasicBlock) *Jump {
	return &Jump{
		Target: target,
	}
}

type Branch struct {
	BaseInstr
	Cond        IValue
	True, False *BasicBlock
}

func NewBranch(cond IValue, bTrue, bFalse *BasicBlock) *Branch {
	if cond.GetType().GetTypeEnum() != I1 {
		panic(NewIRError("wrong condition value type"))
	}
	return &Branch{
		Cond:  cond,
		True:  bTrue,
		False: bFalse,
	}
}

func (b *Branch) GetOpd() []*IValue { return []*IValue{&b.Cond} }

type Call struct {
	BaseInstr
	Func IValue
	Args []IValue
	Ret  IValue // struct that accept return value
}

func NewCall(fun IValue, args []IValue, ret IValue) *Call {
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
		Func: fun,
		Args: args,
		Ret:  ret,
	}
}

type Return struct {
	BaseInstr
	Func   *Func
	Values []IValue
}

func (r *Return) GetOpd() []*IValue {
	use := make([]*IValue, 0)
	for i := range r.Values {
		use = append(use, &r.Values[i])
	}
	return use
}

func NewReturn(fun *Func, values []IValue) *Return {
	ret := fun.Type.(*FuncType).Return.Field
	if len(ret) != len(values) {
		panic(NewIRError(
			fmt.Sprintf("wrong return number, want %d, have %d", len(ret),
				len(values)),
		))
	}
	for i, f := range ret {
		if !f.Type.IsIdentical(values[i].GetType()) {
			panic(NewIRError("invalid return type"))
		}
	}
	return &Return{
		Func:   fun,
		Values: values,
	}
}

type PhiOpd struct {
	pred *BasicBlock
	val  IValue
}

type Phi struct {
	BaseInstr
	OpdList []IValue
	BBToOpd map[*BasicBlock]*IValue
	Result  IValue
}

func NewPhi(operands []PhiOpd, result IValue) *Phi {
	p := &Phi{
		OpdList: make([]IValue, len(operands)),
		BBToOpd: make(map[*BasicBlock]*IValue),
		Result:  result,
	}
	// Enforce an order of operands
	for i := 0; i < len(operands)-1; i++ {
		minStr, minIdx := operands[i].pred.Name, i
		for j := i + 1; j < len(operands); j++ {
			if operands[j].pred.Name < minStr {
				minStr, minIdx = operands[j].pred.Name, j
			}
		}
		if minIdx != i {
			operands[i], operands[minIdx] = operands[minIdx], operands[i]
		}
	}
	// Build phi instruction
	for i, entry := range operands {
		bb := entry.pred
		val := entry.val
		if !val.GetType().IsIdentical(result.GetType()) {
			panic(NewIRError(
				fmt.Sprintf("type of operand %s is incompatible with result %s",
					val.ToString(), result.ToString()),
			))
		}
		p.OpdList[i] = val
		p.BBToOpd[bb] = &p.OpdList[i]
	}
	return p
}

func (p *Phi) GetDef() *IValue { return &p.Result }

func (p *Phi) GetOpd() []*IValue {
	use := make([]*IValue, 0)
	for i := range p.OpdList {
		use = append(use, &p.OpdList[i])
	}
	return use
}
