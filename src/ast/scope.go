package ast

import "fmt"

type Scope struct {
	Parent    *Scope
	Children  []*Scope
	Symbols   *SymbolTable
	OperandId map[*IdExpr]bool // all operand identifiers mentioned in statements
	Global    bool
	Func      *FuncDecl // point back to the function this scope belongs to
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

func (s *Scope) AddSymbol(entry *TableEntry) {
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
	if s.OperandId == nil {
		s.OperandId = make(map[*IdExpr]bool)
	}
	s.OperandId[id] = true
}

// Look up symbol, considering nested scopes
func (s *Scope) Lookup(name string) (entry *TableEntry, scope *Scope) {
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
	_, found := s.Symbols.Table[name]
	return found
}
