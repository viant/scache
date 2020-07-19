package scache_test

import (
	"github.com/viant/scache"
	"log"
)

const (
	InMemoryExample = iota
	MemoryMappedFileExample
	InMemoryEntriesExample
)

func ExampleNew() {

	var cacheUsageType int
	var cache *scache.Cache
	var err error
	switch cacheUsageType {
	case InMemoryExample:
		cache, err = scache.New(&scache.Config{SizeMb: 256})
		//or
		cache, err = scache.NewMemCache(256, 0, 0)
	case MemoryMappedFileExample:
		cache, err = scache.New(&scache.Config{SizeMb: 256, Location: "/tmp/data.sch"})
		//or
		cache, err = scache.NewMmapCache("/tmp/data.sch", 256, 0, 0)
	case InMemoryEntriesExample:
		cache, err = scache.New(&scache.Config{MaxEntries: 5000000, EntrySize: 128})
	}
	if err != nil {
		log.Fatal(err)
	}
	err = cache.Set("keyX", []byte("some value"))
	if err != nil {
		log.Fatal(err)
	}
	value, err := cache.Get("keyX")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("value : %s\n", value)
	err = cache.Delete("keyX")
	if err != nil {
		log.Fatal(err)
	}
	cache.Close()

}
