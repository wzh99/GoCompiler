package ir

type BasicBlock struct {
	// Block label
	Label string
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
		Label: label,
		Head:  nil, // initially an empty list
		Tail:  nil,
		Enter: make([]*BasicBlock, 0),
		Exit:  make([]*BasicBlock, 0),
		Func:  fun,
	}
}
