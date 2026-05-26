//go:build darwin

package embeddings

import (
	"os"

	ort "github.com/yalue/onnxruntime_go"
)

// appendAccelerationProviders configures platform-specific hardware acceleration
// for the ONNX session. On macOS this adds the CoreML execution provider, which
// offloads inference to the Apple Neural Engine / GPU / AMX coprocessor.
//
// The embedder's accelCfg.CoreML field controls this (default: true on darwin).
// Set VECTOS_COREML=0 to override and force CPU-only inference.
func (e *EmbeddedEmbedder) appendAccelerationProviders(opts *ort.SessionOptions) error {
	// Env var overrides config: VECTOS_COREML=0 disables CoreML regardless of config.
	if os.Getenv("VECTOS_COREML") == "0" {
		return nil
	}
	if !e.accelCfg.CoreML {
		return nil
	}
	return opts.AppendExecutionProviderCoreML(0)
}
