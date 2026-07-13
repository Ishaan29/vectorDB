// Package scalar contains the pure-Go vector math kernels.
//
// This package is ALWAYS compiled, regardless of build mode:
//   - Under CGO_ENABLED=0 it is the implementation behind pkg/vectormath.
//   - Under CGO_ENABLED=1 it is the reference implementation the SIMD core
//     is parity-tested against.
//
// Kernels are pure functions: no validation, no logging, no allocation.
// Callers (pkg/vectormath) are responsible for length checks and error
// reporting.
//
// Numerics rule (mirrored exactly by the C++ SIMD core): accumulate in
// float32; cosine similarity = dot / (sqrt(na) * sqrt(nb)).
package scalar

import "math"

// Dot returns the dot product of a and b. Assumes len(a) == len(b).
func Dot(a, b []float32) float32 {
	var sum float32
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}

// SqNorm returns the sum of squares of a (squared L2 norm, no sqrt).
func SqNorm(a []float32) float32 {
	var sum float32
	for _, v := range a {
		sum += v * v
	}
	return sum
}

// CosineSimilarity computes cosine similarity in a single fused pass:
// dot product and both squared norms are accumulated in one loop.
// ok is false when either vector has zero norm (including zero length).
// Assumes len(a) == len(b).
func CosineSimilarity(a, b []float32) (sim float32, ok bool) {
	var dot, na, nb float32
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0, false
	}
	return dot / (float32(math.Sqrt(float64(na))) * float32(math.Sqrt(float64(nb)))), true
}

// SqEuclidean returns the squared Euclidean distance (no sqrt).
// Assumes len(a) == len(b).
func SqEuclidean(a, b []float32) float32 {
	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return sum
}

// CosineSimilarityBatch computes cosine similarity of query against n rows of
// base, where base is a flattened row-major buffer of n*dim floats.
// out[i] is NaN when row i (or the query) has zero norm.
// Assumes len(query) == dim, len(base) == n*dim, len(out) == n.
func CosineSimilarityBatch(query, base []float32, dim int, out []float32) {
	qn := SqNorm(query)
	nan := float32(math.NaN())
	for i := range out {
		if qn == 0 {
			out[i] = nan
			continue
		}
		row := base[i*dim : (i+1)*dim]
		var dot, rn float32
		for j := 0; j < dim; j++ {
			dot += query[j] * row[j]
			rn += row[j] * row[j]
		}
		if rn == 0 {
			out[i] = nan
			continue
		}
		out[i] = dot / (float32(math.Sqrt(float64(qn))) * float32(math.Sqrt(float64(rn))))
	}
}
