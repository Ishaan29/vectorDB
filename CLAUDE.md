# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
```bash
# Build the CLI (CGO_ENABLED=1, uses the C++ SIMD math core)
make build

# Build the pure-Go fallback binary (no C++ toolchain needed)
make build-nocgo

# Build the HTTP server
make build-server

# Run the built binary
make run
# or directly:
./build/vectordb -config path/to/config.yaml
```

### Testing and Quality
```bash
# Run all tests with the C++ SIMD core
make test

# Run all tests with the pure-Go fallback
make test-nocgo

# Both modes (do this before considering a change verified)
make test-all

# Benchmarks (vectormath kernels + HNSW end-to-end)
make bench          # cgo/SIMD
make bench-nocgo    # pure Go

# Prove both CGO modes build and pass tests on Linux (arm64 + amd64)
make docker-verify

# Run linting (requires golangci-lint; needs CGO_ENABLED=1 to typecheck cgo)
make lint

# Clean build artifacts
make clean
```

## Architecture Overview

### Core Components

**Engine (`internal/engine/`)**
- The live engine: orchestrates BadgerDB persistence + HNSW index
- Dual-write pattern: BadgerDB first (durability), then HNSW (searchability)
- Startup recovery rebuilds the index from persisted vectors
- Thread-safe via `sync.RWMutex`

**HNSW Index (`internal/index/`)**
- Wraps `github.com/fogfish/hnsw`; distance injected via `CosineSurface`
  (`hnswIndex.go`), which calls `vectormath.CosineDistance` per pairwise
  comparison — this is the hottest code path in the system
- Hardcoded params: M=16, efConstruction=200, efSearch=50 (adjustable via
  `SetSearchEf`)

**Vector Math (`pkg/vectormath/`) — SIMD core**
- Public API (frozen signatures): `Dot`, `Magnitude`, `CosineSimilarity`,
  `CosineDistance`, `EuclideanDistance`, `NormalizeVector`, plus batched
  `CosineSimilarityBatch`/`CosineSimilarityMany`
- Kernel dispatch by build tag:
  - `kernels_cgo.go` (`//go:build cgo`) → `pkg/vectormath/simd` — C++17 core
    bound via cgo; ARM NEON on aarch64, portable scalar C++ elsewhere
  - `kernels_purego.go` (`//go:build !cgo`) → `pkg/vectormath/scalar` — pure
    Go, always compiled, also serves as the parity reference for SIMD tests
- Kernels are fused (cosine = one pass for dot + both norms = one cgo
  crossing) and batched (1 query vs N vectors per crossing)
- `vectormath.Backend()` reports the active implementation
  (`cgo/neon`, `cgo/scalar`, `go/scalar`); logged at startup
- Parity tests: `pkg/vectormath/simd/parity_cgo_test.go` (simd vs scalar,
  epsilon-tolerance — float32 SIMD reassociation means exact equality is NOT
  expected; keep all float assertions tolerance-based)

**Persistence (`persistence/`)**
- BadgerDB store for full vectors + metadata

**API (`internal/api/`)**
- Gin HTTP server (`cmd/vectordb-server`)

### Conventions for the SIMD core
- Never change `pkg/vectormath` public signatures; callers must not care
  which kernel is active
- Validation/error logging stays in Go; C++ kernels are pure arithmetic
  (no exceptions, no allocation, no retained pointers)
- Every simd wrapper guards `len == 0` before taking `&slice[0]`
- Batch APIs take flattened buffers ([][]float32 must never cross cgo)
- Any kernel change must pass `make test-all` AND the parity tests, both
  locally and via `make docker-verify`

### Known quirks
- `config.yaml` uses relative paths (`data/`, `logs/`) resolved against the
  process working directory — run binaries from the repo root (or pass an
  absolute `-config` path and adjust the paths); tests use `t.TempDir()`

### Configuration System
- Main config file: `config.yaml` (server host/port, storage path, index
  type + dimensions (default 128), database limits, logging)
