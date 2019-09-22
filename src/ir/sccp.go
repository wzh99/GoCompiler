package ir

// Sparse Conditional Constant Propagation
// See Figure 10.9 of EAC and Figure 12.31 of The Whale Book.
type SCCPOpt struct {
	ssaGraph  *SSAGraph
	cfgWL     map[CFGEdge]bool
	ssaWL     map[SSAEdge]bool
	value     map[*SSAVert]LatValue
	edgeExec  map[CFGEdge]bool
	instrExec map[IInstr]bool
}

func NewSCCPOpt() *SCCPOpt {
	return &SCCPOpt{}
}

// In SCCP, it's assumed that one basic block only contain one assignment along with
// several phi instructions.
type CFGEdge struct {
	from, to IInstr
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

func (o *SCCPOpt) Optimize(fun *Func) {
	// Initialize data structures
	if fun.Enter.Head == nil { // empty function, no need to optimize
		return
	}
	o.ssaGraph = NewSSAGraph(fun)
	o.cfgWL = map[CFGEdge]bool{CFGEdge{from: nil, to: fun.Enter.Head}: true}
	o.ssaWL = make(map[SSAEdge]bool)
	o.value = make(map[*SSAVert]LatValue) // default to TOP
	o.edgeExec = make(map[CFGEdge]bool)
	o.instrExec = make(map[IInstr]bool)
	for vert := range o.ssaGraph.vertSet {
		if vert.cnst != nil {
			o.value[vert] = CONST // mark constant vertices in value table
		}
	}

	// Propagate constants iteratively with help of CFG and SSA work lists
	for len(o.cfgWL) > 0 || len(o.ssaWL) > 0 {
		if len(o.cfgWL) > 0 {
			// Possible visit phi instruction depending on whether this edge has been
			// visited
			edge := o.removeOneCFGEdge()
			if o.edgeExec[edge] { // don't execute this edge
				goto AccessSSAList
			}
			instr := edge.to
			o.edgeExec[edge] = true
			if o.instrExec[instr] {
				goto AccessSSAList
			}

			// Visit all phi instructions and one other instruction in the block
			firstNonPhi := o.evalAllPhis(instr)
			if firstNonPhi == nil {
				goto AccessSSAList
			}
			instr = firstNonPhi
			o.instrExec[instr] = true
			switch instr.(type) {
			case *Jump:
				jump := instr.(*Jump)
				o.cfgWL[CFGEdge{from: instr, to: jump.Target.Head}] = true
			case *Branch:
				o.evalBranch(instr.(*Branch))
			case *Return:
				goto AccessSSAList
			default:
				o.evalAssign(instr)
				if next := instr.GetNext(); next != nil {
					o.cfgWL[CFGEdge{from: instr, to: next}] = true
				}
			} // end instruction iteration
		}

	AccessSSAList:
		if len(o.ssaWL) > 0 {
			// Skip instruction that cannot be proved to be reachable
			edge := o.removeOneSSAEdge()
			vert := edge.use
			instr := vert.instr
			if !o.instrExec[instr] {
				continue
			}

			// Evaluate reachable instruction according to its type
			switch instr.(type) {
			case *Phi:
				o.evalPhi(instr.(*Phi))
			case *Branch:
				o.evalBranch(instr.(*Branch))
			default:
				o.evalAssign(instr)
			}
		}
	}

	// Print result
	/*for vert, val := range o.value {
		if len(vert.symbols) == 0 {
			continue
		}
		fmt.Printf("%s: %d, %s\n", pickOneSymbol(vert.symbols).ToString(), val,
			vert.cnst)
	}*/

	// Transform original instructions, if possible.
	blockWL := map[*BasicBlock]bool{fun.Enter: true}
	visited := make(map[*BasicBlock]bool)
	for len(blockWL) > 0 {
		block := removeOneBlock(blockWL)
		if visited[block] {
			continue
		}
		visited[block] = true

		// Replace use with constants
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			instr := iter.Get()
			for _, opd := range instr.GetOpd() {
				switch (*opd).(type) {
				case *Variable:
					sym := (*opd).(*Variable).Symbol
					vert := o.ssaGraph.symToVert[sym]
					if o.value[vert] != CONST {
						continue
					}
					cnst := vert.cnst
					switch cnst.(type) {
					case bool:
						*opd = NewI1Const(cnst.(bool))
					case int:
						*opd = NewI64Const(cnst.(int))
					case float64:
						*opd = NewF64Const(cnst.(float64))
					}
				}
			}
		}

		// Remove unreachable branch
		iter := NewIterFromInstr(block.Tail)
		switch iter.Get().(type) {
		case *Branch:
			branch := iter.Get().(*Branch)
			switch branch.Cond.(type) {
			case *Constant:
				cnst := branch.Cond.(*Constant)
				target, removed := branch.True, branch.False
				if !cnst.Value.(bool) {
					target, removed = branch.False, branch.True
				}
				iter.Remove()
				block.DisconnectTo(removed)
				block.JumpTo(target)
			}
		}

		// Add successors to work list
		for succ := range block.Succ {
			blockWL[succ] = true
		}
	}

	eliminateDeadCode(fun)
	computeDominators(fun) // control flow may be changed
}

func (o *SCCPOpt) removeOneCFGEdge() CFGEdge {
	var edge CFGEdge
	for e := range o.cfgWL {
		edge = e
		break
	}
	delete(o.cfgWL, edge)
	return edge
}

func (o *SCCPOpt) removeOneSSAEdge() SSAEdge {
	var edge SSAEdge
	for e := range o.ssaWL {
		edge = e
		break
	}
	delete(o.ssaWL, edge)
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
	// Computed values are stored in vertices, and they will later be reflected on instructions.
	// Since there is injective mapping from instruction type to vertex type, using instruction
	// type in switch clause is much safer.
	prevVal, prevConst := o.value[vert], vert.cnst
	switch instr.(type) {
	case *Load, *Malloc, *GetPtr, *PtrOffset:
		// values defined by these instructions are considered variables
		o.value[vert] = BOTTOM
	case *Unary:
		unary := instr.(*Unary)
		o.value[vert], vert.cnst = o.evalUnary(unary.Op, vert.opd[0])
	case *Binary:
		binary := instr.(*Binary)
		o.value[vert], vert.cnst = o.evalBinary(binary.Op, vert.opd[0], vert.opd[1])
	}

	// Add uses of value to work list
	if prevVal == o.value[vert] && constEq(prevConst, vert.cnst) { // value not changed
		return
	}
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
	cnst := opd.cnst
	tp := opd.tp.GetTypeEnum()
	switch op {
	case NOT:
		result = !cnst.(bool)
	case NEG:
		switch tp {
		case I64:
			result = -cnst.(int)
		case F64:
			result = -cnst.(float64)
		}
	}
	return
}

func (o *SCCPOpt) evalBinary(op BinaryOp, left, right *SSAVert) (lat LatValue,
	result interface{}) {
	lat, result = BOTTOM, nil // default value

	// Consider six cases of value combination
	tp := left.tp.GetTypeEnum()
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
		// 0 * x = 0, true || x = true, false && x = false
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
		// Only those operators which support short circuit evaluation return a constant
		switch op {
		case MUL:
			switch tp {
			case I64:
				if cVert.cnst.(int) == 0 {
					return CONST, 0
				} else {
					return
				}
			case F64:
				if cVert.cnst.(float64) == 0. {
					return CONST, 0.
				} else {
					return
				}
			}
		case AND:
			switch tp {
			case I1:
				if cVert.cnst.(bool) == false {
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
				if cVert.cnst.(bool) == true {
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
		l, r := left.cnst.(bool), right.cnst.(bool)
		switch op {
		case AND:
			result = l && r
		case OR:
			result = l || r
		}
	case I64:
		l, r := left.cnst.(int), right.cnst.(int)
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
		l, r := left.cnst.(float64), right.cnst.(float64)
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

func (o *SCCPOpt) evalBranch(branch *Branch) {
	// Try to extract constant from condition vertex
	var cnst interface{}
	cond := branch.Cond
	switch cond.(type) {
	case *Constant:
		cnst = cond.(*Constant).Value
	case *Variable:
		sym := cond.(*Variable).Symbol
		vert := o.ssaGraph.symToVert[sym]
		val := o.value[vert]
		switch val {
		case CONST:
			cnst = vert.cnst
		}
	}

	trueEdge := CFGEdge{from: branch, to: branch.True.Head}
	falseEdge := CFGEdge{from: branch, to: branch.False.Head}
	if cnst != nil {
		// Only choose the corresponding block if condition is constant
		if cnst.(bool) {
			o.cfgWL[trueEdge] = true
		} else {
			o.cfgWL[falseEdge] = true
		}
	} else {
		// Add both branches to work list
		o.cfgWL[trueEdge] = true
		o.cfgWL[falseEdge] = true
	}
}

func (o *SCCPOpt) evalPhi(phi *Phi) {
	result := phi.Result.(*Variable)
	rVert := o.ssaGraph.symToVert[result.Symbol]
	if o.value[rVert] == BOTTOM {
		return // cannot propagate
	}
	lat, cnst := TOP, interface{}(nil)
	for _, v := range rVert.opd {
		lat, cnst = o.meet(lat, cnst, o.value[v], v.cnst)
	}
	if lat == o.value[rVert] {
		if lat == TOP || lat == BOTTOM {
			return // TOP and BOTTOM has only one case
		} else if constEq(cnst, rVert.cnst) {
			return // has equal constant value
		}
	}
	o.value[rVert], rVert.cnst = lat, cnst
	for u := range rVert.use {
		o.ssaWL[SSAEdge{def: rVert, use: u}] = true
	}
}

// Evaluate all phi instruction in a basic block, and return the first non-phi instruction
func (o *SCCPOpt) evalAllPhis(instr IInstr) IInstr {
	iter := NewIterFromInstr(instr)
	for { // visit result of all phi instructions
		phi, ok := iter.Get().(*Phi)
		if !ok {
			break
		}
		o.instrExec[phi] = true
		o.evalPhi(phi)
		iter.MoveNext()
	}
	return iter.Get()
}

func (o *SCCPOpt) meet(v1 LatValue, i1 interface{}, v2 LatValue, i2 interface{}) (
	lat LatValue, cnst interface{}) {
	hasOne := func(v LatValue) bool {
		return v == v1 || v == v2
	}
	if hasOne(TOP) {
		otherVal, otherConst := v2, i2
		if otherVal == TOP { // try to find lower one
			otherVal, otherConst = v1, i1
		}
		return otherVal, otherConst
	}
	if hasOne(BOTTOM) {
		return BOTTOM, nil
	}
	// Both constant
	if constEq(i1, i2) {
		return CONST, i1
	} else {
		return BOTTOM, nil
	}
}
