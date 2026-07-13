package vectormath

import (
	"math/rand"
	"testing"
)

// benchDims covers common embedding sizes: small, the config default (128),
// and typical model output dims (768 = BERT, 1536 = OpenAI ada-002).
var benchDims = []int{64, 128, 256, 768, 1536}

// sink defeats dead-code elimination.
var sink float32

func randVec(r *rand.Rand, dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = r.Float32()*2 - 1
	}
	return v
}

func benchPair(b *testing.B, dim int, fn func(v1, v2 []float32)) {
	r := rand.New(rand.NewSource(42))
	v1, v2 := randVec(r, dim), randVec(r, dim)
	b.SetBytes(int64(dim * 4 * 2))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn(v1, v2)
	}
}

func BenchmarkDot(b *testing.B) {
	for _, dim := range benchDims {
		b.Run(dimName(dim), func(b *testing.B) {
			benchPair(b, dim, func(v1, v2 []float32) {
				s, _ := Dot(v1, v2)
				sink = s
			})
		})
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	for _, dim := range benchDims {
		b.Run(dimName(dim), func(b *testing.B) {
			benchPair(b, dim, func(v1, v2 []float32) {
				s, _ := CosineSimilarity(v1, v2)
				sink = s
			})
		})
	}
}

// BenchmarkCosineDistance is the HNSW hot path: fogfish/hnsw calls
// CosineSurface.Distance -> CosineDistance once per pairwise comparison.
func BenchmarkCosineDistance(b *testing.B) {
	for _, dim := range benchDims {
		b.Run(dimName(dim), func(b *testing.B) {
			benchPair(b, dim, func(v1, v2 []float32) {
				s, _ := CosineDistance(v1, v2)
				sink = s
			})
		})
	}
}

func BenchmarkEuclideanDistance(b *testing.B) {
	for _, dim := range benchDims {
		b.Run(dimName(dim), func(b *testing.B) {
			benchPair(b, dim, func(v1, v2 []float32) {
				s, _ := EuclideanDistance(v1, v2)
				sink = s
			})
		})
	}
}

func dimName(dim int) string {
	switch dim {
	case 64:
		return "dim=64"
	case 128:
		return "dim=128"
	case 256:
		return "dim=256"
	case 768:
		return "dim=768"
	case 1536:
		return "dim=1536"
	}
	return "dim=?"
}

// BenchmarkCosineSimilarityBatch measures the brute-force scan shape:
// one query scored against 10k stored vectors (dim=128), comparing a single
// batched kernel invocation against a per-pair loop.
func BenchmarkCosineSimilarityBatch(b *testing.B) {
	const n, dim = 10000, 128
	r := rand.New(rand.NewSource(42))
	query := randVec(r, dim)
	base := randVec(r, n*dim)

	b.Run("batched", func(b *testing.B) {
		b.SetBytes(int64(n * dim * 4))
		for i := 0; i < b.N; i++ {
			sims, err := CosineSimilarityBatch(query, base, dim)
			if err != nil {
				b.Fatal(err)
			}
			sink = sims[0]
		}
	})

	b.Run("per-pair", func(b *testing.B) {
		b.SetBytes(int64(n * dim * 4))
		for i := 0; i < b.N; i++ {
			for j := 0; j < n; j++ {
				s, _ := CosineSimilarity(query, base[j*dim:(j+1)*dim])
				sink = s
			}
		}
	})
}
