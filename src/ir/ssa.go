package ir

type SSAOpt struct{}

func NewSSAOpt() *SSAOpt {
	return &SSAOpt{}
}

func (o *SSAOpt) Optimize(prg *Program) {
	for _, fun := range prg.Funcs {
		o.optimize(fun)
	}
}

// See Chapter 19 of Modern Compiler Implementation in Java, Second Edition.
func (o *SSAOpt) optimize(fun *Func) {
	// Convert to SSA form
	o.computeDominators(fun)
	o.insertPhi(fun)
	o.renameVar(fun)
}

// Use Lengauer-Tarjan Algorithm to build domination tree.
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
	o.clearUnreachable(fun)               // unreachable predecessors will cause error
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
	serial := 0
	fun.Enter.AcceptAsTreeNode(func(block *BasicBlock) {
		block.serial[0] = serial
		serial++
	}, func(block *BasicBlock) {
		block.serial[1] = serial
		serial++
	})

	fun.Enter.PrintDomTree()
}

func (o *SSAOpt) clearUnreachable(fun *Func) {
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
				delete(workList, n)
				break
			}
			// Possibly insert phi instructions for each node in dominance frontiers
			for y := range DF[node] { // for each dominance frontier node
				if APhi[y] == nil {
					APhi[y] = make(map[*Symbol]bool)
				}
				if APhi[y][a] {
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
	iter := NewInstrIterFromBlock(block)
	for iter.IsValid() {
		defList := iter.Cur.GetDef()
		for _, def := range defList {
			switch (*def).(type) {
			case *Variable: // only variables could be defined
				sym := (*def).(*Variable).Symbol
				if sym.Scope.Global {
					continue
				}
				defSymSet[(*def).(*Variable).Symbol] = true
			}
		}
		iter.Next()
	}
	return defSymSet
}

func (o *SSAOpt) renameVar(fun *Func) {
	// Initialize lookup table for latest variable
	latest := make(map[string]*Symbol) // original -> latest renamed
	for _, sym := range fun.Scope.Symbols {
		latest[sym.Name] = sym // the original serves as the first version
	}

	// Rename variables
	var rename func(*BasicBlock)
	rename = func(block *BasicBlock) {
		// Replace use and definitions in current block
		for iter := NewInstrIterFromBlock(block); iter.IsValid(); iter.Next() {
			instr := iter.Cur
			// Replace use in instructions
			if _, isPhi := instr.(*Phi); !isPhi { // not a phi instruction
				for _, use := range o.getVarUse(instr) {
					sym := (*use).(*Variable).Symbol
					top := latest[sym.Name]
					*use = NewVariable(top) // replace use of x with x_i
				}
			}
			// Replace definitions in instructions
			for _, def := range o.getVarDef(instr) {
				sym := (*def).(*Variable).Symbol
				newSym := latest[sym.Name].Rename()
				latest[sym.Name] = newSym
				*def = NewVariable(newSym) // replace definition of x with x_i
			}
		} // end instruction iteration loop

		// Replace use in phi instructions in successors
		for succ := range block.Succ {
			for iter := NewInstrIterFromBlock(succ); iter.IsValid(); iter.Next() {
				phi, isPhi := iter.Cur.(*Phi)
				if !isPhi {
					continue
				}
				sym := (*phi.BBToVal[block]).(*Variable).Symbol
				if sym.Scope.Global {
					continue
				}
				top := latest[sym.Name]
				*phi.BBToVal[block] = NewVariable(top)
			} // end instruction iteration loop
		} // end successor loop

		// Recursively rename variables in children nodes in the dominance tree
		for child := range block.Children {
			rename(child)
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
	for _, use := range instr.GetDef() {
		if v, ok := (*use).(*Variable); ok {
			if v.Symbol.Scope.Global {
				continue
			}
			defList = append(defList, use)
		}
	}
	return defList
}
