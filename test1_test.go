package protobuf_test

import (
	"encoding/hex"
	"fmt"

	"github.com/DmitriyMV/protobuf"
)

type Test1 struct {
	A uint32
}

// This example encodes the simple message defined at the start of
// the Protocol Buffers encoding specification:
// https://developers.google.com/protocol-buffers/docs/encoding
func ExampleEncode_test1() {
	t := Test1{150}
	buf := try(protobuf.Encode(&t))
	fmt.Print(hex.Dump(buf))

	// Output:
	// 00000000  08 96 01                                          |...|
}
