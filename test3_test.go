package protobuf_test

import (
	"encoding/hex"
	"fmt"

	"github.com/DmitriyMV/protobuf"
)

type Test3 struct {
	_ interface{} // = 1
	_ interface{} // = 2
	C Test1       // = 3
}

// This example encodes the Test3 message illustrating embedded messages
// in the Protocol Buffers encoding specification.
func ExampleEncode_test3() {
	t := Test3{C: Test1{150}}
	buf := try(protobuf.Encode(&t))
	fmt.Print(hex.Dump(buf))

	// Output:
	// 00000000  1a 03 08 96 01                                    |.....|
}

func try[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}
