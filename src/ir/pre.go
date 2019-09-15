package ir

import "fmt"

// Partial Redundancy Elimination in SSA Form by Chow et al. [1999].
type PREOpt struct {
	prg *Program
	fun *Func
	DF  map[*BasicBlock]map[*BasicBlock]bool
}

func NewPREOpt(prg *Program) *PREOpt {
	return &PREOpt{prg: prg}
}

// SSAPRE operates on lexically identified expressions.
type LexIdentExpr struct {
	op  string
	opd [2]string
}

func opdToStr(val IValue) string {
	switch val.(type) {
	case *Immediate:
		imm := val.(*Immediate)
		switch imm.Type.GetTypeEnum() {
		case I1:
			val := imm.Value.(bool)
			if val {
				return "1"
			} else {
				return "0"
			}
		case I64:
			return fmt.Sprintf("%d", imm.Value.(int))
		case F64:
			return fmt.Sprintf("%f", imm.Value.(float64))
		}
	case *Variable:
		sym := val.(*Variable).Symbol
		return sym.Name // the SSA versions of the variables are ignored.
	}
	return ""
}

func newLexIdentExpr(instr IInstr) *LexIdentExpr {
	switch instr.(type) {
	case *Unary:
		unary := instr.(*Unary)
		return &LexIdentExpr{
			op:  unaryOpStr[unary.Op],
			opd: [2]string{opdToStr(unary.Result)},
		}
	case *Binary:
		binary := instr.(*Binary)
		return &LexIdentExpr{
			op:  binaryOpStr[binary.Op],
			opd: [2]string{opdToStr(binary.Left), opdToStr(binary.Right)},
		}
	default:
		return nil // other instructions not considered
	}
}

func (e *LexIdentExpr) hasOpd(name string) bool {
	return name == e.opd[0] || name == e.opd[1]
}

func (e *LexIdentExpr) opdIndex(name string) int {
	if name == e.opd[0] {
		return 0
	}
	if name == e.opd[1] {
		return 1
	}
	return -1
}

// Common interface of all evaluations of expression (Section 3.1)
// 1. a real occurrence of expression
// 2. a Phi occurrence
// 3. an assignment to an operand of expression
type IEval interface {
	getPrev() IEval
	setPrev(eval IEval)
	getNext() IEval
	setNext(eval IEval)
}

type BaseEval struct {
	prev, next IEval // as linked list node
}

func (o *BaseEval) getPrev() IEval { return o.prev }

func (o *BaseEval) setPrev(occur IEval) { o.prev = occur }

func (o *BaseEval) getNext() IEval { return o.next }

func (o *BaseEval) setNext(occur IEval) { o.next = occur }

// Common interface for all occurrences of expressions (Section 3.2)
// 1. an real occurrence
// 2. inserted Phi operation
// 3. Phi operands
type IOccur interface {
	getVersion() int
	setVersion(ver int)
	getDef() IOccur
	setDef(occur IOccur)
	isIdentical(o2 IOccur) bool
}

// Shared fields of all occurrences
type BaseOccur struct {
	def     IOccur // reference to the representative occurrence
	version int
}

func (o *BaseOccur) getVersion() int { return o.version }

func (o *BaseOccur) setVersion(ver int) { o.version = ver }

func (o *BaseOccur) getDef() IOccur { return o.def }

func (o *BaseOccur) setDef(occur IOccur) { o.def = occur }

func (o *BaseOccur) isIdentical(o2 IOccur) bool { return o.version == o2.getVersion() }

// Real occurrence, computation in the original program
// Both evaluation and occurrence of an expression
type RealOccur struct {
	BaseEval
	BaseOccur
	instr IInstr
}

func newRealOccur(instr IInstr) *RealOccur {
	return &RealOccur{
		BaseEval: BaseEval{},
		BaseOccur: BaseOccur{
			version: 0,
		},
		instr: instr,
	}
}

// Assignment to an operand of expression
// Just an evaluation of expression, not an occurrence
type OpdAssign struct {
	BaseEval
	opd   string // which operand is assigned
	instr IInstr // related instruction in CFG
}

func newOpdAssign(opd string, instr IInstr) *OpdAssign {
	return &OpdAssign{
		BaseEval: BaseEval{},
		opd:      opd,
		instr:    instr,
	}
}

// Operands in redundancy factoring operation
// Just an occurrence of expression, not an evaluation
type BigPhiOpd struct {
	BaseOccur
}

func newBigPhiOpd() *BigPhiOpd {
	return &BigPhiOpd{
		BaseOccur: BaseOccur{
			version: 0,
		},
	}
}

// Redundancy factoring operator
// Both occurrence and evaluation of expression
type BigPhi struct {
	BaseEval
	BaseOccur
	bbToOccur map[*BasicBlock]*BigPhiOpd
}

func newBigPhi(bbToOccur map[*BasicBlock]*BigPhiOpd) *BigPhi {
	return &BigPhi{
		BaseEval: BaseEval{},
		BaseOccur: BaseOccur{
			version: 0,
		},
		bbToOccur: bbToOccur,
	}
}

// A linked list of all occurrences of
type BlockEvalList struct {
	head, tail IEval
}

func (b *BlockEvalList) pushFront(eval IEval) {
	if b.head == nil {
		b.head, b.tail = eval, eval
		return
	}
	eval.setNext(b.head)
	b.head.setPrev(eval)
	b.head = eval
}

func (b *BlockEvalList) pushBack(eval IEval) {
	if b.tail == nil {
		b.head, b.tail = eval, eval
		return
	}
	eval.setPrev(b.tail)
	b.tail.setNext(eval)
	b.tail = eval
}

type EvalListTable map[*BasicBlock]*BlockEvalList

// Work-list driven PRE, see Section 5.1 of paper
func (o *PREOpt) Optimize(fun *Func) {
	// Initialize work-list
	o.fun = fun
	workList := make(map[LexIdentExpr]bool)
	removeOneExpr := func() LexIdentExpr {
		var expr LexIdentExpr
		for e := range workList {
			expr = e
		}
		delete(workList, expr)
		return expr
	}

	// Collect occurrences of expressions
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			expr := newLexIdentExpr(iter.Get())
			if expr == nil {
				continue
			}
			workList[*expr] = true
		}
	}, DepthFirst)

	// Eliminate redundancies iteratively
	o.DF = computeDF(fun)
	for len(workList) > 0 {
		// Pick one expression and build FRG for that
		expr := new(LexIdentExpr)
		*expr = removeOneExpr()
		o.buildFRG(fun, expr)

		// Perform backward and forward data flow propagation
		// Pinpoint locations for computations to be inserted
		// Transform code to form optimized program
	}
}

func copyBlockSet(set map[*BasicBlock]bool) map[*BasicBlock]bool {
	cp := make(map[*BasicBlock]bool)
	for b := range set {
		cp[b] = true
	}
	return cp
}

func (o *PREOpt) buildFRG(fun *Func, expr *LexIdentExpr) EvalListTable {
	// Build evaluation
	table := make(EvalListTable)           // table of evaluation list
	occurred := make(map[*BasicBlock]bool) // set of blocks where there are real occurrences
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		table[block] = &BlockEvalList{}
		evalList := table[block]
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			instr := iter.Get()
			def := instr.GetDef()
			// Try to build as an assignment to operands
			if def == nil {
				continue // no symbol defined in this instruction. don't care
			}
			sym := (*def).(*Variable).Symbol
			if expr.hasOpd(sym.Name) {
				evalList.pushBack(newOpdAssign(expr.opd[expr.opdIndex(sym.Name)], instr))
				continue // cannot also be an real occurrence
			}
			// Try to build as real occurrence
			instrExpr := newLexIdentExpr(instr)
			if instrExpr == nil {
				continue // this type of instruction is not in consideration
			}
			if *instrExpr != *expr {
				continue // not the expression we build FRG fors
			}
			evalList.pushBack(newRealOccur(instr))
			occurred[block] = true // add to real occurrence set
		}
	}, DepthFirst)

	// Insert Phi functions
	// Insert in iterated dominance frontiers of expression
	inserted := make(map[*BasicBlock]bool)
	workList := copyBlockSet(occurred)
	insertPhi := func(block *BasicBlock) {
		if inserted[block] {
			return
		}
		bbToOpd := make(map[*BasicBlock]*BigPhiOpd)
		for pred := range block.Pred {
			bbToOpd[pred] = newBigPhiOpd()
		}
		table[block].pushFront(newBigPhi(bbToOpd))
		inserted[block] = true
	}
	for len(workList) > 0 {
		block := removeOneBlock(workList)
		for df := range o.DF[block] {
			insertPhi(df)
			if !occurred[df] && !inserted[df] { // there was no occurrence before
				workList[df] = true
			}
		}
	}

	// Insert in blocks where there is a phi instruction of operands of the expression
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			switch iter.Get().(type) {
			case *Phi:
				def := iter.Get().GetDef()
				if def == nil {
					continue
				}
				switch (*def).(type) {
				case *Variable:
					sym := (*def).(*Variable).Symbol
					if expr.hasOpd(sym.Name) {
						insertPhi(block)
					}
				}
			}
		}
	}, DepthFirst)

	// Print inserted Phis
	fmt.Println(expr)
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		fmt.Println(block.Name)
		for cur := table[block].head; cur != nil; cur = cur.getNext() {
			switch cur.(type) {
			case *BigPhi:
				fmt.Println("\tBigPhi", cur.(*BigPhi).version)
			case *RealOccur:
				fmt.Println("\tRealOccur", cur.(*RealOccur).version)
			case *OpdAssign:
				fmt.Println("\tOpdAssign", cur.(*OpdAssign).opd)
			}
		}
	}, DepthFirst)
	fmt.Println()

	return table
}
