package scache

import (
	"github.com/dolthub/swiss"
	"sync"
)

// shardedMap represents sharded map
type shardedMap struct {
	config     Config
	lock       []sync.RWMutex
	maps       []*swiss.Map[uint64, uint32]
	hasher     fnv64a
	shardsHash uint64
}

func (m *shardedMap) getAddress(key string) uint64 {
	hashedKey := m.hasher.Sum64(key)
	index := hashedKey & m.shardsHash
	m.lock[index].RLock()
	if m.maps[index].Count() == 0 {
		m.lock[index].RUnlock()
		return 0
	}
	value, _ := m.maps[index].Get(hashedKey)
	m.lock[index].RUnlock()
	return uint64(value) << 5
}

func (m *shardedMap) put(key string, value uint32) bool {
	hashedKey := m.hasher.Sum64(key)
	index := hashedKey & m.shardsHash
	m.lock[index].Lock()
	_, has := m.maps[index].Get(hashedKey)
	m.maps[index].Put(hashedKey, value)
	m.lock[index].Unlock()
	return has
}

func (m *shardedMap) delete(key string) bool {
	hashedKey := m.hasher.Sum64(key)
	index := hashedKey & m.shardsHash
	m.lock[index].Lock()
	if m.maps[index].Count() == 0 {
		m.lock[index].Unlock()
		return false
	}
	value, _ := m.maps[index].Get(hashedKey)
	m.maps[index].Put(hashedKey, 0)
	m.lock[index].Unlock()
	return value > 0
}

func newShardedMap(config *Config) *shardedMap {
	if config.Shards == 0 {
		config.Shards = 100
	}
	aMap := &shardedMap{
		config:     *config,
		lock:       make([]sync.RWMutex, config.Shards),
		maps:       make([]*swiss.Map[uint64, uint32], config.Shards),
		shardsHash: config.Shards - 1,
	}
	for i := range aMap.maps {
		aMap.maps[i] = swiss.NewMap[uint64, uint32](uint32(config.shardMapSize))
	}
	return aMap
}
