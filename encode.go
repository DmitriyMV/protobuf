// Package protobuf implements Protocol Buffers reflectively
// using Go types to define message formats.
package protobuf

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

// Ufixed32 - message fields declared to have exactly this type
// will be transmitted as fixed-size 32-bit unsigned integers.
type Ufixed32 uint32

// Ufixed64 - message fields declared to have exactly this type
// will be transmitted as fixed-size 64-bit unsigned integers.
type Ufixed64 uint64

// Sfixed32 - message fields declared to have exactly this type
// will be transmitted as fixed-size 32-bit signed integers.
type Sfixed32 int32

// Sfixed64 - message fields declared to have exactly this type
// will be transmitted as fixed-size 64-bit signed integers.
type Sfixed64 int64

type encoder struct {
	bytes.Buffer
}

// Encode a Go struct into protocol buffer format.
// The caller must pass a pointer to the struct to encode.
func Encode(structPtr interface{}) (result []byte, err error) { //nolint:nonamedreturns
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
			result = nil
		}
	}()

	if structPtr == nil {
		return nil, nil
	}

	if bu, ok := structPtr.(encoding.BinaryMarshaler); ok {
		return bu.MarshalBinary()
	}

	en := encoder{}

	val := reflect.ValueOf(structPtr)
	if val.Kind() != reflect.Ptr {
		return nil, errors.New("encode takes a pointer to struct")
	}

	en.message(val.Elem())

	return en.Bytes(), nil
}

func (en *encoder) message(sval reflect.Value) {
	var index *ProtoField

	defer func() {
		if r := recover(); r != nil {
			if index != nil {
				panic(fmt.Sprintf("%s (field %s)", r, index.Field.Name))
			} else {
				panic(r)
			}
		}
	}()
	// Encode all fields in-order

	protoFields := ProtoFields(sval.Type())
	if len(protoFields) == 0 {
		return
	}

	noPublicFields := true

	for _, index = range protoFields {
		field := sval.FieldByIndex(index.Index)
		key := index.ID

		if field.CanSet() { // Skip blank/padding fields
			en.value(key, field)

			noPublicFields = false
		}
	}

	if noPublicFields {
		panic("struct has no serializable fields")
	}
}

var timeType = reflect.TypeOf(time.Time{})

var (
	boolType    = reflect.TypeOf(false)
	intType     = reflect.TypeOf(0)
	int32Type   = reflect.TypeOf(int32(0))
	int64Type   = reflect.TypeOf(int64(0))
	uintType    = reflect.TypeOf(uint(0))
	uint32Type  = reflect.TypeOf(uint32(0))
	uint64Type  = reflect.TypeOf(uint64(0))
	float32Type = reflect.TypeOf(float32(0))
	float64Type = reflect.TypeOf(float64(0))
	stringType  = reflect.TypeOf("")
)

//nolint:gocyclo,cyclop,maintidx
func (en *encoder) value(key protowire.Number, val reflect.Value) {
	// Non-reflectively handle some of the fixed types
	switch val.Type() {
	case boolType:
		en.keyTag(key, protowire.VarintType)
		en.uvarint(protowire.EncodeBool(val.Bool()))

		return

	case intType, int32Type, int64Type:
		en.keyTag(key, protowire.VarintType)
		en.svarint(val.Int())

		return

	case timeType: // Encode time.Time as sfixed64
		t := val.Interface().(time.Time).UnixNano() //nolint:forcetypeassert

		en.keyTag(key, protowire.Fixed64Type)
		en.u64(uint64(t))

		return

	case uintType, uint32Type, uint64Type:
		en.keyTag(key, protowire.VarintType)
		en.uvarint(val.Uint())

		return

	case sfixed32type:
		en.keyTag(key, protowire.Fixed32Type)
		en.u32(uint32(val.Int()))

		return

	case sfixed64type:
		en.keyTag(key, protowire.Fixed64Type)
		en.u64(uint64(val.Int()))

		return

	case ufixed32type:
		en.keyTag(key, protowire.Fixed32Type)
		en.u32(uint32(val.Uint()))

		return

	case ufixed64type:
		en.keyTag(key, protowire.Fixed64Type)
		en.u64(val.Uint())

		return

	case float32Type:
		en.keyTag(key, protowire.Fixed32Type)
		en.u32(math.Float32bits(float32(val.Float())))

		return

	case float64Type:
		en.keyTag(key, protowire.Fixed64Type)
		en.u64(math.Float64bits(val.Float()))

		return

	case stringType:
		en.keyTag(key, protowire.BytesType)

		b := []byte(val.String())

		en.uvarint(uint64(len(b)))
		en.Write(b)

		return
	}

	// Handle pointer or interface values (possibly within slices).
	// Note that this switch has to handle all the cases,
	// because custom type aliases will fail the above typeswitch.
	switch val.Kind() { //nolint:exhaustive
	case reflect.Bool:
		en.keyTag(key, protowire.VarintType)

		v := uint64(0)
		if val.Bool() {
			v = 1
		}

		en.uvarint(v)

	case reflect.Int, reflect.Int32, reflect.Int64:
		// Varint-encoded 32-bit and 64-bit signed integers.
		// Note that protobufs don't support 8- or 16-bit ints.
		en.keyTag(key, protowire.VarintType)
		en.svarint(val.Int())

	case reflect.Uint32, reflect.Uint64:
		// Varint-encoded 32-bit and 64-bit unsigned integers.
		en.keyTag(key, protowire.VarintType)
		en.uvarint(val.Uint())

	case reflect.Float32:
		// Fixed-length 32-bit floats.
		en.keyTag(key, protowire.Fixed32Type)
		en.u32(math.Float32bits(float32(val.Float())))

	case reflect.Float64:
		// Fixed-length 64-bit floats.
		en.keyTag(key, protowire.Fixed64Type)
		en.u64(math.Float64bits(val.Float()))

	case reflect.String:
		// Length-delimited string.
		en.keyTag(key, protowire.BytesType)

		b := []byte(val.String())

		en.uvarint(uint64(len(b)))
		en.Write(b)

	case reflect.Struct:
		var b []byte

		enc, ok := getEncoder(val)
		if ok {
			en.keyTag(key, protowire.BytesType)

			var err error

			b, err = enc.MarshalBinary()
			if err != nil {
				panic(err.Error())
			}
		} else {
			// Embedded messages.
			en.keyTag(key, protowire.BytesType)
			emb := encoder{}
			emb.message(val)
			b = emb.Bytes()
		}

		en.uvarint(uint64(len(b)))
		en.Write(b)
	case reflect.Slice, reflect.Array:
		// Length-delimited slices or byte-vectors.
		en.slice(key, val)

		return

	case reflect.Ptr:
		if val.IsNil() {
			return
		}

		en.value(key, val.Elem())

	case reflect.Interface:
		// Abstract interface field.
		if val.IsNil() {
			return
		}

		// If the object support self-encoding, use that.
		if enc, ok := val.Interface().(encoding.BinaryMarshaler); ok {
			en.keyTag(key, protowire.BytesType)

			bytes, err := enc.MarshalBinary()
			if err != nil {
				panic(err.Error())
			}

			size := len(bytes)

			var id GeneratorID

			im, ok := val.Interface().(InterfaceMarshaler)

			if ok {
				id = im.MarshalID()

				g := generators.get(id)

				ok = g != nil
				if ok {
					// add the length of the type tag
					size += len(id)
				}
			}

			en.uvarint(uint64(size))

			if ok {
				// Only write the tag if a generator exists
				en.Write(id[:])
			}

			en.Write(bytes)

			return
		}

		// Encode from the object the interface points to.
		en.value(key, val.Elem())

	case reflect.Map:
		en.handleMap(key, val)

		return

	default:
		panic(fmt.Sprintf("unsupported field Kind %d", val.Kind()))
	}
}

func getEncoder(val reflect.Value) (encoding.BinaryMarshaler, bool) {
	if enc, ok := val.Interface().(encoding.BinaryMarshaler); ok {
		return enc, true
	}

	if val.CanAddr() {
		if enc, ok := val.Addr().Interface().(encoding.BinaryMarshaler); ok {
			return enc, true
		}
	}

	return nil, false
}

//nolint:gocyclo,cyclop
func (en *encoder) slice(key protowire.Number, slval reflect.Value) {
	// First handle common cases with a direct typeswitch
	sllen := slval.Len()
	packed := encoder{}

	switch slt := slval.Interface().(type) {
	case []bool:
		for i := 0; i < sllen; i++ {
			v := uint64(0)
			if slt[i] {
				v = 1
			}

			packed.uvarint(v)
		}

	case []int32:
		for i := 0; i < sllen; i++ {
			packed.svarint(int64(slt[i]))
		}

	case []int64:
		for i := 0; i < sllen; i++ {
			packed.svarint(slt[i])
		}

	case []uint32:
		for i := 0; i < sllen; i++ {
			packed.uvarint(uint64(slt[i]))
		}

	case []uint64:
		for i := 0; i < sllen; i++ {
			packed.uvarint(slt[i])
		}

	case []Sfixed32:
		for i := 0; i < sllen; i++ {
			packed.u32(uint32(slt[i]))
		}

	case []Sfixed64:
		for i := 0; i < sllen; i++ {
			packed.u64(uint64(slt[i]))
		}

	case []Ufixed32:
		for i := 0; i < sllen; i++ {
			packed.u32(uint32(slt[i]))
		}

	case []Ufixed64:
		for i := 0; i < sllen; i++ {
			packed.u64(uint64(slt[i]))
		}

	case []float32:
		for i := 0; i < sllen; i++ {
			packed.u32(math.Float32bits(slt[i]))
		}

	case []float64:
		for i := 0; i < sllen; i++ {
			packed.u64(math.Float64bits(slt[i]))
		}

	case []byte: // Write the whole byte-slice as one key,value pair
		en.keyTag(key, protowire.BytesType)
		en.uvarint(uint64(sllen))
		en.Write(slt)

		return

	case []string:
		for i := 0; i < sllen; i++ {
			subVal := slval.Index(i)
			subStr := subVal.Interface().(string) //nolint:errcheck,forcetypeassert
			subSlice := []byte(subStr)

			en.keyTag(key, protowire.BytesType)
			en.uvarint(uint64(len(subSlice)))
			en.Write(subSlice)
		}

		return
	default: // We'll need to use the reflective path
		en.sliceReflect(key, slval)

		return
	}

	// Encode packed representation key/value pair
	en.keyTag(key, protowire.BytesType)

	b := packed.Bytes()

	en.uvarint(uint64(len(b)))
	en.Write(b)
}

// Handle the encoding of an arbritrary map[K]V.
func (en *encoder) handleMap(key protowire.Number, mpval reflect.Value) {
	/*
		A map defined as
			map<key_type, value_type> map_field = N;
		is encoded in the same way as
			message MapFieldEntry {
				key_type key = 1;
				value_type value = 2;
			}
			repeated MapFieldEntry map_field = N;
	*/
	for _, mkey := range mpval.MapKeys() {
		mval := mpval.MapIndex(mkey)

		// illegal map entry values
		// - nil message pointers.
		switch kind := mval.Kind(); kind { //nolint:exhaustive
		case reflect.Ptr:
			if mval.IsNil() {
				panic("proto: map has nil element")
			}
		case reflect.Slice, reflect.Array:
			if mval.Type().Elem().Kind() != reflect.Uint8 {
				panic("protobuf: map only support []byte or string as repeated value")
			}
		}

		packed := encoder{}
		packed.value(1, mkey)
		packed.value(2, mval)

		en.keyTag(key, protowire.BytesType)

		b := packed.Bytes()

		en.uvarint(uint64(len(b)))
		en.Write(b)
	}
}

var bytesType = reflect.TypeOf([]byte{})

//nolint:gocognit,gocyclo,cyclop
func (en *encoder) sliceReflect(key protowire.Number, slval reflect.Value) {
	kind := slval.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		panic("no slice passed")
	}

	sllen := slval.Len()
	slelt := slval.Type().Elem()
	packed := encoder{}

	switch slelt.Kind() { //nolint:exhaustive
	case reflect.Bool:
		for i := 0; i < sllen; i++ {
			v := uint64(0)
			if slval.Index(i).Bool() {
				v = 1
			}

			packed.uvarint(v)
		}

	case reflect.Int, reflect.Int32, reflect.Int64:
		for i := 0; i < sllen; i++ {
			packed.svarint(slval.Index(i).Int())
		}

	case reflect.Uint32, reflect.Uint64:
		for i := 0; i < sllen; i++ {
			packed.uvarint(slval.Index(i).Uint())
		}

	case reflect.Float32:
		for i := 0; i < sllen; i++ {
			packed.u32(math.Float32bits(
				float32(slval.Index(i).Float())))
		}

	case reflect.Float64:
		for i := 0; i < sllen; i++ {
			packed.u64(math.Float64bits(slval.Index(i).Float()))
		}

	case reflect.Uint8: // Write the byte-slice as one key,value pair
		en.keyTag(key, protowire.BytesType)
		en.uvarint(uint64(sllen))

		var b []byte

		if slval.Kind() == reflect.Array {
			if slval.CanAddr() {
				sliceVal := slval.Slice(0, sllen)
				b = sliceVal.Convert(bytesType).Interface().([]byte) //nolint:errcheck,forcetypeassert
			} else {
				sliceVal := reflect.MakeSlice(bytesType, sllen, sllen)
				reflect.Copy(sliceVal, slval)
				b = sliceVal.Interface().([]byte) //nolint:errcheck,forcetypeassert
			}
		} else {
			b = slval.Convert(bytesType).Interface().([]byte) //nolint:errcheck,forcetypeassert
		}

		en.Write(b)

		return

	default: // Write each element as a separate key,value pair
		t := slval.Type().Elem()
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			subSlice := t.Elem()
			if subSlice.Kind() != reflect.Uint8 {
				panic("protobuf: no support for 2-dimensional array except for [][]byte")
			}
		}

		for i := 0; i < sllen; i++ {
			en.value(key, slval.Index(i))
		}

		return
	}

	// Encode packed representation key/value pair
	en.keyTag(key, protowire.BytesType)

	b := packed.Bytes()

	en.uvarint(uint64(len(b)))
	en.Write(b)
}

func (en *encoder) uvarint(v uint64) {
	var b [binary.MaxVarintLen64]byte

	en.Write(protowire.AppendVarint(b[:0], v))
}

func (en *encoder) svarint(v int64) {
	en.uvarint(uint64(v))
}

func (en *encoder) keyTag(key protowire.Number, varintType protowire.Type) {
	encoded := protowire.EncodeTag(key, varintType)
	en.uvarint(encoded)
}

func (en *encoder) u32(v uint32) {
	var b [4]byte

	en.Write(protowire.AppendFixed32(b[:0], v))
}

func (en *encoder) u64(v uint64) {
	var b [8]byte

	en.Write(protowire.AppendFixed64(b[:0], v))
}

// 0a 02 08 00 10 02
// 0a 01 00 10 02
