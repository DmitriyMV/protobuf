package protobuf_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/DmitriyMV/protobuf"
)

type Pass struct {
	Other []int
}

type Fail struct {
	Bytes []byte
}

func TestOptionalBytes(t *testing.T) {
	buffP, errP := protobuf.Encode(new(Pass))
	buffF, errF := protobuf.Encode(new(Fail))

	bytes := []byte{1, 2, 3}
	buffFP, errFP := protobuf.Encode(&Fail{bytes})

	t.Log(buffP, errP)
	t.Log(buffF, errF)
	t.Log(buffFP, errFP)

	require.Equal(t, buffF, buffP)
}
