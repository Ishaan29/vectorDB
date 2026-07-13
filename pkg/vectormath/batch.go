package vectormath

import (
	"math"

	"github.com/ishaan29/vectorDB/internal/logger"
)

// CosineSimilarityBatch computes the cosine similarity of query against every
// row of base, where base is a flattened row-major buffer of n*dim float32
// values (n = len(base)/dim). It performs a single kernel invocation (one cgo
// crossing when the SIMD core is active), amortizing call overhead across all
// rows; the query norm is computed once.
//
// The result has one entry per row. Rows with zero norm (or a zero-norm
// query) yield NaN rather than an error, so one bad row cannot fail the whole
// batch — callers should skip NaN entries.
func CosineSimilarityBatch(query, base []float32, dim int) ([]float32, error) {
	if dim <= 0 || len(query) != dim {
		logger.Error("Dimension mismatch in CosineSimilarityBatch", ErrDimensionMismatch)
		return nil, ErrDimensionMismatch
	}
	if len(base)%dim != 0 {
		logger.Error("Base length not a multiple of dim in CosineSimilarityBatch", ErrDimensionMismatch)
		return nil, ErrDimensionMismatch
	}

	n := len(base) / dim
	out := make([]float32, n)
	if n == 0 {
		return out, nil
	}
	cosineBatchKernel(query, base, dim, out)
	return out, nil
}

// CosineSimilarityMany is a convenience wrapper over CosineSimilarityBatch
// for [][]float32 inputs. It flattens vecs into one contiguous buffer
// (required for a single kernel/cgo crossing) and returns one similarity per
// input vector. Rows whose length differs from len(query), rows with zero
// norm, or a zero-norm query yield NaN.
func CosineSimilarityMany(query []float32, vecs [][]float32) ([]float32, error) {
	dim := len(query)
	if dim == 0 {
		logger.Error("Empty query in CosineSimilarityMany", ErrDimensionMismatch)
		return nil, ErrDimensionMismatch
	}

	out := make([]float32, len(vecs))
	if len(vecs) == 0 {
		return out, nil
	}

	// Flatten well-formed rows; mismatched rows are marked NaN and excluded
	// from the kernel call.
	nan := float32(math.NaN())
	flat := make([]float32, 0, len(vecs)*dim)
	rowIdx := make([]int, 0, len(vecs))
	for i, v := range vecs {
		if len(v) != dim {
			out[i] = nan
			continue
		}
		flat = append(flat, v...)
		rowIdx = append(rowIdx, i)
	}
	if len(rowIdx) == 0 {
		return out, nil
	}

	sims := make([]float32, len(rowIdx))
	cosineBatchKernel(query, flat, dim, sims)
	for j, i := range rowIdx {
		out[i] = sims[j]
	}
	return out, nil
}
