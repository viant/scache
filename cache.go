package scache

import (
	"fmt"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	segmentsSize     = 2
	maxSupportedSize = 256 * 1024
)

// Cache represents cache service
type Cache struct {
	config   *Config
	segments [segmentsSize]segment
	index    uint32
	mutex    sync.Mutex
	mmap     *mmap
	OnSegmentSwitch
	*shardedMap
}

func (s *Cache) nextIndex(idx uint32) uint32 {
	next := uint32(0)
	if idx == 0 {
		next = 1
	}
	return next
}

func (s *Cache) newShardedMap() *shardedMap {
	result := s.shardedMap
	s.shardedMap = nil
	if result == nil {
		result = newShardedMap(s.config)
	}
	go func() {
		s.mutex.Lock()
		// preallocate new keys space, for next segment switch
		s.shardedMap = newShardedMap(s.config)
		s.mutex.Unlock()
	}()
	return result
}

// Set sets key with value or error
func (s *Cache) Set(key string, value []byte) error {
	idx := atomic.LoadUint32(&s.index)
	_, isSet := s.segments[idx].set(key, value)
	if !isSet {
		nextIndex := s.nextIndex(idx)
		s.mutex.Lock()
		if currIdx := atomic.LoadUint32(&s.index); currIdx == idx {

			startTime := time.Now()
			fn := s.OnSegmentSwitch
			s.segments[nextIndex].reset()
			atomic.StoreUint32(&s.index, nextIndex)
			if fn != nil {
				fn(idx, atomic.LoadUint32(&s.segments[idx].keys), time.Now().Sub(startTime))
			}
		}
		s.mutex.Unlock()
		idx = atomic.LoadUint32(&s.index)
		if _, ok := s.segments[idx].set(key, value); !ok {
			return errors.Errorf("failed to set key: %v", key)
		}

	}
	return nil
}

// Delete deletes key in the cache
func (s *Cache) Delete(key string) error {
	idx := atomic.LoadUint32(&s.index)
	s.segments[idx].delete(key)
	return nil
}

// Get returns a cache entry for the supplied key or error
func (s *Cache) Get(key string) ([]byte, error) {
	idx := atomic.LoadUint32(&s.index)
	value, has := s.segments[idx].get(key)
	if !has { //if not found in the current segment find in secondary, when  found copy to primary
		nextIndex := s.nextIndex(idx)
		if value, has = s.segments[nextIndex].get(key); has {
			value, _ = s.segments[idx].set(key, value) //return buffer from primary  segment
		}
	}
	if !has {
		return nil, noSuchKeyErr
	}
	return value, nil
}

// Close closes the Cache
func (s *Cache) Close() (err error) {
	for i := range s.segments {
		if e := s.segments[i].close(); e != nil {
			err = e
		}
	}
	return err
}

// New creates a Cache
func New(config *Config) (*Cache, error) {
	config.Init()
	if config.SizeMb > maxSupportedSize {
		//given 32 bytes data alignment max addressable space is 256GB
		return nil, fmt.Errorf("exceeded max supported cache size: 256GB")
	}
	var cache = &Cache{
		config: config,
	}
	for i := range cache.segments {
		cache.segments[i].config = config
		cache.segments[i].shardedMap = newShardedMap(config)
		if err := cache.segments[i].allocate(i); err != nil {
			return nil, err
		}
	}
	cache.shardedMap = newShardedMap(config)
	return cache, nil
}

// NewMemCache creates a memory backed cache
func NewMemCache(sizeMb, maxEntries, entrySize int) (*Cache, error) {
	return New(&Config{SizeMb: sizeMb, EntrySize: entrySize, MaxEntries: maxEntries})
}

// NewMmapCache creates a memory mapped filed backed cache
func NewMmapCache(location string, sizeMb, maxEntries, entrySize int) (*Cache, error) {
	return New(&Config{Location: location, SizeMb: sizeMb, EntrySize: entrySize, MaxEntries: maxEntries})
}
