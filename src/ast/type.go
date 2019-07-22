package ast

type TypeEnum int

const (
	// Integer type
	Bool TypeEnum = iota
	Uint8
	Uint16
	Uint32
	Uint64
	Int8
	Int16
	Int32
	Int64
	Int
	Uint
	Uintptr
	Byte
	Rune
	// Float type
	Float32
	Float64
	// Complex type
	Complex64
	Complex128
	// Pointer type: *int, *float64, ...
	Ptr
	// Nil Type
	Nil
	// String type
	String
	// Function type: func $Name$($Params$) $Result$
	Func
	// Structure type: type $Name$ struct {$Members$}
	Struct
	// Unresolved
	Unresolved
)

const (
	IntegerType = (Rune << 1) - 1
	FloatType   = Float32 | Float64
	ComplexType = Complex64 | Complex128
)

type IType interface {
	ToString() string
	GetTypeEnum() TypeEnum
	IsSameType(tp IType) bool
	GetSize() int
}

type BaseType struct {
	enum TypeEnum
}

func newBaseType(enum TypeEnum) *BaseType {
	return &BaseType{enum: enum}
}

func (t *BaseType) GetTypeEnum() TypeEnum { return t.enum }

// This method should be overridden by non-primitive type
func (t *BaseType) IsSameType(tp IType) bool {
	return t.enum == tp.GetTypeEnum()
}

type UnresolvedType struct {
	BaseType
	name string
}

// When the AST is constructed, some type cannot be resolved at that time.
func NewUnresolvedType(name string) *UnresolvedType {
	return &UnresolvedType{BaseType: *newBaseType(Unresolved), name: name}
}

func (t *UnresolvedType) ToString() string {
	return "Unresolved"
}

func (t *UnresolvedType) GetSize() int { return 0 }
