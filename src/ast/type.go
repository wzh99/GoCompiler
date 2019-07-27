package ast

import "fmt"

type TypeEnum int

// Provide a simpler way to mark types tha using type assertions
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

var TypeToStr = map[TypeEnum]string{
	Bool: "bool", Int8: "int8", Int16: "int16", Int32: "int32", Int64: "int64",
	Uint8: "uint8", Uint16: "uint16", Uint32: "uint32", Uint64: "uint64",
	Int: "int", Uint: "uint", Uintptr: "uintptr", Byte: "byte", Rune: "rune",
	Float32: "float32", Float64: "float64", Complex64: "complex64", Complex128: "complex128",
	String: "string", Struct: "struct", Func: "func", Interface: "interface", Map: "map",
	Channel: "channel", Nil: "nil",
}

// Type interface
type IType interface {
	ToString() string
	GetTypeEnum() TypeEnum
	IsIdentical(tp IType) bool
}

type BaseType struct {
	enum TypeEnum
}

func NewBaseType(enum TypeEnum) *BaseType {
	return &BaseType{enum: enum}
}

func (t *BaseType) ToString() string {
	str, ok := TypeToStr[t.enum]
	if ok {
		return str
	} else {
		return "unknown"
	}
}

func (t *BaseType) GetTypeEnum() TypeEnum { return t.enum }

// This method should be overridden by non-primitive type
func (t *BaseType) IsIdentical(tp IType) bool {
	return t.enum == tp.GetTypeEnum()
}

// When the AST is constructed, some type cannot be resolved at that time.
type UnresolvedType struct {
	BaseType
	name string
}

func NewUnresolvedType(name string) *UnresolvedType {
	return &UnresolvedType{BaseType: *NewBaseType(Unresolved), name: name}
}

func (t *UnresolvedType) ToString() string {
	return fmt.Sprintf("unresolved: %s", t.name)
}

func (t *UnresolvedType) IsIdentical(o IType) bool {
	t2, ok := o.(*UnresolvedType)
	if !ok {
		return false
	}
	return t.name == t2.name
}

// Type alias
type AliasType struct {
	BaseType
	name  string
	under IType
}

func NewAliasType(name string, under IType) *AliasType {
	if alias, ok := under.(*AliasType); ok {
		under = alias.under
	}
	return &AliasType{BaseType: *NewBaseType(under.GetTypeEnum()), name: name, under: under}
}

func (t *AliasType) ToString() string {
	return fmt.Sprintf("%s: %s", t.name, t.under.ToString())
}

func (t *AliasType) IsIdentical(o IType) bool {
	if t2, ok := o.(*AliasType); ok {
		// all alias type, and have identical underlying type
		if t.under.IsIdentical(t2.under) {
			return true
		}
	} else { // the second is not alias type
		// the second is identical to the underlying type of the first
		if t.under.IsIdentical(o) {
			return true
		}
	}
	return false
}

// Value type of integer, float, complex
type PrimType struct {
	BaseType
}

func NewPrimType(enum TypeEnum) *PrimType {
	if (enum & PrimitiveType) == 0 {
		panic("Not primitive type")
	}
	return &PrimType{BaseType: *NewBaseType(enum)}
}

func (t *PrimType) IsIdentical(o IType) bool {
	if alias, ok := o.(*AliasType); ok {
		return alias.IsIdentical(t)
	}
	return t.GetTypeEnum() == o.GetTypeEnum()
}

var PrimTypeSize = map[TypeEnum]int{
	Bool: 1, Int8: 1, Int16: 2, Int32: 4, Int64: 8, Uint8: 1, Uint16: 2, Uint32: 4, Uint64: 8,
	Int: 8, Uint: 8, Uintptr: 8, Byte: 1, Rune: 4, Float32: 4, Float64: 8, Complex64: 8, Complex128: 16,
}

func (t *PrimType) GetSize() int {
	return PrimTypeSize[t.enum]
}

type StructType struct {
	BaseType
	field *SymbolTable
}

func NewStructType(field *SymbolTable) *StructType {
	return &StructType{BaseType: *NewBaseType(Struct), field: field}
}

func (t *StructType) ToString() string {
	str := "struct{"
	for i, f := range t.field.entries {
		if i != 0 {
			str += ", "
		}
		str += f.tp.ToString()
	}
	return str + "}"
}

func (t *StructType) IsIdentical(o IType) bool {
	if alias, ok := o.(*AliasType); ok {
		return alias.IsIdentical(t)
	}
	t2, ok := o.(*StructType)
	if !ok { // not even struct type
		return false
	}
	if len(t2.field.entries) != len(t2.field.entries) {
		return false
	}
	for i, e1 := range t.field.entries {
		e2 := t2.field.entries[i]
		if (!e1.tp.IsIdentical(e2.tp)) || (e1.name != e2.name) {
			return false
		}
	}
	return true
}

// Mainly used in function parameters representation, and assignment semantic analysis
type TupleType struct {
	BaseType
	elem []IType
}

func NewTupleType(elem []IType) *TupleType {
	return &TupleType{BaseType: *NewBaseType(Tuple), elem: elem}
}

func (t *TupleType) ToString() string {
	str := "("
	for i, e := range t.elem {
		if i != 0 {
			str += " "
		}
		str += e.ToString()
	}
	return str + ")"
}

func (t *TupleType) IsIdentical(o IType) bool {
	t2, ok := o.(*TupleType)
	if !ok { // not even tuple type
		return false
	}
	if len(t.elem) != len(t2.elem) { // have different number of elements
		return false
	}
	for i := range t.elem {
		if !t.elem[i].IsIdentical(t2.elem[i]) {
			return false
		}
	}
	return true
	// tuple is not defined in the language, so there is no type alias
}

type FuncType struct {
	BaseType
	param, ret *TupleType
	receiver   IType // optional for methods, not explicitly assigned in constructor
}

func NewFunctionType(params, results []IType) *FuncType {
	return &FuncType{BaseType: *NewBaseType(Func), param: NewTupleType(params),
		ret: NewTupleType(results), receiver: nil}
}

func (t *FuncType) ToString() string {
	return fmt.Sprintf("func (%s) %s %s", t.receiver.ToString(), t.param.ToString(),
		t.ret.ToString())
}

func (t *FuncType) IsIdentical(o IType) bool {
	if alias, ok := o.(*AliasType); ok {
		return alias.IsIdentical(t)
	}
	t2, ok := o.(*FuncType)
	if !ok {
		return false
	}
	identical := t.param.IsIdentical(t2.param) && t.ret.IsIdentical(t2.ret)
	if t.receiver == nil {
		return identical && t2.receiver == nil
	} else {
		return identical && t.receiver.IsIdentical(t2.receiver)
	}
}
