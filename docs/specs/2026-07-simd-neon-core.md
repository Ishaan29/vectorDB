# Spec: SIMD NEON math core

- **Status:** Implemented
- **Date:** 2026-07 (spec written retroactively as the reference example)
- **Author(s):** Ishaan Bajpai (with AI assistance)
- **Supersedes / superseded by:** —

## Problem

All similarity search funnels through `vectormath.CosineDistance`, called
once per pairwise comparison inside HNSW traversal — the hottest path in
the system. The original pure-Go implementation computed cosine as
`Dot` + 2× `Magnitude` (three passes over both vectors), costing ~323 ns
at dim=128 and dominating insert/search latency.

## Goals

- ≥3x speedup on cosine distance at practical embedding dims (64–1536).
- Zero behavior change for callers: same API, same results within float32
  tolerance, identical top-k output.
- Graceful fallback: the project must still build and pass tests with
  `CGO_ENABLED=0` and on non-ARM platforms.

## Non-goals

- amd64 AVX2 kernels (follow-up: [2026-07-avx2-dispatch.md](2026-07-avx2-dispatch.md)).
- Normalizing vectors at insert time (changes stored-data semantics;
  deferred).

## Design

Kernel dispatch by build tag behind the frozen `pkg/vectormath` API:

- `kernels_cgo.go` (`//go:build cgo`) → `pkg/vectormath/simd`, a C++17
  core; 2x-unrolled ARM NEON with FMA on aarch64, portable scalar C++
  elsewhere.
- `kernels_purego.go` (`//go:build !cgo`) → `pkg/vectormath/scalar`, pure
  Go — always compiled, and the parity-test reference.

Kernels are **fused** (cosine = one pass computing dot + both norms = one
cgo crossing; zero-norm reported via NaN sentinel to stay allocation-free)
and **batched** (`CosineSimilarityBatch`/`Many`: 1 query × N vectors per
crossing, flattened buffers — `[][]float32` never crosses cgo).

Rejected alternative: Go assembly (avo/hand-written NEON). Rejected for
maintainability — the C++ core is compiler-vectorizable, testable with
standard tooling, and the `#ifdef` structure extends naturally to AVX2.

## API impact

None. `Dot`, `Magnitude`, `CosineSimilarity`, `CosineDistance`,
`EuclideanDistance`, `NormalizeVector` unchanged; batch APIs added.
`vectormath.Backend()` added to report the active kernel
(`cgo/neon`, `cgo/scalar`, `go/scalar`); logged at startup.

## Performance expectations

Measured (Apple M1 Pro — full data in
[../simd-benchmark-report.md](../simd-benchmark-report.md)):

- Cosine dim=128: 323 ns → 57 ns (5.7x); dim=1536: 5622 ns → 283 ns.
- Batched scoring: 11.4 ns/vector (crossing amortized to nothing).
- End-to-end HNSW 10k: insert 2.96x, search 3.33x.
- Zero heap allocations on all paths.

## Test plan

- Parity: `pkg/vectormath/simd/parity_cgo_test.go` — every kernel vs the
  scalar reference across dims {1…1536}, magnitudes, zero vectors,
  negatives, batch rows. Tolerances 1e-3 relative / 1e-4 absolute
  (cosine); zero-norm/NaN semantics exact.
- `make test-all` (both CGO modes) + `make docker-verify` (linux/arm64
  NEON + linux/amd64 scalar).
- Identical top-k IDs from both binaries on the CLI demo.

## Open questions

None outstanding. Resolved during implementation: `-fno-math-errno` is
rejected by the cgo flag allowlist (dropped); NaN sentinel chosen over an
out-param for zero-norm to keep wrappers allocation-free.
