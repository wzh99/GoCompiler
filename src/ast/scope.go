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
	return &Scope{Symbols: NewSymbolTable(), Parent: nil, Children: make([]*Scope, 0), Global: true}
}

func NewLocalScope(parent *Scope) *Scope {
	s := &Scope{Symbols: NewSymbolTable(), Parent: parent, Children: make([]*Scope, 0), Global: false,
		Func: parent.Func}
	parent.AddChild(s)
	return s
}

func (s *Scope) AddChild(child *Scope) {
	s.Children = append(s.Children, child)
}

func (s *Scope) AddSymbol(entry *Symbol) {
	// Reject unnamed symbol
	if len(entry.Name) == 0 {
		panic(fmt.Errorf("%s unnamed symbol", entry.Loc.ToString()))
	}
	// Reject redefined symbol
	if s.CheckDefined(entry.Name) {
		panic(fmt.Errorf("%s redefined symbol: %s", entry.Loc.ToString(), entry.Name))
	}
	s.Symbols.Add(entry)
}

func (s *Scope) AddOperandId(id *IdExpr) {
	if s.operandId == nil {
		s.operandId = make(map[*IdExpr]bool)
	}
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
