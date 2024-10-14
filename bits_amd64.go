//go:build amd64 && !nosimd

package scache

import (
	_ "unsafe"
)

const (
	groupSize = 16
)
