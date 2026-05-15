package server

import (
	"net/http"
	"time"
)

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed"); return }
	stats, err := s.store.Stats()
	if err != nil { writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "metrics failed"); return }
	metadata, _ := s.store.GetIndexMetadata()
	resp := MetricsResponse{ChunkCount: stats.ChunkCount, FileCount: stats.FileCount, DatabaseSize: stats.DatabaseSize, EmbeddedCount: stats.EmbeddedCount, UnembeddedCount: stats.UnembeddedCount, Provider: metadata.Provider, Model: metadata.Model, Dimensions: metadata.Dimensions, UptimeSeconds: int64(time.Since(s.startTime).Seconds()), WatcherStatus: "disabled"}
	if resp.Provider == "" { resp.Provider = "unknown" }
	if !metadata.UpdatedAt.IsZero() { resp.LastIndexTime = metadata.UpdatedAt.UTC().Format(time.RFC3339) }
	writeJSON(w, http.StatusOK, resp)
}
