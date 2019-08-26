package ir

import "ast"

// Similar to the one in AST, but with simplified data
type Symbol struct {
	Name string
	Type IType
}

// Every function has only one scope. No nested scopes.
type Scope struct {
	// Linear list of entries
	Entries []*Symbol
	// Look up table that helps querying
	table map[string]*Symbol
	// Indicate whether this scope is global
	Global bool
	// Transform AST symbol to IR symbol
	astToIr map[*ast.Symbol]*Symbol
}

func NewGlobalScope() *Scope {
	return &Scope{
		Entries: make([]*Symbol, 0),
		table:   make(map[string]*Symbol),
		Global:  true,
		astToIr: make(map[*ast.Symbol]*Symbol),
	}
}

func NewLocalScope() *Scope {
	return &Scope{
		Entries: make([]*Symbol, 0),
		table:   make(map[string]*Symbol),
		Global:  false,
		astToIr: make(map[*ast.Symbol]*Symbol),
	}
}

func (s *Scope) AddSymbolFromAST(astSym *ast.Symbol, name string, tp IType) *Symbol {
	irSym := &Symbol{
		Name: name,
		Type: tp,
	}
	s.Entries = append(s.Entries, irSym)
	s.table[name] = irSym
	s.astToIr[astSym] = irSym
	return irSym
}

func (s *Scope) AddTempSymbol(name string, tp IType) *Symbol {
	sym := &Symbol{
		Name: name,
		Type: tp,
	}
	s.Entries = append(s.Entries, sym)
	s.table[name] = sym
	return sym
}
