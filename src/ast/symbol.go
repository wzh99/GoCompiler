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

func (e *SymbolEntry) IsNamed() bool { return len(e.name) > 0 }

func (e *SymbolEntry) TypeUnknown() bool { return e.tp == nil }

type SymbolTable struct {
	entries []*SymbolEntry          // for ordered access
	table   map[string]*SymbolEntry // for lookup
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{entries: make([]*SymbolEntry, 0), table: make(map[string]*SymbolEntry)}
}

// This method does not check the validity of entry.
// The check is done in Scope.AddEntry()
func (t *SymbolTable) Add(entries ...*SymbolEntry) {
	for _, e := range entries {
		t.entries = append(t.entries, e)
		t.table[e.name] = e
	}
}

func (t *SymbolTable) Lookup(name string) *SymbolEntry {
	return t.table[name]
}
