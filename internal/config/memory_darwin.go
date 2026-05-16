//go:build darwin

package config

import (
	"encoding/binary"
	"syscall"
)

// availableMemoryMB returns the approximate available memory in megabytes on macOS.
// It uses sysctl to get total physical memory and estimates available by summing
// free + speculative + pageable internal pages from the VM subsystem.
// Falls back to total/3 if page stats are unavailable.
func availableMemoryMB() uint64 {
	total := totalPhysicalMemoryMB()
	if total == 0 {
		return 0
	}

	pageSize := uint64(syscall.Getpagesize())

	freePages := sysctlUint32("vm.page_free_count")
	speculativePages := sysctlUint32("vm.page_speculative_count")
	// page_pageable_internal_count includes dirty/active pages that are not
	// immediately reclaimable, so this estimate can be optimistic. In practice
	// it gives a reasonable approximation for batch-size selection: the worst
	// case is choosing a slightly larger batch than ideal, not a crash.
	// Activity Monitor uses a similar heuristic (free + inactive + speculative).
	pageableInternal := sysctlUint32("vm.page_pageable_internal_count")

	availablePages := uint64(freePages) + uint64(speculativePages) + uint64(pageableInternal)
	if availablePages == 0 {
		return total / 3
	}

	return (availablePages * pageSize) / (1024 * 1024)
}

// totalPhysicalMemoryMB returns total physical RAM in MB via sysctl hw.memsize.
// hw.memsize is CTLTYPE_QUAD (8-byte uint64, little-endian on Apple Silicon).
func totalPhysicalMemoryMB() uint64 {
	val, err := syscall.Sysctl("hw.memsize")
	if err != nil || len(val) == 0 {
		return 0
	}
	// syscall.Sysctl trims trailing null bytes from binary values.
	// Pad to 8 bytes (CTLTYPE_QUAD size).
	raw := padRight([]byte(val), 8)
	if len(raw) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(raw[:8]) / (1024 * 1024)
}

// sysctlUint32 reads a CTLTYPE_INT (4-byte uint32) sysctl by name.
// syscall.Sysctl returns the raw binary bytes with trailing nulls trimmed,
// so we must pad back to 4 bytes.
func sysctlUint32(name string) uint32 {
	val, err := syscall.Sysctl(name)
	if err != nil || len(val) == 0 {
		return 0
	}
	raw := padRight([]byte(val), 4)
	if len(raw) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(raw[:4])
}

// padRight extends b to targetLen bytes by appending zeros.
func padRight(b []byte, targetLen int) []byte {
	if len(b) >= targetLen {
		return b
	}
	padded := make([]byte, targetLen)
	copy(padded, b)
	return padded
}
