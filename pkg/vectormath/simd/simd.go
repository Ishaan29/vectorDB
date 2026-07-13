//go:build cgo

// Package simd binds the C++ SIMD math core (vectormath_simd.cpp) into Go.
//
// The kernels are pure functions over caller-owned float32 buffers: they do
// not retain pointers, allocate, or throw, so passing Go slice memory across
// the boundary is safe under the cgo pointer-passing rules. Every wrapper
// guards the empty-slice case before taking &s[0].
package simd

/*
#cgo CXXFLAGS: -std=c++17 -O3 -fno-exceptions
#cgo linux LDFLAGS: -lstdc++ -lm
#include "vectormath_simd.h"
*/
import "C"

import (
	"math"
	"unsafe"
)

// Enabled reports whether the C++ SIMD core is compiled into this binary.
const Enabled = true

// Variant returns the compiled kernel variant: "neon" or "scalar".
func Variant() string {
	return C.GoString(C.vm_simd_variant())
}

// Dot returns the dot product of a and b. Assumes len(a) == len(b).
func Dot(a, b []float32) float32 {
	if len(a) == 0 {
		return 0
	}
	return float32(C.vm_dot_f32(fptr(a), fptr(b), C.size_t(len(a))))
}

// SqNorm returns the sum of squares of a.
func SqNorm(a []float32) float32 {
	if len(a) == 0 {
		return 0
	}
	return float32(C.vm_sqnorm_f32(fptr(a), C.size_t(len(a))))
}

// CosineSimilarity computes cosine similarity in a single fused pass.
// ok is false when either vector has zero norm (including zero length).
// Assumes len(a) == len(b).
func CosineSimilarity(a, b []float32) (float32, bool) {
	if len(a) == 0 {
		return 0, false
	}
	sim := float32(C.vm_cosine_sim_f32(fptr(a), fptr(b), C.size_t(len(a))))
	if math.IsNaN(float64(sim)) {
		return 0, false
	}
	return sim, true
}

// SqEuclidean returns the squared Euclidean distance between a and b.
// Assumes len(a) == len(b).
func SqEuclidean(a, b []float32) float32 {
	if len(a) == 0 {
		return 0
	}
	return float32(C.vm_sqeuclidean_f32(fptr(a), fptr(b), C.size_t(len(a))))
}

// CosineSimilarityBatch computes cosine similarity of query against n rows of
// base (flattened row-major, n*dim floats) in one cgo crossing.
// out[i] is NaN when row i (or the query) has zero norm.
// Assumes len(query) == dim > 0, len(base) == n*dim, len(out) == n.
func CosineSimilarityBatch(query, base []float32, dim int, out []float32) {
	if len(out) == 0 || dim == 0 {
		return
	}
	C.vm_cosine_sim_batch_f32(fptr(query), fptr(base), C.size_t(dim),
		C.size_t(len(out)), fptr(out))
}

func fptr(s []float32) *C.float {
	return (*C.float)(unsafe.Pointer(&s[0]))
}
