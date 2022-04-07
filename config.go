package scache

const (
	//DefaultCacheSizeMb default cache size
	DefaultCacheSizeMb = 1
	//MinShards min shards
	MinShards = 32
	//DefaultShardMapSize default map shard allocation size.
	DefaultShardMapSize = 32 * 1024
	mb                  = 1024 * 1024
	alignmentSize       = 32
)

//Config represents cache config
type Config struct {
	MaxEntries   int //optional upper entries limit in the cache
	EntrySize    int //optional entry size to estimate SizeMb (MaxEntries * EntrySize) when specified
	KeySize      int
	SizeMb       int    //optional max cache size, default 1
	Shards       uint64 //optional segment shards size,  default MAX(32, MaxEntries / 1024*1024)
	Location     string //optional path to mapped memory file
	shardMapSize int
}

//SegmentDataSize returns segments data size (cache always has 2 segments)
func (c *Config) SegmentDataSize() int {
	return c.SizeMb * (mb / 2)
}

//Init initialises config
func (c *Config) Init() {
	if c.SizeMb == 0 {
		c.SizeMb = DefaultCacheSizeMb
	}

	if c.MaxEntries > 0 && c.EntrySize > 0 {
		estSizeMb := DefaultCacheSizeMb + (2*c.MaxEntries*alignSize(headerSize+c.EntrySize))/mb
		if c.SizeMb < estSizeMb {
			c.SizeMb = estSizeMb
		}
	}

	if c.Shards < MinShards {
		c.Shards = MinShards
		if candidate := c.MaxEntries / mb; candidate > int(c.Shards) {
			c.Shards = uint64(candidate)
		}
	}
	if c.MaxEntries > 0 {
		c.shardMapSize = 2 * (c.MaxEntries / int(c.Shards))
	} else {
		c.shardMapSize = DefaultShardMapSize
	}

}

func alignSize(size int) int {
	return ((size >> 5) + 1) << 5
}
