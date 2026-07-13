# Spec: amd64 AVX2 kernel dispatch

- **Status:** Draft
- **Date:** 2026-07
- **Author(s):** Ishaan Bajpai (with AI assistance)
- **Supersedes / superseded by:** —

## Problem

On amd64 the cgo path currently compiles the portable scalar C++ kernels.
x86 machines with AVX2+FMA leave most of the available FLOPs unused; the
NEON speedups (5.7x cosine at dim=128 on arm64) have no amd64 equivalent.

## Goals

- AVX2/FMA variants of the fused and batched kernels in
  `pkg/vectormath/simd/vectormath_simd.cpp`.
- Runtime dispatch via `__builtin_cpu_supports("avx2")` so one binary
  runs correctly on any amd64 CPU (the `#ifdef` structure is already
  laid out for this).
- `vectormath.Backend()` reports `cgo/avx2` when active.

## Non-goals

- AVX-512 (downclocking trade-offs; revisit separately).
- Any change to the public `pkg/vectormath` API.

## Design

_To be drafted (spec-architect agent) before implementation._

## API impact

None expected; Backend() string value extended.

## Performance expectations

Target ≥3x over portable scalar C++ on cosine dim=128 on an AVX2 machine;
verify via `make docker-verify`-style amd64 benchmarks.

## Test plan

Existing parity tests must pass with the AVX2 kernels active (same
tolerances); `make docker-verify` on linux/amd64 exercises the new path.
Needs a CI or cloud amd64 box with AVX2 for real benchmarks.

## Open questions

- Where to get trustworthy amd64 benchmark hardware (Docker on M1 Pro
  emulates amd64 — numbers are meaningless).
- Dispatch granularity: per-call branch vs function-pointer table set at
  init.
