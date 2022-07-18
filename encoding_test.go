package protobuf_test

import (
	"encoding"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/DmitriyMV/protobuf"
)

type Number interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	Value() int
}

type Int struct {
	N int
}

type Wrapper struct {
	N Number
}

func NewNumber(n int) Number {
	return &Int{n}
}

func (i *Int) Value() int {
	return i.N
}

func (i *Int) MarshalBinary() ([]byte, error) {
	return protowire.AppendVarint(nil, uint64(i.N)), nil
}

func (i *Int) UnmarshalBinary(data []byte) error {
	res, _ := protowire.ConsumeVarint(data)
	i.N = int(res)

	return nil
}

// Check at compile time that we satisfy the interfaces.
var _ encoding.BinaryMarshaler = (*Int)(nil)
var _ encoding.BinaryUnmarshaler = (*Int)(nil)

// Validate that support for self-encoding via the Encoding
// interface works as expected.
func TestBinaryMarshaler(t *testing.T) {
	wrapper := Wrapper{NewNumber(99)}
	buf, err := protobuf.Encode(&wrapper)
	assert.Nil(t, err)

	wrapper2 := Wrapper{NewNumber(0)}
	err = protobuf.Decode(buf, &wrapper2)

	assert.Nil(t, err)
	assert.Equal(t, 99, wrapper2.N.Value())
}

type NumberNoMarshal interface {
	Value() int
}

func NewNumberNoMarshal(n int) NumberNoMarshal {
	return &IntNoMarshal{n}
}

type IntNoMarshal struct {
	N int
}

func (i *IntNoMarshal) Value() int {
	return i.N
}

type WrapperNoMarshal struct {
	N NumberNoMarshal
}

func TestNoBinaryMarshaler(t *testing.T) {
	wrapper := WrapperNoMarshal{NewNumberNoMarshal(99)}
	buf, err := protobuf.Encode(&wrapper)
	assert.Nil(t, err)

	wrapper2 := WrapperNoMarshal{NewNumberNoMarshal(0)}
	err = protobuf.Decode(buf, &wrapper2)

	assert.Nil(t, err)
	assert.Equal(t, 99, wrapper2.N.Value())
}

type WrongSliceInt struct {
	Ints [][]int
}
type WrongSliceUint struct {
	UInts [][]uint16
}

func TestNo2dSlice(t *testing.T) {
	w := &WrongSliceInt{}
	w.Ints = [][]int{{1, 2, 3}, {4, 5, 6}}
	_, err := protobuf.Encode(w)
	assert.NotNil(t, err)

	w2 := &WrongSliceUint{}
	w2.UInts = [][]uint16{{1, 2, 3}, {4, 5, 6}}
	_, err = protobuf.Encode(w2)
	assert.NotNil(t, err)
}

type T struct {
	Buf1, Buf2 []byte
}

func TestByteOverwrite(t *testing.T) {
	t0 := T{
		Buf1: []byte("abc"),
		Buf2: []byte("def"),
	}
	buf, err := protobuf.Encode(&t0)
	assert.Nil(t, err)

	var t1 T
	err = protobuf.Decode(buf, &t1)
	assert.Nil(t, err)

	assert.Equal(t, []byte("abc"), t1.Buf1)
	assert.Equal(t, []byte("def"), t1.Buf2)

	// now we trigger the bug that used to exist, by writing off the end of
	// Buf1, over where the size was (the g and h) and onto the top of Buf2.
	b1 := append(t1.Buf1, 'g', 'h', 'i') //nolint:gocritic
	assert.Equal(t, []byte("abcghi"), b1)
	// Buf2 must be unchanged, even though Buf1 was written to. When the bug
	// was present, Buf2 turns into "ief".
	assert.Equal(t, []byte("def"), t1.Buf2)

	// With the fix in place, the capacities must match the lengths.
	assert.Equal(t, len(t1.Buf1), cap(t1.Buf1))
	assert.Equal(t, len(t1.Buf2), cap(t1.Buf2))
}

type wrapper struct {
	Int *big.Int
}

var (
	zero   = new(big.Int)
	negone = new(big.Int).SetInt64(-1)
)

func (w *wrapper) MarshalBinary() ([]byte, error) {
	sign := []byte{0}
	if w.Int.Cmp(zero) < 0 {
		sign[0] = 1
	}

	return append(sign, w.Int.Bytes()...), nil
}

func (w *wrapper) UnmarshalBinary(in []byte) error {
	if len(in) < 1 {
		w.Int.SetInt64(0)

		return nil
	}

	w.Int.SetBytes(in[1:])

	if in[0] != 0 {
		w.Int.Mul(w.Int, negone)
	}

	return nil
}

func TestBigInt(t *testing.T) {
	v := wrapper{Int: new(big.Int)}
	v2 := wrapper{Int: new(big.Int)}

	v.Int.SetUint64(99)
	buf, err := protobuf.Encode(&v)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0, 99}, buf)
	err = protobuf.Decode(buf, &v2)
	assert.NoError(t, err)
	assert.Equal(t, "99", v2.Int.String())

	v.Int.SetInt64(-99)
	buf, err = protobuf.Encode(&v)
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 99}, buf)
	err = protobuf.Decode(buf, &v2)
	assert.NoError(t, err)
	assert.Equal(t, "-99", v2.Int.String())

	v.Int.SetString("238756834756284658865287462349857298752354", 10)
	buf, err = protobuf.Encode(&v)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x0, 0x2, 0xbd, 0xa4, 0xad, 0xbf, 0x98, 0xbd, 0x70, 0x26, 0xbd, 0x3b, 0x65, 0xe8, 0xae, 0xf3, 0xfa, 0xa3, 0x62}, buf)
	err = protobuf.Decode(buf, &v2)
	assert.NoError(t, err)
	assert.Equal(t, "238756834756284658865287462349857298752354", v2.Int.String())
}

func TestMapVsGeneratedMap(t *testing.T) {
	m := map[string]bool{
		"test": true,
	}

	r := TestRequest{Something: m}

	buf, err := r.MarshalVT()
	assert.NoError(t, err)

	type typ struct {
		M map[string]bool
	}

	t0 := &typ{M: m}

	buf2, err := protobuf.Encode(t0)
	assert.NoError(t, err)

	assert.Equal(t, buf, buf2)
	fmt.Println(hex.Dump(buf))
	fmt.Println(hex.Dump(buf2))
}

func TestStringKey(t *testing.T) {
	type typ struct {
		M map[string]bool
	}

	t0 := &typ{M: map[string]bool{}}

	var k1 string

	k2 := "test"
	k3 := "another"

	t0.M[k1] = true
	t0.M[k2] = true

	buf, err := protobuf.Encode(t0)
	assert.NoError(t, err)
	fmt.Println(hex.Dump(buf))
	assert.Equal(t, 16, len(buf))

	var t1 typ
	err = protobuf.Decode(buf, &t1)
	assert.NoError(t, err)
	assert.True(t, t1.M[k1])
	assert.True(t, t1.M[k2])
	assert.False(t, t1.M[k3])
}

func TestArrayKey(t *testing.T) {
	type typ struct {
		M map[[4]byte]bool
	}

	t0 := &typ{M: make(map[[4]byte]bool)}

	var k1 [4]byte

	k2 := [4]byte{0, 1, 2, 3}
	k3 := [4]byte{5, 6, 7, 8}

	t0.M[k1] = true
	t0.M[k2] = true

	buf, err := protobuf.Encode(t0)
	assert.NoError(t, err)
	assert.Equal(t, 20, len(buf))

	var t1 typ
	err = protobuf.Decode(buf, &t1)
	assert.NoError(t, err)
	assert.True(t, t1.M[k1])
	assert.True(t, t1.M[k2])
	assert.False(t, t1.M[k3])
}

type dummyInterface interface {
	String() string
	encoding.BinaryUnmarshaler
	protobuf.InterfaceMarshaler
}

type dummyStruct struct{}

func (ds *dummyStruct) String() string {
	return "dummy"
}

func (ds *dummyStruct) MarshalBinary() ([]byte, error) {
	return []byte{1, 2, 3}, nil
}

const unmarshalErr = "fail to unmarshal"

func (ds *dummyStruct) UnmarshalBinary(data []byte) error {
	return errors.New(unmarshalErr)
}

func (ds *dummyStruct) MarshalID() [8]byte {
	return [8]byte{'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a'}
}

type dummyWrapper struct {
	D dummyInterface
}

// TestInterface_UnknownType checks that proper errors are returned in
// the worst case scenario.
func TestInterface_UnknownType(t *testing.T) {
	w := &dummyWrapper{D: &dummyStruct{}}
	buf, err := protobuf.Encode(w)

	// encoding doesn't fail because it's the default constructor case
	require.NoError(t, err)
	require.NotNil(t, buf)
	require.Equal(t, "0a03010203", fmt.Sprintf("%x", buf))

	var r dummyWrapper
	err = protobuf.Decode(buf, &r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no constructor")

	// this time there is a tag at encoding time
	protobuf.RegisterInterface(func() interface{} { return &dummyStruct{} })
	buf, err = protobuf.Encode(w)
	require.NoError(t, err)
	require.NotNil(t, buf)
	require.Equal(t, "0a0b6161616161616161010203", fmt.Sprintf("%x", buf))

	// but the data is corrupted for some reason
	err = protobuf.Decode(buf, &r)
	require.Error(t, err)
	require.Contains(t, err.Error(), unmarshalErr)

	// but not at decoding time
	defer protobuf.SetGenerators(protobuf.GetGenerators())
	protobuf.SetGenerators(protobuf.NewInterfaceRegistry())

	r = dummyWrapper{}
	err = protobuf.Decode(buf, &r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no constructor")
}

// a type that can marshal itself, to be used inside of another struct.
type canMarshal struct{ private string }

type hasInternalCanMarshal struct {
	CM            canMarshal
	SomethingElse int
}

func (cm canMarshal) MarshalBinary() ([]byte, error) {
	return []byte(cm.private), nil
}

func (cm *canMarshal) UnmarshalBinary(data []byte) error {
	cm.private = string(data)

	return nil
}

func Test_InternalStructMarshal(t *testing.T) {
	v := hasInternalCanMarshal{
		CM:            canMarshal{private: "hello nurse"},
		SomethingElse: 99,
	}

	var v2 hasInternalCanMarshal

	buf, err := protobuf.Encode(&v)
	assert.NoError(t, err)
	err = protobuf.Decode(buf, &v2)
	assert.NoError(t, err)

	assert.Equal(t, v, v2)
}
