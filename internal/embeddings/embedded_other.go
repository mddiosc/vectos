//go:build !darwin

package embeddings

import (
	ort "github.com/yalue/onnxruntime_go"
)

// appendAccelerationProviders is a no-op on non-darwin platforms.
// Future: add CUDA, DirectML, or OpenVINO here when needed.
func (e *EmbeddedEmbedder) appendAccelerationProviders(opts *ort.SessionOptions) error {
	return nil
}
