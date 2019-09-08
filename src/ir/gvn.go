package ir

import (
	"fmt"
	"os"
)

// Global Value Numbering
// Partition vertices in value graph so that each vertex in a set shares one value number.
// See Fig. 12.21 and 12.22 of Advanced Compiler Design and Implementation
type GVNOpt struct {
	opt   *SSAOpt
	graph *SSAGraph
}

func (o *GVNOpt) optimize(fun *Func) {
	// Build value graph out of SSA
	o.graph = newSSAGraph(fun)

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
			if v.hasSameLabel(o.pickOneSSAVert(part[i])) { // may be congruent
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
		v := o.pickOneSSAVert(set)
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
			for sym := range vert.symbols {
				if rep == nil {
					rep = sym
				}
				repSym[sym] = rep
			}
		}
	}

	// Transform IR according to representative set
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); {
			remove := o.simplify(iter.Cur, repSym)
			if remove {
				iter.Remove() // directly point to next instruction
			} else {
				iter.Next()
			}
		}
	}, DepthFirst)
	o.opt.eliminateDeadCode(fun)
}

func (o *GVNOpt) simplify(instr IInstr, repSym map[*Symbol]*Symbol) bool {
	// Remove redefinitions
	defList := instr.GetDef()
	for _, def := range defList {
		switch (*def).(type) {
		case *Variable:
			sym := (*def).(*Variable).Symbol
			if repSym[sym] != sym { // duplicated definition
				return true
			}
		}
	}

	// Replace symbols in use list with their representative symbols
	useList := instr.GetUse()
	for _, use := range useList {
		switch (*use).(type) {
		case *Variable:
			sym := (*use).(*Variable).Symbol
			if repSym[sym] != nil {
				*use = NewVariable(repSym[sym])
			}
		}
	}

	// Remove an instruction if it satisfy other conditions
	switch instr.(type) {
	case *Move:
		move := instr.(*Move)
		dst := move.Dst.(*Variable)
		switch move.Src.(type) {
		case *Variable:
			src := move.Src.(*Variable)
			if src.Symbol == dst.Symbol { // redundant move
				return true
			}
		}
	}

	return false
}

func (o *GVNOpt) pickOneIndex(set map[int]bool) int {
	for i := range set {
		return i
	}
	return -1
}

func (o *GVNOpt) pickOneSSAVert(set map[*SSAVert]bool) *SSAVert {
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
