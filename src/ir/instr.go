package ir

import "fmt"

type IInstr interface {
	GetPrev() IInstr
	SetPrev(instr IInstr)
	GetNext() IInstr
	SetNext(instr IInstr)
	GetBasicBlock() *BasicBlock
}

type InstrIter struct {
	Cur IInstr
	BB  *BasicBlock
}

// Constructed from a existing instruction
func NewInstrIter(instr IInstr) *InstrIter {
	if instr == nil {
		panic(fmt.Errorf("cannot construct iterator from a nil instruction"))
	}
	return &InstrIter{
		Cur: instr,
		BB:  instr.GetBasicBlock(),
	}
}

// Iterate from the first instruction of basic block
func NewInstrIterFromBlock(bb *BasicBlock) *InstrIter {
	return &InstrIter{
		Cur: bb.Head,
		BB:  bb,
	}
}

func (i *InstrIter) Valid() bool { return i.Cur != nil }

func (i *InstrIter) Next() { i.Cur = i.Cur.GetNext() }

func (i *InstrIter) Prev() { i.Cur = i.Cur.GetPrev() }

func (i *InstrIter) HasNext() bool { return i.Cur.GetNext() != nil }

func (i *InstrIter) HasPrev() bool { return i.Cur.GetPrev() != nil }

func (i *InstrIter) InsertBefore(instr IInstr) {
	prev := i.Cur.GetPrev()
	cur := i.Cur
	if cur == nil { // empty block
		i.Cur = instr
		i.BB.Head = instr
		i.BB.Tail = instr
		return
	}
	if prev != nil { // cur is not the first node
		// prev -> instr - cur
		prev.SetNext(instr)
	} else {
		// head -> instr
		i.BB.Head = instr
	}
	// prev <-> instr - cur
	instr.SetPrev(prev)
	// instr <- cur
	cur.SetPrev(instr)
	// instr <-> cur
	instr.SetNext(cur)
}

func (i *InstrIter) InsertAfter(instr IInstr) {
	next := i.Cur.GetNext()
	cur := i.Cur
	if cur == nil { // empty block
		i.Cur = instr
		i.BB.Head = instr
		i.BB.Tail = instr
		return
	}
	if next != nil { // cur is not the last node
		// cur - instr <- next
		next.SetPrev(instr)
	} else {
		// instr <- tail
		i.BB.Tail = instr
	}
	// cur - instr <-> next
	instr.SetNext(next)
	// cur -> instr
	cur.SetNext(instr)
	// cur <-> instr
	instr.SetPrev(cur)
}

func (i *InstrIter) Remove() {
	if i.Cur == nil {
		return // no instruction to remove
	}
	prev, next := i.Cur.GetPrev(), i.Cur.GetNext()
	if prev != nil { // not the first node
		prev.SetNext(next)
	} else {
		i.BB.Head = next
	}
	if next != nil { // not the last node
		next.SetPrev(prev)
	} else {
		i.BB.Tail = prev
	}
}

type BaseInstr struct {
	// an instruction also serves as a node in the linked list of a basic block
	Prev, Next IInstr
	// the basic block that this instruction lies in
	BB *BasicBlock
}

func NewBaseInstr(bb *BasicBlock) *BaseInstr {
	return &BaseInstr{BB: bb}
}

func (i *BaseInstr) GetPrev() IInstr { return i.Prev }

func (i *BaseInstr) SetPrev(instr IInstr) { i.Prev = instr }

func (i *BaseInstr) GetNext() IInstr { return i.Next }

func (i *BaseInstr) SetNext(instr IInstr) { i.Next = instr }

func (i *BaseInstr) GetBasicBlock() *BasicBlock { return i.BB }
