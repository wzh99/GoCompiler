package ir

import "fmt"

// Partial Redundancy Elimination in SSA Form by Chow et al. [1999].
type PREOpt struct {
	prg *Program
	fun *Func
}

func NewPREOpt(prg *Program) *PREOpt {
	return &PREOpt{prg: prg}
}

// SSAPRE operates on lexically identified expressions.
type LexIdentExpr struct {
	op  string
	opd [2]string
}

func opdToStr(val IValue) string {
	switch val.(type) {
	case *Immediate:
		imm := val.(*Immediate)
		switch imm.Type.GetTypeEnum() {
		case I1:
			val := imm.Value.(bool)
			if val {
				return "1"
			} else {
				return "0"
			}
		case I64:
			return fmt.Sprintf("%d", imm.Value.(int))
		case F64:
			return fmt.Sprintf("%f", imm.Value.(float64))
		}
	case *Variable:
		sym := val.(*Variable).Symbol
		return sym.Name // the SSA versions of the variables are ignored.
	}
	return ""
}

func newLexIdentExpr(instr IInstr) *LexIdentExpr {
	switch instr.(type) {
	case *Unary:
		unary := instr.(*Unary)
		return &LexIdentExpr{
			op:  unaryOpStr[unary.Op],
			opd: [2]string{opdToStr(unary.Result)},
		}
	case *Binary:
		binary := instr.(*Binary)
		expr := &LexIdentExpr{
			op:  binaryOpStr[binary.Op],
			opd: [2]string{opdToStr(binary.Left), opdToStr(binary.Right)},
		}
		// enforce an order for commutative binary operators
		if commutative[binary.Op] && expr.opd[0] > expr.opd[1] {
			expr.opd = [2]string{expr.opd[1], expr.opd[0]}
		}
		return expr
	default:
		return nil // other instructions not considered
	}
}

type IOccur interface {
	getPrev() IOccur
	setPrev(occur IOccur)
	getNext() IOccur
	setNext(occur IOccur)
	getVersion() int
	setVersion(ver int)
}

type BaseOccur struct {
	prev, next IOccur // as linked list node
	version    int
}

func (o *BaseOccur) getPrev() IOccur { return o.prev }

func (o *BaseOccur) setPrev(occur IOccur) { o.prev = occur }

func (o *BaseOccur) getNext() IOccur { return o.next }

func (o *BaseOccur) setNext(occur IOccur) { o.next = occur }

func (o *BaseOccur) getVersion() int { return o.version }

func (o *BaseOccur) setVersion(ver int) { o.version = ver }

type RealOccur struct {
	BaseOccur
	instr IInstr
}

func newRealOccur(instr IInstr) *RealOccur {
	return &RealOccur{
		BaseOccur: BaseOccur{
			prev:    nil,
			next:    nil,
			version: 0,
		},
		instr: instr,
	}
}

// Redundancy factoring operator
type BigPhi struct {
	BaseOccur
	bbToOccur map[*BasicBlock]IOccur
}

func newBigPhi(bbToOccur map[*BasicBlock]IOccur) *BigPhi {
	return &BigPhi{
		BaseOccur: BaseOccur{
			prev:    nil,
			next:    nil,
			version: 0,
		},
		bbToOccur: bbToOccur,
	}
}

type BlockOccur struct {
	head, tail IOccur
}

// Worklist driven PRE, see Section 5.1 of paper
func (o *PREOpt) Optimize(fun *Func) {
	// Initialize worklist
	o.fun = fun
	workList := make(map[LexIdentExpr]bool)
	removeOneExpr := func() LexIdentExpr {
		var expr LexIdentExpr
		for e := range workList {
			expr = e
			break
		}
		delete(workList, expr)
		return expr
	}

	// Collect occurrences of expressions

	// Eliminate redundancies iteratively
	for len(workList) > 0 {
		// Construct factored redundancy graph (FRG)
		expr := new(LexIdentExpr)
		*expr = removeOneExpr()
		// Perform backward and forward data flow propagation
		// Pinpoint locations for computations to be inserted
		// Transform code to form optimized program
	}
}
