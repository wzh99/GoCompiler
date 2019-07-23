package ast

type TypeEnum int

// Some type, notably literals, can be more than one type, depending on context.
const (
	/* Built-in types */
	// Boolean type
	Bool TypeEnum = 1 << iota
	// Integer type
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
	Byte
	Rune
	// Float type
	Float32
	Float64
	// Complex type
	Complex64
	Complex128
	// String type
	String
	// Array type
	Array
	// Slice type
	Slice
	// Structure type: type ... struct {...}
	Struct
	// Pointer type: *int, *float64, ...
	Ptr
	// Function type: func ...(...) ...
	Func
	// Interface type: type ... interface {...}
	Interface
	// Map type: map[...]...
	Map
	// Channel type: channel ...
	Channel
	/* Additional types in compiler */
	// Nil type
	Nil
	// Unresolved
	Unresolved
	// Tuple type (handle multiple values)
	Tuple
)

const (
	SignedType    = Int8 | Int16 | Int32 | Int64 | Int | Rune
	UnsignedType  = Uint8 | Uint16 | Uint32 | Uint64 | Uint | Uintptr | Byte
	IntegerType   = SignedType | UnsignedType
	FloatType     = Float32 | Float64
	ComplexType   = Complex64 | Complex128
	PrimitiveType = IntegerType | FloatType | ComplexType
	CompositeType = Array | Slice | Struct | Ptr | Func | Interface | Map | Channel
)

var TypeStr = map[TypeEnum]string{
	Bool: "bool", Int8: "int8", Int16: "int16", Int32: "int32", Int64: "int64",
	Uint8: "uint8", Uint16: "uint16", Uint32: "uint32", Uint64: "uint64",
	Int: "int", Uint: "uint", Uintptr: "uintptr", Byte: "byte", Rune: "rune",
	Float32: "float32", Float64: "float64", Complex64: "complex64", Complex128: "complex128",
	String: "string", Array: "array", Struct: "struct", Ptr: "ptr", Func: "func",
	Interface: "interface", Map: "map", Channel: "channel",
	Nil: "nil", Unresolved: "unresolved",
}

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
	str, ok := TypeStr[t.enum]
	if ok {
		return str
	} else {
		return "unknown"
	}
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

// Value type of integer, float, complex
type PrimType struct {
	BaseType
}

func NewPrimType(enum TypeEnum) *PrimType {
	if enum&PrimitiveType == 0 {
		panic("Not primitive type.")
	}
	return &PrimType{BaseType: *newBaseType(enum)}
}

var PrimTypeSize = map[TypeEnum]int{
	Bool: 1, Int8: 1, Int16: 2, Int32: 4, Int64: 8, Uint8: 1, Uint16: 2, Uint32: 4, Uint64: 8,
	Int: 8, Uint: 8, Uintptr: 8, Byte: 1, Rune: 4, Float32: 4, Float64: 8, Complex64: 8, Complex128: 16,
}

func (t *PrimType) GetSize() int {
	return PrimTypeSize[t.enum]
}

type TupleType struct {
	BaseType
	elem []IType
}

func NewTupleType(elem []IType) *TupleType {
	return &TupleType{BaseType: *newBaseType(Tuple), elem: elem}
}

func (t *TupleType) ToString() string {
	str := "tuple("
	for i, e := range t.elem {
		if i != 0 {
			str += " "
		}
		str += e.ToString()
	}
	return str + ")"
}

func (t *TupleType) IsSameType(o IType) bool {
	t2, ok := o.(*TupleType)
	if !ok { // not even tuple type
		return false
	}
	if len(t.elem) != len(t2.elem) { // have different number of elements
		return false
	}
	for i := range t.elem {
		if !t.elem[i].IsSameType(t2.elem[i]) {
			return false
		}
	}
	return true
}

func (t *TupleType) GetSize() int {
	size := 0
	for _, t := range t.elem {
		size += t.GetSize()
	}
	return size
}
