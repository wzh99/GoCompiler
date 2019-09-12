package ir

import (
	"fmt"
	"os"
	"strings"
)

// Global Value Numbering
// Partition vertices in value graph so that each vertex in a set shares one value number.
// See Fig. 12.21 and 12.22 in The Whale Book.
type GVNOpt struct {
	graph *SSAGraph
}

func (o *GVNOpt) Optimize(fun *Func) {
	// Build value graph out of SSA
	o.graph = NewSSAGraph(fun)

	// Initialize vertex partition and work list
	part := make([]map[*SSAVert]bool, 0) // partition result: array of sets
	valNum := make(map[*SSAVert]int)     // map vertices to value number
	workList := make(map[int]bool, 0)    // sets to be further partitioned in B

TraverseVertSet:
	for v := range o.graph.vertSet {
		// Create the first set
		if len(part) == 0 {
			part = append(part, map[*SSAVert]bool{v: true})
			valNum[v] = 0
			continue
		}
		// Test whether there is congruence
		for i := 0; i < len(part); i++ {
			if v.hasSameLabel(pickOneSSAVert(part[i])) { // may be congruent
				part[i][v] = true
				valNum[v] = i
				if len(v.opd) > 0 && len(part[i]) > 1 { // depends on operands
					workList[i] = true
				}
				continue TraverseVertSet
			}
		}
		// No congruence is found, add to new set
		n := len(part)
		valNum[v] = n
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
			for i := range v.opd {
				if valNum[v.opd[i]] != valNum[v2.opd[i]] {
					// Not congruent, move to new one.
					delete(set, v2)
					newSet[v2] = true
					break // no need to test more
				}
			}
		}
		if len(newSet) > 0 { // another cut made in current set
			// Update the partition list and value number
			n := len(part)
			for v2 := range newSet {
				valNum[v2] = n
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

	// Build representative symbol lookup table
	repSym := make(map[*Symbol]*Symbol)
TraversePartition:
	for _, set := range part {
		var rep *Symbol
		for vert := range set {
			if vert.label == "param" {
				continue TraversePartition // parameters cannot be merged
			}
			for sym := range vert.symbols { // choose non-temporary symbol first for readability
				if strings.HasPrefix(sym.Name, "_s") && rep == nil {
					rep = sym
					break
				}
			}
			for sym := range vert.symbols {
				if rep == nil { // only temporary symbols
					rep = sym
				}
				repSym[sym] = rep // map this symbol to representative one
			}
		}
	}

	// Simplify instructions according to numbering result
	defOut := make(map[*BasicBlock]map[*Symbol]bool)
	copySet := func(set map[*Symbol]bool) map[*Symbol]bool {
		cp := make(map[*Symbol]bool)
		for s := range set {
			cp[s] = true
		}
		return cp
	}
	fun.Enter.AcceptAsTreeNode(func(block *BasicBlock) {
		if block.ImmDom == nil {
			defOut[block] = make(map[*Symbol]bool)
		} else {
			defOut[block] = copySet(defOut[block.ImmDom])
		}
		// simplification and set construction are executed simultaneously
		for iter := NewIterFromBlock(block); iter.Valid(); {
			remove := o.simplify(iter.Cur, repSym, defOut[block])
			if remove {
				iter.Remove() // directly point to next instruction
			} else {
				iter.MoveNext()
			}
		}
	}, func(*BasicBlock) {})

	eliminateDeadCode(fun)
}

func (o *GVNOpt) simplify(instr IInstr, repSym map[*Symbol]*Symbol,
	defined map[*Symbol]bool) bool {
	// Replace operands with representative symbols
	for _, use := range instr.GetOpd() {
		switch (*use).(type) {
		case *Variable:
			sym := (*use).(*Variable).Symbol
			if repSym[sym] != nil {
				*use = NewVariable(repSym[sym])
			}
		}
	}

	// Replace definitions with representative symbols and remove redefinitions
	def := instr.GetDef()
	if def == nil {
		return false
	}
	rep := repSym[(*def).(*Variable).Symbol]
	if defined[rep] {
		return true
	}
	*def = NewVariable(rep)
	defined[rep] = true

	// Break phi-phi cycle
	// For some vertices a, b, x, y in graph, a = phi(b, x), b = phi(a, y)
	// Vertices of temporary variables may create a phi-phi cycle in SSA graph.
	// The two phi instructions are redundant, so they should be eliminated.
	vert := o.graph.symToVert[rep]
	if !strings.HasPrefix(vert.label, "phi") {
		return false
	}
	if len(vert.use) == 0 { // this phi has no use
		return true
	}
	allPhi := true
	for u1 := range vert.use {
		allPhi = allPhi && strings.HasPrefix(u1.label, "phi")
	}
	if !allPhi { // not all use are phi instructions, cannot remove
		return false
	}
	hasCycle := false
	for u1 := range vert.use {
		for u2 := range u1.use {
			if u2 == vert { // one cycle found
				hasCycle = true // at least there is one cycle
				delete(u1.use, vert)
			}
		}
	}

	return hasCycle
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

func (o *GVNOpt) printPartition(part []map[*SSAVert]bool, valNum map[*SSAVert]int) {
	for _, set := range part {
		for s := range set {
			s.print(os.Stdout, valNum)
		}
		fmt.Println()
	}
	fmt.Println()
}
