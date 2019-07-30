package ast

import "fmt"

type Scope struct {
	parent    *Scope
	children  []*Scope
	symbols   *SymbolTable
	operandId map[*IdExpr]bool // all operand identifiers mentioned in statements
	global    bool
	fun       *FuncDecl // point back to the function this scope belongs to
}

func NewGlobalScope() *Scope {
	return &Scope{symbols: NewSymbolTable(), parent: nil, children: make([]*Scope, 0), global: true}
}

func NewLocalScope(parent *Scope) *Scope {
	s := &Scope{symbols: NewSymbolTable(), parent: parent, children: make([]*Scope, 0), global: false,
		fun: parent.fun}
	parent.AddChild(s)
	return s
}

func (s *Scope) AddChild(child *Scope) {
	s.children = append(s.children, child)
}

func (s *Scope) AddSymbol(entry *SymbolEntry) {
	// Reject unnamed symbol
	if len(entry.name) == 0 {
		panic(fmt.Errorf("%s unnamed symbol", entry.loc.ToString()))
	}
	// Reject redefined symbol
	if s.CheckDefined(entry.name) {
		panic(fmt.Errorf("%s redefined symbol: %s", entry.loc.ToString(), entry.name))
	}
	s.symbols.Add(entry)
}

func (s *Scope) AddOperandId(id *IdExpr) {
	if s.operandId == nil {
		s.operandId = make(map[*IdExpr]bool)
	}
	s.operandId[id] = true
}

// Look up symbol, considering nested scopes
func (s *Scope) Lookup(name string) (entry *SymbolEntry, scope *Scope) {
	scope = s
	for scope != nil {
		entry = scope.symbols.Lookup(name)
		if entry != nil {
			return
		}
		scope = scope.parent
	}
	return
}

func (s *Scope) CheckDefined(name string) bool {
	_, found := s.symbols.table[name]
	return found
}
