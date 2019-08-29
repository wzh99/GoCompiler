package ir

type BasicBlock struct {
	// Block label
	Name string
	// Internally. a block is a linked list of instructions.
	Head, Tail IInstr
	// In a function, a series of basic blocks form a control flow graph.
	// This list can be constructed when instructions are added to the function.
	Enter, Exit []*BasicBlock
	// The function this block lies in.
	Func *Func
}

func NewBasicBlock(label string, fun *Func) *BasicBlock {
	return &BasicBlock{
		Name:  label,
		Head:  nil, // initially an empty list
		Tail:  nil,
		Enter: make([]*BasicBlock, 0),
		Exit:  make([]*BasicBlock, 0),
		Func:  fun,
	}
}

func (b *BasicBlock) Append(instr IInstr) {
	if b.Head == nil {
		b.Head = instr
		b.Tail = instr
		return
	}
	instr.SetPrev(b.Tail) // prev_tail <- instr
	b.Tail.SetNext(instr) // prev_tail <-> instr
	b.Tail = instr        // prev_tail <-> instr <- tail
	instr.SetNext(nil)
}

// Automatically create a jump instruction in the receiver block, and create edges in
// two blocks.
func (b *BasicBlock) JumpTo(b2 *BasicBlock) {
	b.Append(NewJump(b, b2))
	b.Exit = append(b.Exit, b2)
	b2.Enter = append(b2.Enter, b)
}

func (b *BasicBlock) BranchTo(cond IValue, trueBB, falseBB *BasicBlock) {
	b.Append(NewBranch(b, cond, trueBB, falseBB))
	b.Exit = append(b.Exit, trueBB, falseBB)
	trueBB.Enter = append(trueBB.Enter, b)
	falseBB.Enter = append(falseBB.Enter, b)
}
