package scache

import (
	"github.com/dolthub/swiss"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_clear(t *testing.T) {
	aMap := swiss.NewMap[uint64, uint32](10)
	for i := 0; i < 10; i++ {
		aMap.Put(uint64(i), uint32(i))
	}
	for i := 0; i < 10; i++ {
		value, _ := aMap.Get(uint64(i))
		if value != uint32(i) {
			t.Errorf("Expected %d, got %d", i, value)
		}
	}
	clearSwissMap(aMap, make([]uint64, 10), make([]uint32, 10))
	for i := 0; i < 10; i++ {
		_, has := aMap.Get(uint64(i))
		assert.False(t, has)
	}
	for i := 0; i < 20; i++ {
		aMap.Put(uint64(i), uint32(i))
	}
	for i := 0; i < 20; i++ {
		value, _ := aMap.Get(uint64(i))
		if value != uint32(i) {
			t.Errorf("Expected %d, got %d", i, value)
		}
	}
}
