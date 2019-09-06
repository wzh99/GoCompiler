package ir

import (
	"ast"
	"fmt"
)

// Similar to the one in AST, but with simplified data
type Symbol struct {
	Name  string
	Ver   int // version index to support variable renaming
	Type  IType
	Scope *Scope // the scope this symbol is defined
	Param bool   // indicate whether this symbol is parameter
}

func (s *Symbol) Rename() *Symbol {
	sym := &Symbol{
		Name:  s.Name,
		Ver:   s.Ver + 1,
		Type:  s.Type,
		Scope: s.Scope,
		Param: false,
	}
	s.Scope.Symbols[sym] = true
	return sym
}

func (s *Symbol) ToString() string {
	var scopeTag byte
	if s.Scope.Global {
		scopeTag = '@'
	} else {
		scopeTag = '$'
	}
	return fmt.Sprintf("%c%s.%d", scopeTag, s.Name, s.Ver)
}

// Every function has only one scope. No nested scopes.
type Scope struct {
	// Linear list of entries
	Symbols map[*Symbol]bool
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
		Symbols:  make(map[*Symbol]bool),
		Params:   make([]*Symbol, 0),
		ParamIdx: make(map[*Symbol]int),
		Global:   true,
		astToIr:  make(map[*ast.Symbol]*Symbol),
	}
}

func NewLocalScope() *Scope {
	return &Scope{
		Symbols:  make(map[*Symbol]bool),
		Params:   make([]*Symbol, 0),
		ParamIdx: make(map[*Symbol]int),
		Global:   false,
		astToIr:  make(map[*ast.Symbol]*Symbol),
	}
}

func (s *Scope) AddSymbol(name string, tp IType, param bool) *Symbol {
	irSym := &Symbol{
		Name:  name,
		Ver:   0,
		Type:  tp,
		Scope: s,
		Param: param,
	}
	s.Symbols[irSym] = true
	if param {
		index := len(s.Params)
		s.Params = append(s.Params, irSym)
		s.ParamIdx[irSym] = index
	}
	return irSym
}

func (s *Scope) AddFromAST(astSym *ast.Symbol, name string, tp IType, param bool) *Symbol {
	irSym := s.AddSymbol(name, tp, param)
	s.astToIr[astSym] = irSym
	return irSym
}

func (s *Scope) AddTemp(name string, tp IType) *Symbol {
	return s.AddSymbol(name, tp, false)
}
