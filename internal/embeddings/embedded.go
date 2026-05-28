package embeddings

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"vectos/internal/config"

	tokenizerpkg "github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

const (
	DefaultEmbeddedDimensions = 384
	defaultSequenceLength     = 512
	defaultORTVersion         = "1.25.0"
)

var (
	requiredEmbeddedAssets = []string{"model.onnx", "tokenizer.json", "config.json"}
	ortInitMu              sync.Mutex
)

type embeddedAssetSpec struct {
	LocalName  string
	RemotePath string
}

type runtimeArchiveSpec struct {
	ArchiveURL  string
	ArchivePath string
	LocalName   string
}

var embeddedModelAssets = map[string][]embeddedAssetSpec{
	"bge-small-en-v1.5": {
		{LocalName: "config.json", RemotePath: "config.json"},
		{LocalName: "tokenizer.json", RemotePath: "tokenizer.json"},
		{LocalName: "model.onnx", RemotePath: "onnx/model.onnx"},
	},
	"jina-embeddings-v3": {
		{LocalName: "config.json", RemotePath: "config.json"},
		{LocalName: "tokenizer.json", RemotePath: "tokenizer.json"},
		{LocalName: "model.onnx", RemotePath: "onnx/model.onnx"},
		{LocalName: "model.onnx_data", RemotePath: "onnx/model.onnx_data"},
	},
}

var runtimeArchiveSpecs = map[string]runtimeArchiveSpec{
	"darwin/arm64": {
		ArchiveURL:  "https://github.com/microsoft/onnxruntime/releases/download/v1.25.0/onnxruntime-osx-arm64-1.25.0.tgz",
		ArchivePath: "onnxruntime-osx-arm64-1.25.0/lib/libonnxruntime.dylib",
		LocalName:   "onnxruntime.dylib",
	},
	"linux/amd64": {
		ArchiveURL:  "https://github.com/microsoft/onnxruntime/releases/download/v1.25.0/onnxruntime-linux-x64-1.25.0.tgz",
		ArchivePath: "onnxruntime-linux-x64-1.25.0/lib/libonnxruntime.so.1.25.0",
		LocalName:   "onnxruntime.so",
	},
	"linux/arm64": {
		ArchiveURL:  "https://github.com/microsoft/onnxruntime/releases/download/v1.25.0/onnxruntime-linux-aarch64-1.25.0.tgz",
		ArchivePath: "onnxruntime-linux-aarch64-1.25.0/lib/libonnxruntime.so.1.25.0",
		LocalName:   "onnxruntime.so",
	},
}

type ProviderStatus struct {
	Provider   string   `json:"provider"`
	Model      string   `json:"model"`
	Dimensions int      `json:"dimensions"`
	Ready      bool     `json:"ready"`
	ModelDir   string   `json:"model_dir,omitempty"`
	Missing    []string `json:"missing_assets,omitempty"`
	Message    string   `json:"message,omitempty"`
}

type EmbeddedEmbedder struct {
	modelName       string
	modelDir        string
	autoDownload    bool
	assetBaseURL    string
	httpClient      *http.Client
	status          ProviderStatus
	tokenizer       *tokenizerpkg.Tokenizer
	session         *ort.DynamicAdvancedSession
	inputNames      []string
	outputNames     []string
	inputInfo       []ort.InputOutputInfo
	outputInfo      []ort.InputOutputInfo
	sequenceLen     int
	embeddingSize   int
	targetDimension int                    // Matryoshka truncation target; 0 = no truncation
	accelCfg        config.AccelerationConfig // hardware acceleration config
	mu              sync.Mutex
}

func NewEmbeddedEmbedder(cfg config.EmbeddedProviderConfig) (*EmbeddedEmbedder, ProviderInfo, error) {
	embedder, status, err := NewEmbeddedEmbedderWithStatus(cfg)
	if err != nil {
		return nil, ProviderInfo{}, err
	}

	return embedder, providerInfoFromStatus(status), nil
}

func NewEmbeddedEmbedderWithStatus(cfg config.EmbeddedProviderConfig) (*EmbeddedEmbedder, ProviderStatus, error) {
	status := ProviderStatus{
		Provider:   config.ProviderEmbedded,
		Model:      strings.TrimSpace(cfg.ModelName),
		Dimensions: DefaultEmbeddedDimensions,
		ModelDir:   strings.TrimSpace(cfg.ModelDir),
	}

	if !cfg.Enabled {
		status.Message = "embedded provider disabled"
		return nil, status, fmt.Errorf("%s", status.Message)
	}
	if status.Model == "" {
		status.Message = "embedded model name is required"
		return nil, status, fmt.Errorf("%s", status.Message)
	}
	if status.ModelDir == "" {
		status.Message = "embedded model directory is required"
		return nil, status, fmt.Errorf("%s", status.Message)
	}
	if err := config.ValidateEmbeddedDimensions(status.Model, cfg.Dimensions); err != nil {
		status.Message = err.Error()
		return nil, status, err
	}

	timeout := 60 * time.Second
	if cfg.TimeoutS > 0 {
		timeout = time.Duration(cfg.TimeoutS) * time.Second
	}

	embedder := &EmbeddedEmbedder{
		modelName:       status.Model,
		modelDir:        status.ModelDir,
		autoDownload:    cfg.AutoDownload,
		assetBaseURL:    strings.TrimRight(strings.TrimSpace(cfg.AssetBaseURL), "/"),
		httpClient:      &http.Client{Timeout: timeout},
		status:          status,
		sequenceLen:     defaultSequenceLength,
		targetDimension: cfg.Dimensions,
		accelCfg:        cfg.Acceleration,
	}

	if err := embedder.ensureModelReady(); err != nil {
		return nil, embedder.status, err
	}

	// Apply Matryoshka truncation: if the target dimension is smaller than
	// the model's native embedding size, report the truncated dimension.
	if config.SupportsMatryoshkaDimensions(embedder.modelName) && embedder.targetDimension > 0 && embedder.targetDimension < embedder.embeddingSize {
		embedder.status.Dimensions = embedder.targetDimension
	}

	return embedder, embedder.status, nil
}

func (e *EmbeddedEmbedder) GetEmbedding(text string) ([]float32, error) {
	vecs, err := e.GetEmbeddings([]string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// GetEmbeddings tokeniza todos los textos, los agrupa en un solo batch con
// padding al máximo sequence length del batch, ejecuta una sola inferencia
// ONNX con shape [N, max_seq_len], y extrae y normaliza cada vector de salida.
func (e *EmbeddedEmbedder) GetEmbeddings(texts []string) ([][]float32, error) {
	if !e.status.Ready {
		return nil, fmt.Errorf("embedded model %q is not ready in %s", e.modelName, e.modelDir)
	}
	if len(texts) == 0 {
		return nil, nil
	}

	batchSize := len(texts)

	// Tokenize all texts and find the maximum sequence length in the batch.
	type tokenized struct {
		ids  []int64
		mask []int64
	}
	tokenizedTexts := make([]tokenized, batchSize)
	maxLen := 0
	for i, text := range texts {
		ids, mask, _, err := e.tokenizeRaw(text)
		if err != nil {
			return nil, fmt.Errorf("text %d: %w", i, err)
		}
		tokenizedTexts[i] = tokenized{ids: ids, mask: mask}
		if len(ids) > maxLen {
			maxLen = len(ids)
		}
	}

	// Ensure at least one position.
	if maxLen <= 0 {
		maxLen = 1
	}

	// Pad all sequences to maxLen and flatten into single arrays.
	flatIDs := make([]int64, batchSize*maxLen)
	flatMask := make([]int64, batchSize*maxLen)
	maskRefs := make([][]int64, batchSize)

	for i, tok := range tokenizedTexts {
		offset := i * maxLen
		copyLen := len(tok.ids)
		if copyLen > maxLen {
			copyLen = maxLen
		}
		copy(flatIDs[offset:offset+copyLen], tok.ids[:copyLen])
		copy(flatMask[offset:offset+copyLen], tok.mask[:copyLen])
		maskRefs[i] = flatMask[offset : offset+maxLen]
	}

	outputTensor, err := e.runBatchedInference(flatIDs, flatMask, batchSize, maxLen)
	if err != nil {
		return nil, err
	}
	defer outputTensor.Destroy()

	data := outputTensor.GetData()
	shape := outputTensor.GetShape()
	if len(shape) != 3 {
		return nil, fmt.Errorf("unexpected embedded output rank %d", len(shape))
	}

	outSeqLen := int(shape[1])
	hiddenSize := int(shape[2])
	if outSeqLen <= 0 || hiddenSize <= 0 {
		return nil, fmt.Errorf("unexpected embedded output shape %v", shape)
	}

	// Extract per-sequence embeddings.
	results := make([][]float32, batchSize)
	stride := outSeqLen * hiddenSize
	for i := 0; i < batchSize; i++ {
		seqData := data[i*stride : (i+1)*stride]
		embedding := meanPoolAndNormalize(seqData, maskRefs[i], outSeqLen, hiddenSize)
		if len(embedding) == 0 {
			return nil, fmt.Errorf("embedded pooling produced empty vector for text %d", i)
		}
		// Matryoshka truncation: reduce to target dimension and re-normalize.
		if config.SupportsMatryoshkaDimensions(e.modelName) && e.targetDimension > 0 && e.targetDimension < len(embedding) {
			embedding = truncateAndNormalize(embedding, e.targetDimension)
		}
		results[i] = embedding
	}

	return results, nil
}

// tokenizeRaw tokenizes a single text and returns raw (unpadded) token IDs and
// attention mask.
func (e *EmbeddedEmbedder) tokenizeRaw(text string) ([]int64, []int64, []int64, error) {
	encoding, err := e.tokenizer.EncodeSingle(text, true)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to tokenize text: %w", err)
	}

	ids := encoding.GetIds()
	mask := encoding.GetAttentionMask()
	if len(mask) == 0 {
		mask = make([]int, len(ids))
		for i := range mask {
			mask[i] = 1
		}
	}

	// Truncate to configured sequence length.
	seqLimit := e.sequenceLen
	if seqLimit <= 0 {
		seqLimit = defaultSequenceLength
	}
	if len(ids) > seqLimit {
		ids = ids[:seqLimit]
		mask = mask[:seqLimit]
	}

	int64IDs := make([]int64, len(ids))
	int64Mask := make([]int64, len(mask))
	tokenTypeIDs := make([]int64, len(ids))
	for i := range ids {
		int64IDs[i] = int64(ids[i])
		int64Mask[i] = int64(mask[i])
	}
	return int64IDs, int64Mask, tokenTypeIDs, nil
}

// runBatchedInference executes an ONNX inference with batched inputs of shape
// [batchSize, seqLen]. It builds input tensors dynamically based on the model's
// actual input names, supporting both legacy token_type_ids and scalar task_id.
func (e *EmbeddedEmbedder) runBatchedInference(inputIDs, attentionMask []int64, batchSize, seqLen int) (*ort.Tensor[float32], error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	inputShape := ort.NewShape(int64(batchSize), int64(seqLen))
	inputValues := make([]ort.Value, len(e.inputNames))

	for i, name := range e.inputNames {
		switch name {
		case "input_ids":
			tensor, err := ort.NewTensor(inputShape, inputIDs)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s tensor: %w", name, err)
			}
			inputValues[i] = tensor
			defer tensor.Destroy()
		case "attention_mask":
			tensor, err := ort.NewTensor(inputShape, attentionMask)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s tensor: %w", name, err)
			}
			inputValues[i] = tensor
			defer tensor.Destroy()
		case "token_type_ids":
			tokenTypeIDs := make([]int64, batchSize*seqLen)
			tensor, err := ort.NewTensor(inputShape, tokenTypeIDs)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s tensor: %w", name, err)
			}
			inputValues[i] = tensor
			defer tensor.Destroy()
		case "task_id":
			// Jina-embeddings-v3 style scalar task_id.
			// 1 = retrieval passage (used for indexing content).
			taskID := []int64{1}
			tensor, err := ort.NewTensor(ort.NewShape(1), taskID)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s tensor: %w", name, err)
			}
			inputValues[i] = tensor
			defer tensor.Destroy()
		default:
			return nil, fmt.Errorf("unsupported model input %q", name)
		}
	}

	outputs := make([]ort.Value, len(e.outputNames))
	if err := e.session.Run(inputValues, outputs); err != nil {
		return nil, fmt.Errorf("failed to run embedded ONNX session: %w", err)
	}

	if len(outputs) == 0 || outputs[0] == nil {
		return nil, fmt.Errorf("embedded ONNX session returned no outputs")
	}

	tensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		for _, output := range outputs {
			if output != nil {
				_ = output.Destroy()
			}
		}
		return nil, fmt.Errorf("unexpected output type at index 0")
	}

	// Destroy any extra output tensors (e.g., jina-embeddings-v3 has two outputs).
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != nil {
			_ = outputs[i].Destroy()
		}
	}

	return tensor, nil
}

func (e *EmbeddedEmbedder) Status() ProviderStatus {
	return e.status
}

func InspectEmbeddedProvider(cfg config.EmbeddedProviderConfig) ProviderStatus {
	_, status, err := NewEmbeddedEmbedderWithStatus(cfg)
	if err != nil {
		return status
	}
	return status
}

func providerInfoFromStatus(status ProviderStatus) ProviderInfo {
	return ProviderInfo{
		Provider:   status.Provider,
		Model:      status.Model,
		Dimensions: status.Dimensions,
		Ready:      status.Ready,
		Message:    status.Message,
	}
}

func (e *EmbeddedEmbedder) ensureModelReady() error {
	if err := e.downloadAndValidateAssets(); err != nil {
		return err
	}
	if err := e.initializeRuntime(); err != nil {
		return err
	}
	if err := e.createONNXSession(); err != nil {
		return err
	}

	e.status.Ready = true
	e.status.Missing = nil
	e.status.Message = "embedded provider ready"
	return nil
}

// downloadAndValidateAssets ensures the model directory exists and all
// required asset files are present, downloading them if auto-download is enabled.
func (e *EmbeddedEmbedder) downloadAndValidateAssets() error {
	if err := os.MkdirAll(e.modelDir, 0755); err != nil {
		e.status.Message = fmt.Sprintf("failed to create embedded model directory: %v", err)
		return err
	}

	missing := missingEmbeddedAssets(e.modelName, e.modelDir)
	if len(missing) > 0 && e.autoDownload {
		if err := e.downloadMissingAssets(missing); err != nil {
			missing = missingEmbeddedAssets(e.modelName, e.modelDir)
			e.status.Missing = missing
			e.status.Message = err.Error()
			return err
		}
		missing = missingEmbeddedAssets(e.modelName, e.modelDir)
	}

	if len(missing) > 0 {
		e.status.Missing = missing
		e.status.Message = fmt.Sprintf("embedded model assets missing in %s: %s", e.modelDir, strings.Join(missing, ", "))
		return fmt.Errorf("%s", e.status.Message)
	}
	return nil
}

// initializeRuntime ensures the ONNX Runtime shared library is present and
// the runtime environment is initialized.
func (e *EmbeddedEmbedder) initializeRuntime() error {
	runtimePath, err := e.ensureRuntimeLibrary()
	if err != nil {
		e.status.Message = err.Error()
		return err
	}

	if err := ensureORTSession(runtimePath); err != nil {
		e.status.Message = err.Error()
		return err
	}
	return nil
}

// createONNXSession loads the tokenizer, inspects the ONNX model, and
// creates the inference session.
func (e *EmbeddedEmbedder) createONNXSession() error {
	tk, err := pretrained.FromFile(filepath.Join(e.modelDir, "tokenizer.json"))
	if err != nil {
		e.status.Message = fmt.Sprintf("failed to load tokenizer: %v", err)
		return err
	}
	e.tokenizer = tk

	modelPath := filepath.Join(e.modelDir, "model.onnx")
	inputs, outputs, err := ort.GetInputOutputInfo(modelPath)
	if err != nil {
		e.status.Message = fmt.Sprintf("failed to inspect embedded ONNX model: %v", err)
		return err
	}
	if len(inputs) < 2 {
		e.status.Message = "embedded ONNX model must expose at least input_ids and attention_mask"
		return fmt.Errorf("%s", e.status.Message)
	}
	if len(outputs) < 1 {
		e.status.Message = "embedded ONNX model must expose at least one output"
		return fmt.Errorf("%s", e.status.Message)
	}

	e.inputInfo = inputs
	e.outputInfo = outputs
	e.inputNames = collectIONames(inputs)
	e.outputNames = collectIONames(outputs)
	e.sequenceLen = detectSequenceLength(inputs)
	e.embeddingSize = detectEmbeddingSize(outputs)
	if e.embeddingSize > 0 {
		e.status.Dimensions = e.embeddingSize
	}

	opts, err := ort.NewSessionOptions()
	if err != nil {
		e.status.Message = fmt.Sprintf("failed to create session options: %v", err)
		return err
	}
	defer opts.Destroy()

	// ORT_ENABLE_ALL = 99 — full graph optimizations (constant fusion, node
	// elimination, etc). The Go wrapper only exports DisableAll, but the
	// underlying C enum is an int that accepts any GraphOptimizationLevel value.
	opts.SetGraphOptimizationLevel(ort.GraphOptimizationLevel(99))
	opts.SetIntraOpNumThreads(runtime.NumCPU())

	// Append platform-specific acceleration providers (CoreML on darwin, etc).
	// Failure is non-fatal — ONNX falls back to CPU automatically.
	_ = e.appendAccelerationProviders(opts)

	session, err := ort.NewDynamicAdvancedSession(modelPath, e.inputNames, e.outputNames, opts)
	if err != nil {
		e.status.Message = fmt.Sprintf("failed to create embedded ONNX session: %v", err)
		return err
	}
	e.session = session
	return nil
}

func ensureORTSession(sharedLibraryPath string) error {
	ortInitMu.Lock()
	defer ortInitMu.Unlock()

	if ort.IsInitialized() {
		return nil
	}

	if strings.TrimSpace(sharedLibraryPath) != "" {
		if _, err := os.Stat(sharedLibraryPath); err == nil {
			ort.SetSharedLibraryPath(sharedLibraryPath)
		}
	}

	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("failed to initialize ONNX runtime: %w", err)
	}

	return nil
}

func collectIONames(items []ort.InputOutputInfo) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func detectSequenceLength(inputs []ort.InputOutputInfo) int {
	for _, input := range inputs {
		if len(input.Dimensions) >= 2 && input.Dimensions[1] > 0 {
			return int(input.Dimensions[1])
		}
	}
	return defaultSequenceLength
}

func detectEmbeddingSize(outputs []ort.InputOutputInfo) int {
	for _, output := range outputs {
		if len(output.Dimensions) >= 3 && output.Dimensions[2] > 0 {
			return int(output.Dimensions[2])
		}
		if len(output.Dimensions) >= 2 && output.Dimensions[1] > 0 {
			return int(output.Dimensions[1])
		}
	}
	return DefaultEmbeddedDimensions
}

func meanPoolAndNormalize(data []float32, attentionMask []int64, seqLen, hiddenSize int) []float32 {
	if len(data) < seqLen*hiddenSize || hiddenSize <= 0 {
		return nil
	}

	pooled := make([]float32, hiddenSize)
	var tokenCount float32
	for tokenIndex := 0; tokenIndex < seqLen && tokenIndex < len(attentionMask); tokenIndex++ {
		if attentionMask[tokenIndex] == 0 {
			continue
		}
		tokenCount++
		base := tokenIndex * hiddenSize
		for hiddenIndex := 0; hiddenIndex < hiddenSize; hiddenIndex++ {
			pooled[hiddenIndex] += data[base+hiddenIndex]
		}
	}

	if tokenCount == 0 {
		return nil
	}

	var norm float64
	for i := range pooled {
		pooled[i] /= tokenCount
		norm += float64(pooled[i] * pooled[i])
	}

	if norm == 0 {
		return pooled
	}

	denominator := float32(math.Sqrt(norm))
	for i := range pooled {
		pooled[i] /= denominator
	}

	return pooled
}

// truncateAndNormalize implements Matryoshka Representation Learning (MRL)
// truncation: it slices the embedding to the first targetDim dimensions and
// re-normalizes to unit length. Re-normalization is required because the
// original L2 norm was computed over the full vector; truncating changes the
// magnitude, which would break cosine similarity thresholds.
func truncateAndNormalize(embedding []float32, targetDim int) []float32 {
	if targetDim <= 0 || targetDim >= len(embedding) {
		return embedding
	}
	truncated := make([]float32, targetDim)
	copy(truncated, embedding[:targetDim])

	var norm float64
	for _, v := range truncated {
		norm += float64(v * v)
	}
	if norm > 0 {
		denominator := float32(math.Sqrt(norm))
		for i := range truncated {
			truncated[i] /= denominator
		}
	}
	return truncated
}

func (e *EmbeddedEmbedder) ensureRuntimeLibrary() (string, error) {
	platformKey := runtime.GOOS + "/" + runtime.GOARCH
	spec, ok := runtimeArchiveSpecs[platformKey]
	if !ok {
		return "", fmt.Errorf("no bundled ONNX Runtime download configured for %s", platformKey)
	}

	localPath := filepath.Join(e.modelDir, spec.LocalName)
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() {
		return localPath, nil
	}

	if !e.autoDownload {
		return "", fmt.Errorf("onnx runtime library missing in %s and auto-download is disabled", localPath)
	}

	if err := e.downloadRuntimeLibrary(spec, localPath); err != nil {
		return "", err
	}

	return localPath, nil
}

func (e *EmbeddedEmbedder) downloadRuntimeLibrary(spec runtimeArchiveSpec, localPath string) error {
	tmpArchivePath := localPath + ".download"
	resp, err := e.httpClient.Get(spec.ArchiveURL)
	if err != nil {
		return fmt.Errorf("failed to download ONNX Runtime archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download ONNX Runtime archive: status %d", resp.StatusCode)
	}

	// Validate Content-Type to prevent downloading unexpected content.
	if err := validateDownloadContentType(resp); err != nil {
		return fmt.Errorf("refusing to download ONNX Runtime: %w", err)
	}

	if err := downloadArchive(resp.Body, tmpArchivePath); err != nil {
		return err
	}
	defer os.Remove(tmpArchivePath)

	return extractTarMember(tmpArchivePath, spec.ArchivePath, localPath)
}

func downloadArchive(body io.Reader, tmpArchivePath string) error {
	archiveFile, err := os.Create(tmpArchivePath)
	if err != nil {
		return fmt.Errorf("failed to create temporary ONNX Runtime archive: %w", err)
	}

	_, copyErr := io.Copy(archiveFile, body)
	closeErr := archiveFile.Close()
	if copyErr != nil {
		_ = os.Remove(tmpArchivePath)
		return fmt.Errorf("failed to write ONNX Runtime archive: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpArchivePath)
		return fmt.Errorf("failed to finalize ONNX Runtime archive: %w", closeErr)
	}
	return nil
}

func extractTarMember(archivePath, memberPath, localPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open ONNX Runtime archive: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to read ONNX Runtime archive: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to scan ONNX Runtime archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg || header.Name != memberPath {
			continue
		}

		return extractSingleFile(tarReader, localPath)
	}

	return fmt.Errorf("failed to find %s in ONNX Runtime archive", memberPath)
}

func extractSingleFile(tarReader *tar.Reader, localPath string) error {
	tmpPath := localPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create extracted ONNX Runtime library: %w", err)
	}

	_, copyErr := io.Copy(out, tarReader)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to extract ONNX Runtime library: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize extracted ONNX Runtime library: %w", closeErr)
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to set ONNX Runtime library permissions: %w", err)
	}
	if err := os.Rename(tmpPath, localPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to install ONNX Runtime library: %w", err)
	}
	return nil
}

func (e *EmbeddedEmbedder) downloadMissingAssets(missing []string) error {
	if e.assetBaseURL == "" {
		return fmt.Errorf("embedded model assets missing and auto-download has no asset_base_url configured")
	}

	assetMap := e.assetSpecsByLocalName()
	for _, asset := range missing {
		spec, ok := assetMap[asset]
		if !ok {
			spec = embeddedAssetSpec{LocalName: asset, RemotePath: asset}
		}
		if err := e.downloadAsset(spec); err != nil {
			return err
		}
	}

	return nil
}

func (e *EmbeddedEmbedder) downloadAsset(asset embeddedAssetSpec) error {
	remotePath := strings.TrimLeft(asset.RemotePath, "/")
	url := e.assetBaseURL + "/" + remotePath
	tmpPath := filepath.Join(e.modelDir, asset.LocalName+".tmp")
	finalPath := filepath.Join(e.modelDir, asset.LocalName)

	resp, err := e.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download embedded asset %s: %w", asset.LocalName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download embedded asset %s: status %d", asset.LocalName, resp.StatusCode)
	}

	// Validate Content-Type to prevent downloading unexpected content.
	if err := validateDownloadContentType(resp); err != nil {
		return fmt.Errorf("refusing to download embedded asset %s: %w", asset.LocalName, err)
	}

	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp asset %s: %w", asset.LocalName, err)
	}

	_, copyErr := io.Copy(file, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write embedded asset %s: %w", asset.LocalName, copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize embedded asset %s: %w", asset.LocalName, closeErr)
	}

	// Check that we actually downloaded content.
	if info, statErr := os.Stat(tmpPath); statErr == nil && info.Size() == 0 {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("refusing to use empty download for embedded asset %s", asset.LocalName)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to move embedded asset %s into place: %w", asset.LocalName, err)
	}

	return nil
}

func (e *EmbeddedEmbedder) assetSpecsByLocalName() map[string]embeddedAssetSpec {
	assets := embeddedModelAssets[e.modelName]
	if len(assets) == 0 {
		assets = make([]embeddedAssetSpec, 0, len(requiredEmbeddedAssets))
		for _, asset := range requiredEmbeddedAssets {
			assets = append(assets, embeddedAssetSpec{LocalName: asset, RemotePath: asset})
		}
	}

	byName := make(map[string]embeddedAssetSpec, len(assets))
	for _, asset := range assets {
		byName[asset.LocalName] = asset
	}

	return byName
}

func missingEmbeddedAssets(modelName, modelDir string) []string {
	// Build the complete list of required files for this model.
	required := make(map[string]bool)
	for _, asset := range requiredEmbeddedAssets {
		required[asset] = true
	}
	if specs, ok := embeddedModelAssets[modelName]; ok {
		for _, spec := range specs {
			required[spec.LocalName] = true
		}
	}

	missing := make([]string, 0, len(required))
	for asset := range required {
		path := filepath.Join(modelDir, asset)
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			missing = append(missing, asset)
		}
	}
	sort.Strings(missing)
	return missing
}

// validateDownloadContentType checks that the HTTP response Content-Type is an
// expected binary type. Empty Content-Type is accepted (some CDNs omit it).
var allowedDownloadContentTypes = map[string]bool{
	"application/octet-stream": true,
	"application/gzip":         true,
	"application/x-gzip":       true,
	"application/x-tar":        true,
	"text/plain":               true,
	"application/json":         true,
}

func validateDownloadContentType(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return nil
	}
	// Strip parameters (e.g., "application/octet-stream; charset=binary")
	if i := strings.IndexByte(contentType, ';'); i != -1 {
		contentType = strings.TrimSpace(contentType[:i])
	}
	if !allowedDownloadContentTypes[contentType] {
		return fmt.Errorf("unexpected Content-Type: %q", resp.Header.Get("Content-Type"))
	}
	return nil
}
