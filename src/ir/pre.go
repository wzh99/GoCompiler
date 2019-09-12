package ir

import "fmt"

// Partial-Redundancy Elimination with Lazy Code Motion
// See Section 13.3 in The Whale Book and Section 10.3.1 in EAC
type PREOpt struct{}

type Expr struct {
	op  string      // operator label
	opd [2]*SSAVert // operands
}

// Expressions are represented with instruction that define them.
type ExprSet map[Expr]bool

func copySet(set ExprSet) ExprSet {
	cp := make(ExprSet)
	for s := range set {
		cp[s] = true
	}
	return cp
}

// A series of local and global data-flow properties of expressions
type ExprProp struct {
	block             *BasicBlock // the block these properties belong to
	transLoc          ExprSet     // local transparency
	antLoc            ExprSet     // local anticipatability
	antIn, antOut     ExprSet     // global anticipatability
	earlIn, earlOut   ExprSet     // earliestness
	delayIn, delayOut ExprSet     // delayedness
	lateIn            ExprSet     // latestness
	isoIn, isoOut     ExprSet     // isolation
	opt, redun        ExprSet     // optimal and redundant expressions
}

func (o *PREOpt) Optimize(fun *Func) {
	// Determine range of optimization
	graph := NewSSAGraph(fun)
	// immediate must be merged so that expression equality can be correctly tested
	graph.MergeImm()
	univ := make(ExprSet) // the set of all instructions being analyzed
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			instr := iter.Cur
			switch instr.(type) {
			case *Unary:
				unary := instr.(*Unary)
				vert := graph.symToVert[unary.Result.(*Variable).Symbol]
				opd := [2]*SSAVert{vert.opd[0]}
				univ[Expr{op: vert.label, opd: opd}] = true
			case *Binary:
				binary := instr.(*Binary)
				vert := graph.symToVert[binary.Result.(*Variable).Symbol]
				opd := [2]*SSAVert{vert.opd[0], vert.opd[1]}
				univ[Expr{op: vert.label, opd: opd}] = true
			}
		}
	}, DepthFirst)
	for expr := range univ {
		fmt.Println(expr)
	}

	// Build local transparency set
	// An expression's value is locally transparent in a block if there are no assignments in
	// the block to variables that occur in the expression.
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {

	}, DepthFirst)
}
