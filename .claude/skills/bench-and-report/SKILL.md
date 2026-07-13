---
name: bench-and-report
description: Run the vectormath + HNSW benchmarks, compare against the recorded baselines, and update docs/simd-benchmark-report.md. Use after performance work, before merging kernel changes, or when the user asks how fast something is.
---

# Benchmark and update the report

## Steps

1. Run both modes (each takes a few minutes):
   ```bash
   make bench        | tee /tmp/bench-cgo.txt
   make bench-nocgo  | tee /tmp/bench-nocgo.txt
   ```
   For statistically sound comparisons use `-count=6` and `benchstat`
   (see the Reproduce section of the report).

2. Compare against the baselines in `docs/simd-benchmark-report.md`.
   Key reference points (Apple M1 Pro, darwin/arm64):
   - Cosine single-pair dim=128: ~57 ns cgo/NEON, ~124 ns fused Go
   - Batched 1×10k dim=128: ~11.4 ns/vector (one cgo crossing)
   - HNSW 10k index: insert ~242 µs, search ~204 µs (cgo)

3. **Flag any regression >5%** on the cosine single-pair or HNSW rows to
   the user before doing anything else — do not silently update baselines
   over a regression.

4. If numbers changed materially (new kernel, new machine, new Go
   version), append a dated section to `docs/simd-benchmark-report.md`
   rather than rewriting history. Always record: hardware, OS/arch, Go
   version, compiler, `-count`, and whether `benchstat` confirmed
   significance (p value).

## Rules

- Never compare numbers taken on different machines as if they were a
  regression — note the machine change instead.
- Bench on AC power, minimal background load; rerun anything surprising
  before reporting it.
- `vectormath.Backend()` is logged at startup — confirm the run actually
  used the backend you think it did (`cgo/neon` vs `go/scalar`).
