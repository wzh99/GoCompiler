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
	if len(scope.Symbols) == 0 {
		return nil
	}
	for s := range scope.Symbols {
		p.write("%s: %s\n", s.ToString(), s.Type.ToString())
	}
	p.write("\n")
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

	// Print basic blocks with BFS
	fun.Enter.AcceptAsVert(func(block *BasicBlock) {
		p.VisitBasicBlock(block)
	}, BreadthFirst)
	p.write("\n")

	return nil
}

func (p *Printer) VisitBasicBlock(bb *BasicBlock) interface{} {
	p.write("%s:\n", bb.Name)
	iter := NewIterFromBlock(bb)
	for iter.Valid() {
		p.VisitInstr(iter.Get())
		iter.MoveNext()
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
	case *Phi:
		p.VisitPhi(instr.(*Phi))
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
	p.writeInstr("malloc", instr.Result, NewI64Const(instr.Result.GetType().GetSize()))
	return nil
}

func (p *Printer) VisitGetPtr(instr *GetPtr) interface{} {
	p.write("\tgetptr %s [%s] => %s\n", instr.Base.ToString(),
		valueListStr(instr.Indices), instr.Result.ToString())
	return nil
}

func (p *Printer) VisitPtrOffset(instr *PtrOffset) interface{} {
	p.writeInstr("ptroff", instr.Dst, instr.Src, NewI64Const(instr.Offset))
	return nil
}

func (p *Printer) VisitClear(instr *Clear) interface{} {
	p.write("\tclear %s\n", instr.Value.ToString())
	return nil
}

func (p *Printer) VisitUnary(instr *Unary) interface{} {
	p.writeInstr(unaryOpStr[instr.Op], instr.Result, instr.Operand)
	return nil
}

func (p *Printer) VisitBinary(instr *Binary) interface{} {
	p.writeInstr(binaryOpStr[instr.Op], instr.Result, instr.Left, instr.Right)
	return nil
}

func (p *Printer) VisitJump(instr *Jump) interface{} {
	p.write("\tjump %s\n", instr.Target.Name)
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

func (p *Printer) VisitPhi(instr *Phi) interface{} {
	p.write("\tphi")
	i := 0
	for bb, val := range instr.BBToOpd {
		p.write(" [%s, %s]", bb.Name, (*val).ToString())
		i++
	}
	p.write(" => %s\n", instr.Result.ToString())
	return nil
}
