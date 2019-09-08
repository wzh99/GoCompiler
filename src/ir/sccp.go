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
	CONST                  // a known constant, value stored in instruction and SSA vertex.
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

func (o *SCCPOpt) evalAssign(instr IInstr) {}

func (o *SCCPOpt) evalBranch(instr *Branch) {}

func (o *SCCPOpt) evalPhi(instr *Phi) {}

// Evaluate all phi instruction in a basic block, and return the first non-phi instruction
func (o *SCCPOpt) evalAllPhis(block *BasicBlock) IInstr {
	return nil
}

func (o *SCCPOpt) evalOperands(phi *Phi) {}

func (o *SCCPOpt) evalResult(phi *Phi) {}
