# SIMD Core Benchmark Report

Environment: Apple M1 Pro (darwin/arm64), go1.24.6, clang (Xcode CLT).
Method: `go test -bench -count=6`, compared with `benchstat` (p=0.002, n=6
unless noted). Vectors are seeded uniform random in [-1, 1].

Three configurations measured:

1. **old scalar Go** — original code: cosine = `Dot` + 2x `Magnitude`
   (3 passes over both vectors)
2. **fused Go** — pure-Go single-pass kernel (`pkg/vectormath/scalar`),
   what `CGO_ENABLED=0` builds run today
3. **cgo NEON** — C++ core (`pkg/vectormath/simd`), 2x-unrolled NEON with
   FMA, one cgo crossing per call

## Cosine similarity (single pair per call — the HNSW hot path shape)

| dim  | old scalar Go | fused Go | cgo NEON | fused vs old | NEON vs old |
|------|---------------|----------|----------|--------------|-------------|
| 64   | 116.5 ns      | 53.1 ns  | 47.1 ns  | -54%         | -60%        |
| 128  | 322.8 ns      | 124.3 ns | 56.8 ns  | -61%         | -82%        |
| 256  | 735.7 ns      | 277.5 ns | 77.8 ns  | -62%         | -89%        |
| 768  | 2683.5 ns     | 930.8 ns | 158.9 ns | -65%         | -94%        |
| 1536 | 5622.0 ns     | 1908.0 ns| 283.4 ns | -66%         | -95%        |

Notes:
- cgo call overhead is roughly 40-50 ns per crossing; the fused single-call
  design keeps the NEON path ahead of pure Go even at dim=64. Below ~dim 32
  the two are within noise of each other — there is no dim in the practical
  embedding range where the cgo path loses.
- Effective bandwidth at dim=1536: ~43 GB/s (NEON) vs ~6.5 GB/s (old Go).
- Zero heap allocations in all three configurations (the fused C kernel
  reports zero-norm via a NaN sentinel instead of an out-param, keeping the
  wrapper allocation-free).

## Batched scoring (1 query x 10,000 vectors, dim=128, count=3)

| configuration | total | per vector |
|---|---|---|
| cgo NEON, batched (1 crossing) | 114 µs | 11.4 ns |
| cgo NEON, per-pair loop (10k crossings) | 561 µs | 56 ns |
| pure Go fused, batched | 1093 µs | 109 ns |

The batched kernel amortizes the cgo crossing to nothing and lets the NEON
loop stream; it is exposed as the public
`CosineSimilarityBatch`/`CosineSimilarityMany` API.

## End-to-end HNSW (dim=128, M=16, efConstruction=200; count=3, benchtime=1000x)

10k-vector index, k=10, efSearch=50:

| operation | old scalar Go | cgo NEON | speedup |
|---|---|---|---|
| Insert (per vector, growing index) | 716.1 µs | 241.9 µs | 2.96x |
| Search (10k index) | 680.5 µs | 204.1 µs | 3.33x |

The remaining time is graph traversal, allocation, and map bookkeeping in
the fogfish/hnsw library — distance computation is no longer the bottleneck.

## Parity / correctness

- `pkg/vectormath/simd/parity_cgo_test.go` compares every simd kernel
  against `pkg/vectormath/scalar` across dims
  {1,2,3,4,7,8,15,16,17,31,64,127,128,129,256,768,1536}, magnitudes 1 and
  1e4, zero vectors, negatives, and batch rows. Tolerances: 1e-3 relative
  (dot/sqnorm/sqeuclidean), 1e-4 absolute (cosine, bounded in [-1,1]);
  zero-norm/NaN semantics must match exactly.
- Both binaries produce identical top-k IDs on the CLI demo.
- Linux verification: `make docker-verify` builds and tests both CGO modes
  under linux/arm64 (NEON) and linux/amd64 (portable scalar C++).

## Reproduce

```bash
make bench          # cgo/SIMD numbers
make bench-nocgo    # pure-Go numbers
# install benchstat, run each with `| tee out.txt`, then:
benchstat old.txt new.txt
```

## Follow-ups (not in this iteration)

- amd64 AVX2/FMA kernels with runtime `__builtin_cpu_supports` dispatch
  (the `#ifdef` structure in `vectormath_simd.cpp` is ready for it).
- Consider normalizing vectors at insert time so cosine reduces to a dot
  product (halves the FLOPs; changes stored-data semantics).
- Batch the HNSW rescoring loop if k grows beyond ~100 (currently noise).
