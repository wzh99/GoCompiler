package ir

import "ast"

// Similar to the one in AST, but with simplified data
type Symbol struct {
	Name  string
	Type  IType
	Param bool // indicate whether this symbol is parameter
}

// Every function has only one scope. No nested scopes.
type Scope struct {
	// Linear list of entries
	Symbols []*Symbol
	// Look up table that helps querying
	table map[string]*Symbol
	// Parameters should be taken special care
	Params []*Symbol
	// Map parameters back to its index in parameter list
	ParamIdx map[*Symbol]int
	// Indicate whether this scope is global
	Global bool
	// Transform AST symbol to IR symbol
	astToIr map[*ast.Symbol]*Symbol
}

func NewGlobalScope() *Scope {
	return &Scope{
		Symbols:  make([]*Symbol, 0),
		table:    make(map[string]*Symbol),
		Params:   make([]*Symbol, 0),
		ParamIdx: make(map[*Symbol]int),
		Global:   true,
		astToIr:  make(map[*ast.Symbol]*Symbol),
	}
}

func NewLocalScope() *Scope {
	return &Scope{
		Symbols:  make([]*Symbol, 0),
		table:    make(map[string]*Symbol),
		Params:   make([]*Symbol, 0),
		ParamIdx: make(map[*Symbol]int),
		Global:   false,
		astToIr:  make(map[*ast.Symbol]*Symbol),
	}
}

func (s *Scope) AddSymbol(name string, tp IType, param bool) *Symbol {
	irSym := &Symbol{
		Name:  name,
		Type:  tp,
		Param: param,
	}
	s.Symbols = append(s.Symbols, irSym)
	s.table[name] = irSym
	if param {
		index := len(s.Params)
		s.Params = append(s.Params, irSym)
		s.ParamIdx[irSym] = index
	}
	return irSym
}

func (s *Scope) AddSymbolFromAST(astSym *ast.Symbol, name string, tp IType, param bool) *Symbol {
	irSym := s.AddSymbol(name, tp, param)
	s.astToIr[astSym] = irSym
	return irSym
}

func (s *Scope) AddTempSymbol(name string, tp IType) *Symbol {
	sym := &Symbol{
		Name:  name,
		Type:  tp,
		Param: false,
	}
	s.Symbols = append(s.Symbols, sym)
	s.table[name] = sym
	return sym
}
