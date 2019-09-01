package ir

import "fmt"

type BasicBlock struct {
	// Block label
	Name string
	// Internally. a block is a linked list of instructions.
	Head, Tail IInstr
	// In a function, the basic blocks form a control flow graph.
	// This set can be constructed when instructions are added to the function.
	Pred, Succ map[*BasicBlock]bool
	// The function this block lies in.
	Func *Func
}

func NewBasicBlock(label string, fun *Func) *BasicBlock {
	return &BasicBlock{
		Name: label,
		Head: nil, // initially an empty list
		Tail: nil,
		Pred: make(map[*BasicBlock]bool),
		Succ: make(map[*BasicBlock]bool),
		Func: fun,
	}
}

func (b *BasicBlock) PushFront(instr IInstr) {
	instr.SetBasicBlock(b)
	if b.Head == nil {
		b.Head, b.Tail = instr, instr
		return
	}
	instr.SetNext(b.Head) // instr -> prev_head
	b.Head.SetNext(instr) // instr <-> prev_head
	b.Head = instr        // head -> instr
}

// Add instruction to the tail of linked list.
func (b *BasicBlock) PushBack(instr IInstr) {
	switch b.Tail.(type) {
	case *Branch, *Jump:
		panic(NewIRError(
			fmt.Sprintf("cannot add to block %s ended with jump or branch instruction",
				b.Name),
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
	instr := NewJump(b2)
	b.PushBack(instr)
	b.ConnectTo(b2)
}

func (b *BasicBlock) BranchTo(cond IValue, trueBB, falseBB *BasicBlock) {
	instr := NewBranch(cond, trueBB, falseBB)
	b.PushBack(instr)
	b.ConnectTo(trueBB)
	b.ConnectTo(falseBB)
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

func (b *BasicBlock) SplitEdgeTo(to, inserted *BasicBlock) { // b: predecessor, to: successor
	if b.Succ[to] == false {
		panic(NewIRError(fmt.Sprintf("%s is not successor of %s", b.Name, to.Name)))
	}
	b.DisconnectTo(to)  // predecessor -X- successor
	inserted.JumpTo(to) // inserted <-> successor
	switch b.Tail.(type) {
	case *Jump:
		tail := b.Tail.(*Jump)
		tail.Target = inserted
	case *Branch:
		tail := b.Tail.(*Branch)
		if tail.True == to {
			tail.True = to
		} else if tail.False == to {
			tail.False = to
		}
	}
	b.ConnectTo(inserted) // predecessor <-> inserted
}

type Traversal int

const (
	DepthFirst Traversal = iota
	BreadthFirst
)

func (b *BasicBlock) Accept(action func(*BasicBlock), method Traversal) {
	switch method {
	case DepthFirst:
		b.depthFirst(action)
	case BreadthFirst:
		b.breadthFirst(action)
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
