# Segmented Cache (scache) 

[![GoReportCard](https://goreportcard.com/badge/github.com/viant/scache)](https://goreportcard.com/report/github.com/viant/scache)
[![GoDoc](https://godoc.org/github.com/viant/scache?status.svg)](https://godoc.org/github.com/viant/scache)

This library is compatible with Go 1.11+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#motivation)
- [Introduction](#introduction)
- [Contibution](#contributing-to-bqtail)
- [License](#license)

## Motivation

The goal of this cache is to provide low latency, zero allocation cache that is able to retain recently used entries that can work 
with memory or memory mapped files. 


## Introduction

Segmented Cache is operationally zero allocation as it preallocate memory during initialization.
It consists two segments each taking half of the allocated memory:
 - Primary:  read/write active
 - Secondary: read only active
 
In case of entry miss in the active segment, service attempt to locate entry in the secondary segment to rewrite it to active one. 

Once active segment reaches limit of allocated memory, or optionally max entries, the secondary segment is promoted to the active, 
and the active is demoted to the secondary. 


This approach double effective memory, but does not require housekeeping on LRU algorithm overhead.
To boost write performance, every Set operation append data to the data pool, and old address is invalidated.   

This cache has been inspired by [BigCache](https://github.com/allegro/bigcache) and uses map[uint64]uint32 for hash ke to data address. mapping.
Using non pointers in the map make GC ommit map content. 

## Usage

```go

func CacheUsage() {
	var cacheUsageType int
	var cache scache.Service
	var err error
	switch cacheUsageType {
	case InMemoryExample:
		cache, err = scache.New(&scache.Config{SizeMb: 256})
	case MemoryMappedFileExample:
		cache, err = scache.New(&scache.Config{SizeMb: 256, Location: "/tmp/data.sch"})
	case InMemoryEntriesExample:
		cache, err = scache.New(&scache.Config{MaxEntries:5000000, EntrySize:128})
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

}
```

### Benchmark 

Benchmark with 256 payload on OSX (2.4 GHz 8-Core Intel Core i9), SSD


```bash
BenchmarkService_MemGet
BenchmarkService_MemGet-16             	 4000428	       303 ns/op	       7 B/op	       0 allocs/op
BenchmarkService_MMapGet
BenchmarkService_MMapGet-16            	 4093209	       305 ns/op	       7 B/op	       0 allocs/op
BenchmarkService_MMapGetParallel
BenchmarkService_MMapGetParallel-16    	23235844	        46.7 ns/op	       7 B/op	       0 allocs/op
BenchmarkService_MemGetParallel
BenchmarkService_MemGetParallel-16     	29515707	        40.3 ns/op	       7 B/op	       0 allocs/op
BenchmarkService_MemSet
BenchmarkService_MemSet-16             	 5508985	       237 ns/op
BenchmarkService_MMapSet
BenchmarkService_MMapSet-16            	 4018416	       322 ns/op
BenchmarkService_MemSetParallel
BenchmarkService_MemSetParallel-16     	15849211	        70.7 ns/op	       7 B/op	       0 allocs/op
BenchmarkService_MMapSetParallel
BenchmarkService_MMapSetParallel-16    	 2765222	       482 ns/op	       7 B/op	       0 allocs/op
PASS
```


## GoCover

[![GoCover](https://gocover.io/github.com/viant/scache)](https://gocover.io/github.com/viant/scache)


<a name="License"></a>
## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.

<a name="Credits-and-Acknowledgements"></a>

## Contributing to SCache

Scache is an open source project and contributors are welcome!

See [TODO](TODO.md) list

## Credits and Acknowledgements

**Library Author:** Adrian Witas

