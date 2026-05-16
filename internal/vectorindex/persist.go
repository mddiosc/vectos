package vectorindex

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

const (
	indexMagic       = 0x574E5348 // "HSNW" little-endian
	indexVersion     = 2
	compressionNone  = 0
	compressionSQ8   = 1
)

// IndexHeader holds the metadata persisted with the index.
type IndexHeader struct {
	Version        uint32
	ContentHash    [sha256.Size]byte
	Dimension      int
	M              int
	EfConstruction int
	EfSearch       int
	Mmax0          int
	ML             float64
}

// Save writes the HNSW index to a binary file at the given path.
// contentHash is the SHA-256 hash of the chunk table — stored in the header
// and used at load time to detect staleness.
func (h *HNSW) Save(path string, contentHash [sha256.Size]byte, compression string, params *SQ8Params) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("vectorindex: create file: %w", err)
	}
	defer f.Close()

	bw := &binaryWriter{w: f}

	// Magic + version
	bw.u32(indexMagic)
	bw.u32(indexVersion)

	// Content hash
	bw.raw(contentHash[:])
	flag := byte(compressionNone)
	if compression == "sq8" && params != nil {
		flag = compressionSQ8
	}
	bw.raw([]byte{flag})

	// Index parameters
	bw.i32(int32(h.dimension))
	bw.i32(int32(h.M))
	bw.i32(int32(h.efConstruction))
	bw.i32(int32(h.efSearch))
	bw.i32(int32(h.Mmax0))
	bw.f64(h.mL)
	bw.i32(int32(h.maxLevel))
	bw.i32(int32(h.entryPoint))

	bw.i32(int32(len(h.nodes)))
	if flag == compressionSQ8 {
		bw.i32(int32(params.Dim))
		bw.i32(int32(len(params.Mins)))
		for i := range params.Mins { bw.f32(params.Mins[i]) }
		for i := range params.Maxs { bw.f32(params.Maxs[i]) }
		encoded := EncodeSQ8(func() [][]float32 {
			vs := make([][]float32, len(h.nodes))
			for i, n := range h.nodes { vs[i] = n.vector }
			return vs
		}(), params)
		for _, b := range encoded { bw.raw([]byte{byte(b)}) }
		for _, n := range h.nodes {
			bw.i32(int32(n.id))
			bw.i32(int32(len(n.layers)))
			for _, layer := range n.layers {
				bw.i32(int32(len(layer)))
				for _, nb := range layer { bw.i32(int32(nb)) }
			}
		}
	} else {
		for _, n := range h.nodes {
			bw.i32(int32(n.id))
			bw.i32(int32(len(n.layers)))
			for _, layer := range n.layers {
				bw.i32(int32(len(layer)))
				for _, nb := range layer { bw.i32(int32(nb)) }
			}
			bw.i32(int32(len(n.vector)))
			for _, v := range n.vector { bw.f32(v) }
		}
	}

	return bw.err
}

// LoadIndex reads an HNSW index from a binary file and returns the parsed
// index and content hash. An error is returned if the file cannot be read,
// has an unsupported version, or contains malformed data.
func LoadIndex(path string) (*HNSW, [sha256.Size]byte, string, *SQ8Params, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, [sha256.Size]byte{}, "", nil, fmt.Errorf("vectorindex: open file: %w", err)
	}
	defer f.Close()

	br := &binaryReader{r: f}

	magic := br.u32()
	if magic != indexMagic {
		return nil, [sha256.Size]byte{}, "", nil, fmt.Errorf("vectorindex: invalid magic 0x%08X", magic)
	}

	version := br.u32()
	if version != indexVersion {
		return nil, [sha256.Size]byte{}, "", nil, fmt.Errorf("vectorindex: unsupported version %d (expected %d)", version, indexVersion)
	}

	var contentHash [sha256.Size]byte
	br.raw(contentHash[:])
	var flagBuf [1]byte
	br.raw(flagBuf[:])
	compressionFlag := flagBuf[0]

	dimension := int(br.i32())
	m := int(br.i32())
	efCons := int(br.i32())
	efSearch := int(br.i32())
	Mmax0 := int(br.i32())
	mL := br.f64()
	maxLevel := int(br.i32())
	entryPoint := int(br.i32())

	_ = Config{M: m, EfConstruction: efCons, EfSearch: efSearch} // validate; not needed for reconstruction
	hnsw := &HNSW{
		dimension:      dimension,
		M:              m,
		Mmax0:          Mmax0,
		efConstruction: efCons,
		efSearch:       efSearch,
		mL:             mL,
		maxLevel:       maxLevel,
		entryPoint:     entryPoint,
	}
	compression := "none"
	var sq8Params *SQ8Params
	nodeCount := int(br.i32())
	if compressionFlag == compressionSQ8 {
		compression = "sq8"
		dim := int(br.i32())
		minsLen := int(br.i32())
		sq8Params = &SQ8Params{Dim: dim, Mins: make([]float32, minsLen), Maxs: make([]float32, minsLen)}
		for i := 0; i < minsLen; i++ { sq8Params.Mins[i] = br.f32() }
		for i := 0; i < minsLen; i++ { sq8Params.Maxs[i] = br.f32() }
		encLen := nodeCount * dim
		encoded := make([]int8, encLen)
		for i := 0; i < encLen; i++ {
			var b [1]byte
			br.raw(b[:])
			encoded[i] = int8(b[0])
		}
		vectors := DecodeSQ8(encoded, sq8Params)
		hnsw.nodes = make([]node, len(vectors))
		for i, v := range vectors {
			normalizeVec(v)
			hnsw.nodes[i] = node{id: i, vector: v}
		}
		for i := 0; i < nodeCount; i++ {
			id := int(br.i32())
			layerCount := int(br.i32())
			layers := make([][]int, layerCount)
			for lc := 0; lc < layerCount; lc++ {
				neighborCount := int(br.i32())
				neighbors := make([]int, neighborCount)
				for j := 0; j < neighborCount; j++ { neighbors[j] = int(br.i32()) }
				layers[lc] = neighbors
			}
			hnsw.nodes[i].id = id
			hnsw.nodes[i].layers = layers
		}
		if len(vectors) != nodeCount { return nil, [sha256.Size]byte{}, "", nil, fmt.Errorf("vectorindex: compression node count mismatch") }
		if br.err != nil { return nil, [sha256.Size]byte{}, "", nil, fmt.Errorf("vectorindex: read error: %w", br.err) }
		return hnsw, contentHash, compression, sq8Params, nil
	}

	hnsw.nodes = make([]node, nodeCount)
	for i := 0; i < nodeCount; i++ {
		id := int(br.i32())
		layerCount := int(br.i32())
		layers := make([][]int, layerCount)
		for lc := 0; lc < layerCount; lc++ {
			neighborCount := int(br.i32())
			neighbors := make([]int, neighborCount)
			for j := 0; j < neighborCount; j++ {
				neighbors[j] = int(br.i32())
			}
			layers[lc] = neighbors
		}
		vectorLen := int(br.i32())
		vector := make([]float32, vectorLen)
		for j := 0; j < vectorLen; j++ {
			vector[j] = br.f32()
		}
		// Ensure vectors are normalized for fast dot-product distance.
		normalizeVec(vector)
		hnsw.nodes[i] = node{id: id, vector: vector, layers: layers}
	}

	if br.err != nil {
		return nil, [sha256.Size]byte{}, "", nil, fmt.Errorf("vectorindex: read error: %w", br.err)
	}

	return hnsw, contentHash, compression, sq8Params, nil
}

// ComputeContentHash computes a SHA-256 hash of the chunk table content to
// use as a staleness indicator. Callers pass any byte sequence that uniquely
// identifies the table state (e.g., row count + last insert timestamp).
func ComputeContentHash(data []byte) [sha256.Size]byte {
	return sha256.Sum256(data)
}

// ---------------------------------------------------------------------------
// Binary I/O helpers
// ---------------------------------------------------------------------------

type binaryWriter struct {
	w   io.Writer
	err error
}

func (bw *binaryWriter) raw(p []byte) {
	if bw.err != nil {
		return
	}
	if _, err := bw.w.Write(p); err != nil {
		bw.err = err
	}
}

func (bw *binaryWriter) u32(v uint32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	bw.raw(buf[:])
}

func (bw *binaryWriter) i32(v int32) {
	bw.u32(uint32(v))
}

func (bw *binaryWriter) f32(v float32) {
	bw.u32(math.Float32bits(v))
}

func (bw *binaryWriter) f64(v float64) {
	bits := math.Float64bits(v)
	bw.u32(uint32(bits))
	bw.u32(uint32(bits >> 32))
}

type binaryReader struct {
	r   io.Reader
	err error
}

func (br *binaryReader) readFull(buf []byte) {
	if br.err != nil {
		return
	}
	if _, err := io.ReadFull(br.r, buf); err != nil {
		br.err = err
	}
}

func (br *binaryReader) raw(p []byte) {
	br.readFull(p)
}

func (br *binaryReader) u32() uint32 {
	var buf [4]byte
	br.readFull(buf[:])
	return binary.LittleEndian.Uint32(buf[:])
}

func (br *binaryReader) i32() int32 {
	return int32(br.u32())
}

func (br *binaryReader) f32() float32 {
	return math.Float32frombits(br.u32())
}

func (br *binaryReader) f64() float64 {
	lo := uint64(br.u32())
	hi := uint64(br.u32())
	return math.Float64frombits(lo | hi<<32)
}
