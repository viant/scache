package scache

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

func TestCacheMultiOperation(t *testing.T) {

	rand.Seed(time.Now().Unix())
	maxEntries := 100000

	{
		cache, err := NewMemCache(0, maxEntries, 128)
		if !assert.Nil(t, err) {
			return
		}
		runMultiOperationTest(t, maxEntries, cache)
	}
	{
		cache, err := NewMemCache(12, maxEntries, 128)
		if !assert.Nil(t, err) {
			return
		}
		runMultiOperationTest(t, maxEntries, cache)
	}

}

func runMultiOperationTest(t *testing.T, maxEntries int, cache *Cache) {
	waitGroup := &sync.WaitGroup{}
	var runTest = func() {
		defer waitGroup.Done()
		for i := 0; i < maxEntries; i++ {
			r := int(rand.Uint32()) % maxEntries
			key := fmt.Sprintf("key:%v", r)
			value := fmt.Sprintf("val.%v", r)
			if _, err := cache.Get(key); err != nil {
				err = cache.Set(key, []byte(value))
				val, _ := cache.Get(key)
				assert.EqualValues(t, val, value)
			}
			err := cache.Delete(key)
			assert.Nil(t, err)
		}
	}

	for i := 0; i < 10; i++ {
		waitGroup.Add(1)
		runTest()
	}

	waitGroup.Wait()
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

func initCache(entries, entrySize int, location string) *Cache {
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
	b.ReportAllocs()
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



func TestConcurrency(t *testing.T) {
	c, _ := New(&Config{
		MaxEntries: 1024,
		Shards:     32768,
	})
	keys := make([]string, 1024*1024)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < 1024; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					key := keys[rand.Intn(len(keys))]
					if err := c.Set(key, []byte("v")); err != nil {
						t.Errorf("Set(%s, v): %v", key, err)
					}
					//since only 1K entries would be allowed at any time (per config), if may happen that, entry is gone if there were 1k  processing writes
					//thus not such key error may happens, other test Get operation,
					if _, err := c.Get(key); err != nil {
						//t.Errorf("Get(%s): %v", key, err)
					}
				case <-ctx.Done():

					return
				}
			}
		}()
	}
	wg.Wait()
}
