package ir

import "fmt"

// Similar to the one in AST, but with simplified data
type Symbol struct {
	Name string
	Type IType
}

type Scope struct {
	// Point to higher level scope, lower ones are not needed
	Parent *Scope
	// Linear list of entries
	Entries []*Symbol
	// Look up table that helps querying
	table map[string]*Symbol
	// Indicate whether this scope is global
	Global bool
}

func NewGlobalScope() *Scope {
	return &Scope{
		Parent:  nil,
		Entries: make([]*Symbol, 0),
		table:   make(map[string]*Symbol),
		Global:  true,
	}
}

func NewLocalScope(parent *Scope) *Scope {
	return &Scope{
		Parent:  parent,
		Entries: make([]*Symbol, 0),
		table:   make(map[string]*Symbol),
		Global:  false,
	}
}

func (s *Scope) AddSymbol(name string, tp IType) {
	entry := &Symbol{
		Name: name,
		Type: tp,
	}
	s.Entries = append(s.Entries, entry)
	s.table[name] = entry
}

func (s *Scope) Lookup(name string) (symbol *Symbol, scope *Scope) {
	scope = s
	for scope != nil {
		symbol = scope.table[name]
		if symbol != nil {
			return
		}
		scope = scope.Parent
	}
	// symbol must be found, otherwise there is logical error in the compiler
	panic(NewIRError(fmt.Sprintf("symbol %s not found", name)))
}
