/* vectorDB SIMD math core.
 *
 * Dispatch is compile-time:
 *   - aarch64: ARM NEON intrinsics (NEON is architecturally mandatory on
 *     aarch64, so no runtime feature detection is needed). Main loops are
 *     2x unrolled (8 floats per iteration) with FMA accumulation and a
 *     scalar tail for n % 8.
 *   - everything else: plain C++ loops. Built with -O3 these auto-vectorize
 *     (SSE2 on x86-64 baseline) and are correct on any architecture.
 *     AVX2/runtime-cpuid dispatch is a documented follow-up.
 *
 * Constraints: no exceptions, no heap allocation, no C++ stdlib beyond
 * <cmath>/<cstddef> — nothing that could throw across the cgo boundary.
 */
#include "vectormath_simd.h"

#include <cmath>
#include <cstddef>

#if defined(__aarch64__)
#include <arm_neon.h>

extern "C" const char* vm_simd_variant(void) { return "neon"; }

extern "C" float vm_dot_f32(const float* a, const float* b, size_t n) {
    float32x4_t acc0 = vdupq_n_f32(0.0f);
    float32x4_t acc1 = vdupq_n_f32(0.0f);
    size_t i = 0;
    for (; i + 8 <= n; i += 8) {
        acc0 = vfmaq_f32(acc0, vld1q_f32(a + i), vld1q_f32(b + i));
        acc1 = vfmaq_f32(acc1, vld1q_f32(a + i + 4), vld1q_f32(b + i + 4));
    }
    float sum = vaddvq_f32(vaddq_f32(acc0, acc1));
    for (; i < n; i++) {
        sum += a[i] * b[i];
    }
    return sum;
}

extern "C" float vm_sqnorm_f32(const float* a, size_t n) {
    float32x4_t acc0 = vdupq_n_f32(0.0f);
    float32x4_t acc1 = vdupq_n_f32(0.0f);
    size_t i = 0;
    for (; i + 8 <= n; i += 8) {
        float32x4_t v0 = vld1q_f32(a + i);
        float32x4_t v1 = vld1q_f32(a + i + 4);
        acc0 = vfmaq_f32(acc0, v0, v0);
        acc1 = vfmaq_f32(acc1, v1, v1);
    }
    float sum = vaddvq_f32(vaddq_f32(acc0, acc1));
    for (; i < n; i++) {
        sum += a[i] * a[i];
    }
    return sum;
}

/* Fused single pass: dot, |a|^2 and |b|^2 accumulated together so the
 * vectors are read exactly once. */
static float fused_dot_norms(const float* a, const float* b, size_t n,
                             float* na_out, float* nb_out) {
    float32x4_t dot0 = vdupq_n_f32(0.0f), dot1 = vdupq_n_f32(0.0f);
    float32x4_t na0 = vdupq_n_f32(0.0f), na1 = vdupq_n_f32(0.0f);
    float32x4_t nb0 = vdupq_n_f32(0.0f), nb1 = vdupq_n_f32(0.0f);
    size_t i = 0;
    for (; i + 8 <= n; i += 8) {
        float32x4_t va0 = vld1q_f32(a + i);
        float32x4_t vb0 = vld1q_f32(b + i);
        float32x4_t va1 = vld1q_f32(a + i + 4);
        float32x4_t vb1 = vld1q_f32(b + i + 4);
        dot0 = vfmaq_f32(dot0, va0, vb0);
        na0 = vfmaq_f32(na0, va0, va0);
        nb0 = vfmaq_f32(nb0, vb0, vb0);
        dot1 = vfmaq_f32(dot1, va1, vb1);
        na1 = vfmaq_f32(na1, va1, va1);
        nb1 = vfmaq_f32(nb1, vb1, vb1);
    }
    float dot = vaddvq_f32(vaddq_f32(dot0, dot1));
    float na = vaddvq_f32(vaddq_f32(na0, na1));
    float nb = vaddvq_f32(vaddq_f32(nb0, nb1));
    for (; i < n; i++) {
        dot += a[i] * b[i];
        na += a[i] * a[i];
        nb += b[i] * b[i];
    }
    *na_out = na;
    *nb_out = nb;
    return dot;
}

extern "C" float vm_sqeuclidean_f32(const float* a, const float* b, size_t n) {
    float32x4_t acc0 = vdupq_n_f32(0.0f);
    float32x4_t acc1 = vdupq_n_f32(0.0f);
    size_t i = 0;
    for (; i + 8 <= n; i += 8) {
        float32x4_t d0 = vsubq_f32(vld1q_f32(a + i), vld1q_f32(b + i));
        float32x4_t d1 = vsubq_f32(vld1q_f32(a + i + 4), vld1q_f32(b + i + 4));
        acc0 = vfmaq_f32(acc0, d0, d0);
        acc1 = vfmaq_f32(acc1, d1, d1);
    }
    float sum = vaddvq_f32(vaddq_f32(acc0, acc1));
    for (; i < n; i++) {
        float diff = a[i] - b[i];
        sum += diff * diff;
    }
    return sum;
}

#else /* portable scalar fallback (auto-vectorized by -O3) */

extern "C" const char* vm_simd_variant(void) { return "scalar"; }

extern "C" float vm_dot_f32(const float* a, const float* b, size_t n) {
    float sum = 0.0f;
    for (size_t i = 0; i < n; i++) {
        sum += a[i] * b[i];
    }
    return sum;
}

extern "C" float vm_sqnorm_f32(const float* a, size_t n) {
    float sum = 0.0f;
    for (size_t i = 0; i < n; i++) {
        sum += a[i] * a[i];
    }
    return sum;
}

static float fused_dot_norms(const float* a, const float* b, size_t n,
                             float* na_out, float* nb_out) {
    float dot = 0.0f, na = 0.0f, nb = 0.0f;
    for (size_t i = 0; i < n; i++) {
        dot += a[i] * b[i];
        na += a[i] * a[i];
        nb += b[i] * b[i];
    }
    *na_out = na;
    *nb_out = nb;
    return dot;
}

extern "C" float vm_sqeuclidean_f32(const float* a, const float* b, size_t n) {
    float sum = 0.0f;
    for (size_t i = 0; i < n; i++) {
        float diff = a[i] - b[i];
        sum += diff * diff;
    }
    return sum;
}

#endif /* arch dispatch */

/* Arch-independent entry points built on the kernels above. */

extern "C" float vm_cosine_sim_f32(const float* a, const float* b, size_t n) {
    float na, nb;
    float dot = fused_dot_norms(a, b, n, &na, &nb);
    if (na == 0.0f || nb == 0.0f) {
        return NAN;
    }
    return dot / (sqrtf(na) * sqrtf(nb));
}

extern "C" void vm_cosine_sim_batch_f32(const float* query, const float* base,
                                        size_t dim, size_t n_vecs, float* out) {
    const float qn = vm_sqnorm_f32(query, dim);
    if (qn == 0.0f) {
        for (size_t i = 0; i < n_vecs; i++) {
            out[i] = NAN;
        }
        return;
    }
    const float qmag = sqrtf(qn);
    for (size_t i = 0; i < n_vecs; i++) {
        const float* row = base + i * dim;
        float na, rn;
        /* na recomputes |query|^2 alongside the row pass; the redundant work
         * is one FMA per lane and keeps a single fused kernel for all cases. */
        float dot = fused_dot_norms(query, row, dim, &na, &rn);
        out[i] = (rn == 0.0f) ? NAN : dot / (qmag * sqrtf(rn));
    }
}
