//go:build !cgo

package vectormath

import "github.com/ishaan29/vectorDB/pkg/vectormath/scalar"

// Pure-Go kernel bindings, used when the binary is built with CGO_ENABLED=0.

func dotKernel(a, b []float32) float32            { return scalar.Dot(a, b) }
func sqNormKernel(a []float32) float32            { return scalar.SqNorm(a) }
func cosineKernel(a, b []float32) (float32, bool) { return scalar.CosineSimilarity(a, b) }
func sqEuclideanKernel(a, b []float32) float32    { return scalar.SqEuclidean(a, b) }
func cosineBatchKernel(q, base []float32, dim int, out []float32) {
	scalar.CosineSimilarityBatch(q, base, dim, out)
}

// Backend reports the active math implementation.
func Backend() string { return "go/scalar" }
