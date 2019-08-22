package ir

import (
	"fmt"
)

// Types specified for IR, different from ones in AST.
type TypeEnum int

const (
	I64 TypeEnum = 1 << iota
	F64
	Struct
	Array
	Ptr // don't care the type it points to
)

type IType interface {
	GetTypeEnum() TypeEnum
	GetSize() int
}

// Serves as both primitive type struct and base struct for other types
type BaseType struct {
	Enum TypeEnum
}

func NewBaseType(enum TypeEnum) *BaseType {
	return &BaseType{Enum: enum}
}

func (t *BaseType) GetTypeEnum() TypeEnum { return t.Enum }

func (t *BaseType) GetSize() int {
	switch t.Enum {
	case I64, F64, Ptr:
		return 8
	default:
		panic(fmt.Errorf("size unknown"))
	}
}

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
