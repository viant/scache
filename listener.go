package scache

import "time"

//OnSegmentSwitch function to call when segment switches primary to secondary role
type OnSegmentSwitch func(index, keys uint32, timeTaken time.Duration)
