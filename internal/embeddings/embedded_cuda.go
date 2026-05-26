package embeddings

import (
	"os"

	ort "github.com/yalue/onnxruntime_go"
)

// appendCUDAProvider adds the CUDA execution provider to the ONNX session
// options when VECTOS_CUDA=1. This requires a CUDA-capable ONNX Runtime build
// (onnxruntime-gpu) and compatible NVIDIA drivers/CUDA toolkit installed.
// If the CUDA provider is not available in the current ONNX Runtime build,
// the function returns nil silently and ONNX falls back to CPU.
//
// Set VECTOS_CUDA=1 to enable. Future: VECTOS_CUDA_DEVICE for multi-GPU setups.
func (e *EmbeddedEmbedder) appendCUDAProvider(opts *ort.SessionOptions) error {
	if os.Getenv("VECTOS_CUDA") != "1" {
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
