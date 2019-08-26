package ast

import "fmt"

type Scope struct {
	Parent   *Scope
	Children []*Scope
	Symbols  *SymbolTable
	Global   bool
	// point back to the function this scope belongs to
	Func *FuncDecl
	// all operand identifiers mentioned in statements, which helps closure construction
	operandId map[*IdExpr]bool
}

func NewGlobalScope() *Scope {
	return &Scope{Symbols: NewSymbolTable(),
		Parent:    nil,
		Children:  make([]*Scope, 0),
		Global:    true,
		operandId: make(map[*IdExpr]bool),
	}
}

func NewLocalScope(parent *Scope) *Scope {
	s := &Scope{
		Symbols:   NewSymbolTable(),
		Parent:    parent,
		Children:  make([]*Scope, 0),
		Global:    false,
		Func:      parent.Func,
		operandId: make(map[*IdExpr]bool),
	}
	parent.AddChild(s)
	return s
}

func (s *Scope) AddChild(child *Scope) {
	s.Children = append(s.Children, child)
}

func (s *Scope) AddSymbol(symbol *Symbol) {
	// Reject unnamed symbol
	if len(symbol.Name) == 0 {
		panic(fmt.Errorf("%s unnamed symbol", symbol.Loc.ToString()))
	}
	// Reject redefined symbol
	if s.CheckDefined(symbol.Name) {
		panic(fmt.Errorf("%s redefined symbol: %s", symbol.Loc.ToString(), symbol.Name))
	}
	symbol.Scope = s
	s.Symbols.Add(symbol)
}

func (s *Scope) AddOperandId(id *IdExpr) {
	s.operandId[id] = true
}

// Look up symbol, considering nested scopes
func (s *Scope) Lookup(name string) (entry *Symbol, scope *Scope) {
	scope = s
	for scope != nil {
		entry = scope.Symbols.Lookup(name)
		if entry != nil {
			return
		}
		scope = scope.Parent
	}
	return
}

func (s *Scope) CheckDefined(name string) bool {
	_, found := s.Symbols.table[name]
	return found
}
