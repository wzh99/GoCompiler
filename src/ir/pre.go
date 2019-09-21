package ir

import (
	"fmt"
	"strconv"
	"strings"
)

// Partial Redundancy Elimination in SSA Form by Chow et al. [1999].
type PREOpt struct {
	fun    *Func
	dfPlus map[*BasicBlock]BlockSet
}

func NewPREOpt() *PREOpt {
	return &PREOpt{}
}

// SSA-PRE operates on lexically identified expressions.
type LexIdentExpr struct {
	op  string
	opd [2]string
}

var immPrefix = "imm_"

func opdToStr(val IValue) string {
	switch val.(type) {
	case *Immediate:
		imm := val.(*Immediate)
		switch imm.Type.GetTypeEnum() {
		case I1:
			val := imm.Value.(bool)
			if val {
				return immPrefix + "1"
			} else {
				return immPrefix + "0"
			}
		case I64:
			return fmt.Sprintf("%s%d", immPrefix, imm.Value.(int))
		case F64:
			return fmt.Sprintf("%s%f", immPrefix, imm.Value.(float64))
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

func (e *LexIdentExpr) isUnary() bool {
	return len(e.opd[1]) == 0 // not have second operand
}

func (e *LexIdentExpr) hasImm() bool {
	if e.isUnary() {
		return false // unary instruction has no immediate
	}
	return strings.HasPrefix(e.opd[0], immPrefix) || strings.HasPrefix(e.opd[1], immPrefix)
}

func (e *LexIdentExpr) immIndex() int {
	if strings.HasPrefix(e.opd[0], immPrefix) {
		return 0
	}
	if strings.HasPrefix(e.opd[1], immPrefix) {
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
	remove()
}

type BaseEval struct {
	prev, next IEval // as linked list node
	list       *BlockEvalList
}

func (o *BaseEval) getPrev() IEval { return o.prev }

func (o *BaseEval) setPrev(occur IEval) { o.prev = occur }

func (o *BaseEval) getNext() IEval { return o.next }

func (o *BaseEval) setNext(occur IEval) { o.next = occur }

func (o *BaseEval) remove() {
	if o.prev != nil {
		o.prev.setNext(o.next)
	} else {
		o.list.head = o.next
	}
	if o.next != nil {
		o.next.setPrev(o.prev)
	} else {
		o.list.tail = o.prev
	}
	o.prev = nil
	o.next = nil
}

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
	getBasicBlock() *BasicBlock
}

// Shared fields of all occurrences
type BaseOccur struct {
	def     IOccur      // reference to the representative occurrence
	version int         // redundancy class number
	bb      *BasicBlock // the basic block this occurrence lies in
}

func (o *BaseOccur) getVersion() int { return o.version }

func (o *BaseOccur) setVersion(ver int) { o.version = ver }

func (o *BaseOccur) getDef() IOccur { return o.def }

func (o *BaseOccur) setDef(occur IOccur) { o.def = occur }

func (o *BaseOccur) isIdentical(o2 IOccur) bool { return o.version == o2.getVersion() }

func (o *BaseOccur) getBasicBlock() *BasicBlock { return o.bb }

// Real occurrence, computation in the original program
// Both evaluation and occurrence of an expression
type RealOccur struct {
	BaseEval
	BaseOccur
	instr  IInstr // if nil, this occurrence is an inserted one (not in original program)
	reload bool   // whether its value should be reloaded from temporary
	save   bool   // whether its value should be saved to temporary
}

func newRealOccur(list *BlockEvalList, instr IInstr) *RealOccur {
	return &RealOccur{
		BaseEval: BaseEval{
			list: list,
		},
		BaseOccur: BaseOccur{
			version: 0,
			bb:      instr.GetBasicBlock(),
		},
		instr: instr,
		save:  false,
	}
}

func (o *RealOccur) getOpdVer() [2]int {
	switch o.instr.(type) {
	case *Unary:
		unary := o.instr.(*Unary)
		return [2]int{getValVer(unary.Operand)}
	case *Binary:
		binary := o.instr.(*Binary)
		return [2]int{getValVer(binary.Left), getValVer(binary.Right)}
	default:
		return [2]int{}
	}
}

func getValVer(val IValue) int {
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

// Inserted occurrence during finalize step of algorithm
type InsertedOccur struct {
	BaseEval
	BaseOccur
}

// Assignment to an operand of expression
// Just an evaluation of expression, not an occurrence
type OpdAssign struct {
	BaseEval
	index int    // which operand is assigned
	instr IInstr // related instruction in CFG
}

func newOpdAssign(list *BlockEvalList, index int, instr IInstr) *OpdAssign {
	return &OpdAssign{
		BaseEval: BaseEval{
			list: list,
		},
		index: index,
		instr: instr,
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
	processed  bool // whether this operand is processed during FRG minimization
}

func newBigPhiOpd() *BigPhiOpd {
	return &BigPhiOpd{
		BaseOccur: BaseOccur{
			version: 0,   // negative value indicates unavailability
			bb:      nil, // assigned when Phi occurrence is inserted
		},
		hasRealUse: false, // keep false until we find a real use
		processed:  false,
	}
}

func (o *BigPhiOpd) isAvailable() bool { return o.version < 0 }

// Redundancy factoring operator
// Both occurrence and evaluation of expression
type BigPhi struct {
	BaseEval
	BaseOccur
	bbToOpd     map[*BasicBlock]*BigPhiOpd
	downSafe    bool // evaluated before program exit or altered by operand redefinition
	canBeAvail  bool // expression value can be safely be made available
	later       bool // whether expression value can be later provided
	willBeAvail bool // the exact point where expression value will be available
	extraneous  bool // whether this Phi occurrence should be removed
}

func newBigPhi(list *BlockEvalList, bb *BasicBlock,
	bbToOpd map[*BasicBlock]*BigPhiOpd) *BigPhi {
	for bb, opd := range bbToOpd { // assign basic block to operand
		// Phi operands are considered as occurring at their corresponding predecessors
		opd.bb = bb
	}
	return &BigPhi{
		BaseEval: BaseEval{
			list: list,
		},
		BaseOccur: BaseOccur{
			version: 0,
			bb:      bb,
		},
		bbToOpd:    bbToOpd,
		downSafe:   true, // keep true until propagated to be false
		canBeAvail: true,
	}
}

type BigPhiSet map[*BigPhi]bool

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

func (b *BlockEvalList) popFront() {
	if b.head == nil {
		return
	}
	head := b.head
	next := b.head.getNext()
	head.setPrev(nil)
	head.setNext(nil)
	if next != nil {
		next.setPrev(nil)
	} else {
		b.tail = nil
	}
	b.head = next
}

type EvalListTable map[*BasicBlock]*BlockEvalList

type BlockSet map[*BasicBlock]bool

func copyBlockSet(set BlockSet) BlockSet {
	cp := make(map[*BasicBlock]bool)
	for b := range set {
		cp[b] = true
	}
	return cp
}

// Factored Redundancy Graph
type FRG struct {
	table     EvalListTable // map basic block to its evaluation list
	bigPhi    BigPhiSet     // set of all Phi occurrences
	realOccur []*RealOccur  // list of all real occurrences
	nClass    int           // number of redundancy class
}

// Compute iterated dominance frontier set (DF+)
func computeDFPlus(fun *Func) map[*BasicBlock]BlockSet {
	dfTable := computeDF(fun)
	dfPlus := make(map[*BasicBlock]BlockSet)
	for bb, df := range dfTable {
		dfp := copyBlockSet(df)
		for {
			prevLen := len(dfp)
			for f := range dfp { // traverse all existing elements
				for d := range dfTable[f] { // no occurrence before
					dfp[d] = true
				}
			}
			curLen := len(dfp)
			if prevLen == curLen { // fixed point reached
				break
			}
		}
		dfPlus[bb] = dfp
	}
	return dfPlus
}

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
	o.dfPlus = computeDFPlus(fun)
	for len(workList) > 0 {
		// Pick one expression and build FRG for that
		expr := new(LexIdentExpr)
		*expr = removeOneExpr()
		frg := o.buildFRG(expr)
		// Perform backward and forward data flow propagation
		o.downSafety(fun, frg.bigPhi)
		o.willBeAvail(frg.bigPhi)
		// Pinpoint locations for computations to be inserted
		o.finalize(frg)
		//o.printEval(expr, frg.table)
		// Transform code to form optimized program
		o.codeMotion(frg, expr)
	}

	propagateCopy(fun)
	eliminateDeadCode(fun)
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

func newOpdStack() *OpdStack {
	return &OpdStack{
		stack: []int{0},
	}
}

func (s *OpdStack) push(v int) { s.stack = append(s.stack, v) }

func (s *OpdStack) pop() { s.stack = s.stack[:len(s.stack)-1] }

func (s *OpdStack) top() int { return s.stack[len(s.stack)-1] }

func (o *PREOpt) buildFRG(expr *LexIdentExpr) *FRG {
	// Build evaluation
	table := make(EvalListTable)           // table of evaluation list
	bigPhi := make(BigPhiSet)              // set of all Phi occurrences
	realOccur := make([]*RealOccur, 0)     // list of all real occurrences
	occurred := make(map[*BasicBlock]bool) // set of blocks where there are real occurrences
	o.fun.Enter.AcceptAsVert(func(block *BasicBlock) {
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
				evalList.pushBack(newOpdAssign(evalList, expr.opdIndex(sym.Name), instr))
				continue // cannot also be an real occurrence
			}
			// Try to build as real occurrence
			instrExpr := newLexIdentExpr(instr)
			if instrExpr == nil {
				continue // this type of instruction is not in consideration
			}
			if *instrExpr != *expr {
				continue // not the expression we build FRG for
			}
			occur := newRealOccur(evalList, instr)
			evalList.pushBack(occur)
			realOccur = append(realOccur, occur)
			occurred[block] = true // add to real occurrence set
		}
	}, DepthFirst)

	// Insert Phi functions
	// Insert in iterated dominance frontiers of expression
	inserted := make(map[*BasicBlock]bool)
	insertPhi := func(block *BasicBlock) {
		if inserted[block] {
			return
		}
		bbToOpd := make(map[*BasicBlock]*BigPhiOpd)
		for pred := range block.Pred {
			bbToOpd[pred] = newBigPhiOpd()
		}
		f := newBigPhi(table[block], block, bbToOpd)
		table[block].pushFront(f)
		bigPhi[f] = true
		inserted[block] = true
	}
	for bb := range occurred {
		for dfp := range o.dfPlus[bb] {
			insertPhi(dfp)
		}
	}

	// Insert in blocks where there is a phi instruction of operands of the expression
	o.fun.Enter.AcceptAsVert(func(block *BasicBlock) {
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
	opdStacks := [2]*OpdStack{newOpdStack(), newOpdStack()}

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
		exprPush := 0 // record push times to restore stack after visiting children
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
				occur.def = occur

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
				if !exprStack.empty() && occur.getOpdVer() == exprStack.top().opdVer {
					top := exprStack.top()
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
						opdVer: occur.getOpdVer(),
						occur:  occur,
						rep:    occur,
					})
					occur.version = exprStack.latest
					occur.def = exprStack.top().rep
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
					opd.def = nil    // definition not found
				}
			}
		}

		// Visit all children in dominance tree
		for child := range block.Children {
			visit(child)
		}

		// Clear down safety flag if reaching exit blocks
		if o.fun.Exit[block] {
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
	visit(o.fun.Enter)

	return &FRG{
		table:     table,
		bigPhi:    bigPhi,
		realOccur: realOccur,
		nClass:    exprStack.latest,
	}
}

func (o *PREOpt) downSafety(fun *Func, set BigPhiSet) {
	for f := range set {
		if !f.downSafe {
			for _, opd := range f.bbToOpd {
				o.resetDownSafety(opd)
			}
		}
	}
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

func (o *PREOpt) willBeAvail(set BigPhiSet) {
	// Compute if expression can be available
	for f := range set {
		if !f.downSafe && f.canBeAvail {
			bottom := false
			for _, opd := range f.bbToOpd {
				if opd.def == nil {
					bottom = true
					break
				}
			}
			if bottom { // has one operand that is bottom
				o.resetCanBeAvail(f, set)
			}
		}
	}

	// Compute if expression can be inserted later
	for f := range set {
		f.later = f.canBeAvail
	}
	for f := range set {
		if !f.later {
			continue
		}
		for _, opd := range f.bbToOpd {
			if opd.def != nil && opd.hasRealUse {
				o.resetLater(f, set)
				break
			}
		}
	}

	// Compute if expression will be available
	for f := range set {
		f.willBeAvail = f.canBeAvail && !f.later // determine final insertion point
	}
}

func (o *PREOpt) resetCanBeAvail(g *BigPhi, set BigPhiSet) {
	g.canBeAvail = false
	for f := range set {
		for _, opd := range f.bbToOpd {
			if opd.def != g || opd.hasRealUse {
				continue
			}
			if !f.downSafe && f.canBeAvail {
				o.resetCanBeAvail(f, set)
				break
			}
		}
	}
}

func (o *PREOpt) resetLater(g *BigPhi, set BigPhiSet) {
	g.later = false
	for f := range set {
		for _, opd := range f.bbToOpd {
			if opd.def != g {
				continue
			}
			if f.later {
				o.resetLater(f, set)
			} else {
				break
			}
		}
	}
}

func (o *PREOpt) finalize(frg *FRG) {
	// Traverse FRG and insert occurrence if needed
	availDef := make([]IOccur, frg.nClass+1) // class number is 1-indexed
	o.fun.Enter.AcceptAsTreeNode(func(block *BasicBlock) {
		list := frg.table[block]
		for cur := list.head; cur != nil; cur = cur.getNext() {
			switch cur.(type) {
			case *BigPhi:
				occur := cur.(*BigPhi)
				if occur.willBeAvail {
					availDef[occur.version] = occur
				}

			case *RealOccur:
				occur := cur.(*RealOccur)
				ver := occur.version
				def := availDef[ver]
				if def == nil || !def.getBasicBlock().Dominates(block) {
					occur.reload = false
					availDef[ver] = occur
				} else {
					occur.reload = true
					occur.def = def
				}
			}
		}

		for succ := range block.Succ {
			head := frg.table[succ].head
			switch head.(type) {
			case *BigPhi:
				f := head.(*BigPhi)
				if !f.willBeAvail {
					continue
				}
				opd := f.bbToOpd[block]
				if opd.def == nil {
					frg.nClass++ // assign a new class to inserted expression
					inserted := &InsertedOccur{
						BaseEval: BaseEval{
							list: list,
						},
						BaseOccur: BaseOccur{
							version: frg.nClass,
							bb:      block,
						},
					}
					inserted.def = inserted
					list.pushBack(inserted)
					opd.def = inserted
					opd.version = inserted.version
				} else {
					opd.def = availDef[opd.version]
				}
			}
		}
	}, func(*BasicBlock) {})

	// Remove extraneous Phi occurrence (FRG minimization)
	for f := range frg.bigPhi {
		f.extraneous = f.willBeAvail
	}
	for _, occur := range frg.realOccur {
		if occur.reload {
			o.setSave(occur.def, frg)
		}
	}
	for f := range frg.bigPhi {
		for _, opd := range f.bbToOpd {
			opd.processed = false
		}
	}
	for f := range frg.bigPhi {
		if !f.willBeAvail {
			o.removeBigPhi(f, frg)
			continue
		}
		if !f.extraneous {
			continue
		}
		for _, opd := range f.bbToOpd {
			def := opd.def
			switch def.(type) {
			case *RealOccur, *InsertedOccur:
				o.setReplacement(f, def, frg)
			case *BigPhi:
				if !def.(*BigPhi).extraneous {
					o.setReplacement(f, def, frg)
				}
			}
		}
	}
}

func (o *PREOpt) setSave(occur IOccur, frg *FRG) {
	switch occur.(type) {
	case *RealOccur:
		occur.(*RealOccur).save = true
	case *BigPhi:
		for _, opd := range occur.(*BigPhi).bbToOpd {
			if !opd.processed {
				opd.processed = true
				o.setSave(opd.def, frg)
			}
		}
	}
	switch occur.(type) {
	case *RealOccur, *InsertedOccur:
		for f := range frg.bigPhi {
			if o.dfPlus[occur.getBasicBlock()][f.bb] && f.willBeAvail {
				f.extraneous = false
			}
		}
	}
}

func (o *PREOpt) setReplacement(g *BigPhi, replace IOccur, frg *FRG) {
	for f := range frg.bigPhi {
		if !f.willBeAvail {
			continue
		}
		for bb, opd := range f.bbToOpd {
			if opd.def != g || opd.processed {
				continue
			}
			opd.processed = true
			if f.extraneous {
				o.setReplacement(f, replace, frg)
			} else {
				f.bbToOpd[bb].def = replace
			}
		}
	}
	for _, occur := range frg.realOccur {
		if !occur.reload || occur.def != g {
			continue
		}
		occur.def = replace
	}
	o.removeBigPhi(g, frg)
}

func (o *PREOpt) removeBigPhi(g *BigPhi, frg *FRG) {
	for _, opd := range g.bbToOpd {
		switch opd.def.(type) {
		case *InsertedOccur: // remove inserted occurrence that define this operand
			opd.def.(*InsertedOccur).remove()
		}
	}
	frg.table[g.bb].popFront()
	delete(frg.bigPhi, g)
}

type SymbolStack struct {
	stack []*Symbol
}

func newSymbolStack() *SymbolStack {
	return &SymbolStack{
		stack: make([]*Symbol, 0),
	}
}

func (s *SymbolStack) push(sym *Symbol) { s.stack = append(s.stack, sym) }

func (s *SymbolStack) pop(num int) { s.stack = s.stack[:len(s.stack)-num] }

func (s *SymbolStack) top() *Symbol { return s.stack[len(s.stack)-1] }

func (o *PREOpt) codeMotion(frg *FRG, expr *LexIdentExpr) {
	// Scan basic blocks and get instruction related information
	var resType IType
	hasImm := expr.hasImm()
	var immVal *Immediate // expression with immediate share this value
	immIdx := expr.immIndex()
	o.fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			instr := iter.Get()
			thisExpr := newLexIdentExpr(instr)
			if thisExpr == nil {
				continue
			}
			// Get result type
			if resType == nil && *expr == *thisExpr {
				resType = (*instr.GetDef()).(*Variable).Symbol.Type
			}
			// Extract immediate value if necessary
			if hasImm && immVal == nil {
				if *expr != *thisExpr { // not the expression considered
					continue
				}
				immVal = (*instr.GetOpd()[immIdx]).(*Immediate)
			}
		}
	}, DepthFirst)

	// Initialize temporary list and symbol stacks
	tmpList := make([]*Symbol, frg.nClass+1) // temporary for each redundancy class
	symStack := [2]*SymbolStack{newSymbolStack(), newSymbolStack()}
	for _, sym := range o.fun.Scope.Params {
		if expr.hasOpd(sym.Name) {
			symStack[expr.opdIndex(sym.Name)].push(sym)
		}
	}

	// Insert incomplete phi instructions
	phiInserted := make([]bool, frg.nClass+1) // whether phi for temporaries are inserted
	o.fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		list := frg.table[block]
		switch list.head.(type) {
		case *BigPhi:
			bigPhi := list.head.(*BigPhi)
			ver := bigPhi.version
			if !phiInserted[ver] { // insert a phi instruction for temporary
				tmp := o.newTemp(resType)
				tmpList[ver] = tmp
				dummy := &Symbol{ // dummy symbol as placeholder
					Name:  "_dummy",
					Ver:   0,
					Type:  resType,
					Scope: nil,
					Param: false,
				}
				phiOpd := make([]PhiOpd, 0)
				for pred := range block.Pred {
					phiOpd = append(phiOpd, PhiOpd{
						pred: pred,
						val:  NewVariable(dummy),
					})
				}
				block.PushFront(NewPhi(phiOpd, NewVariable(tmp)))
				phiInserted[ver] = true
			}
		}
	}, DepthFirst)

	// Visit basic blocks and perform code motion
	var visit func(*BasicBlock)
	visit = func(block *BasicBlock) {
		// Visit CFG and update symbol stack
		nPush := [2]int{0, 0}
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			instr := iter.Get()
			def := instr.GetDef()
			if def == nil {
				continue
			}
			sym := (*def).(*Variable).Symbol
			if expr.hasOpd(sym.Name) {
				idx := expr.opdIndex(sym.Name)
				symStack[idx].push(sym)
				nPush[idx]++
			}
		}

		// Visit evaluation list of current block
		list := frg.table[block]
		for cur := list.head; cur != nil; cur = cur.getNext() {
			switch cur.(type) {
			case *RealOccur:
				occur := cur.(*RealOccur)
				instr := occur.instr
				def := instr.GetDef()
				iter := NewIterFromInstr(instr)
				if occur.save { // generate a save of the result to a new temporary
					sym := (*def).(*Variable).Symbol
					tmp := o.newTemp(resType)
					tmpList[occur.version] = tmp
					iter.InsertAfter(NewMove(NewVariable(sym), NewVariable(tmp)))
				}
				if occur.reload { // replace computation by a use of temporary
					iter.Replace(NewMove(NewVariable(tmpList[occur.version]), *def))
				}

			case *InsertedOccur: // generate a save of the result to a new temporary
				inserted := cur.(*InsertedOccur)
				tmp := o.newTemp(resType)
				tmpList[inserted.version] = tmp
				if expr.isUnary() {
					block.PushBack(NewUnary(unaryStrToOp[expr.op],
						NewVariable(symStack[0].top()), NewVariable(tmp)))
				} else {
					opd := [2]IValue{nil, nil}
					for i := range expr.opd {
						if i == immIdx {
							opd[i] = immVal
						} else {
							opd[i] = NewVariable(symStack[i].top())
						}
					}
					block.PushBack(NewBinary(binaryStrToOp[expr.op], opd[0], opd[1],
						NewVariable(tmp)))
				}
			} // end evaluation type switch
		} // end evaluation iteration

		// Update phi operand in successor blocks
		for succ := range block.Succ {
			head := frg.table[succ].head
			switch head.(type) {
			case *BigPhi:
				// Replace value in phi for temporary according to version in Phi occurrence
				// operand
				bigPhi := head.(*BigPhi)
				phi := succ.Head.(*Phi)
				bigPhiOpd := bigPhi.bbToOpd[block]
				*phi.BBToVal[block] = NewVariable(tmpList[bigPhiOpd.version])
			} // end evaluation head type switch
		} // end successor iteration

		// Visit children in dominance tree
		for child := range block.Children {
			visit(child)
		}

		// Restore variable stack
		for i, n := range nPush {
			symStack[i].pop(n)
		}
	}
	visit(o.fun.Enter)
}

func (o *PREOpt) newTemp(tp IType) *Symbol {
	name := fmt.Sprintf("_t%d", o.fun.nTmp)
	o.fun.nTmp++
	sym := o.fun.Scope.AddTemp(name, tp)
	return sym
}

func (o *PREOpt) printEval(expr *LexIdentExpr, table EvalListTable) {
	fmt.Println(expr)
	o.fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		fmt.Println(block.Name)
		for cur := table[block].head; cur != nil; cur = cur.getNext() {
			switch cur.(type) {
			case *BigPhi:
				bigPhi := cur.(*BigPhi)
				fmt.Printf("\tBigPhi %d %s %s %s %s %s", bigPhi.version,
					strconv.FormatBool(bigPhi.downSafe),
					strconv.FormatBool(bigPhi.canBeAvail),
					strconv.FormatBool(bigPhi.later),
					strconv.FormatBool(bigPhi.willBeAvail),
					strconv.FormatBool(bigPhi.extraneous))
				for bb, opd := range bigPhi.bbToOpd {
					fmt.Printf(" [%s: %d %s]", bb.Name, opd.version,
						strconv.FormatBool(opd.hasRealUse))
				}
				fmt.Println()
			case *RealOccur:
				occur := cur.(*RealOccur)
				fmt.Printf("\tRealOccur %d %s %s\n", occur.version,
					strconv.FormatBool(occur.reload),
					strconv.FormatBool(occur.save))
			case *InsertedOccur:
				inserted := cur.(*InsertedOccur)
				fmt.Printf("\tInsertedOccur %d\n", inserted.version)
			case *OpdAssign:
				fmt.Printf("\tOpdAssign %s\n", expr.opd[cur.(*OpdAssign).index])
			}
		}
	}, ReversePostOrder)
	fmt.Println()
}
