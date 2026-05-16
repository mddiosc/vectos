package config

import (
	"runtime"
	"testing"
)

func TestAvailableMemoryMB(t *testing.T) {
	mem := availableMemoryMB()
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if mem == 0 {
			t.Fatalf("availableMemoryMB() returned 0 on %s; expected a positive value", runtime.GOOS)
		}
		t.Logf("Available memory: %d MB", mem)
		// Sanity check: should be between 100 MB and 1 TB
		if mem < 100 {
			t.Errorf("availableMemoryMB() = %d MB; suspiciously low", mem)
		}
		if mem > 1024*1024 {
			t.Errorf("availableMemoryMB() = %d MB; suspiciously high (>1TB)", mem)
		}
	} else {
		// On unsupported platforms, should return 0
		if mem != 0 {
			t.Errorf("availableMemoryMB() = %d on unsupported platform %s; expected 0", mem, runtime.GOOS)
		}
	}
}

func TestAdaptiveBatchSize(t *testing.T) {
	bs := AdaptiveBatchSize()
	t.Logf("AdaptiveBatchSize: %d (available memory: %d MB)", bs, availableMemoryMB())

	// Must be one of the valid values
	validSizes := map[int]bool{4: true, 8: true, 16: true, 32: true}
	if !validSizes[bs] {
		t.Errorf("AdaptiveBatchSize() = %d; expected one of [4, 8, 16, 32]", bs)
	}

	// On a modern dev machine, should be at least 16
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if bs < 8 {
			t.Logf("Warning: batch size %d suggests very low available memory", bs)
		}
	}
}

func TestAdaptiveBatchSizeThresholds(t *testing.T) {
	// Test the logic directly by verifying the function's behavior
	// matches the documented thresholds. Since we can't mock availableMemoryMB
	// easily, we just verify the returned value is consistent.
	mem := availableMemoryMB()
	bs := AdaptiveBatchSize()

	if mem == 0 {
		if bs != 16 {
			t.Errorf("with unknown memory (0), expected batch size 16, got %d", bs)
		}
		return
	}

	switch {
	case mem >= 4096:
		if bs != 32 {
			t.Errorf("with %d MB available, expected batch size 32, got %d", mem, bs)
		}
	case mem >= 2048:
		if bs != 16 {
			t.Errorf("with %d MB available, expected batch size 16, got %d", mem, bs)
		}
	case mem >= 1024:
		if bs != 8 {
			t.Errorf("with %d MB available, expected batch size 8, got %d", mem, bs)
		}
	default:
		if bs != 4 {
			t.Errorf("with %d MB available, expected batch size 4, got %d", mem, bs)
		}
	}
}
