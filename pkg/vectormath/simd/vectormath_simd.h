/* C API of the vectorDB SIMD math core.
 *
 * All kernels operate on float32 buffers with runtime length n. Semantics
 * mirror pkg/vectormath/scalar exactly (float32 accumulation; cosine =
 * dot / (sqrtf(na) * sqrtf(nb))), modulo SIMD reassociation of the sums.
 */
#ifndef VECTORMATH_SIMD_H
#define VECTORMATH_SIMD_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Dot product of a and b (n elements each). */
float vm_dot_f32(const float* a, const float* b, size_t n);

/* Sum of squares of a (squared L2 norm, no sqrt). */
float vm_sqnorm_f32(const float* a, size_t n);

/* Fused cosine similarity: dot product and both squared norms accumulated in
 * a single pass. Returns the similarity, or NaN if either vector has zero
 * norm (including n == 0). The NaN sentinel (rather than an out-param)
 * keeps the Go wrapper allocation-free on the hot path. */
float vm_cosine_sim_f32(const float* a, const float* b, size_t n);

/* Squared Euclidean distance (no sqrt). */
float vm_sqeuclidean_f32(const float* a, const float* b, size_t n);

/* Cosine similarity of query against n_vecs rows of base, a flattened
 * row-major buffer of n_vecs * dim floats. The query norm is computed once.
 * out[i] = similarity of row i, or NaN when row i (or the query) has zero
 * norm. */
void vm_cosine_sim_batch_f32(const float* query, const float* base,
                             size_t dim, size_t n_vecs, float* out);

/* Compiled kernel variant: "neon" or "scalar". */
const char* vm_simd_variant(void);

#ifdef __cplusplus
}
#endif

#endif /* VECTORMATH_SIMD_H */
