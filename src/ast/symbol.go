package ast

type EntryFlag int

const (
	VarEntry   EntryFlag = iota
	ConstEntry           // variables and constants should be treated differently
	FuncEntry            // directly defined functions, excluding lambdas
	TypeEntry
)

type Symbol struct {
	Loc   *Loc
	Name  string
	Flag  EntryFlag
	Type  IType       // if nil, the type of current symbol is unknown
	Val   interface{} // reserved for constant expression
	Scope *Scope      // the scope where this symbol is defined
}

func NewSymbol(loc *Loc, name string, flag EntryFlag, tp IType, val interface{}) *Symbol {
	return &Symbol{
		Loc:  loc,
		Name: name,
		Flag: flag,
		Type: tp,
		Val:  val,
	}
}

func (e *Symbol) IsNamed() bool { return len(e.Name) > 0 }

func (e *Symbol) TypeUnknown() bool { return e.Type == nil }

type SymbolTable struct {
	Entries []*Symbol          // for ordered access
	table   map[string]*Symbol // for lookup
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{Entries: make([]*Symbol, 0), table: make(map[string]*Symbol)}
}

// This method does not check the validity of entry.
// The check is done in Scope.AddEntry()
func (t *SymbolTable) Add(entries ...*Symbol) {
	for _, e := range entries {
		t.Entries = append(t.Entries, e)
		t.table[e.Name] = e
	}
}

func (t *SymbolTable) Lookup(name string) *Symbol {
	return t.table[name]
}
