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
	tp   IType // if nil, the type of current symbol is unknown
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
