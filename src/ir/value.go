package ir

type IValue interface {
	GetType() IType
}

type BaseValue struct {
	Type IType
}

func NewBaseValue(tp IType) *BaseValue {
	return &BaseValue{Type: tp}
}

func (v *BaseValue) GetType() IType { return v.Type }

// Values stored in symbol table of scopes
type SymbolValue struct {
	BaseValue
	Symbol *Symbol
	Scope  *Scope
}

func NewSymbolValue(symbol *Symbol, scope *Scope) *SymbolValue {
	return &SymbolValue{
		BaseValue: *NewBaseValue(symbol.Type),
		Symbol:    symbol,
		Scope:     scope,
	}
}

// Immediate values, refer to constants in AST
type ImmValue struct {
	BaseValue
	Value interface{}
}

func NewI1Imm(value bool) *ImmValue {
	return &ImmValue{
		BaseValue: *NewBaseValue(NewBaseType(I1)),
		Value:     value,
	}
}

func NewI64Imm(value int) *ImmValue {
	return &ImmValue{
		BaseValue: *NewBaseValue(NewBaseType(I64)),
		Value:     value,
	}
}

func NewF64Imm(value float64) *ImmValue {
	return &ImmValue{
		BaseValue: *NewBaseValue(NewBaseType(F64)),
		Value:     value,
	}
}

// IR version of function, directly callable.
type Func struct {
	BaseValue
	// Function label
	Label string
	// A function has only one begin block and one end block
	Begin, End *BasicBlock
	// Base scope of current function, may have nested scopes
	Scope *Scope
}

func NewFunc(tp *FuncType, label string, scope *Scope) *Func {
	return &Func{
		BaseValue: *NewBaseValue(tp),
		Label:     label,
		Begin:     nil, // to be assigned later
		End:       nil, // to be assigned later
		Scope:     scope,
	}
}
