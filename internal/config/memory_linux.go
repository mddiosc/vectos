//go:build linux

package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// availableMemoryMB returns the available memory in megabytes on Linux.
// It reads /proc/meminfo and uses MemAvailable (kernel 3.14+), falling
// back to MemFree + Buffers + Cached on older kernels.
func availableMemoryMB() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	var memAvailable, memFree, buffers, cached uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemAvailable:"):
			memAvailable = parseMeminfoKB(line)
		case strings.HasPrefix(line, "MemFree:"):
			memFree = parseMeminfoKB(line)
		case strings.HasPrefix(line, "Buffers:"):
			buffers = parseMeminfoKB(line)
		case strings.HasPrefix(line, "Cached:"):
			cached = parseMeminfoKB(line)
		}
	}

	if memAvailable > 0 {
		return memAvailable / 1024
	}
	// Fallback for kernels < 3.14
	return (memFree + buffers + cached) / 1024
}

// parseMeminfoKB extracts the numeric value (in kB) from a /proc/meminfo line.
// Format: "FieldName:     12345 kB"
func parseMeminfoKB(line string) uint64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	val, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0
	}
	return val
}
