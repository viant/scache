package scache

import (
	"encoding/binary"
	"sync/atomic"
)

/*
Idea taken from
https://dev.to/douglasmakey/how-bigcache-avoids-expensive-gc-cycles-and-speeds-up-concurrent-access-in-go-12bb
*/

const headerSize = 4

type segment struct {
	*shardedMap
	config *Config
	index  uint32
	data   []byte
	dataSize uint32
	tail   uint32
	keys   uint32
	mmap   *mmap


}

func (s *segment) close() error {
	if s.mmap != nil {
		return s.mmap.close()
	}
	return nil
}

func (s *segment) reset(aMap *shardedMap) {
	for i :=  range aMap.maps {
		s.shardedMap.lock[i].Lock()
		s.shardedMap.maps[i] = aMap.maps[i]
		s.shardedMap.lock[i].Unlock()
	}
	atomic.StoreUint32(&s.tail, 1)
	atomic.StoreUint32(&s.keys, 0)
}


func (s *segment) get(key string) ([]byte, bool) {
	shardedMap := s.getShardedMap()
	headerAddress := shardedMap.getAddress(key)
	if headerAddress == 0 {
		return nil, false
	}
	headerAddressEnd := headerAddress+headerSize
	if headerAddressEnd > s.dataSize {
		return nil, false
	}
	entrySize := binary.LittleEndian.Uint32(s.data[headerAddress : headerAddressEnd])
	if headerAddressEnd > s.dataSize {
		return nil, false
	}
	dataAddress := headerAddress + headerSize
	dataAddressEnd := dataAddress + entrySize
	if dataAddressEnd > atomic.LoadUint32(&s.tail) {
		return nil, false
	}
	return s.data[dataAddress : dataAddressEnd], true
}

func (s *segment) delete(key string) {
	shardedMap := s.getShardedMap()
	if shardedMap.delete(key) {
	updateKeys:
		if keys := atomic.LoadUint32(&s.keys); keys > 1 {
			if !atomic.CompareAndSwapUint32(&s.keys, keys, keys-1) {
				goto updateKeys
			}
		}
	}
}

func (s *segment) getShardedMap() *shardedMap {
	result := s.shardedMap
	return result
}


func (s *segment) set(key string, value []byte) bool {
	if maxEntries := s.config.MaxEntries; maxEntries > 0 && int(atomic.LoadUint32(&s.keys)) > maxEntries {
		return false
	}
	shardedMap := s.getShardedMap()
	blobSize := len(value) + headerSize
	nextAddress := int(atomic.AddUint32(&s.tail, uint32(blobSize)))
	if nextAddress >= len(s.data) { //out of memory,
		return false
	}
	headerAddress := nextAddress - blobSize
	binary.LittleEndian.PutUint32(s.data[headerAddress:headerAddress+headerSize], uint32(len(value)))
	entryAddress := headerAddress + headerSize
	copy(s.data[entryAddress:entryAddress+len(value)], value)
	if hadKey := shardedMap.put(key, uint32(headerAddress)); !hadKey {
		atomic.AddUint32(&s.keys, 1)
	}
	return true
}

func (s *segment) allocate(idx int) error {
	s.index = uint32(idx)
	s.tail = 1
	segmentDataSize := s.config.SegmentDataSize()
	if s.config.Location == "" {
		s.data = make([]byte, segmentDataSize)
		s.dataSize = uint32(len(s.data))
		return nil
	}
	s.mmap = newMmap(s.config.Location, s.config.SizeMb*mb)
	err := s.mmap.open()
	if err == nil {
		s.mmap.size = segmentDataSize
		offset := int64(idx * segmentDataSize)
		err = s.mmap.assign(offset, &s.data)
		s.dataSize = uint32(len(s.data))
	}
	return err
}
