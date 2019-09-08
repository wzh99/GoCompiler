package ir

// Sparse Conditional Constant Propagation
// See Figure 10.9 of Engineering a Compiler, Second Edition.
type SCCPOpt struct {
	opt      *SSAOpt
	ssaGraph *SSAGraph
	cfgWL    map[CFGEdge]bool
	ssaWL    map[SSAEdge]bool
	value    map[*SSAVert]LatValue
}

// In SCCP, it's assumed that one basic block only contain one assignment, along with
// some possible phi instructions. However, a basic block containing several assignment
// does not interfere with the algorithm. Therefore, it's fairly enough to only store
// edges that connect actual basic blocks, and ignore those between linearly executed
// instructions.
type CFGEdge struct {
	from, to *BasicBlock
}

type SSAEdge struct {
	// data flow: def (where a value is defined) -> use (where it's used)
	def, use *SSAVert
}

type LatValue int

const (
	TOP    LatValue = iota // uninitialized
	CONST                  // a known constant, stored in SSA vertex
	BOTTOM                 // variable (assigned more than once during execution)
)

func (o *SCCPOpt) optimize(fun *Func) {
	// Initialize data structures
	o.ssaGraph = newSSAGraph(fun)
	o.cfgWL = map[CFGEdge]bool{CFGEdge{from: nil, to: fun.Enter}: true}
	o.ssaWL = make(map[SSAEdge]bool)
	o.value = make(map[*SSAVert]LatValue) // default to TOP
	edgeReached := make(map[CFGEdge]bool)
	blockReached := make(map[*BasicBlock]bool)

	// Propagate constants iteratively with help of CFG and SSA work lists
	for len(o.cfgWL) > 0 && len(o.ssaWL) > 0 {
		if len(o.cfgWL) > 0 {
			// Possible visit phi instruction depending on whether this edge has been visited
			edge := o.removeOneCFGEdge()
			if edgeReached[edge] { // don't execute this edge
				goto AccessSSAWorkList
			}
			block := edge.to
			edgeReached[edge] = true
			firstNonPhi := o.evalAllPhis(block)

			// Test whether this block has been visited before
			if blockReached[block] {
				goto AccessSSAWorkList
			}

			// Visit all non-phi instructions in the basic block
			// Here we allow multiple non-phi instructions, thus saving the compiler
			// from visiting every edge between linearly executed instructions.
			for iter := NewIterFromInstr(firstNonPhi); iter.Valid(); iter.Next() {
				instr := iter.Cur
				switch instr.(type) {
				case *Jump:
					jump := instr.(*Jump)
					// an actual block encountered, add it to work list
					o.cfgWL[CFGEdge{from: block, to: jump.Target}] = true
				case *Branch:
					o.evalBranch(instr.(*Branch))
				default:
					o.evalAssign(instr)
				}
			} // end instruction iteration
		}

	AccessSSAWorkList:
		if len(o.ssaWL) > 0 {
			// Skip instruction that cannot be proved to be reachable
			edge := o.removeOneSSAEdge()
			vert := edge.use
			instr := vert.instr
			block := instr.GetBasicBlock()
			if !blockReached[block] {
				// if a basic block is unreachable, then its every instruction cannot be
				// reachable.
				continue
			}

			// Evaluate reachable instruction according to its type
			switch instr.(type) {
			case *Phi:
				o.evalPhi(instr.(*Phi))
			case *Jump:
				jump := instr.(*Jump)
				o.cfgWL[CFGEdge{from: block, to: jump.Target}] = true
			case *Branch:
				o.evalBranch(instr.(*Branch))
			default:
				o.evalAssign(instr)
			}
		}
	}
}

func (o *SCCPOpt) removeOneCFGEdge() CFGEdge {
	var edge CFGEdge
	for e := range o.cfgWL {
		edge = e
		break
	}
	o.cfgWL[edge] = false
	return edge
}

func (o *SCCPOpt) removeOneSSAEdge() SSAEdge {
	var edge SSAEdge
	for e := range o.ssaWL {
		edge = e
		break
	}
	o.ssaWL[edge] = false
	return edge
}

func (o *SCCPOpt) evalAssign(instr IInstr) {
	// Decide whether this instruction should be evaluated
	def := instr.GetDef()
	if def == nil {
		return // no new value is defined
	}
	sym := (*def).(*Variable).Symbol
	vert := o.ssaGraph.symToVert[sym]
	if o.value[vert] == BOTTOM {
		return // a variable, no need to evaluate
	}

	// Propagate constant according to instruction type
	// Computed values are stored in vertices, and they will later be reflected on
	// instructions.
	// Since there is injective mapping from instruction type to vertex type, using
	// instruction type in switch clause is much safer.
	var newVal LatValue
	switch instr.(type) {
	case *Load, *Malloc, *GetPtr, *PtrOffset:
		// values defined by these instructions are considered variables
		newVal = BOTTOM
	case *Clear:
		enum := sym.Type.GetTypeEnum()
		switch enum {
		case I1, I64, F64:
			newVal, vert.imm = CONST, o.getZeroValue(sym.Type.GetTypeEnum())
		default:
			newVal = BOTTOM
		}
	case *Unary:
		unary := instr.(*Unary)
		newVal, vert.imm = o.evalUnary(unary.Op, vert.opd[0])
	case *Binary:
		binary := instr.(*Binary)
		newVal, vert.imm = o.evalBinary(binary.Op, vert.opd[0], vert.opd[1])
	}

	// Add uses of value to work list
	if newVal == o.value[vert] { // value not changed
		return
	}
	o.value[vert] = newVal
	for u := range vert.use {
		o.ssaWL[SSAEdge{def: vert, use: u}] = true
	}
}

func (o *SCCPOpt) evalUnary(op UnaryOp, opd *SSAVert) (lat LatValue, result interface{}) {
	opdVal := o.value[opd]
	if opdVal == TOP || opdVal == BOTTOM { // operand is not sure to be constant
		return opdVal, nil
	}
	lat = CONST
	imm := opd.imm
	tp := o.opt.pickOneSymbol(opd.symbols).Type.GetTypeEnum()
	switch op {
	case NOT:
		result = !imm.(bool)
	case NEG:
		switch tp {
		case I64:
			result = -imm.(int)
		case F64:
			result = -imm.(float64)
		}
	}
	return
}

func (o *SCCPOpt) evalBinary(op BinaryOp, left, right *SSAVert) (lat LatValue,
	result interface{}) {
	lat, result = BOTTOM, nil // default value

	// Consider six cases of value combination
	tp := o.opt.pickOneSymbol(left.symbols).Type.GetTypeEnum()
	lVal, rVal := o.value[left], o.value[right]
	isComb := func(c1, c2 LatValue) bool {
		return (lVal == c1 && rVal == c2) || (lVal == c2 && rVal == c1)
	}
	if isComb(TOP, TOP) || isComb(TOP, CONST) {
		return TOP, nil
	}
	if isComb(BOTTOM, BOTTOM) {
		return
	}
	if isComb(TOP, BOTTOM) {
		// Only those operators which support short circuit evaluation returns TOP
		switch op {
		// 0 * x = 0, true || x = true, false AND x = false
		case MUL:
			return TOP, nil
		case AND, OR:
			if tp == I1 { // only works for boolean value
				return TOP, nil
			} else {
				return
			}
		default:
			return
		}
	}
	if isComb(CONST, BOTTOM) {
		var cVert *SSAVert
		if lVal == CONST {
			cVert = left
		} else {
			cVert = right
		}
		// Only those operators which support short circuit evaluation returns a constant
		switch op {
		case MUL:
			switch tp {
			case I64:
				if cVert.imm.(int) == 0 {
					return CONST, 0
				} else {
					return
				}
			case F64:
				if cVert.imm.(float64) == 0. {
					return CONST, 0.
				} else {
					return
				}
			}
		case AND:
			switch tp {
			case I1:
				if cVert.imm.(bool) == false {
					return CONST, false
				} else {
					return
				}
			default:
				return
			}
		case OR:
			switch tp {
			case I1:
				if cVert.imm.(bool) == true {
					return CONST, true
				} else {
					return
				}
			default:
				return
			}
		default:
			return
		}
	}
	// CONST, CONST
	lat = CONST
	switch tp {
	case I1:
		l, r := left.imm.(bool), right.imm.(bool)
		switch op {
		case AND:
			result = l && r
		case OR:
			result = l || r
		}
	case I64:
		l, r := left.imm.(int), right.imm.(int)
		switch op {
		case ADD:
			result = l + r
		case SUB:
			result = l - r
		case MUL:
			result = l * r
		case DIV:
			result = l / r
		case MOD:
			result = l % r
		case AND:
			result = l & r
		case OR:
			result = l | r
		case XOR:
			result = l ^ r
		case SHL:
			result = l << uint(r)
		case SHR:
			result = l >> uint(r)
		case EQ:
			result = l == r
		case NE:
			result = l != r
		case LT:
			result = l < r
		case LE:
			result = l <= r
		case GT:
			result = l > r
		case GE:
			result = l >= r
		}
	case F64:
		l, r := left.imm.(float64), right.imm.(float64)
		switch op {
		case ADD:
			result = l + r
		case SUB:
			result = l - r
		case MUL:
			result = l * r
		case DIV:
			result = l / r
		case EQ:
			result = l == r
		case NE:
			result = l != r
		case LT:
			result = l < r
		case LE:
			result = l <= r
		case GT:
			result = l > r
		case GE:
			result = l >= r
		}
	}
	return
}

func (o *SCCPOpt) getZeroValue(enum TypeEnum) interface{} {
	switch enum {
	case I1:
		return false
	case I64:
		return 0
	case F64:
		return 0.
	default:
		return nil
	}
}

func (o *SCCPOpt) evalBranch(branch *Branch) {}

func (o *SCCPOpt) evalPhi(phi *Phi) {}

// Evaluate all phi instruction in a basic block, and return the first non-phi instruction
func (o *SCCPOpt) evalAllPhis(block *BasicBlock) IInstr {
	return nil
}

func (o *SCCPOpt) evalOperands(phi *Phi) {}

func (o *SCCPOpt) evalResult(phi *Phi) {}
