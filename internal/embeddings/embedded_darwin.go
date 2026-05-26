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
// Set VECTOS_COREML=0 to disable and force CPU-only inference.
func (e *EmbeddedEmbedder) appendAccelerationProviders(opts *ort.SessionOptions) error {
	if os.Getenv("VECTOS_COREML") == "0" {
		return nil
	}
	return opts.AppendExecutionProviderCoreML(0)
}
