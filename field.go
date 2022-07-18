package protobuf

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/protobuf/encoding/protowire"
)

// ParseTag parses the protobuf tag of the given struct field.
func ParseTag(field reflect.StructField) (id int, name string) { //nolint:nonamedreturns
	tag := field.Tag.Get("protobuf")
	if tag == "" {
		return
	}

	parts := strings.Split(tag, ",")

	for _, part := range parts {
		i, err := strconv.Atoi(part)
		if err != nil {
			name = part
		} else {
			id = i
		}
	}

	return
}

// ProtoField contains cached reflected metadata for struct fields.
//nolint:govet
type ProtoField struct {
	ID    protowire.Number
	Name  string // If non-empty, tag-defined field name.
	Index []int
	Field reflect.StructField
}

var (
	cache     = map[reflect.Type][]*ProtoField{}
	cacheLock sync.Mutex
)

// ProtoFields returns a list of ProtoFields for the given struct type.
func ProtoFields(t reflect.Type) []*ProtoField {
	cacheLock.Lock()
	idx, ok := cache[t]
	cacheLock.Unlock()

	if ok {
		return idx
	}

	id := 0
	idx = innerFieldIndexes(&id, t)

	seen := map[protowire.Number]struct{}{}
	for _, i := range idx {
		if _, ok := seen[i.ID]; ok {
			panic(fmt.Sprintf("protobuf ID %d reused in %s.%s", i.ID, t.PkgPath(), t.Name()))
		}

		seen[i.ID] = struct{}{}
	}

	cacheLock.Lock()
	defer cacheLock.Unlock()

	cache[t] = idx

	return idx
}

func innerFieldIndexes(id *int, v reflect.Type) []*ProtoField {
	if v.Kind() == reflect.Ptr {
		return innerFieldIndexes(id, v.Elem())
	}

	out := []*ProtoField{}

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		*id++

		tid, name := ParseTag(f)
		if tid != 0 {
			*id = tid
		}

		if f.Anonymous {
			*id--
			for _, inner := range innerFieldIndexes(id, f.Type) {
				inner.Index = append([]int{i}, inner.Index...)
				out = append(out, inner)
			}
		} else {
			out = append(out, &ProtoField{
				ID:    protowire.Number(*id),
				Name:  name,
				Index: []int{i},
				Field: f,
			})
		}
	}

	return out
}
