package protobuf_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/DmitriyMV/protobuf"
)

type A struct {
	Value int
}

func (a *A) MarshalBinary() ([]byte, error) {
	res := protowire.AppendTag(nil, 1, protowire.VarintType)

	return protowire.AppendVarint(res, uint64(a.Value)), nil
}

func (a *A) Print() string {
	return ""
}

type B struct {
	AValue A
	AInt   int
}

func TestMarshal(t *testing.T) {
	a := A{-149}
	b := B{a, 300}

	bufA := must(protobuf.Encode(&a))(t)
	bufB := must(protobuf.Encode(&b))(t)

	t.Logf("%s", hex.Dump(bufA))
	t.Logf("%s", hex.Dump(bufB))

	testA := A{}
	testB := B{}

	require.NoError(t, protobuf.Decode(bufA, &testA))
	require.NoError(t, protobuf.Decode(bufB, &testB))

	assert.Equal(t, a, testA)
	assert.Equal(t, b, testB)
}
