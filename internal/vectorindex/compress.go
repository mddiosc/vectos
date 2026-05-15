package vectorindex

import "math"

// SQ8Params stores per-dimension min/max for quantization.
type SQ8Params struct {
	Mins []float32
	Maxs []float32
	Dim  int
}

// ComputeSQ8Params computes min/max per dimension across all vectors.
func ComputeSQ8Params(vectors [][]float32) *SQ8Params {
	if len(vectors) == 0 {
		return &SQ8Params{}
	}
	dim := len(vectors[0])
	params := &SQ8Params{Mins: make([]float32, dim), Maxs: make([]float32, dim), Dim: dim}
	for i := 0; i < dim; i++ {
		params.Mins[i] = float32(math.MaxFloat32)
		params.Maxs[i] = float32(-math.MaxFloat32)
	}
	for _, v := range vectors {
		if len(v) != dim {
			panic("vectorindex: vector dimension mismatch")
		}
		for i, x := range v {
			if x < params.Mins[i] { params.Mins[i] = x }
			if x > params.Maxs[i] { params.Maxs[i] = x }
		}
	}
	return params
}

// EncodeSQ8 quantizes vectors to int8 using the params.
func EncodeSQ8(vectors [][]float32, params *SQ8Params) []int8 {
	if len(vectors) == 0 || params == nil || params.Dim == 0 {
		return nil
	}
	out := make([]int8, 0, len(vectors)*params.Dim)
	for _, v := range vectors {
		for i, x := range v {
			min, max := params.Mins[i], params.Maxs[i]
			if max == min {
				out = append(out, 0)
				continue
			}
			n := math.Round(float64((x-min)/(max-min)*255 - 128))
			if n < -128 { n = -128 }
			if n > 127 { n = 127 }
			out = append(out, int8(n))
		}
	}
	return out
}

// DecodeSQ8 reconstructs float32 vectors from int8 using the params.
func DecodeSQ8(encoded []int8, params *SQ8Params) [][]float32 {
	if params == nil || params.Dim == 0 || len(encoded) == 0 {
		return nil
	}
	count := len(encoded) / params.Dim
	out := make([][]float32, count)
	for v := 0; v < count; v++ {
		vec := make([]float32, params.Dim)
		for i := 0; i < params.Dim; i++ {
			x := float64(encoded[v*params.Dim+i])
			min, max := float64(params.Mins[i]), float64(params.Maxs[i])
			if max == min { vec[i] = float32(min); continue }
			vec[i] = float32(((x+128)/255)*(max-min) + min)
		}
		out[v] = vec
	}
	return out
}
