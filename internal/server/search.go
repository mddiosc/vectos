package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"vectos/internal/storage"
)

func (s *Server) handleSearchCode(w http.ResponseWriter, r *http.Request) { s.handleSearch(w, r, "code") }
func (s *Server) handleSearchDocs(w http.ResponseWriter, r *http.Request) { s.handleSearch(w, r, "docs") }

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, source string) {
	if r.Method != http.MethodPost { writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed"); return }
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeAPIError(w, http.StatusBadRequest, "INVALID_QUERY", "invalid JSON body"); return }
	if req.Limit <= 0 { req.Limit = 10 }
	if req.Limit > 100 { req.Limit = 100 }
	if err := validateSearchRequest(req); err != nil { if ve, ok := err.(*ValidationError); ok { writeAPIError(w, http.StatusBadRequest, ve.Code, ve.Message) }; return }
	queryVector, err := s.embedFunc(req.Query)
	if err != nil {
		results, textErr := s.store.SearchText(req.Query)
		if textErr != nil { writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "search failed"); return }
		items := make([]SearchResultItem, 0, len(results))
		for _, c := range results {
			items = append(items, SearchResultItem{FilePath: c.FilePath, FileName: filepath.Base(c.FilePath), Language: c.Language, Relevance: c.Score, LineRanges: []LineRange{{Start: c.StartLine, End: c.EndLine}}, Signatures: []string{c.Signature}})
		}
		writeJSON(w, http.StatusOK, SearchResult{Results: items, Mode: "text", Total: len(items)}); return
	}
	includeDocs := source == "docs"
	results, err := s.store.SearchSemantic(queryVector, req.Limit*3, includeDocs)
	if err != nil { writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "search failed"); return }
	fileResults := storage.CollapseFileResults(results, 5)
	items := make([]SearchResultItem, 0, len(fileResults))
	for _, fr := range fileResults {
		item := SearchResultItem{FilePath: fr.FilePath, FileName: filepath.Base(fr.FilePath), Language: fr.Language, Relevance: fr.Relevance, LineRanges: make([]LineRange, 0, len(fr.LineRanges)), Signatures: fr.Signatures}
		for _, lr := range fr.LineRanges { item.LineRanges = append(item.LineRanges, LineRange{Start: lr.Start, End: lr.End}) }
		items = append(items, item)
	}
	if len(items) > req.Limit { items = items[:req.Limit] }
	writeJSON(w, http.StatusOK, SearchResult{Results: items, Mode: "semantic_hybrid", Total: len(items)})
}
