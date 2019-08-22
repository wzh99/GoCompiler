package ir

type BasicBlock struct {
	// internally. a block is a linked list of instructions
	Head, Tail IInstr
	// in a function, a series of basic blocks form a control flow graph
	Enter, Exit []*BasicBlock
	// the function this block lies in
	Func *Func
}
