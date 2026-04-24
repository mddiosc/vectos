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
	modelName     string
	modelDir      string
	autoDownload  bool
	assetBaseURL  string
	httpClient    *http.Client
	status        ProviderStatus
	tokenizer     *tokenizerpkg.Tokenizer
	session       *ort.DynamicAdvancedSession
	inputNames    []string
	outputNames   []string
	inputInfo     []ort.InputOutputInfo
	outputInfo    []ort.InputOutputInfo
	sequenceLen   int
	embeddingSize int
	mu            sync.Mutex
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

	timeout := 60 * time.Second
	if cfg.TimeoutS > 0 {
		timeout = time.Duration(cfg.TimeoutS) * time.Second
	}

	embedder := &EmbeddedEmbedder{
		modelName:    status.Model,
		modelDir:     status.ModelDir,
		autoDownload: cfg.AutoDownload,
		assetBaseURL: strings.TrimRight(strings.TrimSpace(cfg.AssetBaseURL), "/"),
		httpClient:   &http.Client{Timeout: timeout},
		status:       status,
		sequenceLen:  defaultSequenceLength,
	}

	if err := embedder.ensureModelReady(); err != nil {
		return nil, embedder.status, err
	}

	return embedder, embedder.status, nil
}

func (e *EmbeddedEmbedder) GetEmbedding(text string) ([]float32, error) {
	if !e.status.Ready {
		return nil, fmt.Errorf("embedded model %q is not ready in %s", e.modelName, e.modelDir)
	}

	inputIDs, attentionMask, tokenTypeIDs, err := e.encodeText(text)
	if err != nil {
		return nil, err
	}

	outputTensor, err := e.runInference(inputIDs, attentionMask, tokenTypeIDs)
	if err != nil {
		return nil, err
	}
	defer outputTensor.Destroy()

	data := outputTensor.GetData()
	shape := outputTensor.GetShape()
	if len(shape) != 3 {
		return nil, fmt.Errorf("unexpected embedded output rank %d", len(shape))
	}

	seqLen := int(shape[1])
	hiddenSize := int(shape[2])
	if seqLen <= 0 || hiddenSize <= 0 {
		return nil, fmt.Errorf("unexpected embedded output shape %v", shape)
	}

	embedding := meanPoolAndNormalize(data, attentionMask, seqLen, hiddenSize)
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedded pooling produced empty vector")
	}

	return embedding, nil
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
	if err := os.MkdirAll(e.modelDir, 0755); err != nil {
		e.status.Message = fmt.Sprintf("failed to create embedded model directory: %v", err)
		return err
	}

	missing := missingEmbeddedAssets(e.modelDir)
	if len(missing) > 0 && e.autoDownload {
		if err := e.downloadMissingAssets(missing); err != nil {
			missing = missingEmbeddedAssets(e.modelDir)
			e.status.Missing = missing
			e.status.Message = err.Error()
			return err
		}
		missing = missingEmbeddedAssets(e.modelDir)
	}

	if len(missing) > 0 {
		e.status.Missing = missing
		e.status.Message = fmt.Sprintf("embedded model assets missing in %s: %s", e.modelDir, strings.Join(missing, ", "))
		return fmt.Errorf("%s", e.status.Message)
	}

	runtimePath, err := e.ensureRuntimeLibrary()
	if err != nil {
		e.status.Message = err.Error()
		return err
	}

	if err := ensureORTSession(runtimePath); err != nil {
		e.status.Message = err.Error()
		return err
	}

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

	session, err := ort.NewDynamicAdvancedSession(modelPath, e.inputNames, e.outputNames, nil)
	if err != nil {
		e.status.Message = fmt.Sprintf("failed to create embedded ONNX session: %v", err)
		return err
	}
	e.session = session

	e.status.Ready = true
	e.status.Missing = nil
	e.status.Message = "embedded provider ready"
	return nil
}

func (e *EmbeddedEmbedder) encodeText(text string) ([]int64, []int64, []int64, error) {
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

	sequenceLen := e.sequenceLen
	if sequenceLen <= 0 {
		sequenceLen = len(ids)
	}
	if sequenceLen <= 0 {
		sequenceLen = defaultSequenceLength
	}

	inputIDs := make([]int64, sequenceLen)
	attentionMask := make([]int64, sequenceLen)
	tokenTypeIDs := make([]int64, sequenceLen)
	limit := minInt(len(ids), sequenceLen)
	for i := 0; i < limit; i++ {
		inputIDs[i] = int64(ids[i])
		attentionMask[i] = int64(mask[i])
	}

	return inputIDs, attentionMask, tokenTypeIDs, nil
}

func (e *EmbeddedEmbedder) runInference(inputIDs, attentionMask, tokenTypeIDs []int64) (*ort.Tensor[float32], error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	inputShape := ort.NewShape(1, int64(len(inputIDs)))
	inputTensor, err := ort.NewTensor(inputShape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputTensor.Destroy()

	maskTensor, err := ort.NewTensor(inputShape, attentionMask)
	if err != nil {
		return nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	defer maskTensor.Destroy()

	tokenTypeTensor, err := ort.NewTensor(inputShape, tokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create token_type_ids tensor: %w", err)
	}
	defer tokenTypeTensor.Destroy()

	inputs := []ort.Value{inputTensor, maskTensor, tokenTypeTensor}
	outputs := make([]ort.Value, len(e.outputNames))
	if err := e.session.Run(inputs, outputs); err != nil {
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
		return nil, fmt.Errorf("embedded ONNX output is not a float32 tensor")
	}

	for i := 1; i < len(outputs); i++ {
		if outputs[i] != nil {
			_ = outputs[i].Destroy()
		}
	}

	return tensor, nil
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

	archiveFile, err := os.Create(tmpArchivePath)
	if err != nil {
		return fmt.Errorf("failed to create temporary ONNX Runtime archive: %w", err)
	}

	_, copyErr := io.Copy(archiveFile, resp.Body)
	closeErr := archiveFile.Close()
	if copyErr != nil {
		_ = os.Remove(tmpArchivePath)
		return fmt.Errorf("failed to write ONNX Runtime archive: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpArchivePath)
		return fmt.Errorf("failed to finalize ONNX Runtime archive: %w", closeErr)
	}
	defer os.Remove(tmpArchivePath)

	if err := extractTarMember(tmpArchivePath, spec.ArchivePath, localPath); err != nil {
		return err
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

	return fmt.Errorf("failed to find %s in ONNX Runtime archive", memberPath)
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

func missingEmbeddedAssets(modelDir string) []string {
	missing := make([]string, 0, len(requiredEmbeddedAssets))
	for _, asset := range requiredEmbeddedAssets {
		path := filepath.Join(modelDir, asset)
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			missing = append(missing, asset)
		}
	}
	sort.Strings(missing)
	return missing
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
