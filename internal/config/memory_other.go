//go:build !darwin && !linux

package config

// availableMemoryMB returns 0 on unsupported platforms, which causes
// AdaptiveBatchSize to fall back to the conservative default.
func availableMemoryMB() uint64 {
	return 0
}
