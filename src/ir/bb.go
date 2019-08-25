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
	instr.SetPrev(b.Tail) // prev_tail <- instr
	b.Tail.SetNext(instr) // prev_tail <-> instr
	b.Tail = instr        // prev_tail <-> instr <- tail
	instr.SetNext(nil)
}
