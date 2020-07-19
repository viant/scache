package scache

import (
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
)

const segmentsSize = 2

//Service represents service
type Service interface {
	//Set sets key
	Set(key string, value []byte) error
	//Delete deletes the key
	Delete(key string) error
	//Delete deletes the key
	Get(key string) ([]byte, error)
	//Close closes the cache
	Close() error
}

type service struct {
	config   *Config
	data     []byte
	segments [segmentsSize]segment
	index    uint32
	mutex    sync.Mutex
	mmap     *mmap
	*shardedMap
}

func (s *service) nextIndex(idx uint32) uint32 {
	next := uint32(0)
	if idx == 0 {
		next = 1
	}
	return next
}

func (s *service) newShardedMap() *shardedMap {
	result := s.shardedMap
	s.shardedMap = nil
	if result == nil {
		result = newShardedMap(s.config)
	}
	go func() {
		// preallocate new keys space, for next segment switch
		s.shardedMap = newShardedMap(s.config)
	}()
	return result
}

//Set sets key with value or error
func (s *service) Set(key string, value []byte) error {
	idx := atomic.LoadUint32(&s.index)
	if !s.segments[idx].set(key, value) {
		nextIndex := s.nextIndex(idx)
		s.mutex.Lock()
		if currIdx := atomic.LoadUint32(&s.index); currIdx == idx {
			s.segments[nextIndex].reset(s.newShardedMap())
			atomic.StoreUint32(&s.index, nextIndex)
		}
		s.mutex.Unlock()
		if !s.segments[nextIndex].set(key, value) {
			return errors.Errorf("failed to set key: %v", key)
		}

	}
	return nil
}

//Delete deletes key in the cache
func (s *service) Delete(key string) error {
	idx := atomic.LoadUint32(&s.index)
	s.segments[idx].delete(key)
	return nil
}

//Get returns a cache entry for the supplied key or error
func (s *service) Get(key string) ([]byte, error) {
	idx := atomic.LoadUint32(&s.index)
	value, has := s.segments[idx].get(key)
	if !has {
		nextIndex := s.nextIndex(idx)
		if value, has = s.segments[nextIndex].get(key); has {
			s.segments[idx].set(key, value)
		}
	}
	if !has {
		return nil, noSuchKeyErr
	}
	return value, nil
}

//Close closes the service
func (s *service) Close() (err error) {
	for i := range s.segments {
		if e := s.segments[i].close(); e != nil {
			err = e
		}
	}
	return err
}

//New creates a service
func New(config *Config) (Service, error) {
	config.Init()
	var cache = &service{
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
