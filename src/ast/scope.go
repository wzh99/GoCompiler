package ast

type Scope struct {
	parent   *Scope
	children []*Scope
	symbols  *SymbolTable
	global   bool
}

func NewGlobalScope() *Scope {
	return &Scope{symbols: NewSymbolTable(), parent: nil, children: make([]*Scope, 0), global: true}
}

func NewLocalScope(parent *Scope) *Scope {
	return &Scope{symbols: NewSymbolTable(), parent: parent, children: make([]*Scope, 0), global: false}
}

func (s *Scope) AddChild(child *Scope) {
	s.children = append(s.children, child)
}

func (s *Scope) AddSymbol(entry *SymbolEntry) { s.symbols.Add(entry) }

func (s *Scope) BuildTable() { s.symbols.Build() }

// Look up symbol, considering nested scopes
func (s *Scope) Lookup(name string) (entry *SymbolEntry, scope *Scope) {
	cur := s
	for cur != nil {
		entry := cur.symbols.Lookup(name)
		if entry != nil {
			return entry, cur
		}
		cur = cur.parent
	}
	return nil, nil
}

func (s *Scope) CheckRedefined(name string) bool {
	_, found := s.symbols.table[name]
	return found
}
