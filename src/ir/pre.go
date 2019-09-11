package ir

// Partial-Redundancy Elimination with Lazy Code Motion
// See Section 13.3 in The Whale Book and Section 10.3.1 in EAC
type PREOpt struct {
	opt *SSAOpt
}

// Expressions are represented with instruction that define them.
type ExprSet map[IInstr]bool

// A series of local and global data-flow properties of expressions
type ExprProp struct {
	block             *BasicBlock // the basic these properties belong to
	transLoc          ExprSet     // local transparency
	antLoc            ExprSet     // local anticipatability
	antIn, antOut     ExprSet     // global anticipatability
	earlIn, earlOut   ExprSet     // earliestness
	delayIn, delayOut ExprSet     // delayedness
	lateIn            ExprSet     // latestness
	isoIn, isoOut     ExprSet     // isolation
	opt, redun        ExprSet     // optimal and redundant expressions
}
