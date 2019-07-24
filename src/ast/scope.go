package ast

type Scope struct {
	table    *SymbolTable
	parent   *Scope
	children []*Scope
	global   bool
}

func NewScope(table *SymbolTable, parent *Scope, global bool) *Scope {
	return &Scope{table: table, parent: parent, children: make([]*Scope, 0), global: global}
}

func (s *Scope) AddChild(child *Scope) {
	s.children = append(s.children, child)
}

// Look up symbol, considering nested scopes
func (s *Scope) Lookup(name string) *SymbolEntry {
	cur := s
	for cur != nil {
		entry := cur.table.Lookup(name)
		if entry != nil {
			return entry
		}
		cur = cur.parent
	}
	return nil
}
