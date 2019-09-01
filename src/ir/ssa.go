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

// From Chapter 19 of Modern Compiler Implementation in Java, 2nd Edition.
func (o *SSAOpt) optimize(fun *Func) {
	// Convert to SSA form
	o.computeDominators(fun)
}

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

// Use Lengauer-Tarjan Algorithm to build domination tree.
// This version has time complexity O(NlogN), not optimal but easier to understand.
func (o *SSAOpt) computeDominators(fun *Func) {
	// Build depth first tree
	nodes := make([]*DFTreeNode, 0)
	bbToNode := make(map[*BasicBlock]*DFTreeNode) // maps vertices in CFG to nodes in DF tree
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
			// Predecessors of vertices are visited. If a predecessor is unreachable, it will not be
			// visited in DFS, and it cannot be found in the map.
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
