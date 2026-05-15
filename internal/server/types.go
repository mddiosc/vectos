package server

type SearchRequest struct {
	Query   string `json:"query"`
	Project string `json:"project,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type SearchResult struct {
	Results []SearchResultItem `json:"results"`
	Mode    string             `json:"mode"`
	Total   int                `json:"total"`
}

type SearchResultItem struct {
	FilePath   string      `json:"file_path"`
	FileName   string      `json:"file_name"`
	Language   string      `json:"language"`
	Relevance  float64     `json:"relevance"`
	LineRanges []LineRange `json:"line_ranges"`
	Signatures []string    `json:"signatures"`
}

type LineRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type MetricsResponse struct {
	ChunkCount      int64  `json:"chunk_count"`
	FileCount       int64  `json:"file_count"`
	DatabaseSize    int64  `json:"database_size_bytes"`
	EmbeddedCount   int64  `json:"embedded_count"`
	UnembeddedCount int64  `json:"unembedded_count"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	Dimensions      int    `json:"dimensions"`
	UptimeSeconds   int64  `json:"uptime_seconds"`
	LastIndexTime   string `json:"last_index_time,omitempty"`
	WatcherStatus   string `json:"watcher_status"`
}

type ProjectStatusResponse struct {
	Project      string `json:"project"`
	Indexed      bool   `json:"indexed"`
	ChunkCount   int64  `json:"chunk_count,omitempty"`
	FileCount    int64  `json:"file_count,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	DatabasePath string `json:"database_path,omitempty"`
}
