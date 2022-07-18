package protobuf_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DmitriyMV/protobuf"
)

type Private struct {
	a int
}

type Public struct {
	A int
}

type Empty struct {
	Empty *string
}

func TestPrivate(t *testing.T) {
	private := Private{37}
	public := Public{37}
	str := "b"
	empty := Empty{&str}

	bufPrivate, errS := protobuf.Encode(&private)
	require.Error(t, errS)
	bufPublic := must(protobuf.Encode(&public))(t)
	bufEmpty := must(protobuf.Encode(&empty))(t)

	t.Log(hex.Dump(bufPrivate))
	t.Log(hex.Dump(bufPublic))
	t.Log(hex.Dump(bufEmpty))

	assert.Equal(t, []byte(nil), bufPrivate)
	assert.Equal(t, []byte{0x8, 0x25}, bufPublic)
	assert.Equal(t, []byte{0xa, 0x1, 0x62}, bufEmpty)
}
