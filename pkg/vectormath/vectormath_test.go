package vectormath

import (
	"errors"
	"math"
	"testing"
)

// Tolerance for float comparisons. Kernel implementations (fused/SIMD) may
// differ from the naive sequential loop in the last ulps due to reassociation,
// so all assertions are tolerance-based rather than exact.
const eps = 1e-5

func almostEqual(a, b float32) bool {
	return math.Abs(float64(a-b)) <= eps
}

func TestDot(t *testing.T) {
	tests := []struct {
		name    string
		v1, v2  []float32
		want    float32
		wantErr error
	}{
		{"basic", []float32{1, 2, 3}, []float32{4, 5, 6}, 32, nil},
		{"negative", []float32{1, -2, 3}, []float32{-4, 5, -6}, -32, nil},
		{"zeros", []float32{0, 0, 0}, []float32{1, 2, 3}, 0, nil},
		{"single", []float32{2}, []float32{3}, 6, nil},
		{"nil both", nil, nil, 0, nil},
		{"empty both", []float32{}, []float32{}, 0, nil},
		{"dim mismatch", []float32{1, 2}, []float32{1, 2, 3}, 0, ErrDimensionMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Dot(tt.v1, tt.v2)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Dot() error = %v, want %v", err, tt.wantErr)
			}
			if !almostEqual(got, tt.want) {
				t.Errorf("Dot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagnitude(t *testing.T) {
	tests := []struct {
		name string
		v    []float32
		want float32
	}{
		{"3-4-5", []float32{3, 4}, 5},
		{"unit", []float32{1, 0, 0}, 1},
		{"zero", []float32{0, 0, 0}, 0},
		{"nil", nil, 0},
		{"negative", []float32{-3, -4}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Magnitude(tt.v); !almostEqual(got, tt.want) {
				t.Errorf("Magnitude() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name    string
		v1, v2  []float32
		want    float32
		wantErr error
	}{
		{"identical", []float32{1, 2, 3}, []float32{1, 2, 3}, 1, nil},
		{"orthogonal", []float32{1, 0}, []float32{0, 1}, 0, nil},
		{"opposite", []float32{1, 2, 3}, []float32{-1, -2, -3}, -1, nil},
		{"scaled identical", []float32{1, 2, 3}, []float32{2, 4, 6}, 1, nil},
		{"dim mismatch", []float32{1, 2}, []float32{1, 2, 3}, 0, ErrDimensionMismatch},
		{"zero v1", []float32{0, 0, 0}, []float32{1, 2, 3}, 0, ErrZeroVector},
		{"zero v2", []float32{1, 2, 3}, []float32{0, 0, 0}, 0, ErrZeroVector},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CosineSimilarity(tt.v1, tt.v2)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CosineSimilarity() error = %v, want %v", err, tt.wantErr)
			}
			if !almostEqual(got, tt.want) {
				t.Errorf("CosineSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCosineDistance(t *testing.T) {
	tests := []struct {
		name    string
		v1, v2  []float32
		want    float32
		wantErr error
	}{
		{"identical", []float32{1, 2, 3}, []float32{1, 2, 3}, 0, nil},
		{"orthogonal", []float32{1, 0}, []float32{0, 1}, 1, nil},
		{"opposite", []float32{1, 0}, []float32{-1, 0}, 2, nil},
		{"dim mismatch", []float32{1, 2}, []float32{1, 2, 3}, 0, ErrDimensionMismatch},
		{"zero vector", []float32{0, 0}, []float32{1, 2}, 0, ErrZeroVector},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CosineDistance(tt.v1, tt.v2)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CosineDistance() error = %v, want %v", err, tt.wantErr)
			}
			if !almostEqual(got, tt.want) {
				t.Errorf("CosineDistance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEuclideanDistance(t *testing.T) {
	tests := []struct {
		name    string
		v1, v2  []float32
		want    float32
		wantErr error
	}{
		{"identical", []float32{1, 2, 3}, []float32{1, 2, 3}, 0, nil},
		{"3-4-5", []float32{0, 0}, []float32{3, 4}, 5, nil},
		{"single", []float32{1}, []float32{4}, 3, nil},
		{"dim mismatch", []float32{1, 2}, []float32{1, 2, 3}, 0, ErrDimensionMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EuclideanDistance(tt.v1, tt.v2)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("EuclideanDistance() error = %v, want %v", err, tt.wantErr)
			}
			if !almostEqual(got, tt.want) {
				t.Errorf("EuclideanDistance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeVector(t *testing.T) {
	t.Run("normalizes in place", func(t *testing.T) {
		v := []float32{3, 4}
		if err := NormalizeVector(v); err != nil {
			t.Fatalf("NormalizeVector() error = %v", err)
		}
		if !almostEqual(v[0], 0.6) || !almostEqual(v[1], 0.8) {
			t.Errorf("NormalizeVector() = %v, want [0.6 0.8]", v)
		}
		if !almostEqual(Magnitude(v), 1) {
			t.Errorf("Magnitude after normalize = %v, want 1", Magnitude(v))
		}
	})
	t.Run("zero vector", func(t *testing.T) {
		v := []float32{0, 0, 0}
		if err := NormalizeVector(v); !errors.Is(err, ErrZeroVector) {
			t.Fatalf("NormalizeVector() error = %v, want ErrZeroVector", err)
		}
	})
}

func TestCosineSimilarityBatch(t *testing.T) {
	query := []float32{1, 0}
	base := []float32{
		1, 0, // identical -> 1
		0, 1, // orthogonal -> 0
		-1, 0, // opposite -> -1
		0, 0, // zero row -> NaN
	}
	sims, err := CosineSimilarityBatch(query, base, 2)
	if err != nil {
		t.Fatalf("CosineSimilarityBatch() error = %v", err)
	}
	if len(sims) != 4 {
		t.Fatalf("len = %d, want 4", len(sims))
	}
	for i, want := range []float32{1, 0, -1} {
		if !almostEqual(sims[i], want) {
			t.Errorf("sims[%d] = %v, want %v", i, sims[i], want)
		}
	}
	if !math.IsNaN(float64(sims[3])) {
		t.Errorf("sims[3] = %v, want NaN (zero-norm row)", sims[3])
	}

	// Parity with the single-pair path.
	r := []float32{0.3, -0.7}
	single, _ := CosineSimilarity(query, r)
	batch, _ := CosineSimilarityBatch(query, r, 2)
	if !almostEqual(single, batch[0]) {
		t.Errorf("batch/single mismatch: %v vs %v", batch[0], single)
	}

	// Errors.
	if _, err := CosineSimilarityBatch(query, base, 3); !errors.Is(err, ErrDimensionMismatch) {
		t.Errorf("base not multiple of dim: err = %v, want ErrDimensionMismatch", err)
	}
	if _, err := CosineSimilarityBatch(query, base, 0); !errors.Is(err, ErrDimensionMismatch) {
		t.Errorf("dim=0: err = %v, want ErrDimensionMismatch", err)
	}
}

func TestCosineSimilarityMany(t *testing.T) {
	query := []float32{1, 0}
	vecs := [][]float32{
		{1, 0},    // 1
		{0, 2},    // 0
		{1, 2, 3}, // dim mismatch -> NaN
		{-3, 0},   // -1
		{0, 0},    // zero -> NaN
	}
	sims, err := CosineSimilarityMany(query, vecs)
	if err != nil {
		t.Fatalf("CosineSimilarityMany() error = %v", err)
	}
	if len(sims) != 5 {
		t.Fatalf("len = %d, want 5", len(sims))
	}
	wants := []struct {
		val float32
		nan bool
	}{{1, false}, {0, false}, {0, true}, {-1, false}, {0, true}}
	for i, w := range wants {
		if w.nan {
			if !math.IsNaN(float64(sims[i])) {
				t.Errorf("sims[%d] = %v, want NaN", i, sims[i])
			}
		} else if !almostEqual(sims[i], w.val) {
			t.Errorf("sims[%d] = %v, want %v", i, sims[i], w.val)
		}
	}

	// Empty inputs.
	if _, err := CosineSimilarityMany(nil, vecs); !errors.Is(err, ErrDimensionMismatch) {
		t.Errorf("empty query: err = %v, want ErrDimensionMismatch", err)
	}
	out, err := CosineSimilarityMany(query, nil)
	if err != nil || len(out) != 0 {
		t.Errorf("no vecs: out = %v, err = %v; want empty, nil", out, err)
	}
}
