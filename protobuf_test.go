package protobuf_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/DmitriyMV/protobuf"
)

//nolint:govet
type emb struct {
	I32 int32
	S   string
}

// test custom type-aliases.
type (
	mybool    bool
	myint32   int32
	myint64   int64
	myuint32  uint32
	myuint64  uint64
	myfloat32 float32
	myfloat64 float64
	mybytes   []byte
	mystring  string
)

//nolint:govet
type test struct {
	Bool   bool `protobuf:"boolean,opt"`
	I      int
	I32    int32
	I64    int64
	U32    uint32
	U64    uint64
	SX32   protobuf.Sfixed32
	SX64   protobuf.Sfixed64
	UX32   protobuf.Ufixed32
	UX64   protobuf.Ufixed64
	F32    float32
	F64    float64
	Bytes  []byte
	Array  [2]byte
	String string
	Struct emb

	OBool   *mybool `protobuf:"50"`
	OI32    *myint32
	OI64    *myint64
	OU32    *myuint32
	OU64    *myuint64
	OF32    *myfloat32
	OF64    *myfloat64
	OBytes  *mybytes
	OString *mystring
	OStruct *test

	SBool   []mybool `protobuf:"100"`
	SI32    []myint32
	SI64    []myint64
	SU32    []myuint32
	SU64    []myuint64
	SSX32   []protobuf.Sfixed32
	SSX64   []protobuf.Sfixed64
	SUX32   []protobuf.Ufixed32
	SUX64   []protobuf.Ufixed64
	SF32    []myfloat32
	SF64    []myfloat64
	SBytes  []mybytes
	SString []mystring
	SStruct []emb
}

func TestProtobuf(t *testing.T) {
	b0 := mybool(true)
	i1 := myint32(-1)
	i2 := myint64(-2)
	i3 := myuint32(3)
	i4 := myuint64(4)
	f5 := myfloat32(5.5)
	f6 := myfloat64(6.6)
	b7 := mybytes("789")
	s8 := mystring("ABC")
	e9 := test{Bytes: []byte{}}

	t1 := test{
		true,
		0,
		-1,
		-2,
		3,
		4,
		-11,
		-22,
		33,
		44,
		5.0,
		6.0,
		[]byte("789"),
		[2]byte{1, 2},
		"abc",
		emb{123, "def"},
		&b0,
		&i1,
		&i2,
		&i3,
		&i4,
		&f5,
		&f6,
		&b7,
		&s8,
		&e9,
		[]mybool{true, false, true},
		[]myint32{1, -2, 3},
		[]myint64{2, -3, 4},
		[]myuint32{3, 4, 5},
		[]myuint64{4, 5, 6},
		[]protobuf.Sfixed32{11, -22, 33},
		[]protobuf.Sfixed64{22, -33, 44},
		[]protobuf.Ufixed32{33, 44, 55},
		[]protobuf.Ufixed64{44, 55, 66},
		[]myfloat32{5.5, 6.6, 7.7},
		[]myfloat64{6.6, 7.7, 8.8},
		[]mybytes{[]byte("the"), []byte("quick"), []byte("brown"), []byte("fox")},
		[]mystring{"the", "quick", "brown", "fox"},
		[]emb{{-1, "a"}, {-2, "b"}, {-3, "c"}},
	}
	buf, err := protobuf.Encode(&t1)
	assert.NoError(t, err)

	t2 := test{}
	err = protobuf.Decode(buf, &t2)
	assert.NoError(t, err)
	assert.Equal(t, t1, t2)
}

//nolint:govet
type simpleFilledInput struct {
	Bytes []mybytes
	I     string
	Ptr   *mybool
}

func TestProtobuf_FilledInput(t *testing.T) {
	b0 := mybool(true)
	b1 := mybool(false)

	t1 := simpleFilledInput{
		[]mybytes{[]byte("the"), []byte("quick"), []byte("brown"), []byte("fox")},
		"intermediate value",
		&b0,
	}
	buf, err := protobuf.Encode(&t1)
	assert.NoError(t, err)

	t2 := simpleFilledInput{
		[]mybytes{[]byte("the"), []byte("quick"), []byte("brown"), []byte("fox")},
		"intermediate value",
		&b1,
	}
	err = protobuf.Decode(buf, &t2)
	assert.NoError(t, err)
	assert.Equal(t, t1, t2)

	t1 = simpleFilledInput{}
	buf, err = protobuf.Encode(&t1)
	assert.NoError(t, err)

	err = protobuf.Decode(buf, &t2)
	assert.NoError(t, err)
	assert.Equal(t, t1, t2)
}

type padded struct {
	Field1 int32    // = 1
	_      struct{} // = 2
	Field3 int32    // = 3
	_      int      // = 4
	Field5 int32    // = 5
}

func TestPadded(t *testing.T) {
	t1 := padded{}
	t1.Field1 = 10
	t1.Field3 = 30
	t1.Field5 = 50
	buf, err := protobuf.Encode(&t1)
	assert.NoError(t, err)

	t2 := padded{}
	if err = protobuf.Decode(buf, &t2); err != nil {
		panic(err.Error())
	}

	if t1 != t2 {
		panic("decode didn't reproduce identical struct")
	}
}

type TimeTypes struct {
	Time     time.Time
	Duration time.Duration
}

const shortForm = "2006-Jan-02"

func TestTimeTypesEncodeDecode(t *testing.T) {
	tt := must(time.Parse(shortForm, "2013-Feb-03"))(t)
	in := &TimeTypes{
		Time:     tt,
		Duration: time.Second * 30,
	}
	buf, err := protobuf.Encode(in)
	assert.NoError(t, err)

	out := &TimeTypes{}

	err = protobuf.Decode(buf, out)
	assert.NoError(t, err)
	assert.Equal(t, in.Time.UnixNano(), out.Time.UnixNano())
	assert.Equal(t, in.Duration, out.Duration)
}

/* encoding of testMsg is equivalent to the encoding to the following in
  a .proto file:
	message cipherText {
	  int32 a = 1;
	  int32 b = 2;
	  }

	  message MapFieldEntry {
	  uint32 key = 1;
	  cipherText value = 2;
	  }

	  message testMsg {
	  repeated MapFieldEntry map_field = 1;
	  }
  for details see:
https://developers.google.com/protocol-buffers/docs/proto#backwards-compatibility */
type wrongTestMsg struct {
	M map[uint32][]cipherText
}

type rightTestMsg struct {
	M map[uint32]*cipherText
}
type cipherText struct {
	A, B int32
}

func TestMapSliceStruct(t *testing.T) {
	cv := []cipherText{{}, {}}
	msg := &wrongTestMsg{
		M: map[uint32][]cipherText{1: cv},
	}

	_, err := protobuf.Encode(msg)
	assert.Error(t, err)

	msg2 := &rightTestMsg{
		M: map[uint32]*cipherText{1: {4, 5}},
	}

	buff, err := protobuf.Encode(msg2)
	assert.NoError(t, err)

	dec := &rightTestMsg{}
	err = protobuf.Decode(buff, dec)
	assert.NoError(t, err)

	assert.True(t, reflect.DeepEqual(dec, msg2))
}
