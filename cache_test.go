package scache

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNewCache(t *testing.T) {

	var useCases = []struct {
		description string
		config      *Config
		keys        int
		entrySize   int
	}{
		{
			description: "single key",
			entrySize:   12,
			config:      &Config{SizeMb: 1},
			keys:        1,
		},
		{
			description: "multi segment switch",
			entrySize:   1024,
			config:      &Config{SizeMb: 12},
			keys:        32 * 1024,
		},
		{
			description: "memory mapped file",
			entrySize:   1024,
			config:      &Config{SizeMb: 12, Location: "/tmp/scache"},
			keys:        32 * 1024,
		},
	}

	for _, useCase := range useCases {
		cache, err := New(useCase.config)
		assert.Nil(t, err)
		for i := 0; i < useCase.keys; i++ {
			key := fmt.Sprintf("key%v\n", i)
			_, err := cache.Get(key)
			assert.True(t, err != nil, useCase.description)
			data := strings.Repeat("xy", useCase.entrySize/2)
			cache.Set(key, []byte(data))
			actual, err := cache.Get(key)
			assert.True(t, err == nil, useCase.description)
			assert.EqualValues(t, data, string(actual), useCase.description)
			err = cache.Delete(key)
			assert.Nil(t, err)
			_, err = cache.Get(key)
			assert.True(t, err != nil, useCase.description)
		}
		cache.Close()
	}
}

func BenchmarkService_MemGet(b *testing.B) {
	readFromCache(b, []byte(strings.Repeat("?", 256)), "")
}

func BenchmarkService_MMapGet(b *testing.B) {
	readFromCache(b, []byte(strings.Repeat("?", 256)), "/tmp/scacheGet.mmap")
}

func readFromCache(b *testing.B, payload []byte, location string) {
	maxEntries := b.N
	if b.N > 10000000 {
		maxEntries = 10000000
	}

	cache := initCache(maxEntries, len(payload), location)
	for i := 0; i < maxEntries; i++ {
		cache.Set(strconv.Itoa(i), payload)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.ReportAllocs()
		cache.Get(strconv.Itoa(i % maxEntries))
	}

}

func BenchmarkService_MMapGetParallel(b *testing.B) {
	readFromCacheParallel(b, []byte(strings.Repeat("?", 256)), "/tmp/scacheGetP.mmap")
}

func BenchmarkService_MemGetParallel(b *testing.B) {
	readFromCacheParallel(b, []byte(strings.Repeat("?", 256)), "")
}

func readFromCacheParallel(b *testing.B, payload []byte, location string) {
	maxEntries := b.N
	if b.N > 10000000 {
		maxEntries = 10000000
	}
	cache := initCache(maxEntries, len(payload), location)
	for i := 0; i < maxEntries; i++ {
		cache.Set(strconv.Itoa(i), payload)
	}
	b.ResetTimer()
	counter := int64(-1)
	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		for pb.Next() {
			cache.Get(strconv.Itoa(int(atomic.AddInt64(&counter, 1)) % maxEntries))

		}
	})
}

func initCache(entries, entrySize int, location string) Service {
	cache, _ := New(&Config{
		Location:   location,
		Shards:     256,
		EntrySize:  entrySize,
		MaxEntries: 2 * entries,
	})
	return cache
}

func BenchmarkService_MemSet(b *testing.B) {
	writeToCache(b, []byte(strings.Repeat("?", 256)), "")
}

func BenchmarkService_MMapSet(b *testing.B) {
	writeToCache(b, []byte(strings.Repeat("?", 256)), "/tmp/scacheSet.mmap")
}

func writeToCache(b *testing.B, payload []byte, location string) {
	maxEntries := b.N
	if b.N > 10000000 {
		maxEntries = 10000000
	}

	cache := initCache(maxEntries, len(payload), location)
	b.ResetTimer()
	for i := 0; i < maxEntries; i++ {
		cache.Set(strconv.Itoa(i), payload)
	}

}

func BenchmarkService_MemSetParallel(b *testing.B) {
	writeToCacheParallel(b, []byte(strings.Repeat("?", 256)), "")
}

func BenchmarkService_MMapSetParallel(b *testing.B) {
	writeToCacheParallel(b, []byte(strings.Repeat("?", 256)), "/tmp/scacheSetP.mmap")
}

func writeToCacheParallel(b *testing.B, payload []byte, location string) {
	maxEntries := b.N
	if b.N > 10000000 {
		maxEntries = 10000000
	}
	cache := initCache(maxEntries, len(payload), location)
	b.ResetTimer()
	counter := int64(-1)
	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		for pb.Next() {
			cache.Set(strconv.Itoa(int(atomic.AddInt64(&counter, 1))%maxEntries), payload)

		}
	})
}
