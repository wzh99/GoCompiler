package ir

import (
	"fmt"
)

type IValue interface {
	GetType() IType
	ToString() string
}

type BaseValue struct {
	Type IType
}

func NewBaseValue(tp IType) *BaseValue {
	return &BaseValue{Type: tp}
}

func (v *BaseValue) GetType() IType { return v.Type }

func (v *BaseValue) ToString() string { return "" }

// Variables, whose related symbols should be stored in scopes
type Variable struct {
	BaseValue
	Symbol *Symbol
}

func NewVariable(symbol *Symbol) *Variable {
	return &Variable{
		BaseValue: *NewBaseValue(symbol.Type),
		Symbol:    symbol,
	}
}

func (v *Variable) ToString() string {
	return fmt.Sprintf("%s: %s", v.Symbol.ToString(), v.Type.ToString())
}

// Immediate values, refer to constants in AST
type Immediate struct {
	BaseValue
	Value interface{}
}

func NewI1Imm(value bool) *Immediate {
	return &Immediate{
		BaseValue: *NewBaseValue(NewBaseType(I1)),
		Value:     value,
	}
}

func NewI64Imm(value int) *Immediate {
	return &Immediate{
		BaseValue: *NewBaseValue(NewBaseType(I64)),
		Value:     value,
	}
}

func NewF64Imm(value float64) *Immediate {
	return &Immediate{
		BaseValue: *NewBaseValue(NewBaseType(F64)),
		Value:     value,
	}
}

func NewNullPtr() *Immediate {
	return &Immediate{
		BaseValue: *NewBaseValue(NewPtrType(NewBaseType(Void))),
		Value:     nil,
	}
}

func (v *Immediate) ToString() string {
	switch v.Type.GetTypeEnum() {
	case I1:
		val := v.Value.(bool)
		if val {
			return "1: i1"
		} else {
			return "0: i1"
		}
	case I64:
		return fmt.Sprintf("%d: i64", v.Value.(int))
	case F64:
		return fmt.Sprintf("%f: f64", v.Value.(float64))
	case Pointer:
		return "nullptr"
	}
	return ""
}

// IR version of function, directly callable.
type Func struct {
	BaseValue
	// Function label
	Name string
	// A function has one entrance block, but can have several exit blocks
	Enter *BasicBlock
	Exit  map[*BasicBlock]bool
	// Base scope of current function, may have nested scopes
	Scope *Scope
	// Temporary variables count
	nTmp int
}

func NewFunc(tp *FuncType, name string, scope *Scope) *Func {
	return &Func{
		BaseValue: *NewBaseValue(tp),
		Name:      name,
		Enter:     nil,                        // to be assigned later
		Exit:      make(map[*BasicBlock]bool), // to be assigned later
		Scope:     scope,
		nTmp:      0,
	}
}

func (f *Func) ToString() string { return f.Name }
