//go:build !darwin

package embeddings

import (
	ort "github.com/yalue/onnxruntime_go"
)

// appendAccelerationProviders configures platform-specific hardware acceleration
// for non-darwin platforms. Currently supports CUDA when VECTOS_CUDA=1.
// The CUDA provider requires a CUDA-capable ONNX Runtime build.
func (e *EmbeddedEmbedder) appendAccelerationProviders(opts *ort.SessionOptions) error {
	return e.appendCUDAProvider(opts)
}
