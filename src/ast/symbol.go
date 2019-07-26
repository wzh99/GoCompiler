package ast

type EntryFlag int

const (
	VarEntry   EntryFlag = iota
	ConstEntry           // variables and constants should be treated differently
	FuncEntry            // directly defined functions, excluding lambdas
	TypeEntry
)

type SymbolEntry struct {
	loc  *Location
	name string
	flag EntryFlag
	tp   IType       // if nil, the type of current symbol is unknown
	val  interface{} // reserved for constant expression
}

func NewSymbolEntry(loc *Location, name string, flag EntryFlag, tp IType, val interface{}) *SymbolEntry {
	return &SymbolEntry{loc: loc, name: name, flag: flag, tp: tp, val: val}
}

type SymbolTable struct {
	entries []*SymbolEntry          // for ordered access
	table   map[string]*SymbolEntry // for lookup
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{entries: make([]*SymbolEntry, 0)}
}

func (t *SymbolTable) Add(entry *SymbolEntry) {
	t.entries = append(t.entries, entry)
}

// Only called when entries are completely added.
func (t *SymbolTable) Build() {
	for i, e := range t.entries {
		t.table[e.name] = t.entries[i]
	}
}

func (t *SymbolTable) Lookup(name string) *SymbolEntry {
	if t.table == nil {
		panic("Table not constructed.")
	}
	return t.table[name]
}
