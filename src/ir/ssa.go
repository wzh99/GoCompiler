package ir

import (
	"fmt"
)

type SSAOpt struct {
	opt []IOpt
	prg *Program
}

func NewSSAOpt(opt ...IOpt) *SSAOpt {
	return &SSAOpt{opt: opt}
}

func (o *SSAOpt) Optimize(prg *Program) {
	o.prg = prg
	for _, fun := range prg.Funcs {
		// Convert to SSA form
		o.splitEdge(fun)
		computeDominators(fun)
		o.insertPhi(fun)
		o.renameVar(fun)

		// Apply optimizations to each function
		for _, opt := range o.opt {
			opt.Optimize(fun)
		}
	}
}

type IOpt interface {
	Optimize(*Func)
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
func computeDominators(fun *Func) {
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
		// Initialize dominance tree members in basic block
		cur.ImmDom = nil
		cur.Children = make(map[*BasicBlock]bool)

		// Build depth first tree node
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

		// Recursively build children nodes
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
	removeDeadBlocks(fun) // unreachable predecessors will cause error
	for i := len(nodes) - 1; i > 0; i-- { // back to forth, ignore root node
		node := nodes[i]
		parent := node.parent
		semi := parent
		for v := range node.bb.Pred {
			// Predecessors of vertices are visited. If a predecessor is unreachable, it will
			// not be visited in DFS, and it cannot be found in the map.
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
			if anc.semi == v.semi { // use dominator theorem
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

func removeDeadBlocks(fun *Func) {
	// Mark blocks that is sure to be reachable
	reached := make(map[*BasicBlock]bool)
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		reached[block] = true
	}, DepthFirst)
	// Remove all unreachable predecessors to reachable blocks
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for pred := range block.Pred {
			if !reached[pred] {
				pred.DisconnectTo(block)
			}
		}
	}, DepthFirst)
}

func removeOneBlock(set map[*BasicBlock]bool) *BasicBlock {
	var block *BasicBlock
	for b := range set {
		block = b
		break
	}
	delete(set, block)
	return block
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
				// Add to work list if variable a is defined in a block that it previously
				// was not
				if !AOrig[y][a] {
					workList[y] = true
				}
			} // end dominance frontier loop
		} // end work list loop
	} // end variable loop
}

func (o *SSAOpt) getDefSymbolSet(block *BasicBlock) map[*Symbol]bool {
	defSymSet := make(map[*Symbol]bool)
	for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
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
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
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
			for iter := NewIterFromBlock(succ); iter.Valid(); iter.MoveNext() {
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
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
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

func getDefUseInfo(fun *Func) map[*Symbol]*DefUseInfo {
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
		for iter := NewIterFromBlock(block); iter.Valid(); iter.MoveNext() {
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
// See Algorithm 19.12 of The Tiger Book.
func eliminateDeadCode(fun *Func) {
	// Initialize work list and def-use info from scope
	workList := make(map[*Symbol]bool)
	for sym := range fun.Scope.Symbols {
		workList[sym] = true
	}
	defUse := getDefUseInfo(fun)

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
		if defInstr == nil { // an undefined symbol
			continue // skip removing its defining instruction
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

	// Break phi-phi cycle
	// For some vertices a, b, x, y in graph, a = phi(b, x), b = phi(a, y)
	// Vertices of temporary variables may create a phi-phi cycle in SSA graph.
	// The two phi instructions are useless, so they should be eliminated.
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		for iter := NewIterFromBlock(block); iter.Valid(); {
			switch iter.Cur.(type) {
			case *Phi:
				phi := iter.Cur.(*Phi)
				sym := phi.Result.(*Variable).Symbol
				if len(defUse[sym].useSet) != 1 {
					iter.MoveNext()
					continue
				}
				useInstr := pickOneInstr(defUse[sym].useSet)
				switch useInstr.(type) {
				case *Phi:
					sym2 := useInstr.(*Phi).Result.(*Variable).Symbol
					if defUse[sym2].useSet[iter.Cur] {
						iter.Remove()
						continue
					}
				}
			}
			iter.MoveNext()
		}
	}, DepthFirst)
}

func pickOneInstr(set map[IInstr]bool) IInstr {
	for i := range set {
		return i
	}
	return nil
}

func pickOneSymbol(set map[*Symbol]bool) *Symbol {
	for s := range set {
		return s
	}
	return nil
}
