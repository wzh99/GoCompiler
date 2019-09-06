package ir

import (
	"fmt"
	"io"
	"os"
)

type SSAOpt struct {
	prg *Program
}

func NewSSAOpt() *SSAOpt {
	return &SSAOpt{}
}

func (o *SSAOpt) Optimize(prg *Program) {
	o.prg = prg
	for _, fun := range prg.Funcs {
		o.optimize(fun)
	}
}

// See Chapter 19 of Modern Compiler Implementation in Java, Second Edition.
func (o *SSAOpt) optimize(fun *Func) {
	// Convert to SSA form
	o.splitEdge(fun)
	o.computeDominators(fun)
	o.insertPhi(fun)
	o.renameVar(fun)

	// Apply optimizing transformations to function
	o.globalValueNumber(fun)
}

// Transform tp an edge-split CFG
func (o *SSAOpt) splitEdge(fun *Func) {
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		if len(block.Succ) <= 1 {
			return
		}
		for succ := range block.Succ {
			if len(succ.Pred) > 1 {
				block.SplitEdgeTo(succ, o.newBasicBlock("EdgeSplit", fun))
			}
		}
	}, DepthFirst)
}

func (o *SSAOpt) newBasicBlock(name string, fun *Func) *BasicBlock {
	tag := fmt.Sprintf("_B%d", o.prg.nBlock)
	o.prg.nBlock++
	return NewBasicBlock(fmt.Sprintf("%s_%s", tag, name), fun)
}

// Use Lengauer-Tarjan Algorithm to build dominator tree.
// This version has time complexity O(NlogN), not optimal but easier to understand.
func (o *SSAOpt) computeDominators(fun *Func) {
	// Define depth first tree node
	type DFTreeNode struct {
		bb       *BasicBlock
		dfNum    int
		parent   *DFTreeNode          // parent of node in depth first tree
		ancestor *DFTreeNode          // represent the spanning forest
		semi     *DFTreeNode          // semi-dominator
		best     *DFTreeNode          // help path compression
		sameDom  *DFTreeNode          // node which shared the same dominator with this node
		bucket   map[*DFTreeNode]bool // all the nodes this node semi-dominates
	}
	nodes := make([]*DFTreeNode, 0)
	bbToNode := make(map[*BasicBlock]*DFTreeNode) // maps vertices in CFG to nodes in DF tree

	// Build depth first tree
	var dfs func(parent, cur *BasicBlock)
	dfs = func(parent, cur *BasicBlock) {
		node := &DFTreeNode{
			bb:       cur,
			dfNum:    len(nodes),
			parent:   bbToNode[parent],
			ancestor: nil,
			semi:     nil,
			best:     nil,
			sameDom:  nil,
			bucket:   make(map[*DFTreeNode]bool),
		}
		nodes = append(nodes, node)
		bbToNode[cur] = node
		for s := range cur.Succ {
			if bbToNode[s] == nil {
				dfs(cur, s)
			}
		}
	}
	dfs(nil, fun.Enter)
	if len(nodes) <= 1 {
		return
	} // no point in building dominator tree

	// Compute auxiliary values for determining dominators
	var ancestorWithLowestSemi func(node *DFTreeNode) *DFTreeNode
	ancestorWithLowestSemi = func(node *DFTreeNode) *DFTreeNode {
		anc := node.ancestor
		if anc.ancestor != nil {
			best := ancestorWithLowestSemi(anc)
			node.ancestor = anc.ancestor
			if best.semi.dfNum < node.best.semi.dfNum {
				node.best = best
			}
		}
		return node.best
	}
	o.removeDeadBlocks(fun)               // unreachable predecessors will cause error
	for i := len(nodes) - 1; i > 0; i-- { // back to forth, ignore root node
		node := nodes[i]
		parent := node.parent
		semi := parent
		for v := range node.bb.Pred {
			// Predecessors of vertices are visited. If a predecessor is unreachable, it will not
			// be visited in DFS, and it cannot be found in the map.
			pred := bbToNode[v]
			var newSemi *DFTreeNode
			if pred.dfNum <= node.dfNum { // use semi-dominator theorem
				newSemi = pred
			} else {
				newSemi = ancestorWithLowestSemi(pred).semi
			}
			if newSemi.dfNum < semi.dfNum {
				semi = newSemi
			}
		}
		node.semi = semi
		// node's dominator is deferred until the path is linked into the forest
		semi.bucket[node] = true
		node.ancestor = parent
		node.best = node
		for v := range parent.bucket {
			anc := ancestorWithLowestSemi(v) // link into spanning forest
			if anc.semi == v.semi {          // use dominator theorem
				v.bb.SetImmDom(parent.bb)
			} else { // defer until dominator is known
				v.sameDom = anc
			}
		}
		parent.bucket = make(map[*DFTreeNode]bool)
	}

	// Finish deferred immediate dominator computation
	for i, n := range nodes {
		if i == 0 {
			continue
		}
		if n.sameDom != nil {
			n.bb.SetImmDom(n.sameDom.bb.ImmDom)
		}
	}

	// Visit dominance tree and mark in and out serial of nodes
	fun.Enter.NumberDomTree()
	fun.Enter.PrintDomTree()
}

func (o *SSAOpt) removeDeadBlocks(fun *Func) {
	reachable := make(map[*BasicBlock]bool)
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		reachable[block] = true
	}, DepthFirst)
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for pred := range block.Pred {
			if !reachable[pred] {
				pred.DisconnectTo(block)
			}
		}
	}, DepthFirst)
}

func (o *SSAOpt) insertPhi(fun *Func) {
	// Compute dominate frontiers of basic blocks
	DF := make(map[*BasicBlock]map[*BasicBlock]bool)
	var computeDF func(*BasicBlock)
	computeDF = func(node *BasicBlock) {
		dfSet := make(map[*BasicBlock]bool)
		for succ := range node.Succ { // DF_local
			if succ.ImmDom != node {
				dfSet[succ] = true
			}
		}
		for child := range node.Children {
			computeDF(child)
			childDF := DF[child]
			for w := range childDF {
				if !node.Dominates(w) || node == w { // DF_up
					dfSet[w] = true
				}
			}
		}
		DF[node] = dfSet
	}
	computeDF(fun.Enter)

	// Compute set A_orig of symbols defined in each block
	// the set of variables the phi instructions of whom are inserted
	APhi := make(map[*BasicBlock]map[*Symbol]bool)
	// the set of variables defined in a block
	AOrig := make(map[*BasicBlock]map[*Symbol]bool)
	// the basic blocks where a certain variable is defined
	defSite := make(map[*Symbol]map[*BasicBlock]bool)
	fun.Enter.AcceptAsVert(func(node *BasicBlock) {
		AOrig[node] = o.getDefSymbolSet(node)
		for v := range AOrig[node] {
			if defSite[v] == nil {
				defSite[v] = make(map[*BasicBlock]bool)
			}
			defSite[v][node] = true
		}
	}, DepthFirst)
	for a, site := range defSite { // for each variable
		workList := make(map[*BasicBlock]bool)
		// Copy defined site of symbol to work list
		for node := range site {
			workList[node] = true
		}
		// Use work list algorithm to iteratively insert phi instructions
		for len(workList) > 0 { // for each involved node in work list
			// Arbitrarily remove an element from list
			var node *BasicBlock
			for n := range workList {
				node = n
				break
			}
			delete(workList, node)
			// Possibly insert phi instructions for each node in dominance frontiers
			for y := range DF[node] { // for each dominance frontier node
				if APhi[y] == nil {
					APhi[y] = make(map[*Symbol]bool)
				}
				if APhi[y][a] { // phi instruction already inserted for a in y
					continue
				}
				// Insert phi instruction at top of block y
				bbToSym := make(map[*BasicBlock]IValue)
				for pred := range y.Pred {
					bbToSym[pred] = NewVariable(a)
				}
				y.PushFront(NewPhi(bbToSym, NewVariable(a)))
				APhi[y][a] = true
				// Add to work list if variable a is defined in a block that it previously was not
				if !AOrig[y][a] {
					workList[y] = true
				}
			} // end dominance frontier loop
		} // end work list loop
	} // end variable loop
}

func (o *SSAOpt) getDefSymbolSet(block *BasicBlock) map[*Symbol]bool {
	defSymSet := make(map[*Symbol]bool)
	for iter := NewInstrIter(block); iter.Valid(); iter.Next() {
		defList := iter.Cur.GetDef()
		for _, def := range defList {
			switch (*def).(type) {
			case *Variable: // only variables could be defined
				sym := (*def).(*Variable).Symbol
				if sym.Scope.Global {
					continue
				}
				defSymSet[sym] = true
			}
		}
	}
	return defSymSet
}

type VarVersion struct {
	latest *Symbol
	stack  []*Symbol
}

func (i *VarVersion) push(s *Symbol) { i.stack = append(i.stack, s) }

func (i *VarVersion) top() *Symbol { return i.stack[len(i.stack)-1] }

func (i *VarVersion) pop() { i.stack = i.stack[:len(i.stack)-1] }

func (i *VarVersion) rename() *Symbol {
	sym := i.latest.Rename()
	i.latest = sym
	i.push(sym)
	return sym
}

func (o *SSAOpt) renameVar(fun *Func) {
	// Initialize lookup table for latest variable
	ver := make(map[string]*VarVersion)
	for sym := range fun.Scope.Symbols {
		ver[sym.Name] = &VarVersion{
			latest: sym,
			stack:  []*Symbol{sym},
		} // the original serves as the first version
	}

	// Rename variables
	var rename func(*BasicBlock)
	rename = func(block *BasicBlock) {
		// Replace use and definitions in current block
		for iter := NewInstrIter(block); iter.Valid(); iter.Next() {
			instr := iter.Cur
			// Replace use in instructions
			if _, isPhi := instr.(*Phi); !isPhi { // not a phi instruction
				for _, use := range o.getVarUse(instr) {
					sym := (*use).(*Variable).Symbol
					top := ver[sym.Name].top()
					*use = NewVariable(top) // replace use of x with x_i
				}
			}
			// Replace definitions in instructions
			for _, def := range o.getVarDef(instr) {
				sym := (*def).(*Variable).Symbol
				newSym := ver[sym.Name].rename()
				*def = NewVariable(newSym) // replace definition of x with x_i
			}
		} // end instruction iteration loop

		// Replace use in phi instructions in successors
		for succ := range block.Succ {
			for iter := NewInstrIter(succ); iter.Valid(); iter.Next() {
				phi, isPhi := iter.Cur.(*Phi)
				if !isPhi {
					continue
				}
				sym := (*phi.BBToVal[block]).(*Variable).Symbol
				if sym.Scope.Global {
					continue
				}
				top := ver[sym.Name].top()
				*phi.BBToVal[block] = NewVariable(top)
			} // end instruction iteration loop
		} // end successor loop

		// Recursively rename variables in children nodes in the dominance tree
		for child := range block.Children {
			rename(child)
		}

		// Remove symbol renamed in this frame
		for iter := NewInstrIter(block); iter.Valid(); iter.Next() {
			for _, def := range o.getVarDef(iter.Cur) {
				sym := (*def).(*Variable).Symbol
				ver[sym.Name].pop()
			}
		}
	}
	rename(fun.Enter)
}

func (o *SSAOpt) getVarUse(instr IInstr) []*IValue {
	useList := make([]*IValue, 0)
	for _, use := range instr.GetUse() {
		if v, ok := (*use).(*Variable); ok {
			if v.Symbol.Scope.Global {
				continue
			}
			useList = append(useList, use)
		}
	}
	return useList
}

func (o *SSAOpt) getVarDef(instr IInstr) []*IValue {
	defList := make([]*IValue, 0)
	for _, def := range instr.GetDef() {
		if v, ok := (*def).(*Variable); ok {
			if v.Symbol.Scope.Global {
				continue
			}
			defList = append(defList, def)
		}
	}
	return defList
}

// Format of labels:
// Value vertices:
// param, imm.
// Instruction vertices:
// move, load, malloc, getptr, ptroff, clear;
// neg, not, add, sub, mul, div, mod, and, or, xor, shl, shr, eq, ne, lt, le, gt, ge;
// call@%s, phi.
// Store, return, jump and branch don't define values, so they are not included in value graph.
type ValueVert struct {
	label    string
	symbols  map[*Symbol]bool // set of symbols this vertex maps to
	imm      interface{}      // store immediate
	operands []*ValueVert
}

func newValueVert(label string, sym *Symbol, imm interface{}, opd ...*ValueVert) *ValueVert {
	vert := &ValueVert{
		label:    label,
		imm:      imm,
		symbols:  make(map[*Symbol]bool),
		operands: opd,
	}
	if sym != nil {
		vert.symbols[sym] = true
	}
	return vert
}

func newTempVert(sym *Symbol) *ValueVert {
	return &ValueVert{
		symbols: map[*Symbol]bool{sym: true},
	}
}

func (v *ValueVert) appendInfo(label string, imm interface{}, opd ...*ValueVert) {
	v.label = label
	v.imm = imm
	v.operands = append(v.operands, opd...)
}

func (v *ValueVert) print(writer io.Writer) {
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
	for i, opd := range v.operands {
		if i != 0 {
			str += ", "
		}
		str += opd.label
	}
	_, _ = fmt.Fprintln(writer, str+"} }")
}

func (v *ValueVert) hasSameLabel(v2 *ValueVert) bool {
	if v.label != v2.label {
		return false
	}
	switch v.label {
	case "imm":
		switch v.imm.(type) { // immediate value should be considered as part of label
		case bool:
			return v.imm.(bool) == v2.imm.(bool)
		case int:
			return v.imm.(int) == v2.imm.(int)
		case float64:
			return v.imm.(float64) == v2.imm.(float64)
		default:
			return false
		}
	}
	return true
}

type ValueGraph struct {
	vertSet map[*ValueVert]bool // set of all vertices in the graph
	// edges are stored in vertices, not stored globally
	symToVert map[*Symbol]*ValueVert // maps symbols to vertices
}

func newValueGraph(fun *Func) *ValueGraph {
	// Initialize data structures
	g := &ValueGraph{
		vertSet:   make(map[*ValueVert]bool),
		symToVert: make(map[*Symbol]*ValueVert),
	}

	// Add symbols as temporary vertices
	for sym := range fun.Scope.Symbols {
		if sym.Param {
			g.addVert(newValueVert("param", sym, nil))
		} else {
			g.addVert(newTempVert(sym))
		}
	}

	// Visit instructions of SSA form
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewInstrIter(block); iter.Valid(); iter.Next() {
			g.processInstr(iter.Cur)
		}
	}, DepthFirst)
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewInstrIter(block); iter.Valid(); iter.Next() {
			g.fixPhi(iter.Cur)
		}
	}, DepthFirst)

	// Mark unlabelled vertices
	for v := range g.vertSet {
		if len(v.label) == 0 { // unlabelled
			v.label = "undef"
		}
	}

	return g
}

func (g *ValueGraph) print(writer io.Writer) {
	for vert := range g.vertSet {
		vert.print(writer)
	}
	_, _ = fmt.Fprintln(writer)
}

func (g *ValueGraph) addVert(vert *ValueVert) {
	g.vertSet[vert] = true
	for sym := range vert.symbols {
		g.symToVert[sym] = vert
	}
}

func (g *ValueGraph) addSymbolToVert(sym *Symbol, vert *ValueVert) {
	if sym == nil {
		panic(NewIRError("cannot add nil symbol to vertex"))
	}
	g.symToVert[sym] = vert
	vert.symbols[sym] = true
}

func (g *ValueGraph) mergeVert(target, prey *Symbol) {
	targVert := g.symToVert[target]
	preyVert := g.symToVert[prey]
	g.addSymbolToVert(prey, targVert)
	delete(g.vertSet, preyVert)
}

func (g *ValueGraph) appendInfoToVert(sym *Symbol, label string, imm interface{},
	opd ...*ValueVert) {
	g.symToVert[sym].appendInfo(label, imm, opd...)
}

func (g *ValueGraph) valToVert(val IValue) *ValueVert {
	var vert *ValueVert
	switch val.(type) {
	case *ImmValue:
		vert = newValueVert("imm", nil, val.(*ImmValue).Value)
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

func (g *ValueGraph) processInstr(instr IInstr) {
	switch instr.(type) {
	case *Move:
		move := instr.(*Move)
		dst := move.Dst.(*Variable)
		switch move.Src.(type) {
		case *ImmValue:
			imm := move.Src.(*ImmValue).Value
			g.appendInfoToVert(dst.Symbol, "imm", imm)
		case *Variable:
			src := move.Src.(*Variable)
			g.mergeVert(src.Symbol, dst.Symbol)
		}

	case *Load:
		load := instr.(*Load)
		dst := load.Dst.(*Variable)
		g.appendInfoToVert(dst.Symbol, "load", nil, g.valToVert(load.Src))

	case *Malloc:
		malloc := instr.(*Malloc)
		dst := malloc.Result.(*Variable)
		g.appendInfoToVert(dst.Symbol, "malloc", nil)

	case *GetPtr:
		getptr := instr.(*GetPtr)
		dst := getptr.Result.(*Variable)
		opd := []*ValueVert{g.valToVert(getptr.Base)}
		for _, val := range getptr.Indices {
			opd = append(opd, g.valToVert(val))
		}
		g.appendInfoToVert(dst.Symbol, "getptr", nil, opd...)

	case *PtrOffset:
		ptroff := instr.(*PtrOffset)
		dst := ptroff.Dst.(*Variable)
		g.appendInfoToVert(dst.Symbol, "ptroff", nil, g.valToVert(ptroff.Src))

	case *Clear:
		clear := instr.(*Clear)
		val := clear.Value.(*Variable)
		g.appendInfoToVert(val.Symbol, "clear", nil)

	case *Unary:
		unary := instr.(*Unary)
		opd := unary.Operand.(*Variable)
		result := unary.Result.(*Variable)
		g.appendInfoToVert(result.Symbol, unaryOpStr[unary.Op], nil, g.valToVert(opd))

	case *Binary:
		binary := instr.(*Binary)
		result := binary.Result.(*Variable)
		g.appendInfoToVert(result.Symbol, binaryOpStr[binary.Op], nil,
			g.valToVert(binary.Left), g.valToVert(binary.Right))

	case *Phi:
		phi := instr.(*Phi)
		result := phi.Result.(*Variable)
		opd := make([]*ValueVert, 0)
		for _, val := range phi.ValList {
			opd = append(opd, g.valToVert(val))
		}
		g.appendInfoToVert(result.Symbol, fmt.Sprintf("phi@%s", phi.BB.Name),
			nil, opd...)
	}
}

// Update operands in phi instructions with latest vertices
func (g *ValueGraph) fixPhi(instr IInstr) {
	switch instr.(type) {
	case *Phi:
		phi := instr.(*Phi)
		result := phi.Result.(*Variable)
		vert := g.symToVert[result.Symbol]
		for i, opd := range vert.operands {
			if len(opd.label) == 0 {
				vert.operands[i] = g.valToVert(phi.ValList[i])
			}
		}
	}
}

// Partition vertices in value graph so that each vertex in a set shares one value number.
// See Fig. 12.21 and 12.22 of Advanced Compiler Design and Implementation
func (o *SSAOpt) globalValueNumber(fun *Func) {
	// Build value graph out of SSA
	graph := newValueGraph(fun)

	// Initialize vertex partition and work list
	part := make([]map[*ValueVert]bool, 0) // partition result: array of sets
	valNum := make(map[*ValueVert]int)     // map vertices to value number
	workList := make(map[int]bool, 0)      // sets to be further partitioned in B

TraverseVertSet:
	for v := range graph.vertSet {
		// Create the first set
		if len(part) == 0 {
			part = append(part, map[*ValueVert]bool{v: true})
			continue
		}
		// Test whether there is congruence
		for i := 0; i < len(part); i++ {
			if v.hasSameLabel(o.pickOneVert(part[i])) { // may be congruent
				part[i][v] = true
				valNum[v] = i
				if len(v.operands) > 0 && len(part[i]) > 1 { // depends on operands
					workList[i] = true
				}
				continue TraverseVertSet
			}
		}
		part = append(part, map[*ValueVert]bool{v: true}) // no congruence is found
	}
	o.printPartition(part)
	fmt.Println(workList)

	// Further partition the vertex set until a fixed point is reached
}

func (o *SSAOpt) pickOneIndex(set map[int]bool) int {
	for i := range set {
		return i
	}
	return -1
}

func (o *SSAOpt) pickOneVert(set map[*ValueVert]bool) *ValueVert {
	for v := range set {
		return v
	}
	return nil
}

func (o *SSAOpt) printPartition(part []map[*ValueVert]bool) {
	for _, set := range part {
		for s := range set {
			s.print(os.Stdout)
		}
		fmt.Println()
	}
	fmt.Println()
}

func (o *SSAOpt) copySet(set map[*ValueVert]bool) map[*ValueVert]bool {
	cp := make(map[*ValueVert]bool)
	for v := range set {
		cp[v] = true
	}
	return cp
}
