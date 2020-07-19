package scache

const (
	//DefaultCacheSizeMb default cache size
	DefaultCacheSizeMb = 1
	//MinShards min shards
	MinShards = 32
	//DefaultShardMapSize default map shard allocation size.
	DefaultShardMapSize = 32 * 1024
	mb                  = 1024 * 1024
)

//Config represents cache config
type Config struct {
	MaxEntries   int    //optional upper entries limit in the cache
	EntrySize    int    //optional entry size to estimate SizeMb (MaxEntries * EntrySize) when specified
	SizeMb       int    //optional max cache size, default 1
	Shards       int    //optional segment shards size,  default MAX(32, MaxEntries / 1024*1024)
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
		if c.MaxEntries > 0 && c.EntrySize > 0 {
			c.SizeMb += 2 * c.MaxEntries * c.EntrySize / mb
		}
	}
	if c.SizeMb > 4096*mb { //currently max supported memory
		c.SizeMb = 4096 * mb
	}
	if c.Shards < MinShards {
		c.Shards = MinShards
		if candidate := c.MaxEntries / mb; candidate > c.Shards {
			c.Shards = candidate
		}
	}

	if c.MaxEntries > 0 {
		c.shardMapSize = 2 * (c.MaxEntries / c.Shards)
	} else {
		c.shardMapSize = DefaultShardMapSize
	}
}
