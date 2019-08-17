package ast

type EntryFlag int

const (
	VarEntry   EntryFlag = iota
	ConstEntry           // variables and constants should be treated differently
	FuncEntry            // directly defined functions, excluding lambdas
	TypeEntry
)

type TableEntry struct {
	Loc  *Loc
	Name string
	Flag EntryFlag
	Type IType       // if nil, the type of current symbol is unknown
	Val  interface{} // reserved for constant expression
}

func NewSymbolEntry(loc *Loc, name string, flag EntryFlag, tp IType,
	val interface{}) *TableEntry {
	return &TableEntry{Loc: loc, Name: name, Flag: flag, Type: tp, Val: val}
}

func (e *TableEntry) IsNamed() bool { return len(e.Name) > 0 }

func (e *TableEntry) TypeUnknown() bool { return e.Type == nil }

type SymbolTable struct {
	Entries []*TableEntry          // for ordered access
	Table   map[string]*TableEntry // for lookup
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{Entries: make([]*TableEntry, 0), Table: make(map[string]*TableEntry)}
}

// This method does not check the validity of entry.
// The check is done in Scope.AddEntry()
func (t *SymbolTable) Add(entries ...*TableEntry) {
	for _, e := range entries {
		t.Entries = append(t.Entries, e)
		t.Table[e.Name] = e
	}
}

func (t *SymbolTable) Lookup(name string) *TableEntry {
	return t.Table[name]
}
