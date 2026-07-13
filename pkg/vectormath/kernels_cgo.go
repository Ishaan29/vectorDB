//go:build cgo

package vectormath

import "github.com/ishaan29/vectorDB/pkg/vectormath/simd"

// cgo kernel bindings: arithmetic runs in the C++ SIMD core
// (pkg/vectormath/simd). Validation and error reporting stay in Go.

func dotKernel(a, b []float32) float32            { return simd.Dot(a, b) }
func sqNormKernel(a []float32) float32            { return simd.SqNorm(a) }
func cosineKernel(a, b []float32) (float32, bool) { return simd.CosineSimilarity(a, b) }
func sqEuclideanKernel(a, b []float32) float32    { return simd.SqEuclidean(a, b) }
func cosineBatchKernel(q, base []float32, dim int, out []float32) {
	simd.CosineSimilarityBatch(q, base, dim, out)
}

// Backend reports the active math implementation.
func Backend() string { return "cgo/" + simd.Variant() }
