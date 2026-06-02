package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"vectos/internal/config"
)

func runConfigCommand(app appContext, args []string) {
	if hasHelpFlag(args) {
		printSubcommandHelp("config")
		os.Exit(0)
	}

	if err := app.flags.configCmd.Parse(args); err != nil {
		fatalErr(err)
	}

	if !*app.flags.configInit {
		fmt.Println("Usage: vectos config --init")
		fmt.Println()
		fmt.Println("Run 'vectos config --init' to start the interactive setup wizard,")
		fmt.Println("or 'vectos config --help' for details.")
		fmt.Println()
		fmt.Println("Configuration is stored in ~/.vectos/config.jsonc")
		fmt.Println("You can also create it manually — all keys have sensible defaults.")
		os.Exit(0)
	}

	runConfigInit(app.embedConfig)
}

func runConfigInit(defaults config.EmbeddingConfig) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not determine home directory: %v\n", err)
		os.Exit(1)
	}
	configPath := filepath.Join(homeDir, ".vectos", "config.jsonc")

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("A configuration file already exists at %s\n", configPath)
		fmt.Print("Overwrite? [y/N] ")
		if !askConfirm(false) {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
	}

	fmt.Println()
	fmt.Println("  Vectos config wizard — let's set up your ~/.vectos/config.jsonc")
	fmt.Println("  Press Enter to accept the default [shown in brackets].")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	cfg := defaults

	// ── Model ──
	fmt.Println("  Embedding model")
	for i, m := range config.SupportedEmbeddedModels {
		marker := " "
		if m == cfg.Embedded.ModelName {
			marker = "*"
		}
		fmt.Printf("    %d. %s %s\n", i+1, m, marker)
	}
	fmt.Printf("  Choice [%s]: ", cfg.Embedded.ModelName)
	choice := readLine(reader)
	switch strings.TrimSpace(choice) {
	case "1":
		cfg.Embedded.ModelName = config.SupportedEmbeddedModels[0]
	case "2":
		cfg.Embedded.ModelName = config.SupportedEmbeddedModels[1]
	case "":
		// keep default
	default:
		cfg.Embedded.ModelName = choice
	}
	cfg.Embedded.ModelDir = filepath.Join(homeDir, ".vectos", "models", cfg.Embedded.ModelName)
	cfg.Embedded.AssetBaseURL = assetBaseURLForModel(cfg.Embedded.ModelName)
	cfg.Embedded.Dimensions = config.DefaultEmbeddedDimensionsForModel(cfg.Embedded.ModelName)
	fmt.Printf("  → Using model: %s\n\n", cfg.Embedded.ModelName)

	// ── Dimensions ──
	if config.SupportsMatryoshkaDimensions(cfg.Embedded.ModelName) {
		fmt.Println("  Embedding dimensions (Matryoshka — smaller = faster, smaller storage)")
		for _, d := range config.MatryoshkaDimensions {
			marker := " "
			if d == cfg.Embedded.Dimensions {
				marker = "*"
			}
			fmt.Printf("    %s %d\n", marker, d)
		}
		fmt.Printf("  Dimensions [%d]: ", cfg.Embedded.Dimensions)
		dimStr := readLine(reader)
		if dimStr != "" {
			if dim, err := strconv.Atoi(strings.TrimSpace(dimStr)); err == nil {
				if config.IsValidMatryoshkaDimension(dim) {
					cfg.Embedded.Dimensions = dim
				} else {
					fmt.Printf("  ⚠ %d is not a valid Matryoshka dimension; using default %d\n", dim, cfg.Embedded.Dimensions)
				}
			}
		}
		fmt.Printf("  → Dimensions: %d\n\n", cfg.Embedded.Dimensions)
	}

	// ── Batch size ──
	fmt.Println("  Batch size (chunks per ONNX inference round)")
	fmt.Println("    Higher = faster indexing but more memory. 32 is good for Apple Silicon.")
	fmt.Printf("  Batch size [%d]: ", config.DefaultEmbeddedBatchSize)
	bsStr := readLine(reader)
	if bsStr != "" {
		if bs, err := strconv.Atoi(strings.TrimSpace(bsStr)); err == nil && bs > 0 {
			cfg.Embedded.BatchSize = bs
		}
	}
	if cfg.Embedded.BatchSize == 0 {
		fmt.Printf("  → Batch size: auto (based on available RAM)\n\n")
	} else {
		fmt.Printf("  → Batch size: %d\n\n", cfg.Embedded.BatchSize)
	}

	// ── Vector index (advanced) ──
	fmt.Println("  Vector index (advanced — press Enter for defaults)")
	fmt.Printf("    HNSW M (connections per node) [%d]: ", cfg.VectorIndex.HNSW_M)
	if mStr := readLine(reader); mStr != "" {
		if m, err := strconv.Atoi(strings.TrimSpace(mStr)); err == nil && m > 0 {
			cfg.VectorIndex.HNSW_M = m
		}
	}
	fmt.Printf("    HNSW ef_construction (build-time search depth) [%d]: ", cfg.VectorIndex.HNSW_EfConstruction)
	if ecStr := readLine(reader); ecStr != "" {
		if ec, err := strconv.Atoi(strings.TrimSpace(ecStr)); err == nil && ec > 0 {
			cfg.VectorIndex.HNSW_EfConstruction = ec
		}
	}
	fmt.Printf("    HNSW ef_search (query-time search depth) [%d]: ", cfg.VectorIndex.HNSW_EfSearch)
	if esStr := readLine(reader); esStr != "" {
		if es, err := strconv.Atoi(strings.TrimSpace(esStr)); err == nil && es > 0 {
			cfg.VectorIndex.HNSW_EfSearch = es
		}
	}
	fmt.Printf("    Compression (none / sq8) [%s]: ", cfg.VectorIndex.Compression)
	comp := strings.TrimSpace(readLine(reader))
	if comp == "sq8" {
		cfg.VectorIndex.Compression = "sq8"
	} else if comp != "" {
		fmt.Printf("  ⚠ Unknown compression %q; using default %q\n", comp, cfg.VectorIndex.Compression)
	}
	fmt.Printf("  → Vector index: %s (M=%d, efConstruction=%d, efSearch=%d)\n\n",
		cfg.VectorIndex.IndexType, cfg.VectorIndex.HNSW_M,
		cfg.VectorIndex.HNSW_EfConstruction, cfg.VectorIndex.HNSW_EfSearch)

	// ── Summary ──
	fmt.Println("  ── Summary ──")
	fmt.Printf("  Model:           %s\n", cfg.Embedded.ModelName)
	if config.SupportsMatryoshkaDimensions(cfg.Embedded.ModelName) {
		fmt.Printf("  Dimensions:      %d\n", cfg.Embedded.Dimensions)
	}
	if cfg.Embedded.BatchSize > 0 {
		fmt.Printf("  Batch size:      %d\n", cfg.Embedded.BatchSize)
	} else {
		fmt.Println("  Batch size:      auto")
	}
	fmt.Printf("  Vector index:    %s (M=%d)\n", cfg.VectorIndex.IndexType, cfg.VectorIndex.HNSW_M)
	fmt.Println()

	fmt.Printf("  Save to %s? [Y/n] ", configPath)
	if !askConfirm(true) {
		fmt.Println("Aborted.")
		os.Exit(0)
	}

	// ── Write config ──
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating config directory: %v\n", err)
		os.Exit(1)
	}
	if err := writeConfigJSONC(configPath, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  ✅ Config saved to %s\n", configPath)
}

// ── Helpers ──

func readLine(r *bufio.Reader) string {
	text, _ := r.ReadString('\n')
	return strings.TrimSpace(text)
}

func readBool(r *bufio.Reader, def bool) bool {
	text := strings.ToLower(readLine(r))
	if text == "" {
		return def
	}
	switch text {
	case "true", "yes", "y", "1", "on":
		return true
	case "false", "no", "n", "0", "off":
		return false
	}
	return def
}

func askConfirm(def bool) bool {
	text := strings.ToLower(readLine(bufio.NewReader(os.Stdin)))
	if text == "" {
		return def
	}
	switch text {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	}
	return def
}

func boolLabel(v bool) string {
	if v {
		return "enabled"
	}
	return "disabled"
}

func assetBaseURLForModel(modelName string) string {
	switch modelName {
	case "jina-embeddings-v3":
		return config.DefaultEmbeddedAssetBaseURL
	case "bge-small-en-v1.5":
		return config.DefaultBGEAssetBaseURL
	default:
		return ""
	}
}

// writeConfigJSONC writes the configuration to a JSONC file with inline
// documentation comments explaining each setting.
func writeConfigJSONC(path string, cfg config.EmbeddingConfig) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprint(f, `{
  // Embedded embeddings configuration
  // Vectos uses ONNX Runtime for local embedding inference.
  // Supported models: jina-embeddings-v3 (code+text, 8192 tokens),
  //                    bge-small-en-v1.5 (lighter, English-only).
  "embeddings": {
    "default_provider": "embedded",
    "fallback_order": ["embedded", "remote"],
    "embedded": {
      "enabled": true,
`)
	fmt.Fprintf(f, `      "model_name": %q,
`, cfg.Embedded.ModelName)
	fmt.Fprintf(f, `      "model_dir": %q,
`, cfg.Embedded.ModelDir)
	fmt.Fprintf(f, `      "auto_download": %v,
`, cfg.Embedded.AutoDownload)
	fmt.Fprintf(f, `      "asset_base_url": %q,
`, cfg.Embedded.AssetBaseURL)
	fmt.Fprintf(f, `      "timeout_seconds": %d,
`, cfg.Embedded.TimeoutS)
	if cfg.Embedded.BatchSize > 0 {
		fmt.Fprintf(f, `      // Batch size for ONNX inference. 0 = auto-detect based on available RAM.
      "batch_size": %d,
`, cfg.Embedded.BatchSize)
	} else {
		fmt.Fprint(f, `      // Batch size 0 = auto-detect based on available RAM (recommended).
      "batch_size": 0,
`)
	}
	if config.SupportsMatryoshkaDimensions(cfg.Embedded.ModelName) {
		fmt.Fprintf(f, `      // Matryoshka truncation (jina-embeddings-v3 only).
      // Valid: 32, 64, 128, 256, 512, 768, 1024. Lower = faster + smaller storage.
      "dimensions": %d,
`, cfg.Embedded.Dimensions)
	}
	fmt.Fprint(f, `    },
    "vector_index": {
      // HNSW (Hierarchical Navigable Small World) vector index parameters.
      // Higher M = better recall at cost of memory.
      // Higher ef = more thorough search at cost of speed.
      "index_type": "hnsw",
`)
	fmt.Fprintf(f, `      "hnsw_m": %d,
`, cfg.VectorIndex.HNSW_M)
	fmt.Fprintf(f, `      "hnsw_ef_construction": %d,
`, cfg.VectorIndex.HNSW_EfConstruction)
	fmt.Fprintf(f, `      "hnsw_ef_search": %d,
`, cfg.VectorIndex.HNSW_EfSearch)
	fmt.Fprint(f, `      // Compression: "none" or "sq8" (4× smaller, lossy).
`)
	fmt.Fprintf(f, `      "compression": %q
`, cfg.VectorIndex.Compression)
	fmt.Fprint(f, `    },
    "remote": {
      // Remote (API-based) embedding provider.
      // Disabled by default; Vectos uses embedded inference.
      "enabled": false,
      "base_url": "",
      "model": "text-embedding-nomic-embed-text-v1.5",
      "timeout_seconds": 30
    }
  },
  // Index exclusion patterns.
  // Patterns use glob syntax (** = recursive, * = single-level).
  // These merge with project-level vectos.config.json and .gitignore.
  "index": {
    "docs": { "exclude": [] },
    "code": { "exclude": [] }
  }
}
`)
	return nil
}
