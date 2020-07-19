package scache

import (
	"sync"
)

//shardedMap represents sharded map
type shardedMap struct {
	config *Config
	lock   []sync.RWMutex
	maps   []map[uint64]uint32
	hasher fnv64a
}

func (m *shardedMap) get(key string) (uint32, bool) {
	hashedKey := m.hasher.Sum64(key)
	index := hashedKey & uint64(m.config.Shards-1)
	m.lock[index].RLock()
	if len(m.maps[index]) == 0 {
		m.lock[index].RUnlock()
		return 0, false
	}
	value := m.maps[index][hashedKey]
	m.lock[index].RUnlock()
	return value, value > 0
}

func (m *shardedMap) put(key string, value uint32) bool {
	hashedKey := m.hasher.Sum64(key)
	index := hashedKey & uint64(m.config.Shards-1)
	m.lock[index].Lock()
	_, has := m.maps[index][hashedKey]
	m.maps[index][hashedKey] = value
	m.lock[index].Unlock()
	return has
}

func (m *shardedMap) delete(key string) bool {
	hashedKey := m.hasher.Sum64(key)
	index := hashedKey & uint64(m.config.Shards-1)
	m.lock[index].Lock()
	if len(m.maps[index]) == 0 {
		m.lock[index].Unlock()
		return false
	}
	has := m.maps[index][hashedKey] > 0
	m.maps[index][hashedKey] = 0
	m.lock[index].Unlock()
	return has
}

func newShardedMap(config *Config) *shardedMap {
	aMap := &shardedMap{
		config: config,
		lock:   make([]sync.RWMutex, config.Shards),
		maps:   make([]map[uint64]uint32, config.Shards),
	}
	for i := range aMap.maps {
		aMap.maps[i] = make(map[uint64]uint32, config.shardMapSize)
	}
	return aMap
}
