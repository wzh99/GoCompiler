package ir

type Func struct {
	// A function has only one begin block and one end block
	Begin, End *BasicBlock
	// base scope of current function, may have nested scopes
	Scope *Scope
	// parameter list
	Param []*Symbol
	// return type
	Return []IType
}
