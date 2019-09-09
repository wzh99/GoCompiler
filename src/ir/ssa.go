package ir

import (
	"fmt"
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

	// Apply optimizations to each function
	gvn := GVNOpt{opt: o} // global value numbering
	gvn.optimize(fun)
	//sccp := SCCPOpt{opt: o} // sparse conditional constant propagation
	//sccp.optimize(fun)
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
}

func (o *SSAOpt) removeDeadBlocks(fun *Func) {
	reachable := make(map[*BasicBlock]bool)
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		reachable[block] = true
	}, DepthFirst)
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for pred := range block.Pred {
			if reachable[pred] {
				continue
			}
			pred.DisconnectTo(block)
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
				bbToSym := make([]PhiOpd, 0)
				for pred := range y.Pred {
					bbToSym = append(bbToSym, PhiOpd{
						pred: pred,
						val:  NewVariable(a),
					})
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
	for iter := NewIterFromBlock(block); iter.Valid(); iter.Next() {
		def := iter.Cur.GetDef()
		if def == nil {
			continue
		}
		switch (*def).(type) {
		case *Variable: // only variables could be defined
			sym := (*def).(*Variable).Symbol
			if sym.Scope.Global {
				continue
			}
			defSymSet[sym] = true
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
		for iter := NewIterFromBlock(block); iter.Valid(); iter.Next() {
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
			def := o.getVarDef(instr)
			if def != nil {
				sym := (*def).(*Variable).Symbol
				newSym := ver[sym.Name].rename()
				*def = NewVariable(newSym) // replace definition of x with x_i
			}

		} // end instruction iteration loop

		// Replace use in phi instructions in successors
		for succ := range block.Succ {
			for iter := NewIterFromBlock(succ); iter.Valid(); iter.Next() {
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
		for iter := NewIterFromBlock(block); iter.Valid(); iter.Next() {
			def := o.getVarDef(iter.Cur)
			if def != nil {
				sym := (*def).(*Variable).Symbol
				ver[sym.Name].pop()
			}
		}
	}
	rename(fun.Enter)
}

func (o *SSAOpt) getVarUse(instr IInstr) []*IValue {
	useList := make([]*IValue, 0)
	for _, use := range instr.GetOpd() {
		if v, ok := (*use).(*Variable); ok {
			if v.Symbol.Scope.Global {
				continue
			}
			useList = append(useList, use)
		}
	}
	return useList
}

func (o *SSAOpt) getVarDef(instr IInstr) *IValue {
	def := instr.GetDef()
	if def == nil {
		return nil
	}
	if v, ok := (*def).(*Variable); ok {
		if v.Symbol.Scope.Global {
			return nil
		}
		return def
	} else {
		return nil
	}
}

type DefUseInfo struct {
	def    IInstr          // in SSA form, each value is defined only once
	useSet map[IInstr]bool // a value may be used in different places
}

func (o *SSAOpt) getDefUseInfo(fun *Func) map[*Symbol]*DefUseInfo {
	// Initialize def-use lookup table
	defUse := make(map[*Symbol]*DefUseInfo)
	for sym := range fun.Scope.Symbols {
		defUse[sym] = &DefUseInfo{
			def:    nil,
			useSet: make(map[IInstr]bool),
		}
	}

	// Visit instructions and fill in the table
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); iter.Next() {
			instr := iter.Cur
			def := instr.GetDef()
			if def != nil {
				defUse[(*def).(*Variable).Symbol].def = instr
			}
			for _, use := range instr.GetOpd() {
				switch (*use).(type) {
				case *Variable:
					sym := (*use).(*Variable).Symbol
					defUse[sym].useSet[instr] = true
				}
			}
		}
	}, DepthFirst)

	return defUse
}

// Dead Code Elimination algorithm. Can be used as a subroutine in multiple passes.
// See Algorithm 19.12 of Modern Compiler Implementation in Java, Second Edition
func (o *SSAOpt) eliminateDeadCode(fun *Func) {
	// Initialize work list and def-use info from scope
	workList := make(map[*Symbol]bool)
	for sym := range fun.Scope.Symbols {
		workList[sym] = true
	}
	defUse := o.getDefUseInfo(fun)

	// Iteratively eliminate dead code and symbols in function scope
	for len(workList) > 0 {
		sym := pickOneSymbol(workList)
		delete(workList, sym)
		dUInfo := defUse[sym]
		if len(dUInfo.useSet) > 0 { // still being used
			continue
		}
		if !sym.Param { // remove this symbol from scope if it is not a parameter
			delete(fun.Scope.Symbols, sym)
		}
		defInstr := dUInfo.def
		if defInstr == nil { // no instruction defined this symbol
			continue
		}
		NewIterFromInstr(defInstr).Remove() // remove this instruction from function
		for _, use := range defInstr.GetOpd() {
			switch (*use).(type) {
			case *Variable:
				x := (*use).(*Variable).Symbol
				delete(defUse[x].useSet, defInstr)
				workList[x] = true
			}
		}
	}
}

func pickOneSymbol(set map[*Symbol]bool) *Symbol {
	for s := range set {
		return s
	}
	return nil
}
