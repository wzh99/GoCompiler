package ir

import (
	"fmt"
	"io"
)

// Format of labels:
// Value vertices:
// param, imm.
// Instruction vertices:
// load, malloc, getptr, ptroff, clear;
// neg, not, add, sub, mul, div, mod, and, or, xor, shl, shr, eq, ne, lt, le, gt, ge;
// call@%s, phi@%s.
// store, return, jump and branch don't define values, so they are not included in value graph.
type SSAVert struct {
	// Instruction that define this value. Can be used to map from SSA graph to CFG
	instr IInstr
	// String label to distinguish different kind of vertices
	label string
	// Set of symbols this vertex maps to
	symbols map[*Symbol]bool
	// Common place where immediate is stored
	imm interface{}
	// Operands that this value uses (use -> def)
	opd []*SSAVert
	// Uses of this value (def -> use)
	use map[*SSAVert]bool
}

func newSSAVert(instr IInstr, label string, sym *Symbol, imm interface{},
	opd ...*SSAVert) *SSAVert {
	vert := &SSAVert{
		instr:   instr,
		label:   label,
		imm:     imm,
		symbols: make(map[*Symbol]bool),
		opd:     opd,
		use:     make(map[*SSAVert]bool),
	}
	for _, v2 := range opd { // add use point to operands
		v2.use[vert] = true
	}
	if sym != nil {
		vert.symbols[sym] = true
	}
	return vert
}

func newTempVert(sym *Symbol) *SSAVert {
	return &SSAVert{
		symbols: map[*Symbol]bool{sym: true},
		use:     make(map[*SSAVert]bool),
	}
}

func (v *SSAVert) appendInfo(instr IInstr, label string, imm interface{}, opd ...*SSAVert) {
	v.instr = instr
	v.label = label
	v.imm = imm
	v.opd = opd
	for _, v2 := range opd { // add use point to operands
		v2.use[v] = true
	}
}

func (v *SSAVert) print(writer io.Writer, valNum map[*SSAVert]int) {
	str := fmt.Sprintf("{ label: %s, symbols: {", v.label)
	i := 0
	for s := range v.symbols {
		if i != 0 {
			str += ", "
		}
		str += s.ToString()
		i++
	}
	str += "}, operands: {"
	for i, opd := range v.opd {
		if i != 0 {
			str += ", "
		}
		str += fmt.Sprintf("%d", valNum[opd])
	}
	_, _ = fmt.Fprintln(writer, str+"} }")
}

func (v *SSAVert) hasSameLabel(v2 *SSAVert) bool {
	if v.label != v2.label {
		return false
	}
	switch v.label {
	case "imm": // immediate value should be considered as part of label
		return immEq(v.imm, v2.imm)
	}
	return true
}

func immEq(i1, i2 interface{}) bool {
	switch i1.(type) { // immediate value should be considered as part of label
	case bool:
		switch i2.(type) {
		case bool:
			return i1.(bool) == i2.(bool)
		}
	case int:
		switch i2.(type) {
		case int:
			return i1.(int) == i2.(int)
		}
	case float64:
		switch i2.(type) {
		case float64:
			return i1.(float64) == i2.(float64)
		}
	}
	return false
}

type SSAGraph struct {
	vertSet map[*SSAVert]bool // set of all vertices in the graph
	// edges are stored in vertices, not stored globally
	symToVert map[*Symbol]*SSAVert // maps symbols to vertices
}

func NewSSAGraph(fun *Func) *SSAGraph {
	// Initialize data structures
	g := &SSAGraph{
		vertSet:   make(map[*SSAVert]bool),
		symToVert: make(map[*Symbol]*SSAVert),
	}

	// Add symbols as temporary vertices
	for sym := range fun.Scope.Symbols {
		if sym.Param {
			g.addVert(newSSAVert(nil, "param", sym, nil))
		} else {
			g.addVert(newTempVert(sym))
		}
	}

	// Visit instructions of SSA form
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			g.processInstr(iter.Cur)
		}
	}, DepthFirst)
	/*fun.Enter.AcceptAsTreeNode(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
			g.fixPhi(iter.Cur)
		}
	}, func(*BasicBlock) {})*/

	// Mark unlabelled vertices
	for v := range g.vertSet {
		if len(v.label) == 0 { // unlabelled
			v.label = "undef_" + pickOneSymbol(v.symbols).Type.ToString()
		}
	}

	return g
}

func (g *SSAGraph) print(writer io.Writer, valNum map[*SSAVert]int) {
	for vert := range g.vertSet {
		vert.print(writer, valNum)
	}
	_, _ = fmt.Fprintln(writer)
}

func (g *SSAGraph) addVert(vert *SSAVert) {
	g.vertSet[vert] = true
	for sym := range vert.symbols {
		g.symToVert[sym] = vert
	}
}

func (g *SSAGraph) addSymbolToVert(sym *Symbol, vert *SSAVert) {
	if sym == nil {
		panic(NewIRError("cannot add nil symbol to vertex"))
	}
	g.symToVert[sym] = vert
	vert.symbols[sym] = true
}

func (g *SSAGraph) mergeVert(target, prey *Symbol) {
	targVert := g.symToVert[target]
	preyVert := g.symToVert[prey]
	g.addSymbolToVert(prey, targVert)
	delete(g.vertSet, preyVert)
}

func (g *SSAGraph) appendInfoToVert(instr IInstr, sym *Symbol, label string, imm interface{},
	opd ...*SSAVert) {
	g.symToVert[sym].appendInfo(instr, label, imm, opd...)
}

func (g *SSAGraph) valToVert(val IValue) *SSAVert {
	var vert *SSAVert
	switch val.(type) {
	case *ImmValue:
		vert = newSSAVert(nil, "imm", nil, val.(*ImmValue).Value)
		g.addVert(vert)
	case *Variable:
		sym := val.(*Variable).Symbol
		vert = g.symToVert[sym]
		if vert == nil {
			panic(NewIRError(fmt.Sprintf("vertex not found for symbol %s",
				sym.ToString())))
		}
	}
	return vert
}

func (g *SSAGraph) processInstr(instr IInstr) {
	switch instr.(type) {
	case *Move:
		move := instr.(*Move)
		dst := move.Dst.(*Variable)
		switch move.Src.(type) {
		case *ImmValue:
			imm := move.Src.(*ImmValue).Value
			g.appendInfoToVert(instr, dst.Symbol, "imm", imm)
		case *Variable:
			src := move.Src.(*Variable)
			g.mergeVert(src.Symbol, dst.Symbol)
		}

	case *Load:
		load := instr.(*Load)
		dst := load.Dst.(*Variable)
		g.appendInfoToVert(instr, dst.Symbol, "load", nil, g.valToVert(load.Src))

	case *Malloc:
		malloc := instr.(*Malloc)
		dst := malloc.Result.(*Variable)
		g.appendInfoToVert(instr, dst.Symbol, "malloc", nil)

	case *GetPtr:
		getptr := instr.(*GetPtr)
		dst := getptr.Result.(*Variable)
		opd := []*SSAVert{g.valToVert(getptr.Base)}
		for _, val := range getptr.Indices {
			opd = append(opd, g.valToVert(val))
		}
		g.appendInfoToVert(instr, dst.Symbol, "getptr", nil, opd...)

	case *PtrOffset:
		ptroff := instr.(*PtrOffset)
		dst := ptroff.Dst.(*Variable)
		g.appendInfoToVert(instr, dst.Symbol, "ptroff", nil, g.valToVert(ptroff.Src))

	case *Clear:
		clear := instr.(*Clear)
		val := clear.Value.(*Variable)
		g.appendInfoToVert(instr, val.Symbol, "clear", nil)

	case *Unary:
		unary := instr.(*Unary)
		opd := unary.Operand.(*Variable)
		result := unary.Result.(*Variable)
		g.appendInfoToVert(instr, result.Symbol, unaryOpStr[unary.Op], nil, g.valToVert(opd))

	case *Binary:
		binary := instr.(*Binary)
		result := binary.Result.(*Variable)
		g.appendInfoToVert(instr, result.Symbol, binaryOpStr[binary.Op], nil,
			g.valToVert(binary.Left), g.valToVert(binary.Right))

	case *Phi:
		phi := instr.(*Phi)
		result := phi.Result.(*Variable)
		opd := make([]*SSAVert, 0)
		for _, val := range phi.ValList {
			opd = append(opd, g.valToVert(val))
		}
		g.appendInfoToVert(instr, result.Symbol, fmt.Sprintf("phi@%s", phi.BB.Name),
			nil, opd...)
	}
}

// Update operands in phi instructions with latest vertices
func (g *SSAGraph) fixPhi(instr IInstr) {
	switch instr.(type) {
	case *Phi:
		phi := instr.(*Phi)
		result := phi.Result.(*Variable)
		vert := g.symToVert[result.Symbol]
		for i, opd := range vert.opd {
			if len(opd.label) == 0 {
				opd := g.valToVert(phi.ValList[i])
				vert.opd[i] = opd
				opd.use[vert] = true
			}
		}
	}
}

// Merge vertices with equal immediate value
func (g *SSAGraph) MergeImm() {
	workList := make(map[*SSAVert]bool)
	for v := range g.vertSet {
		workList[v] = true
	}
	for len(workList) > 0 {
		v1 := pickOneSSAVert(workList)
		delete(workList, v1)
		for v2 := range g.vertSet {
			if v1 == v2 || v1.label != "imm" || v2.label != "imm" {
				continue
			}
			if immEq(v1.imm, v2.imm) {
				g.mergeTwoImm(v1, v2)
				delete(workList, v2)
				delete(g.vertSet, v2)
			}
		}
	}
}

func (g *SSAGraph) mergeTwoImm(v1, v2 *SSAVert) {
	for s := range v2.symbols {
		v1.symbols[s] = true // union of two symbol set
		g.symToVert[s] = v1  // map to the the first vertex
	}
	for use := range v2.use {
		v1.use[use] = true // union of two use set
		for i, opd := range use.opd { // change operands in use to merged vertex
			if opd == v2 {
				use.opd[i] = v1
			}
		}
	}
}
