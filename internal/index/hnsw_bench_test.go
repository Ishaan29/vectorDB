package index

import (
	"fmt"
	"math/rand"
	"testing"
)

const benchDim = 128

func benchRandVec(r *rand.Rand, dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = r.Float32()*2 - 1
	}
	return v
}

// BenchmarkHNSWInsert measures end-to-end index build cost. Each iteration
// inserts one vector into a growing index (fresh index every 5000 inserts to
// keep graph size bounded and comparable across runs).
func BenchmarkHNSWInsert(b *testing.B) {
	r := rand.New(rand.NewSource(42))
	vecs := make([][]float32, 5000)
	for i := range vecs {
		vecs[i] = benchRandVec(r, benchDim)
	}

	idx := NewHNSWIndex(benchDim, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%5000 == 0 && i > 0 {
			b.StopTimer()
			idx = NewHNSWIndex(benchDim, nil)
			b.StartTimer()
		}
		_ = idx.Add(fmt.Sprintf("v%d", i), vecs[i%5000])
	}
}

// BenchmarkHNSWSearch measures query latency against a pre-built 10k index.
func BenchmarkHNSWSearch(b *testing.B) {
	r := rand.New(rand.NewSource(42))
	idx := NewHNSWIndex(benchDim, nil)
	for i := 0; i < 10000; i++ {
		if err := idx.Add(fmt.Sprintf("v%d", i), benchRandVec(r, benchDim)); err != nil {
			b.Fatal(err)
		}
	}
	queries := make([][]float32, 100)
	for i := range queries {
		queries[i] = benchRandVec(r, benchDim)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := idx.Search(queries[i%len(queries)], 10)
		if err != nil {
			b.Fatal(err)
		}
		if len(results) == 0 {
			b.Fatal("no results")
		}
	}
}
