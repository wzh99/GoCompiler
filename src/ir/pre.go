package ir

import (
	"fmt"
	"strconv"
)

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
	index int    // which operand is assigned
	instr IInstr // related instruction in CFG
}

func newOpdAssign(index int, instr IInstr) *OpdAssign {
	return &OpdAssign{
		BaseEval: BaseEval{},
		index:    index,
		instr:    instr,
	}
}

func (a *OpdAssign) getSymbolVersion() int {
	return (*a.instr.GetDef()).(*Variable).Symbol.Ver
}

// Operands in redundancy factoring operation
// Just an occurrence of expression, not an evaluation
type BigPhiOpd struct {
	BaseOccur
	hasRealUse bool // whether there is real occurrence before this operand
}

func newBigPhiOpd() *BigPhiOpd {
	return &BigPhiOpd{
		BaseOccur: BaseOccur{
			version: 0, // negative value indicates unavailability
		},
		hasRealUse: false, // keep false until we find a real use
	}
}

func (o *BigPhiOpd) isAvailable() bool { return o.version < 0 }

// Redundancy factoring operator
// Both occurrence and evaluation of expression
type BigPhi struct {
	BaseEval
	BaseOccur
	bbToOpd  map[*BasicBlock]*BigPhiOpd
	downSafe bool // evaluated before program exit or altered by operand redefinition
}

func newBigPhi(bbToOccur map[*BasicBlock]*BigPhiOpd) *BigPhi {
	return &BigPhi{
		BaseEval: BaseEval{},
		BaseOccur: BaseOccur{
			version: 0,
		},
		bbToOpd:  bbToOccur,
		downSafe: true, // keep true until propagated to be false
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
		table := o.buildFRG(fun, expr)

		// Perform backward and forward data flow propagation
		o.downSafety(fun, table)
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

type ExprStackElem struct {
	exprVer    int
	opdVer     [2]int
	occur, rep IOccur
}

func (e *ExprStackElem) matchOpdVersion(opdStack [2]*OpdStack) bool {
	return e.opdVer[0] == opdStack[0].top() && e.opdVer[1] == opdStack[1].top()
}

type ExprStack struct {
	latest int
	stack  []*ExprStackElem
}

func newExprStack() *ExprStack {
	return &ExprStack{
		latest: 0,
		stack:  make([]*ExprStackElem, 0),
	}
}

func (s *ExprStack) rename(elem *ExprStackElem) {
	s.latest++
	elem.exprVer = s.latest
	s.push(elem)
}

func (s *ExprStack) push(elem *ExprStackElem) { s.stack = append(s.stack, elem) }

func (s *ExprStack) pop() { s.stack = s.stack[:len(s.stack)-1] }

func (s *ExprStack) top() *ExprStackElem { return s.stack[len(s.stack)-1] }

func (s *ExprStack) empty() bool { return len(s.stack) == 0 }

type OpdStack struct {
	stack []int
}

func newVarStack() *OpdStack {
	return &OpdStack{
		stack: []int{0},
	}
}

func (s *OpdStack) push(v int) { s.stack = append(s.stack, v) }

func (s *OpdStack) pop() { s.stack = s.stack[:len(s.stack)-1] }

func (s *OpdStack) top() int { return s.stack[len(s.stack)-1] }

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
				evalList.pushBack(newOpdAssign(expr.opdIndex(sym.Name), instr))
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

	// Rename occurrences in blocks, see Section 3.2
	// Also perform down safety initialization, see Section 3.3
	exprStack := newExprStack()
	opdStacks := [2]*OpdStack{newVarStack(), newVarStack()}

	getOpdVer := func() [2]int {
		return [2]int{opdStacks[0].top(), opdStacks[1].top()}
	}
	clearDownSafe := func() {
		if exprStack.empty() {
			return
		}
		topOccur := exprStack.top().occur
		switch topOccur.(type) {
		case *BigPhi: // no real occurrence of previous version to here
			topOccur.(*BigPhi).downSafe = false
		}
	}

	var visit func(block *BasicBlock)
	visit = func(block *BasicBlock) {
		// Traverse evaluation list in the basic block
		exprPush := 0 // record expression push times to restore stack after visiting children
		opdPush := [2]int{0, 0}
		evalList := table[block]
		for cur := evalList.head; cur != nil; cur = cur.getNext() {
			switch cur.(type) {
			case *BigPhi: // assign a new expression version (redundancy class number)
				occur := cur.(*BigPhi)
				exprStack.rename(&ExprStackElem{
					opdVer: getOpdVer(),
					occur:  occur,
					rep:    occur,
				})
				exprPush++
				occur.version = exprStack.latest

			case *OpdAssign:
				assign := cur.(*OpdAssign)
				ver := assign.getSymbolVersion()
				switch assign.instr.(type) {
				case *Phi:
					opdStacks[assign.index].push(ver) // push operand version
					if !exprStack.empty() { // update operand in top occurrence
						exprStack.top().opdVer[assign.index] = ver
					}
				default: // only push operand version
					opdStacks[assign.index].push(ver)
				}
				opdPush[assign.index]++

			case *RealOccur:
				occur := cur.(*RealOccur)
				// Compare operands in occurrence with ones in available expression
				top := exprStack.top()
				if !exprStack.empty() && o.getRealOccurVer(occur) == top.opdVer {
					occur.version = top.exprVer // assign same class number
					occur.def = top.rep         // refer to representative occurrence
					exprStack.push(&ExprStackElem{
						exprVer: top.exprVer,
						opdVer:  top.opdVer,
						occur:   occur,
						rep:     top.rep,
					})
				} else {
					clearDownSafe() // examine the top occurrence
					exprStack.rename(&ExprStackElem{ // assign a new class number
						opdVer: o.getRealOccurVer(occur),
						occur:  occur,
						rep:    occur,
					})
					occur.version = exprStack.latest
					occur.def = top.rep
				}
				exprPush++
			} // end evaluation switch
		} // end evaluation list iteration

		// Visit Phi occurrence in successors
		for succ := range block.Succ {
			head := table[succ].head
			switch head.(type) {
			case *BigPhi: // there is at most one occurrence in each block
				bigPhi := head.(*BigPhi)
				opd := bigPhi.bbToOpd[block]
				if !exprStack.empty() && exprStack.top().opdVer == getOpdVer() {
					top := exprStack.top()
					opd.version = top.exprVer
					opd.def = top.rep
					switch top.occur.(type) {
					case *RealOccur: // a real occurrence found
						opd.hasRealUse = true
					}
				} else {
					clearDownSafe()  // examine the top occurrence
					opd.version = -1 // value of expression is unavailable
				}
			}
		}

		// Visit all children in dominance tree
		for child := range block.Children {
			visit(child)
		}

		// Clear down safety flag if reaching exit blocks
		if fun.Exit[block] {
			clearDownSafe()
		}

		// Restore stack before exiting this block
		for i := 0; i < exprPush; i++ {
			exprStack.pop()
		}
		for i, s := range opdPush {
			for j := 0; j < s; j++ {
				opdStacks[i].pop()
			}
		}
	}
	visit(fun.Enter)

	// Print inserted Phis
	o.printEval(fun, expr, table)

	return table
}

func (o *PREOpt) getRealOccurVer(occur *RealOccur) [2]int {
	instr := occur.instr
	switch instr.(type) {
	case *Unary:
		unary := instr.(*Unary)
		return [2]int{o.getOpdVer(unary.Operand)}
	case *Binary:
		binary := instr.(*Binary)
		return [2]int{o.getOpdVer(binary.Left), o.getOpdVer(binary.Right)}
	default:
		return [2]int{}
	}
}

func (o *PREOpt) getOpdVer(val IValue) int {
	switch val.(type) {
	case *Immediate:
		return 0 // immediate has only one version
	case *Variable:
		sym := val.(*Variable).Symbol
		return sym.Ver
	default:
		return 0
	}
}

func (o *PREOpt) downSafety(fun *Func, table EvalListTable) {
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		evalList := table[block]
		switch evalList.head.(type) {
		case *BigPhi:
			bigPhi := evalList.head.(*BigPhi)
			if !bigPhi.downSafe {
				for _, opd := range bigPhi.bbToOpd {
					o.resetDownSafety(opd)
				}
			}
		}
	}, PostOrder)
}

func (o *PREOpt) resetDownSafety(opd *BigPhiOpd) {
	if opd.hasRealUse {
		return
	}
	switch opd.def.(type) {
	case *BigPhi:
		def := opd.def.(*BigPhi)
		if !def.downSafe {
			return
		}
		def.downSafe = false
		for _, opd := range def.bbToOpd {
			o.resetDownSafety(opd)
		}
	}
}

func (o *PREOpt) printEval(fun *Func, expr *LexIdentExpr, table EvalListTable) {
	fmt.Println(expr)
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		fmt.Println(block.Name)
		for cur := table[block].head; cur != nil; cur = cur.getNext() {
			switch cur.(type) {
			case *BigPhi:
				bigPhi := cur.(*BigPhi)
				fmt.Printf("\tBigPhi %d %s", bigPhi.version,
					strconv.FormatBool(bigPhi.downSafe))
				for bb, opd := range bigPhi.bbToOpd {
					fmt.Printf(" [%s: %d %s]", bb.Name, opd.version,
						strconv.FormatBool(opd.hasRealUse))
				}
				fmt.Println()
			case *RealOccur:
				fmt.Printf("\tRealOccur %d\n", cur.(*RealOccur).version)
			case *OpdAssign:
				fmt.Printf("\tOpdAssign %s\n", expr.opd[cur.(*OpdAssign).index])
			}
		}
	}, ReversePostOrder)
	fmt.Println()
}
