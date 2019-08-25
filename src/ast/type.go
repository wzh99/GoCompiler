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
	PrimitiveType = Bool | IntegerType | FloatType | ComplexType | String
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

func (e TypeEnum) Match(m TypeEnum) bool { return (e & m) != 0 }

// Type interface
type IType interface {
	GetLoc() *Loc
	ToString() string
	GetTypeEnum() TypeEnum
	IsIdentical(tp IType) bool
}

type BaseType struct {
	Loc  *Loc
	Enum TypeEnum
}

func NewBaseType(loc *Loc, enum TypeEnum) *BaseType {
	return &BaseType{Loc: loc, Enum: enum}
}

func (t *BaseType) GetLoc() *Loc { return t.Loc }

func (t *BaseType) ToString() string {
	str, ok := TypeToStr[t.Enum]
	if ok {
		return str
	} else {
		return "unknown"
	}
}

func (t *BaseType) GetTypeEnum() TypeEnum { return t.Enum }

// This method should be overridden by non-primitive type
func (t *BaseType) IsIdentical(tp IType) bool {
	return t.Enum == tp.GetTypeEnum()
}

// When the AST is constructed, some type cannot be resolved at that time.
type UnresolvedType struct {
	BaseType
	Name string
}

func NewUnresolvedType(loc *Loc, name string) *UnresolvedType {
	return &UnresolvedType{BaseType: *NewBaseType(loc, Unresolved), Name: name}
}

func (t *UnresolvedType) ToString() string {
	return fmt.Sprintf("unresolved: %s", t.Name)
}

func (t *UnresolvedType) IsIdentical(o IType) bool {
	t2, ok := o.(*UnresolvedType)
	if !ok {
		return false
	}
	return t.Name == t2.Name
}

// Type alias
type AliasType struct {
	BaseType
	Name  string
	Under IType
}

func NewAliasType(loc *Loc, name string, under IType) *AliasType {
	if alias, ok := under.(*AliasType); ok {
		under = alias.Under
	}
	return &AliasType{
		BaseType: *NewBaseType(loc, under.GetTypeEnum()),
		Name:     name,
		Under:    under,
	}
}

func (t *AliasType) ToString() string {
	return fmt.Sprintf("%s", t.Name)
}

func (t *AliasType) IsIdentical(o IType) bool {
	if t2, ok := o.(*AliasType); ok {
		// all alias type, and have identical underlying type
		if t.Under.IsIdentical(t2.Under) {
			return true
		}
	} else { // the second is not alias type
		// the second is identical to the underlying type of the first
		if t.Under.IsIdentical(o) {
			return true
		}
	}
	return false
}

// Value type of integer, float, complex
type PrimType struct {
	BaseType
}

var StrToPrimType = map[string]TypeEnum{
	"bool": Bool, "int8": Int8, "int16": Int16, "int32": Int32, "int64": Int64,
	"uint8": Uint8, "uint16": Uint16, "uint32": Uint32, "uint64": Uint64,
	"int": Int, "uint": Uint, "uintptr": Uintptr, "byte": Byte, "rune": Rune,
	"float32": Float32, "float64": Float64, "complex64": Complex64, "complex128": Complex128,
	"string": String,
}

func NewPrimType(loc *Loc, enum TypeEnum) *PrimType {
	if (enum & PrimitiveType) == 0 {
		panic("Not primitive type")
	}
	return &PrimType{
		BaseType: *NewBaseType(loc, enum),
	}
}

func (t *PrimType) IsIdentical(o IType) bool {
	if alias, ok := o.(*AliasType); ok {
		return alias.IsIdentical(t)
	}
	return t.GetTypeEnum() == o.GetTypeEnum()
}

var PrimTypeSize = map[TypeEnum]int{
	Bool: 1, Int8: 1, Int16: 2, Int32: 4, Int64: 8,
	Uint8: 1, Uint16: 2, Uint32: 4, Uint64: 8,
	Int: 8, Uint: 8, Uintptr: 8, Byte: 1, Rune: 4, Float32: 4, Float64: 8,
	Complex64: 8, Complex128: 16,
}

func (t *PrimType) GetSize() int {
	return PrimTypeSize[t.Enum]
}

type NilType struct {
	BaseType
}

func NewNilType(loc *Loc) *NilType {
	return &NilType{BaseType: *NewBaseType(loc, Nil)}
}

func (t *NilType) IsIdentical(o IType) bool {
	_, ok := o.(*NilType)
	return ok
}

type PtrType struct {
	BaseType
	Base IType
}

func NewPtrType(loc *Loc, ref IType) *PtrType {
	return &PtrType{BaseType: *NewBaseType(loc, Ptr), Base: ref}
}

func (t *PtrType) ToString() string {
	return fmt.Sprintf("*%s", t.Base.ToString())
}

func (t *PtrType) IsIdentical(o IType) bool {
	if alias, ok := o.(*AliasType); ok {
		return alias.IsIdentical(t)
	}
	t2, ok := o.(*PtrType)
	return ok && t.Base.IsIdentical(t2.Base)
}

type ArrayType struct {
	BaseType
	Elem    IType
	Len     int
	lenExpr IExprNode // may be a constant expression
}

func NewArrayType(loc *Loc, elem IType, lenExpr IExprNode) *ArrayType {
	return &ArrayType{
		BaseType: *NewBaseType(loc, Array),
		Elem:     elem,
		lenExpr:  lenExpr,
	}
}

func (t *ArrayType) ToString() string {
	return fmt.Sprintf("[%d]%s", t.lenExpr, t.Elem.ToString())
}

func (t *ArrayType) IsIdentical(o IType) bool {
	if alias, ok := o.(*AliasType); ok {
		return alias.IsIdentical(t)
	}
	t2, ok := o.(*ArrayType)
	return ok && t.Elem.IsIdentical(t2.Elem) && t.lenExpr == t2.lenExpr
}

type SliceType struct {
	BaseType
	Elem IType
}

func NewSliceType(loc *Loc, elem IType) *SliceType {
	return &SliceType{BaseType: *NewBaseType(loc, Slice), Elem: elem}
}

func (t *SliceType) ToString() string {
	return fmt.Sprintf("[]%s", t.Elem.ToString())
}

func (t *SliceType) IsIdentical(o IType) bool {
	if alias, ok := o.(*AliasType); ok {
		return alias.IsIdentical(t)
	}
	t2, ok := o.(*SliceType)
	return ok && t.Elem.IsIdentical(t2.Elem)
}

type MapType struct {
	BaseType
	Key, Val IType
}

func NewMapType(loc *Loc, key, val IType) *MapType {
	return &MapType{BaseType: *NewBaseType(loc, Map), Key: key, Val: val}
}

func (t *MapType) ToString() string {
	return fmt.Sprintf("[%s]%s", t.Key.ToString(), t.Val.ToString())
}

func (t *MapType) IsIdentical(o IType) bool {
	if alias, ok := o.(*AliasType); ok {
		return alias.IsIdentical(t)
	}
	t2, ok := o.(*MapType)
	return ok && t.Key.IsIdentical(t2.Key) && t2.Val.IsIdentical(t2.Val)
}

type StructType struct {
	BaseType
	Field *SymbolTable
}

func NewStructType(loc *Loc, field *SymbolTable) *StructType {
	return &StructType{BaseType: *NewBaseType(loc, Struct), Field: field}
}

func (t *StructType) ToString() string {
	str := "struct{"
	for i, f := range t.Field.Entries {
		if i != 0 {
			str += ", "
		}
		str += f.Type.ToString()
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
	if len(t2.Field.Entries) != len(t2.Field.Entries) {
		return false
	}
	for i, e1 := range t.Field.Entries {
		e2 := t2.Field.Entries[i]
		if (!e1.Type.IsIdentical(e2.Type)) || (e1.Name != e2.Name) {
			return false
		}
	}
	return true
}

// Mainly used in function parameters representation, and assignment semantic analysis
type TupleType struct {
	BaseType
	Elem []IType
}

func NewTupleType(elem []IType) *TupleType {
	return &TupleType{BaseType: *NewBaseType(nil, Tuple), Elem: elem}
}

func (t *TupleType) ToString() string {
	str := "("
	for i, e := range t.Elem {
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
	if len(t.Elem) != len(t2.Elem) { // have different number of elements
		return false
	}
	for i := range t.Elem {
		if !t.Elem[i].IsIdentical(t2.Elem[i]) {
			return false
		}
	}
	return true
	// tuple is not an actual type in the language, so there is no type alias
}

type FuncType struct {
	BaseType
	Param, Result *TupleType
	Receiver      IType // optional for methods, not explicitly assigned in constructor
}

func NewFunctionType(loc *Loc, param, result []IType) *FuncType {
	return &FuncType{BaseType: *NewBaseType(loc, Func), Param: NewTupleType(param),
		Result: NewTupleType(result), Receiver: nil}
}

func (t *FuncType) ToString() string {
	return fmt.Sprintf("func (%s) %s %s", t.Receiver.ToString(), t.Param.ToString(),
		t.Result.ToString())
}

func (t *FuncType) IsIdentical(o IType) bool {
	if alias, ok := o.(*AliasType); ok {
		return alias.IsIdentical(t)
	}
	t2, ok := o.(*FuncType)
	if !ok {
		return false
	}
	identical := t.Param.IsIdentical(t2.Param) && t.Result.IsIdentical(t2.Result)
	if t.Receiver == nil {
		return identical && t2.Receiver == nil
	} else {
		return identical && t.Receiver.IsIdentical(t2.Receiver)
	}
}

func (t *FuncType) GetParamType() []IType { return t.Param.Elem }

func (t *FuncType) GetResultType() []IType { return t.Result.Elem }
