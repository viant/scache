package scache

import (
	"fmt"
	"github.com/dolthub/swiss"
	"github.com/mhmtszr/concurrent-swiss-map/maphash"
	"github.com/viant/xunsafe"
	"reflect"
	"unsafe"
)

const (
	groupSize      = 8
	empty     int8 = -128 // 0b1000_0000
)

type metadata [groupSize]int8
type group[K comparable, V any] struct {
	keys   [groupSize]K
	values [groupSize]V
}

type SwissMap[K comparable, V any] struct {
	ctrl     []metadata
	groups   []group[K, V]
	hash     maphash.Hasher[K]
	resident uint32
	dead     uint32
	limit    uint32
}

var emptyControl = make([]metadata, 1000)

func init() {
	for i := range emptyControl {
		emptyControl[i] = newEmptyMetadata()
	}

	ensureSwissMapType()

}

func ensureSwissMapType() {
	destStructType := reflect.TypeOf(SwissMap[uint64, uint32]{})
	srcStructType := reflect.TypeOf(swiss.Map[uint64, uint32]{})
	ensureAssignable("SwissMap", destStructType, srcStructType)
}

func ensureAssignable(fieldName string, destFieldType reflect.Type, srcFieldType reflect.Type) {
	switch destFieldType.Kind() {
	case reflect.Slice:
		destFieldType = destFieldType.Elem()
		srcFieldType = srcFieldType.Elem()
		ensureAssignable(fieldName, destFieldType, srcFieldType)
	case reflect.Ptr:
		destFieldType = destFieldType.Elem()
		srcFieldType = srcFieldType.Elem()
		ensureAssignable(fieldName, destFieldType, srcFieldType)
	case reflect.Array:
		destFieldType = destFieldType.Elem()
		srcFieldType = srcFieldType.Elem()
		if destFieldType != srcFieldType && !destFieldType.ConvertibleTo(srcFieldType) {
			panic(fmt.Sprintf("SwissMap and swiss.Map have different field %v %v %v\n", fieldName, destFieldType, srcFieldType))
		}
	case reflect.Struct:
		var destStruct = xunsafe.NewStruct(srcFieldType)
		var srcStruct = xunsafe.NewStruct(destFieldType)
		if len(destStruct.Fields) != len(srcStruct.Fields) {
			panic("SwissMap and swiss.Map have different fields")
		}
		for i := range destStruct.Fields {
			destFieldType := destStruct.Fields[i].Type
			srcFieldType := srcStruct.Fields[i].Type
			if destFieldType != srcFieldType {
				ensureAssignable(destStruct.Fields[i].Name, destFieldType, srcFieldType)
			}
		}
	default:
		if destFieldType != srcFieldType && !destFieldType.ConvertibleTo(srcFieldType) {
			panic(fmt.Sprintf("SwissMap and swiss.Map have different field %v %v %v\n", fieldName, destFieldType, srcFieldType))
		}
	}
}

// Reset removes all elements from the Map.
func clearSwissMap[K comparable, V any](aMap *swiss.Map[K, V], emptyK []K, emptyV []V) {

	m := (*SwissMap[K, V])(unsafe.Pointer(aMap))
	if m.resident == 0 && m.dead == 0 {
		return
	}
	for i := 0; i < len(m.ctrl); i += len(emptyControl) {
		if i+len(emptyControl) < len(m.ctrl) {
			copy(m.ctrl[i:i+len(emptyControl)], emptyControl)
		} else {
			copy(m.ctrl[i:], emptyControl[:len(m.ctrl)-i])
		}
	}

	// Reset keys and values in chunks using copy
	emptyKLen := len(emptyK)
	emptyVLen := len(emptyV)
	for _, g := range m.groups {
		// Reset keys
		keysLen := len(g.keys)
		for k := 0; k < keysLen; k += emptyKLen {
			if k+emptyKLen < keysLen {
				copy(g.keys[k:k+emptyKLen], emptyK)
				copy(g.values[k:k+emptyVLen], emptyV)
			} else {
				copy(g.keys[k:], emptyK[:keysLen-k])
				copy(g.values[k:], emptyV[:keysLen-k])
			}
		}
	}
	m.resident, m.dead = 0, 0
}

func newEmptyMetadata() (meta metadata) {
	for i := range meta {
		meta[i] = empty
	}
	return
}

var keys = make([]uint64, 1000)
var values = make([]uint32, 1000)
