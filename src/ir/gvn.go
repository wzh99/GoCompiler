package ir

// Global Value Numbering
// Partition vertices in value graph so that each vertex in a set shares one value number.
// See Fig. 12.21 and 12.22 in The Whale Book.
type GVNOpt struct {
	graph  *SSAGraph
	valNum map[*SSAVert]int
}

func NewGVNOpt() *GVNOpt {
	return &GVNOpt{}
}

func (o *GVNOpt) Optimize(fun *Func) {
	// Build value graph out of SSA
	o.graph = NewSSAGraph(fun)

	// Initialize vertex partition and work list
	part := make([]map[*SSAVert]bool, 0) // partition result: array of sets
	o.valNum = make(map[*SSAVert]int)    // map vertices to value number
	workList := make(map[int]bool, 0)    // sets to be further partitioned in B

TraverseVertSet:
	for v := range o.graph.vertSet {
		// Create the first set
		if len(part) == 0 {
			part = append(part, map[*SSAVert]bool{v: true})
			o.valNum[v] = 0
			continue
		}
		// Test whether there is congruence
		for i := 0; i < len(part); i++ {
			if v.hasSameLabel(pickOneSSAVert(part[i])) { // may be congruent
				part[i][v] = true
				o.valNum[v] = i
				if len(v.opd) > 0 && len(part[i]) > 1 { // depends on operands
					workList[i] = true
				}
				continue TraverseVertSet
			}
		}
		// No congruence is found, add to new set
		n := len(part)
		o.valNum[v] = n
		part = append(part, map[*SSAVert]bool{v: true})
	}

	// Further partition the vertex set until a fixed point is reached
	for len(workList) > 0 {
		// Pick up one node set
		wi := o.pickOneIndex(workList)
		delete(workList, wi)
		set := part[wi]
		// Pick up one vertex and test it against others in the set
		v := pickOneSSAVert(set)
		newSet := make(map[*SSAVert]bool)
		for v2 := range set {
			if v == v2 {
				continue // one vertex must be congruent to itself
			}
			if !o.opdEq(v, v2) {
				delete(set, v2)
				newSet[v2] = true
			}
		}
		if len(newSet) > 0 { // another cut made in current set
			// Update the partition list and value number
			n := len(part)
			for v2 := range newSet {
				o.valNum[v2] = n
				for u := range v2.use { // add uses to work list
					// the value number of operands have changed, so the number of their
					// uses may also change
					workList[o.valNum[u]] = true
				}
			}
			part = append(part, newSet)
			part[wi] = set
			// Add original and new set to work list
			if len(set) > 1 {
				workList[wi] = true
			}
			if len(newSet) > 1 {
				workList[n] = true
			}
		}
	}

	// Map symbols to index
	symNum := make(map[*Symbol]int)
	for i, set := range part {
		for vert := range set {
			for sym := range vert.symbols {
				symNum[sym] = i
			}
		}
	}

	// Apply transformation to instructions
	repSym := make([]*Symbol, len(part))
	replaceOpd := func(opd *IValue) {
		switch (*opd).(type) {
		case *Variable:
			sym := (*opd).(*Variable).Symbol
			num := symNum[sym]
			rep := repSym[num]
			if rep != nil && rep != sym {
				*opd = NewVariable(rep)
			}
		}
	}
	var visit func(*BasicBlock)
	visit = func(block *BasicBlock) {
		setSym := make([]bool, len(part)) // whether a representative symbol is set
		for iter := NewIterFromBlock(block); iter.Valid(); {
			// Define new symbol or remove instruction
			instr := iter.Get()
			def := instr.GetDef()
			if def != nil {
				sym := (*def).(*Variable).Symbol
				num := symNum[sym]
				if repSym[num] == nil { // first definition of symbol in this class
					repSym[num] = sym
					setSym[num] = true
				} else { // other symbols in this class defined before
					iter.Remove() // eliminate this definition
					continue
				}
			}

			// Replace operands with representative symbols
			switch instr.(type) {
			case *Phi:
				iter.MoveNext()
				continue // skip all phi operands
			}
			for _, opd := range instr.GetOpd() {
				replaceOpd(opd)
			}
			iter.MoveNext()
		}

		// Replace phi operands in successors
		for succ := range block.Succ {
		InstrIter:
			for iter := NewIterFromBlock(succ); iter.Valid(); iter.MoveNext() {
				instr := iter.Get()
				switch instr.(type) {
				case *Phi:
					phi := instr.(*Phi)
					opd := phi.BBToOpd[block]
					replaceOpd(opd)
				default:
					break InstrIter
				}
			}
		}

		// Visit children in dominance tree
		for child := range block.Children {
			visit(child)
		}

		// Restore representative symbols
		for i, set := range setSym {
			if set {
				repSym[i] = nil
			}
		}
	}
	visit(fun.Enter)
}

func (o *GVNOpt) opdEq(v1, v2 *SSAVert) bool {
	/*if strings.HasPrefix(v1.label, "phi") {
		for _, o1 := range v1.opd {
			found := false
			for _, o2 := range v2.opd {
				if o.valNum[o1] == o.valNum[o2] {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}*/
	for i := range v1.opd {
		if o.valNum[v1.opd[i]] != o.valNum[v2.opd[i]] {
			return false
		}
	}
	return true
}

func (o *GVNOpt) pickOneIndex(set map[int]bool) int {
	for i := range set {
		return i
	}
	return -1
}

func pickOneSSAVert(set map[*SSAVert]bool) *SSAVert {
	for v := range set {
		return v
	}
	return nil
}
