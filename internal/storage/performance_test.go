package storage

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestStreamingStress_100kEmbeddingsMemoryStable(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	const (
		embeddingCount = 100000
		dimension      = 8
		resultLimit    = 10
		streamMaxDelta = 64 << 20
		searchMaxDelta = 96 << 20
	)

	insertStart := time.Now()
	for i := 0; i < embeddingCount; i++ {
		vector := make([]float32, dimension)
		vector[i%dimension] = 1
		if _, err := store.SaveChunk(CodeChunk{
			FilePath:  fmt.Sprintf("stress/file-%06d.go", i),
			Content:   fmt.Sprintf("stress-token-%06d", i),
			StartLine: 1,
			EndLine:   1,
			Language:  "go",
			Vector:    vector,
		}); err != nil {
			t.Fatalf("SaveChunk(%d): %v", i, err)
		}
	}
	insertDuration := time.Since(insertStart)

	query := []float32{1, 0, 0, 0, 0, 0, 0, 0}
	baselineStream := currentHeapAlloc()
	streamStart := time.Now()
	var streamed int
	var streamPeak atomic.Uint64
	streamPeak.Store(baselineStream)
	err := store.ForEachEmbedding(func(_ int, vector []float32) error {
		streamed++
		if len(vector) != dimension {
			return fmt.Errorf("unexpected vector dimension %d", len(vector))
		}
		if streamed%1000 == 0 {
			updatePeakAlloc(&streamPeak)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachEmbedding: %v", err)
	}
	streamDuration := time.Since(streamStart)
	streamPeakDelta := streamPeak.Load() - baselineStream
	if streamed != embeddingCount {
		t.Fatalf("streamed %d embeddings, want %d", streamed, embeddingCount)
	}
	if streamPeakDelta > streamMaxDelta {
		t.Fatalf("ForEachEmbedding peak heap delta = %d bytes, want <= %d", streamPeakDelta, streamMaxDelta)
	}

	baselineSearch := currentHeapAlloc()
	searchStart := time.Now()
	var results []CodeChunk
	searchPeak, err := samplePeakHeapAlloc(func() error {
		var searchErr error
		results, searchErr = store.searchLinearScan(query, resultLimit, true)
		return searchErr
	})
	if err != nil {
		t.Fatalf("searchLinearScan: %v", err)
	}
	searchDuration := time.Since(searchStart)
	searchPeakDelta := searchPeak - baselineSearch
	if len(results) != resultLimit {
		t.Fatalf("searchLinearScan returned %d results, want %d", len(results), resultLimit)
	}
	if searchPeakDelta > searchMaxDelta {
		t.Fatalf("searchLinearScan peak heap delta = %d bytes, want <= %d", searchPeakDelta, searchMaxDelta)
	}

	t.Logf("inserted %d embeddings in %v", embeddingCount, insertDuration)
	t.Logf("ForEachEmbedding streamed %d embeddings in %v with peak heap delta %.2f MiB", streamed, streamDuration, float64(streamPeakDelta)/(1<<20))
	t.Logf("searchLinearScan returned %d results in %v with peak heap delta %.2f MiB", len(results), searchDuration, float64(searchPeakDelta)/(1<<20))
}

func currentHeapAlloc() uint64 {
	runtime.GC()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return mem.Alloc
}

func samplePeakHeapAlloc(fn func() error) (uint64, error) {
	baseline := currentHeapAlloc()
	var peak atomic.Uint64
	peak.Store(baseline)

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			updatePeakAlloc(&peak)
			return peak.Load(), err
		case <-ticker.C:
			updatePeakAlloc(&peak)
		}
	}
}

func updatePeakAlloc(peak *atomic.Uint64) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	for {
		current := peak.Load()
		if mem.Alloc <= current {
			return
		}
		if peak.CompareAndSwap(current, mem.Alloc) {
			return
		}
	}
}
