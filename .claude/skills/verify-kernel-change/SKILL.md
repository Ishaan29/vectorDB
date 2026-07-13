---
name: verify-kernel-change
description: Full verification ritual for any change touching pkg/vectormath, the C++ SIMD core, or internal/index. Use after modifying kernels, wrappers, build tags, or cgo flags — and before declaring any such change done.
---

# Verify a kernel / vectormath change

Run every step. A kernel change is not verified until all of them pass.

## Steps

1. **Both build modes test clean:**
   ```bash
   make test-all        # runs make test (cgo) + make test-nocgo (pure Go)
   ```

2. **Parity explicitly:** the simd kernels must match the pure-Go scalar
   reference within tolerance:
   ```bash
   CGO_ENABLED=1 go test ./pkg/vectormath/... -run Parity -v
   ```

3. **Linux, both architectures:** proves NEON (arm64) and portable scalar
   C++ (amd64) both build and pass off-macOS:
   ```bash
   make docker-verify
   ```

4. **Benchmark sanity (only if the hot path changed):** run
   `make bench` and eyeball cosine dim=128 against the baseline in
   `docs/simd-benchmark-report.md` (~57 ns cgo/NEON on M1 Pro). A big
   regression means the fusion or batching broke.

## Invariants to check in the diff itself

- `pkg/vectormath` public signatures are frozen — callers must never care
  which kernel is active.
- Float assertions in tests are tolerance-based, never exact equality:
  float32 SIMD reassociation legitimately changes low bits. Parity
  tolerances: 1e-3 relative (dot/sqnorm/sqeuclidean), 1e-4 absolute
  (cosine); zero-norm/NaN semantics must match exactly.
- No `[][]float32` crosses cgo — batch APIs take flattened buffers.
- Every simd wrapper guards `len == 0` before taking `&slice[0]`.
- C++ kernels stay pure arithmetic: no exceptions, no allocation, no
  retained pointers; validation and error logging stay on the Go side.
- New cgo flags must be on the cgo allowlist (`-fno-math-errno` is NOT —
  this was hit before).

## On failure

Do not paper over a parity failure by widening tolerances — widen only if
you can explain the numerical reason (e.g. changed reduction order) and
the new bound is still tight. Otherwise the kernel is wrong; fix it.
