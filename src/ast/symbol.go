package ast

type EntryFlag int

const (
	VarEntry   EntryFlag = iota
	ConstEntry           // variables and constants should be treated differently
	FuncEntry            // directly defined functions, excluding lambdas
	TypeEntry
)

type SymbolEntry struct {
	Loc  *Location
	Name string
	Flag EntryFlag
	Type IType       // if nil, the type of current symbol is unknown
	Val  interface{} // reserved for constant expression
}

func NewSymbolEntry(loc *Location, name string, flag EntryFlag, tp IType,
	val interface{}) *SymbolEntry {
	return &SymbolEntry{Loc: loc, Name: name, Flag: flag, Type: tp, Val: val}
}

func (e *SymbolEntry) IsNamed() bool { return len(e.Name) > 0 }

func (e *SymbolEntry) TypeUnknown() bool { return e.Type == nil }

type SymbolTable struct {
	Entries []*SymbolEntry          // for ordered access
	Table   map[string]*SymbolEntry // for lookup
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{Entries: make([]*SymbolEntry, 0), Table: make(map[string]*SymbolEntry)}
}

// This method does not check the validity of entry.
// The check is done in Scope.AddEntry()
func (t *SymbolTable) Add(entries ...*SymbolEntry) {
	for _, e := range entries {
		t.Entries = append(t.Entries, e)
		t.Table[e.Name] = e
	}
}

func (t *SymbolTable) Lookup(name string) *SymbolEntry {
	return t.Table[name]
}
