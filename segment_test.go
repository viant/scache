package scache

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestSegment_get(t *testing.T) {

	var useCases = []struct {
		description string
		sizeMb      int
		keys        int
		pattern     string
		entrySize   int
	}{
		{
			description: "single key",
			entrySize:   12,
			sizeMb:      1,
			pattern:     "xy",
			keys:        1,
		},
		{
			description: "multi key",
			entrySize:   26,
			sizeMb:      1,
			pattern:     "ab",

			keys: 5,
		},
		{
			description: "large entry",
			pattern:     "uI",
			entrySize:   1024,
			sizeMb:      1,
			keys:        128,
		},
		{
			description: "large entry",
			pattern:     "uI",
			entrySize:   16,
			sizeMb:      48,
			keys:        1024 * 1024,
		},
	}

	for _, useCase := range useCases {
		config := &Config{SizeMb: useCase.sizeMb}
		config.Init()
		segment := &segment{
			config:     config,
			shardedMap: newShardedMap(config),
		}

		err := segment.allocate(0)

		assert.Nil(t, err)
		for i := 0; i < useCase.keys; i++ {
			key := fmt.Sprintf("key%v", i)
			_, has := segment.get(key)
			assert.False(t, has, useCase.description)
			data := strings.Repeat(useCase.pattern, useCase.entrySize/2)
			added := segment.set(key, []byte(data))
			assert.True(t, added, useCase.description)

			actual, has := segment.get(key)
			assert.True(t, has, useCase.description)

			assert.EqualValues(t, data, string(actual), useCase.description)
		}

		for i := 0; i < useCase.keys; i++ {
			key := fmt.Sprintf("key%v", i)
			_, has := segment.get(key)
			assert.True(t, has, useCase.description)
			segment.delete(key)
			_, has = segment.get(key)
			assert.False(t, has, useCase.description)
		}
	}
}
