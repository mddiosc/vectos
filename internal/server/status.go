package server

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed"); return }
	project := sanitizeProject(strings.TrimPrefix(r.URL.Path, "/status/"))
	if project == "" { writeAPIError(w, http.StatusBadRequest, "INVALID_PROJECT", "invalid project name"); return }
	stats, err := s.store.Stats()
	if err != nil { writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "status failed"); return }
	resp := ProjectStatusResponse{Project: project, Indexed: stats.FileCount > 0}
	if resp.Indexed { resp.ChunkCount, resp.FileCount, resp.DatabasePath = stats.ChunkCount, stats.FileCount, stats.DatabasePath }
	if meta, err := s.store.GetIndexMetadata(); err == nil && !meta.UpdatedAt.IsZero() { resp.LastModified = meta.UpdatedAt.UTC().Format(time.RFC3339) }
	writeJSON(w, http.StatusOK, resp)
}
