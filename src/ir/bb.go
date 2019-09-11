package ir

import "fmt"

type BasicBlock struct {
	// Block label
	Name string
	// The function this block lies in.
	Func *Func
	// Internally. a block is a linked list of instructions.
	Head, Tail IInstr
	// In a function, the basic blocks form a control flow graph.
	// This set can be constructed when instructions are added to the function.
	Pred, Succ map[*BasicBlock]bool
	// Dominance tree can be constructed from CFG.
	ImmDom   *BasicBlock          // immediate dominator of this block
	Children map[*BasicBlock]bool // blocks that this immediately dominates
	serial   [2]int               // pre-order serial [in, out] that determine dominance
}

func NewBasicBlock(label string, fun *Func) *BasicBlock {
	return &BasicBlock{
		Name:     label,
		Func:     fun,
		Head:     nil, // initially an empty list
		Tail:     nil,
		Pred:     make(map[*BasicBlock]bool),
		Succ:     make(map[*BasicBlock]bool),
		Children: make(map[*BasicBlock]bool),
	}
}

func (b *BasicBlock) PushFront(instr IInstr) {
	instr.SetBasicBlock(b)
	if b.Head == nil {
		b.Head, b.Tail = instr, instr
		return
	}
	instr.SetNext(b.Head) // instr -> prev_head
	b.Head.SetPrev(instr) // instr <-> prev_head
	b.Head = instr        // head -> instr
}

// Add instruction to the tail of linked list.
func (b *BasicBlock) PushBack(instr IInstr) {
	switch b.Tail.(type) {
	case *Branch, *Jump:
		panic(NewIRError(fmt.Sprintf(
			"cannot add to block %s ended with jump or branch instruction", b.Name),
		))
	}
	instr.SetBasicBlock(b)
	if b.Head == nil {
		b.Head, b.Tail = instr, instr
		return
	}
	instr.SetPrev(b.Tail) // prev_tail <- instr
	b.Tail.SetNext(instr) // prev_tail <-> instr
	b.Tail = instr        // instr <- tail
}

// Automatically create a jump instruction in the receiver block, and create edges in
// two blocks.
func (b *BasicBlock) JumpTo(b2 *BasicBlock) {
	b.Succ = make(map[*BasicBlock]bool) // clear successors
	instr := NewJump(b2)
	b.PushBack(instr)
	b.ConnectTo(b2)
}

func (b *BasicBlock) BranchTo(cond IValue, trueBB, falseBB *BasicBlock) {
	b.Succ = make(map[*BasicBlock]bool) // clear successors
	instr := NewBranch(cond, trueBB, falseBB)
	b.PushBack(instr)
	b.ConnectTo(trueBB)
	b.ConnectTo(falseBB)
}

func (b *BasicBlock) Terminates() bool {
	switch b.Tail.(type) {
	case *Jump, *Branch, *Return:
		return true
	default:
		return false
	}
}

// This method only modifies the predecessor and successor set, and has nothing to do with
// instructions in the blocks.
func (b *BasicBlock) ConnectTo(to *BasicBlock) {
	b.Succ[to] = true
	to.Pred[b] = true
}

func (b *BasicBlock) DisconnectTo(to *BasicBlock) {
	delete(b.Succ, to)
	delete(to.Pred, b)
}

// Change successor to a new block, and modify target in jump or branch instruction
func (b *BasicBlock) ChangeSuccTo(prev, new *BasicBlock) {
	if b.Succ[prev] == false {
		panic(NewIRError(fmt.Sprintf("%s is not successor of %s", b.Name, new.Name)))
	}
	b.DisconnectTo(prev) // predecessor -X- successor
	b.ConnectTo(new)     // predecessor <-> inserted
	switch b.Tail.(type) {
	case *Jump:
		tail := b.Tail.(*Jump)
		tail.Target = new
	case *Branch:
		tail := b.Tail.(*Branch)
		if tail.True == prev {
			tail.True = new
		} else if tail.False == prev {
			tail.False = new
		}
	}
}

func (b *BasicBlock) SplitEdgeTo(to, inserted *BasicBlock) {
	b.ChangeSuccTo(to, inserted)
	inserted.JumpTo(to) // inserted <-> successor
}

type GraphTrav int

const (
	DepthFirst GraphTrav = iota
	BreadthFirst
	PostOrder
	ReversePostOrder
)

// Accept current basic block as vertex in a graph
func (b *BasicBlock) AcceptAsVert(action func(*BasicBlock), method GraphTrav) {
	switch method {
	case DepthFirst:
		b.depthFirst(action)
	case BreadthFirst:
		b.breadthFirst(action)
	case PostOrder:
		b.postOrder(action)
	case ReversePostOrder:
		b.reversePostOrder(action)
	}
}

func (b *BasicBlock) depthFirst(action func(*BasicBlock)) {
	stack := []*BasicBlock{b}
	visited := make(map[*BasicBlock]bool)
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1] // pop an element
		if visited[top] {
			continue
		}
		action(top)
		visited[top] = true
		for bb := range top.Succ {
			stack = append(stack, bb)
		}
	}
}

func (b *BasicBlock) breadthFirst(action func(*BasicBlock)) {
	queue := []*BasicBlock{b}
	visited := make(map[*BasicBlock]bool)
	for len(queue) > 0 {
		top := queue[0]
		queue = queue[1:] // dequeue an element
		if visited[top] {
			continue
		}
		action(top)
		visited[top] = true // mark as visited
		for bb := range top.Succ {
			queue = append(queue, bb)
		}
	}
}

func (b *BasicBlock) postOrder(action func(*BasicBlock)) {
	visited := make(map[*BasicBlock]bool)
	var visit func(*BasicBlock)
	visit = func(block *BasicBlock) {
		if visited[block] {
			return
		}
		visited[block] = true
		for succ := range block.Succ {
			visit(succ)
		}
		action(block)
	}
	visit(b)
}

func (b *BasicBlock) reversePostOrder(action func(*BasicBlock)) {
	postOrder := make([]*BasicBlock, 0)
	b.postOrder(func(block *BasicBlock) {
		postOrder = append(postOrder, block)
	})
	for i := len(postOrder) - 1; i >= 0; i-- {
		action(postOrder[i])
	}
}

func (b *BasicBlock) SetImmDom(b2 *BasicBlock) {
	b.ImmDom = b2
	b2.Children[b] = true
}

func (b *BasicBlock) AcceptAsTreeNode(pre, post func(*BasicBlock)) {
	pre(b)
	for child := range b.Children {
		child.AcceptAsTreeNode(pre, post)
	}
	post(b)
}

func (b *BasicBlock) PrintDomTree() {
	depth := 0
	b.AcceptAsTreeNode(func(block *BasicBlock) {
		for i := 0; i < depth; i++ {
			fmt.Print("\t")
		}
		fmt.Println(block.Name)
		depth++
	}, func(block *BasicBlock) {
		depth--
	})
	fmt.Print("\n")
}

// Number dominance with serials to enable O(1) parent-child judgement
func (b *BasicBlock) NumberDomTree() {
	serial := 0
	b.AcceptAsTreeNode(func(block *BasicBlock) {
		block.serial[0] = serial
		serial++
	}, func(block *BasicBlock) {
		block.serial[1] = serial
		serial++
	})
}

func (b *BasicBlock) Dominates(b2 *BasicBlock) bool {
	return b.serial[0] < b2.serial[0] && b.serial[1] > b2.serial[1]
}
