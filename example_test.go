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
	var cache scache.Service
	var err error
	switch cacheUsageType {
	case InMemoryExample:
		cache, err = scache.New(&scache.Config{SizeMb: 256})
	case MemoryMappedFileExample:
		cache, err = scache.New(&scache.Config{SizeMb: 256, Location: "/tmp/data.sch"})
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
