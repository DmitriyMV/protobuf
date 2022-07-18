package protobuf_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DmitriyMV/protobuf"
)

//nolint:govet
type TestNestedOuter struct {
	A int32
	*TestNestedInner
}

type TestNestedInner struct {
	A int32
	B int32 `protobuf:"10"`
	C int32 `protobuf:"renamed"`
}

func TestEncodeNested(t *testing.T) {
	s := &TestNestedOuter{
		A: 13,
		TestNestedInner: &TestNestedInner{
			A: 12,
			B: 14,
			C: 15,
		},
	}
	v := reflect.ValueOf(s).Elem()

	actual := protobuf.ProtoFields(v.Type())
	for _, f := range actual {
		f.Field = reflect.StructField{}
	}

	expected := []*protobuf.ProtoField{
		{1, "", []int{0}, reflect.StructField{}},
		{2, "", []int{1, 0}, reflect.StructField{}},
		{10, "", []int{1, 1}, reflect.StructField{}},
		{11, "renamed", []int{1, 2}, reflect.StructField{}},
	}
	assert.Equal(t, expected, actual)
	assert.Equal(t, v.FieldByIndex(actual[0].Index).Int(), int64(13))
	assert.Equal(t, v.FieldByIndex(actual[1].Index).Int(), int64(12))
	assert.Equal(t, v.FieldByIndex(actual[2].Index).Int(), int64(14))
	assert.Equal(t, v.FieldByIndex(actual[3].Index).Int(), int64(15))
}

type TestDuplicateID struct {
	A int32 `protobuf:"1"`
	B int32 `protobuf:"1"`
}

func TestDuplicateIDNotAllowed(t *testing.T) {
	assert.Panics(t, func() {
		v := reflect.TypeOf(&TestDuplicateID{})
		protobuf.ProtoFields(v)
	})
}
