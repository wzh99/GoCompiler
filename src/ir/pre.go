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
		expr := &LexIdentExpr{
			op:  binaryOpStr[binary.Op],
			opd: [2]string{opdToStr(binary.Left), opdToStr(binary.Right)},
		}
		// enforce an order for commutative binary operators
		if commutative[binary.Op] && expr.opd[0] > expr.opd[1] {
			expr.opd = [2]string{expr.opd[1], expr.opd[0]}
		}
		return expr
	default:
		return nil // other instructions not considered
	}
}

type IOccur interface {
	getPrev() IOccur
	setPrev(occur IOccur)
	getNext() IOccur
	setNext(occur IOccur)
	getVersion() int
	setVersion(ver int)
	isIdentical(o2 IOccur) bool
}

type BaseOccur struct {
	prev, next IOccur // as linked list node
	version    int
}

func (o *BaseOccur) getPrev() IOccur { return o.prev }

func (o *BaseOccur) setPrev(occur IOccur) { o.prev = occur }

func (o *BaseOccur) getNext() IOccur { return o.next }

func (o *BaseOccur) setNext(occur IOccur) { o.next = occur }

func (o *BaseOccur) getVersion() int { return o.version }

func (o *BaseOccur) setVersion(ver int) { o.version = ver }

func (o *BaseOccur) isIdentical(o2 IOccur) bool { return o.version == o2.getVersion() }

type RealOccur struct {
	BaseOccur
	instr IInstr
}

func newRealOccur(instr IInstr) *RealOccur {
	return &RealOccur{
		BaseOccur: BaseOccur{
			prev:    nil,
			next:    nil,
			version: 0,
		},
		instr: instr,
	}
}

type BigPhiOpd struct {
	BaseOccur
}

func newBigPhiOpd() *BigPhiOpd {
	return &BigPhiOpd{
		BaseOccur: BaseOccur{
			prev:    nil,
			next:    nil,
			version: 0,
		},
	}
}

// Redundancy factoring operator
type BigPhi struct {
	BaseOccur
	bbToOccur map[*BasicBlock]IOccur
}

func newBigPhi(bbToOccur map[*BasicBlock]IOccur) *BigPhi {
	return &BigPhi{
		BaseOccur: BaseOccur{
			prev:    nil,
			next:    nil,
			version: 0,
		},
		bbToOccur: bbToOccur,
	}
}

type BlockOccur struct {
	head, tail IOccur
}

func (b *BlockOccur) pushFront(occur IOccur) {
	if b.head == nil {
		b.head, b.tail = occur, occur
		return
	}
	occur.setNext(b.head)
	b.head.setPrev(occur)
	b.head = occur
}

func (b *BlockOccur) pushBack(occur IOccur) {
	if b.tail == nil {
		b.head, b.tail = occur, occur
		return
	}
	occur.setPrev(b.tail)
	b.tail.setNext(occur)
	b.tail = occur
}

type OccurTable map[*BasicBlock]*BlockOccur

// Worklist driven PRE, see Section 5.1 of paper
func (o *PREOpt) Optimize(fun *Func) {
	// Initialize worklist
	o.fun = fun
	workList := make(map[LexIdentExpr]OccurTable)
	removeOnePair := func() (LexIdentExpr, OccurTable) {
		var expr LexIdentExpr
		var occurTable map[*BasicBlock]*BlockOccur
		for e, oc := range workList {
			expr, occurTable = e, oc
		}
		delete(workList, expr)
		return expr, occurTable
	}

	// Collect occurrences of expressions
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			expr := newLexIdentExpr(iter.Get())
			if expr == nil {
				continue
			}
			if workList[*expr] == nil {
				workList[*expr] = make(OccurTable)
			}
			occurTable := workList[*expr]
			if occurTable[block] == nil {
				occurTable[block] = &BlockOccur{}
			}
			occurTable[block].pushBack(newRealOccur(iter.Get()))
		}
	}, DepthFirst)

	// Eliminate redundancies iteratively
	o.DF = computeDF(fun)
	for len(workList) > 0 {
		// Pick one expression and build FRG for that
		expr := new(LexIdentExpr)
		var table OccurTable
		*expr, table = removeOnePair()
		o.buildFRG(fun, expr, table)

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

func (o *PREOpt) buildFRG(fun *Func, expr *LexIdentExpr, table OccurTable) {
	// Build occurrence set of expression
	occurred := make(map[*BasicBlock]bool)
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		if table[block] == nil { // fill in missing entries with empty records
			table[block] = &BlockOccur{}
		} else { // add to definition set
			occurred[block] = true
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
		bbToOpd := make(map[*BasicBlock]IOccur)
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
					if sym.Name == expr.opd[0] || sym.Name == expr.opd[1] {
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
				fmt.Println("\tBigPhi", cur.getVersion())
			case *RealOccur:
				fmt.Println("\tRealOccur", cur.getVersion())
			}
		}
	}, DepthFirst)
	fmt.Println()
}
