//go:build !amd64 || nosimd

package scache

const (
	groupSize = 8
)
