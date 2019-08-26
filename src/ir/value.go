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
}

func NewSymbolValue(symbol *Symbol) *SymbolValue {
	return &SymbolValue{
		BaseValue: *NewBaseValue(symbol.Type),
		Symbol:    symbol,
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

func NewNullPtr() *ImmValue {
	return &ImmValue{
		BaseValue: *NewBaseValue(NewPtrType(NewBaseType(Void))),
		Value:     nil,
	}
}

// IR version of function, directly callable.
type Func struct {
	BaseValue
	// Function label
	Name string
	// A function has one entrance block, but can have several exit blocks
	Enter *BasicBlock
	Exit  []*BasicBlock
	// Base scope of current function, may have nested scopes
	Scope *Scope
}

func NewFunc(tp *FuncType, name string, scope *Scope) *Func {
	return &Func{
		BaseValue: *NewBaseValue(tp),
		Name:      name,
		Enter:     nil,                    // to be assigned later
		Exit:      make([]*BasicBlock, 0), // to be assigned later
		Scope:     scope,
	}
}

func (f *Func) AddExitBlock(exit *BasicBlock) { f.Exit = append(f.Exit, exit) }
