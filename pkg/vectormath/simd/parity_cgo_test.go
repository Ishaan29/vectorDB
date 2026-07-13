//go:build cgo

package simd_test

import (
	"math"
	"math/rand"
	"runtime"
	"testing"

	"github.com/ishaan29/vectorDB/pkg/vectormath/scalar"
	"github.com/ishaan29/vectorDB/pkg/vectormath/simd"
)

// Dims cover: below/at/above NEON lane width (4), the unrolled step (8),
// non-multiples forcing the scalar tail, the config default (128), and
// large embedding sizes where float32 reassociation error accumulates.
var parityDims = []int{1, 2, 3, 4, 7, 8, 15, 16, 17, 31, 64, 127, 128, 129, 256, 768, 1536}

// relTol checks |got-want| <= tol * max(1, |want|): relative for large
// magnitudes, absolute near zero.
func relTol(got, want float32, tol float64) bool {
	diff := math.Abs(float64(got - want))
	scale := math.Max(1, math.Abs(float64(want)))
	return diff <= tol*scale
}

func fill(r *rand.Rand, n int, scale float32) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = (r.Float32()*2 - 1) * scale
	}
	return v
}

func TestVariant(t *testing.T) {
	t.Logf("SIMD variant: %s", simd.Variant())
	if !simd.Enabled {
		t.Fatal("simd.Enabled = false in a cgo build")
	}
	// On arm64 NEON must be active; anything else means the arch dispatch in
	// vectormath_simd.cpp silently fell through to the scalar path.
	if runtime.GOARCH == "arm64" && simd.Variant() != "neon" {
		t.Fatalf("Variant() = %q on arm64, want \"neon\"", simd.Variant())
	}
}

func TestParityRandom(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	for _, dim := range parityDims {
		for _, scale := range []float32{1, 1e4} {
			a := fill(r, dim, scale)
			b := fill(r, dim, scale)

			if got, want := simd.Dot(a, b), scalar.Dot(a, b); !relTol(got, want, 1e-3) {
				t.Errorf("Dot dim=%d scale=%g: simd=%v scalar=%v", dim, scale, got, want)
			}
			if got, want := simd.SqNorm(a), scalar.SqNorm(a); !relTol(got, want, 1e-3) {
				t.Errorf("SqNorm dim=%d scale=%g: simd=%v scalar=%v", dim, scale, got, want)
			}
			if got, want := simd.SqEuclidean(a, b), scalar.SqEuclidean(a, b); !relTol(got, want, 1e-3) {
				t.Errorf("SqEuclidean dim=%d scale=%g: simd=%v scalar=%v", dim, scale, got, want)
			}

			gotSim, gotOK := simd.CosineSimilarity(a, b)
			wantSim, wantOK := scalar.CosineSimilarity(a, b)
			if gotOK != wantOK {
				t.Errorf("CosineSimilarity ok dim=%d scale=%g: simd=%v scalar=%v", dim, scale, gotOK, wantOK)
			}
			// Cosine is bounded in [-1, 1]; absolute tolerance.
			if math.Abs(float64(gotSim-wantSim)) > 1e-4 {
				t.Errorf("CosineSimilarity dim=%d scale=%g: simd=%v scalar=%v", dim, scale, gotSim, wantSim)
			}
		}
	}
}

func TestParityEdgeCases(t *testing.T) {
	t.Run("zero vectors", func(t *testing.T) {
		for _, dim := range []int{1, 8, 128} {
			zero := make([]float32, dim)
			nonzero := fill(rand.New(rand.NewSource(1)), dim, 1)

			if _, ok := simd.CosineSimilarity(zero, nonzero); ok {
				t.Errorf("dim=%d: ok=true for zero first arg", dim)
			}
			if _, ok := simd.CosineSimilarity(nonzero, zero); ok {
				t.Errorf("dim=%d: ok=true for zero second arg", dim)
			}
			if got := simd.Dot(zero, nonzero); got != 0 {
				t.Errorf("dim=%d: Dot(zero, v) = %v, want 0", dim, got)
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		if got := simd.Dot(nil, nil); got != 0 {
			t.Errorf("Dot(nil, nil) = %v, want 0", got)
		}
		if got := simd.SqNorm(nil); got != 0 {
			t.Errorf("SqNorm(nil) = %v, want 0", got)
		}
		if _, ok := simd.CosineSimilarity(nil, nil); ok {
			t.Error("CosineSimilarity(nil, nil) ok = true, want false")
		}
	})

	t.Run("all negative", func(t *testing.T) {
		a := []float32{-1, -2, -3, -4, -5, -6, -7, -8, -9}
		sim, ok := simd.CosineSimilarity(a, a)
		if !ok || math.Abs(float64(sim-1)) > 1e-5 {
			t.Errorf("self-similarity = %v (ok=%v), want 1", sim, ok)
		}
	})

	t.Run("single element", func(t *testing.T) {
		if got := simd.Dot([]float32{3}, []float32{-4}); got != -12 {
			t.Errorf("Dot([3],[-4]) = %v, want -12", got)
		}
	})
}

func TestParityBatch(t *testing.T) {
	r := rand.New(rand.NewSource(43))
	for _, dim := range []int{4, 31, 128, 768} {
		const n = 17
		query := fill(r, dim, 1)
		base := fill(r, n*dim, 1)
		// Row 5 zeroed to exercise the NaN path.
		for j := 0; j < dim; j++ {
			base[5*dim+j] = 0
		}

		got := make([]float32, n)
		want := make([]float32, n)
		simd.CosineSimilarityBatch(query, base, dim, got)
		scalar.CosineSimilarityBatch(query, base, dim, want)

		for i := 0; i < n; i++ {
			gNaN := math.IsNaN(float64(got[i]))
			wNaN := math.IsNaN(float64(want[i]))
			if gNaN != wNaN {
				t.Errorf("dim=%d row=%d: NaN mismatch simd=%v scalar=%v", dim, i, got[i], want[i])
				continue
			}
			if !gNaN && math.Abs(float64(got[i]-want[i])) > 1e-4 {
				t.Errorf("dim=%d row=%d: simd=%v scalar=%v", dim, i, got[i], want[i])
			}
		}

		// Batch row must also match the fused single-pair kernel (same code
		// path family — tight bound).
		row0 := base[:dim]
		single, _ := simd.CosineSimilarity(query, row0)
		if math.Abs(float64(got[0]-single)) > 1e-6 {
			t.Errorf("dim=%d: batch[0]=%v vs single=%v", dim, got[0], single)
		}
	}

	t.Run("zero query", func(t *testing.T) {
		out := make([]float32, 3)
		simd.CosineSimilarityBatch(make([]float32, 8), fill(rand.New(rand.NewSource(2)), 24, 1), 8, out)
		for i, v := range out {
			if !math.IsNaN(float64(v)) {
				t.Errorf("out[%d] = %v, want NaN for zero query", i, v)
			}
		}
	})
}
