package ir

import (
	"fmt"
	"io"
)

type Printer struct {
	BaseVisitor
	writer io.Writer
}

func NewPrinter(out io.Writer) *Printer {
	return &Printer{
		writer: out,
	}
}

func (p *Printer) write(format string, args ...interface{}) {
	_, err := fmt.Fprintf(p.writer, format, args...)
	if err != nil {
		panic(NewIRError(fmt.Sprintf("error writing IR: %s", err.Error())))
	}
}

func valueListStr(list []IValue) string {
	str := ""
	for i, v := range list {
		if i != 0 {
			str += ", "
		}
		str += v.ToString()
	}
	return str
}

func (p *Printer) VisitProgram(prg *Program) interface{} {
	p.write("program %s\n\n", prg.Name)
	p.VisitScope(prg.Global)
	for _, f := range prg.Funcs {
		p.VisitFunc(f)
	}
	return nil
}

func (p *Printer) VisitScope(scope *Scope) interface{} {
	for _, s := range scope.Symbols {
		p.write("%s: %s\n", s.Name, s.Type.ToString())
	}
	return nil
}

func (p *Printer) VisitFunc(fun *Func) interface{} {
	// Print function signature
	p.write("func %s(", fun.Name)
	for i, param := range fun.Scope.Params {
		if i != 0 {
			p.write(", ")
		}
		p.write("%s: %s", param.Name, param.Type.ToString())
	}
	p.write("): \n")

	// Print basic blocks with pre-order traversal
	stack := []*BasicBlock{fun.Enter}
	visited := make(map[*BasicBlock]bool)
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1] // pop an element from stack
		if visited[top] {
			continue
		}
		p.VisitBasicBlock(top)
		visited[top] = true // mark as visited
		for _, bb := range top.Exit {
			stack = append(stack, bb)
		}
	}
	p.write("\n")

	return nil
}

func (p *Printer) VisitBasicBlock(bb *BasicBlock) interface{} {
	p.write("%s:\n", bb.Name)
	iter := NewInstrIterFromBlock(bb)
	for iter.IsValid() {
		p.VisitInstr(iter.Cur)
		iter.Next()
	}
	return nil
}

func (p *Printer) VisitInstr(instr IInstr) interface{} {
	switch instr.(type) {
	case *Move:
		p.VisitMove(instr.(*Move))
	case *Load:
		p.VisitLoad(instr.(*Load))
	case *Store:
		p.VisitStore(instr.(*Store))
	case *Malloc:
		p.VisitMalloc(instr.(*Malloc))
	case *GetPtr:
		p.VisitGetPtr(instr.(*GetPtr))
	case *PtrOffset:
		p.VisitPtrOffset(instr.(*PtrOffset))
	case *Clear:
		p.VisitClear(instr.(*Clear))
	case *Unary:
		p.VisitUnary(instr.(*Unary))
	case *Binary:
		p.VisitBinary(instr.(*Binary))
	case *Jump:
		p.VisitJump(instr.(*Jump))
	case *Branch:
		p.VisitBranch(instr.(*Branch))
	case *Call:
		p.VisitCall(instr.(*Call))
	case *Return:
		p.VisitReturn(instr.(*Return))
	}
	return nil
}

func (p *Printer) writeInstr(op string, dst IValue, src ...IValue) {
	p.write("\t%s ", op)
	for i, v := range src {
		if i != 0 {
			p.write(", ")
		}
		p.write(v.ToString())
	}
	p.write(" => %s\n", dst.ToString())
}

func (p *Printer) VisitMove(instr *Move) interface{} {
	p.writeInstr("move", instr.Dst, instr.Src)
	return nil
}

func (p *Printer) VisitLoad(instr *Load) interface{} {
	p.writeInstr("load", instr.Dst, instr.Src)
	return nil
}

func (p *Printer) VisitStore(instr *Store) interface{} {
	p.writeInstr("store", instr.Dst, instr.Src)
	return nil
}

func (p *Printer) VisitMalloc(instr *Malloc) interface{} {
	p.writeInstr("malloc", instr.Return, NewI64Imm(instr.Size))
	return nil
}

func (p *Printer) VisitGetPtr(instr *GetPtr) interface{} {
	p.write("\tgetptr %s [%s] => %s\n", instr.Base.ToString(),
		valueListStr(instr.Indices), instr.Result.ToString())
	return nil
}

func (p *Printer) VisitPtrOffset(instr *PtrOffset) interface{} {
	p.writeInstr("ptroff", instr.Dst, instr.Src, NewI64Imm(instr.Offset))
	return nil
}

func (p *Printer) VisitClear(instr *Clear) interface{} {
	p.write("\tclear %s\n", instr.Value.ToString())
	return nil
}

var unaryOpStr = map[UnaryOp]string{
	NEG: "neg", NOT: "not",
}

func (p *Printer) VisitUnary(instr *Unary) interface{} {
	p.writeInstr(unaryOpStr[instr.Op], instr.Operand, instr.Result)
	return nil
}

var binaryOpStr = map[BinaryOp]string{
	ADD: "add", SUB: "sub", MUL: "mul", DIV: "div", AND: "and", OR: "or", XOR: "xor",
	SHL: "shl", SHR: "shr", EQ: "eq", NE: "ne", LT: "lt", LE: "le", GT: "gt", GE: "ge",
}

func (p *Printer) VisitBinary(instr *Binary) interface{} {
	p.writeInstr(binaryOpStr[instr.Op], instr.Result, instr.Left, instr.Right)
	return nil
}

func (p *Printer) VisitJump(instr *Jump) interface{} {
	p.write("\tjump %s\n", instr.BB.Name)
	return nil
}

func (p *Printer) VisitBranch(instr *Branch) interface{} {
	p.write("\tbranch %s ? %s : %s\n", instr.Cond.ToString(), instr.True.Name,
		instr.False.Name)
	return nil
}

func (p *Printer) VisitCall(instr *Call) interface{} {
	p.write("\tcall %s(%s) => %s\n", instr.Func.ToString(), valueListStr(instr.Args),
		instr.Ret.ToString())
	return nil
}

func (p *Printer) VisitReturn(instr *Return) interface{} {
	p.write("\treturn %s\n", valueListStr(instr.Values))
	return nil
}
