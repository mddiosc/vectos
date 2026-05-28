package embeddings

import (
	"os"

	ort "github.com/yalue/onnxruntime_go"
)

// appendCUDAProvider adds the CUDA execution provider to the ONNX session
// options. The embedder's accelCfg.CUDA field controls this (default: false).
// Set VECTOS_CUDA=1 to override and enable CUDA regardless of config.
//
// Requires a CUDA-capable ONNX Runtime build (onnxruntime-gpu) and compatible
// NVIDIA drivers/CUDA toolkit installed. If the CUDA provider is not available
// in the current ONNX Runtime build, the function returns nil silently.
func (e *EmbeddedEmbedder) appendCUDAProvider(opts *ort.SessionOptions) error {
	// Env var overrides config: VECTOS_CUDA=1 enables CUDA regardless of config.
	cudaEnabled := e.accelCfg.CUDA
	if os.Getenv("VECTOS_CUDA") == "1" {
		cudaEnabled = true
	}
	if !cudaEnabled {
		return nil
	}

	cudaOpts, err := ort.NewCUDAProviderOptions()
	if err != nil {
		// CUDA provider not compiled into this ONNX Runtime build — non-fatal.
		return nil
	}
	if err := cudaOpts.Update(map[string]string{"device_id": "0"}); err != nil {
		cudaOpts.Destroy()
		return nil
	}
	err = opts.AppendExecutionProviderCUDA(cudaOpts)
	cudaOpts.Destroy()
	return err
}
