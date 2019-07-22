package ast

type TypeEnum int

// Some type, notably literals, can be more than one type, depending on context.
const (
	// Integer type
	Bool TypeEnum = 1 << iota
	Int8
	Int16
	Int32
	Int64
	Uint8
	Uint16
	Uint32
	Uint64
	Int
	Uint
	Uintptr
	// Rune type
	Byte
	Rune
	// Float type
	Float32
	Float64
	// Complex type
	Complex64
	Complex128
	// Nil Type
	Nil
	// String type
	String
	// Pointer type: *int, *float64, ...
	Ptr
	// Function type: func $Name$($Params$) $Result$
	Func
	// Structure type: type $Name$ struct {$Members$}
	Struct
	// Unresolved
	Unresolved
)

var TypeStr = [...]string{
	"bool", "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64",
	"int", "uint", "uintptr", "byte", "rune", "float32", "floar64", "complex64", "complex128",
	"nil", "string", "*", "func", "struct", "unresolved",
}

const (
	IntegerType = (Uintptr << 1) - 1
	RuneType    = Byte | Rune
	FloatType   = Float32 | Float64
	ComplexType = Complex64 | Complex128
)

// Type interface
type IType interface {
	ToString() string
	GetTypeEnum() TypeEnum
	IsSameType(tp IType) bool
	GetSize() int // in bytes
}

type BaseType struct {
	enum TypeEnum
}

func newBaseType(enum TypeEnum) *BaseType {
	return &BaseType{enum: enum}
}

func (t *BaseType) ToString() string {
	switch t.enum {
	case IntegerType:
		return "IntegerType"
	case RuneType:
		return "RuneType"
	case FloatType:
		return "FloatType"
	case ComplexType:
		return "ComplexType"
	default:
		for i := 0; i < len(TypeStr); i++ {
			if (1<<uint(i))&t.enum != 0 {
				return TypeStr[i]
			}
		}
	}
	return "unknown"
}

func (t *BaseType) GetTypeEnum() TypeEnum { return t.enum }

// This method should be overridden by non-primitive type
func (t *BaseType) IsSameType(tp IType) bool {
	return t.enum == tp.GetTypeEnum()
}

// When the AST is constructed, some type cannot be resolved at that time.
type UnresolvedType struct {
	BaseType
	name string
}

func NewUnresolvedType(name string) *UnresolvedType {
	return &UnresolvedType{BaseType: *newBaseType(Unresolved), name: name}
}

func (t *UnresolvedType) GetSize() int { return 0 }

// Value type of integer, float, complex and string
type PrimType struct {
	BaseType
}

func NewPrimType(enum TypeEnum) *PrimType {
	if enum&(IntegerType|RuneType|FloatType|ComplexType|Nil) == 0 {
		panic("Not primitive type.")
	}
	return &PrimType{BaseType: *newBaseType(enum)}
}

var PrimTypeSize = map[TypeEnum]int{
	Bool: 1, Int8: 1, Int16: 2, Int32: 4, Int64: 8, Uint8: 1, Uint16: 2, Uint32: 4, Uint64: 8,
	Int: 8, Uint: 8, Uintptr: 8, Byte: 1, Rune: 4, Float32: 4, Float64: 8, Complex64: 8, Complex128: 16,
	Nil: 8, IntegerType: 8, RuneType: 4, FloatType: 8, ComplexType: 16,
}

func (t *PrimType) GetSize() int {
	return PrimTypeSize[t.enum]
}
