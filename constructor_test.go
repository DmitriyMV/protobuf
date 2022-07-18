package protobuf_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/DmitriyMV/protobuf"
)

func TestConstructorString(t *testing.T) {
	c := &protobuf.Constructors{
		reflect.TypeOf(int64(0)): func() interface{} { return int64(0) },
	}
	if !strings.HasPrefix(c.String(), "int64=>(func() interface {}") {
		t.Fatal("unexpected constructor string: ", c)
	}
}
