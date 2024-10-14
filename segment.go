package scache

import (
	"encoding/binary"
	"sync/atomic"
)

/*
Idea taken from
https://dev.to/douglasmakey/how-bigcache-avoids-expensive-gc-cycles-and-speeds-up-concurrent-access-in-go-12bb
*/

const (
	headerSize  = 5
	controlByte = 0x9A
)

type segment struct {
	*shardedMap
	config   *Config
	index    uint32
	data     []byte
	dataSize uint64
	tail     uint64
	keys     uint32
	mmap     *mmap
}

func (s *segment) close() error {
	if s.mmap != nil {
		return s.mmap.close()
	}
	return nil
}

func (s *segment) reset() {
	for i := range s.maps {
		s.shardedMap.lock[i].Lock()
		clearSwissMap(s.shardedMap.maps[i], keys, values)
		s.shardedMap.lock[i].Unlock()
	}
	atomic.StoreUint64(&s.tail, 32)
	atomic.StoreUint32(&s.keys, 0)

}

func (s *segment) get(key string) ([]byte, bool) {
	shardedMap := s.getShardedMap()
	headerAddress := shardedMap.getAddress(key)
	if headerAddress == 0 {
		return nil, false
	}
	headerAddressEnd := headerAddress + headerSize
	if headerAddressEnd > s.dataSize {
		return nil, false
	}
	entrySize := binary.LittleEndian.Uint32(s.data[headerAddress+1 : headerAddressEnd])
	if headerAddressEnd > atomic.LoadUint64(&s.tail) {
		return nil, false
	}
	dataAddress := headerAddress + headerSize
	dataAddressEnd := dataAddress + uint64(entrySize)
	if dataAddressEnd > s.dataSize {
		return nil, false
	}
	result := s.data[dataAddress:dataAddressEnd]
	if s.data[headerAddress] != controlByte {
		return nil, false
	}
	return result, true
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

func (s *segment) set(key string, value []byte) ([]byte, bool) {
	if maxEntries := s.config.MaxEntries; maxEntries > 0 && 1+int(atomic.LoadUint32(&s.keys)) > maxEntries {
		return nil, false
	}
	shardedMap := s.getShardedMap()
	blobSize := len(value) + headerSize
	alignBlobSize := ((blobSize >> 5) + 1) << 5
	nextAddress := int(atomic.AddUint64(&s.tail, uint64(alignBlobSize)))

	if nextAddress >= len(s.data) { //out of memory,
		atomic.SwapUint64(&s.tail, s.dataSize-1)
		return nil, false
	}
	headerAddress := nextAddress - alignBlobSize
	s.data[headerAddress] = controlByte
	binary.LittleEndian.PutUint32(s.data[headerAddress+1:headerAddress+headerSize], uint32(len(value)))
	entryAddress := headerAddress + headerSize
	entryAddressOffset := entryAddress + len(value)
	copy(s.data[entryAddress:entryAddressOffset], value)
	if hadKey := shardedMap.put(key, uint32(headerAddress>>5)); !hadKey {
		atomic.AddUint32(&s.keys, 1)
	}
	return s.data[entryAddress:entryAddressOffset], true
}

func (s *segment) allocate(idx int) error {
	s.index = uint32(idx)
	s.tail = 32
	segmentDataSize := s.config.SegmentDataSize()
	if s.config.Location == "" {
		s.data = make([]byte, segmentDataSize)
		s.dataSize = uint64(segmentDataSize)
		return nil
	}
	s.mmap = newMmap(s.config.Location, s.config.SizeMb*mb)
	err := s.mmap.open()
	if err == nil {
		s.mmap.size = segmentDataSize
		offset := int64(idx * segmentDataSize)
		err = s.mmap.assign(offset, &s.data)
		s.dataSize = uint64(len(s.data))
	}
	return err
}
