package ir

import (
	"fmt"
)

// Types specified for IR, different from ones in AST.
type TypeEnum int

const (
	Void TypeEnum = 1 << iota
	I1
	I64
	F64
	Struct
	Array
	Pointer
	Function
)

const (
	Integer = I1 | I64
	Float   = F64
)

func (e TypeEnum) Match(m TypeEnum) bool { return (e & m) != 0 }

const BitWidth = 64 // x86_64

type IType interface {
	GetTypeEnum() TypeEnum
	GetSize() int
	IsIdentical(tp IType) bool
	ToString() string
}

// Serves as both primitive type descriptor and base struct for other types
type BaseType struct {
	Enum TypeEnum
}

func NewBaseType(enum TypeEnum) *BaseType {
	return &BaseType{Enum: enum}
}

func (t *BaseType) GetTypeEnum() TypeEnum { return t.Enum }

func (t *BaseType) GetSize() int {
	switch t.Enum {
	case Void:
		return 0
	case I1:
		return 1
	case I64, F64:
		return 8
	default:
		panic(fmt.Errorf("size unknown"))
	}
}

func (t *BaseType) IsIdentical(t2 IType) bool {
	return t.Enum == t2.GetTypeEnum()
}

var typeEnumToStr = map[TypeEnum]string{
	Void: "void", I1: "i1", I64: "i64", F64: "f64",
}

func (t *BaseType) ToString() string { return typeEnumToStr[t.Enum] }

// Struct field in IR, no names are involved.
type FieldSpec struct {
	Type   IType
	Size   int
	Offset int
}

type StructType struct {
	BaseType
	Field []FieldSpec
	size  int
}

func NewStructType(field []IType) *StructType {
	t := &StructType{
		BaseType: *NewBaseType(Struct),
		Field:    make([]FieldSpec, 0),
	}
	offset := 0
	for _, tp := range field {
		size := tp.GetSize()
		t.Field = append(t.Field, FieldSpec{Type: tp, Size: size, Offset: offset})
		offset += size
	}
	t.size = offset
	return t
}

func (t *StructType) GetSize() int { return t.size }

func (t *StructType) At(i int) IType { return t.Field[i].Type }

func (t *StructType) IsIdentical(o IType) bool {
	t2, ok := o.(*StructType)
	if !ok {
		return false
	}
	if len(t.Field) != len(t2.Field) {
		return false
	}
	for i := range t.Field {
		if !t.Field[i].Type.IsIdentical(t2.Field[i].Type) {
			return false
		}
	}
	return true
}

func (t *StructType) ToString() string {
	str := "{"
	for i, f := range t.Field {
		if i != 0 {
			str += ", "
		}
		str += f.Type.ToString()
	}
	return str + "}"
}

type ArrayType struct {
	BaseType
	Elem IType
	Len  int // number of elements, not of bytes
}

func NewArrayType(elem IType, len int) *ArrayType {
	return &ArrayType{
		BaseType: *NewBaseType(Array),
		Elem:     elem,
		Len:      len,
	}
}

func (t *ArrayType) GetSize() int { return t.Len * t.Elem.GetSize() }

func (t *ArrayType) IsIdentical(o IType) bool {
	t2, ok := o.(*ArrayType)
	if !ok {
		return false
	}
	return t.Elem.IsIdentical(t2.Elem) && t.Len == t2.Len
}

func (t *ArrayType) ToString() string {
	return fmt.Sprintf("[%d]%s", t.Len, t.Elem.ToString())
}

type PtrType struct {
	BaseType
	Base IType
}

func NewPtrType(target IType) *PtrType {
	return &PtrType{
		BaseType: *NewBaseType(Pointer),
		Base:     target,
	}
}

func (t *PtrType) GetSize() int { return BitWidth / 8 }

func (t *PtrType) IsIdentical(o IType) bool {
	t2, ok := o.(*PtrType)
	if !ok {
		return false
	}
	// Pointer to void is equivalent to pointer to other type.
	// This may be unsafe, but allows for more flexibility, when the base type of pointer
	// is not clear.
	if t.Base.GetTypeEnum() == Void || t2.Base.GetTypeEnum() == Void {
		return true
	}
	return t.Base.IsIdentical(t2.Base)
}

func (t *PtrType) ToString() string {
	return fmt.Sprintf("*%s", t.Base.ToString())
}

type FuncType struct {
	BaseType
	Param  []IType
	Return *StructType // multiple values should be packed into a struct
}

func NewFuncType(param []IType, ret *StructType) *FuncType {
	return &FuncType{
		BaseType: *NewBaseType(Function),
		Param:    param,
		Return:   ret,
	}
}

func (t *FuncType) GetSize() int { return BitWidth / 8 }

func (t *FuncType) IsIdentical(o IType) bool {
	t2, ok := o.(*FuncType)
	if !ok {
		return false
	}
	for i := range t.Param {
		if !t.Param[i].IsIdentical(t2.Param[i]) {
			return false
		}
	}
	return t.Return.IsIdentical(t2.Return)
}

func (t *FuncType) ToString() string {
	str := "("
	for i, p := range t.Param {
		if i != 0 {
			str += ", "
		}
		str += p.ToString()
	}
	return str + fmt.Sprintf(")->%s", t.Return.ToString())
}
