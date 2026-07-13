# VectorDB

A high-performance cloud native, distributed vector database implementation in Go.
With hyper focus in text and code embeddings. 

## Features

- Fast vector similarity search (HNSW index)
- C++ SIMD math core (ARM NEON) with a pure-Go fallback
- Metadata storage and filtering
- REST API (gRPC coming soon)
- Persistence and recovery (BadgerDB)
- Production-ready configuration

## Project Structure

```
vectorDB/
├── cmd/                   # Binaries: vectordb (CLI), vectordb-server (HTTP), bench (e2e harness)
├── internal/              # Engine, HNSW index glue, HTTP API, config, logger
├── pkg/
│   ├── types/             # Vector, SearchResult
│   └── vectormath/        # Distance math: public API + kernel dispatch
│       ├── scalar/        #   pure-Go kernels (CGO_ENABLED=0 + parity reference)
│       └── simd/          #   C++17 SIMD core via cgo (NEON on aarch64)
├── persistence/           # BadgerDB store
├── proto/                 # Protocol buffer definitions
├── docs/                  # SIMD benchmark report
├── test/                  # Integration tests
├── Dockerfile.verify      # Linux build+test verification (both CGO modes, arm64+amd64)
└── Makefile               # Build automation (cgo build matrix)
```
# Architecture (v0.2 — SIMD core)

```
                 ┌─────────────┐      ┌──────────────────┐
                 │  CLI        │      │  HTTP API (gin)  │
                 │ cmd/vectordb│      │ cmd/vectordb-    │
                 │             │      │ server           │
                 └──────┬──────┘      └────────┬─────────┘
                        │                      │
                        ▼                      ▼
                 ┌────────────────────────────────────┐
                 │        Engine (internal/engine)    │
                 │  dual-write · startup recovery ·   │
                 │  RWMutex thread safety             │
                 └───────┬────────────────────┬───────┘
             insert/search                    │ persist/hydrate
                         ▼                    ▼
        ┌───────────────────────┐   ┌──────────────────────┐
        │  HNSW Index           │   │  BadgerDB            │
        │  (internal/index →    │   │  (persistence/)      │
        │   fogfish/hnsw)       │   │  full vectors +      │
        │  CosineSurface calls  │   │  metadata            │
        │  distance per pair ──┐│   └──────────────────────┘
        └──────────────────────┼┘
                               │  hottest path in the system
                               ▼
        ┌──────────────────────────────────────────────────┐
        │        pkg/vectormath  (public API, frozen)      │
        │   validation + errors stay in Go; arithmetic     │
        │   dispatches to a kernel chosen at build time    │
        │                                                  │
        │   CGO_ENABLED=1              CGO_ENABLED=0       │
        │  ┌────────────────┐        ┌────────────────┐    │
        │  │ kernels_cgo.go │        │kernels_purego.go│   │
        │  └───────┬────────┘        └───────┬────────┘    │
        └──────────┼─────────────────────────┼─────────────┘
                   │ cgo boundary            │
        ═══════════▼═══════════              │ (plain Go call)
        ┌─────────────────────┐    ┌─────────▼──────────┐
        │ pkg/vectormath/simd │    │pkg/vectormath/scalar│
        │ C++17 core          │    │ pure Go, fused      │
        │ ├ aarch64: NEON+FMA │    │ single-pass kernels │
        │ │  (2x unrolled)    │    │ (always compiled —  │
        │ └ other: scalar C++ │    │  also the parity    │
        │   (auto-vectorized) │    │  reference)         │
        └─────────────────────┘    └────────────────────┘
```

  * BadgerDB: persistent storage for full vectors with metadata
  * HNSW Index: fast in-memory similarity search (fogfish/hnsw, distance
    injected via `CosineSurface`)
  * Engine: orchestrates both, dual-write (durability before searchability),
    startup recovery rebuilds the index from persisted data
  * SIMD core: all distance math funnels through `pkg/vectormath`; the
    kernel behind it is selected at build time (C++ NEON vs pure Go)

# e2e Flow
  Insert: BadgerDB → HNSW (each insert costs O(M·efConstruction) distance calls)
  Search: HNSW (O(ef·hops) distance calls) → BadgerDB (hydration)
  Startup: BadgerDB → HNSW (rebuild)

## SIMD Math Core

All vector distance math (`pkg/vectormath`) runs on a pluggable kernel
selected at build time:

```
pkg/vectormath            public API (validation, errors) — frozen signatures
├── scalar/               pure-Go kernels: always compiled; the CGO_ENABLED=0
│                         implementation AND the parity reference for SIMD
└── simd/                 C++17 core bound via cgo (CGO_ENABLED=1)
    ├── vectormath_simd.cpp   ARM NEON on aarch64; portable scalar C++
    │                         (compiler auto-vectorized) elsewhere
    └── simd.go / simd_stub.go  cgo bindings / !cgo stub
```

### Why this is faster — theory of the change

The optimization attacks three independent costs. Each one compounds with
the next:

**1. Kernel fusion — 3 memory passes become 1.**
The old cosine similarity was composed of three separate loops:
`Dot(a,b)` + `Magnitude(a)` + `Magnitude(b)` — both vectors were streamed
from memory three times, with three function-call setups, before a single
division. The fused kernel accumulates the dot product and both squared
norms *in the same loop iteration*, so every float is loaded exactly once.
This alone is the ~2.6x "fused Go" column below, and it is why the
`CGO_ENABLED=0` fallback is also faster than the original code.

**2. SIMD width + FMA — 8 floats per iteration instead of 1.**
On aarch64, NEON registers hold 4 float32 lanes, and fused-multiply-add
(`vfmaq_f32`) does a multiply and an add in one instruction. The kernels
process two NEON vectors per iteration (2x unrolled = 8 floats) with
independent accumulator registers so consecutive FMAs don't stall on each
other's results; a horizontal reduction happens once at the end, and a
scalar tail handles `n % 8`. Dimensionality stays a runtime value — nothing
is compiled for a fixed dim.

**3. Amortizing the cgo boundary.**
A Go→C call costs roughly 40–50 ns. That fact shaped the API:

- *Fusion doubles as crossing-elimination*: one `CosineSimilarity` call is
  one crossing, not three. At dim=128 the NEON work is ~15 ns, so the
  crossing dominates — but the total (~57 ns) still beats pure Go (~124 ns)
  from dim≈64 upward, and the gap widens with dimension.
- *The HNSW hot path cannot batch*: fogfish/hnsw asks the injected
  `CosineSurface` for one pair at a time during graph traversal, so the
  per-pair call has to be cheap by itself. Fusion is the mitigation there.
- *Paths we control do batch*: `CosineSimilarityBatch`/`CosineSimilarityMany`
  score one query against N vectors in a **single** crossing (flattened
  row-major buffer, query norm computed once) — 10,000 vectors cost one
  crossing instead of 10,000, which is the 561 µs → 114 µs row in the
  tables.
- *Zero allocations*: the fused kernel reports zero-norm inputs via a NaN
  sentinel instead of an out-pointer, because any Go pointer passed to C
  escapes to the heap — the sentinel keeps the hot path allocation-free.

Two supporting design rules: input validation and error logging stay on the
Go side (the C++ kernels are pure arithmetic — no exceptions, no
allocation, no retained pointers, per cgo pointer-passing rules), and SIMD
results are parity-tested against the pure-Go kernels with epsilon
tolerances rather than exact equality, since vectorized accumulation
reassociates float32 additions.

### Key design points

- **Frozen public API** — every caller (`internal/index`, tests) is
  untouched; the kernel swap is invisible above `pkg/vectormath`.
- **Pure-Go fallback** — `CGO_ENABLED=0` builds and passes the full test
  suite anywhere Go runs, no C++ toolchain required.
- **Parity gate** — `pkg/vectormath/simd/parity_cgo_test.go` checks SIMD vs
  scalar across 17 dims (1…1536, including non-multiples of the SIMD
  width), zero vectors, large magnitudes, and batch rows; on arm64 it also
  asserts the NEON path is actually compiled in.
- The active backend is logged at startup (`cgo/neon`, `cgo/scalar`, or
  `go/scalar`) via `vectormath.Backend()`.

### Build matrix

| Platform | CGO_ENABLED=1 kernel | Status |
|---|---|---|
| darwin/arm64 (Apple Silicon) | NEON | tested natively |
| linux/arm64 | NEON | verified via `make docker-verify` |
| linux/amd64 | scalar C++ (compiler-vectorized SSE2) | verified via `make docker-verify` |
| any | `CGO_ENABLED=0` → pure Go | tested |

AVX2/runtime-cpuid dispatch for amd64 is a planned follow-up
(`pkg/vectormath/simd/vectormath_simd.cpp` is structured for it).

### Measured results (Apple M1 Pro, go1.24, count=6, benchstat)

Cosine similarity, one pair per call (the shape of the HNSW hot path —
column 2 isolates the fusion win, column 3 adds SIMD + cgo):

| dim | old scalar Go (3-pass) | fused Go | cgo NEON | total speedup |
|---|---|---|---|---|
| 64 | 116 ns | 53 ns | 47 ns | 2.5x |
| 128 | 323 ns | 124 ns | 57 ns | 5.7x |
| 256 | 736 ns | 277 ns | 78 ns | 9.4x |
| 768 | 2684 ns | 931 ns | 159 ns | 16.9x |
| 1536 | 5622 ns | 1908 ns | 283 ns | 19.9x |

Batched scoring, 1 query x 10,000 vectors at dim=128 (the brute-force scan
shape — shows what amortizing the cgo crossing is worth):

| configuration | total | per vector | effective bandwidth |
|---|---|---|---|
| cgo NEON, batched (1 crossing) | 114 µs | 11.4 ns | ~45 GB/s |
| cgo NEON, per-pair (10k crossings) | 561 µs | 56 ns | ~9 GB/s |
| pure Go fused, batched | 1093 µs | 109 ns | ~4.7 GB/s |

End-to-end HNSW (dim=128, M=16, efConstruction=200, 10k index, k=10) —
whole-operation latency including graph traversal and bookkeeping:

| operation | old scalar Go | cgo NEON | speedup |
|---|---|---|---|
| Insert | 716 µs | 242 µs | 3.0x |
| Search | 680 µs | 204 µs | 3.3x |

After the change, distance computation is no longer the bottleneck — the
remaining time is graph traversal and allocations inside the HNSW library.

Reproduce the kernel/index numbers with `make bench` / `make bench-nocgo`;
run the full end-to-end harness (throughput, latency percentiles, recall,
concurrent QPS) with `make bench-e2e`. Full methodology and analysis in
[docs/simd-benchmark-report.md](docs/simd-benchmark-report.md).

### End-to-end baseline (100k vectors, Apple M1 Pro, cgo/neon)

Measured 2026-07-13 at `19ec07a` with the `cmd/bench` harness: 100,000 ×
128-dim clustered vectors (fixed seed), 1,000 held-out queries, recall
against exact brute-force ground truth. "Index" is the raw HNSW; "engine"
is the full path including BadgerDB hydration of results.

| Metric (k=10, efSearch=50) | Index level | Engine level |
|---|---|---|
| Build / ingest throughput | 2,027 vec/s | 1,700 vec/s |
| Recall@10 | 0.681 | 0.655 |
| QPS, single-threaded | 4,252 | 1,666 |
| Latency p50 / p99 | 0.23 / 0.66 ms | 0.57 / 1.24 ms |
| QPS, 10 concurrent workers | 19,063 | 6,555 |
| Index memory overhead | 344 B/vec | — |

Known limits captured by this baseline: recall is bounded by the current
HNSW library's graph quality (only 0.72 even at efSearch=256), recall@100
drops to 0.40 because `efSearch` isn't clamped to `k`, and engine-level
search pays one Badger `Get` per result. Full tables, the efSearch
recall/latency curve, and the version-comparison log live in
[bench/README.md](bench/README.md).

## Getting Started

### Prerequisites

- Go 1.25 or later
- Make
- A C++ toolchain for the SIMD core (clang ships with Xcode CLT on macOS,
  `g++` on Linux). Not needed for `make build-nocgo`.

### Building

```bash
make build         # with C++ SIMD core (CGO_ENABLED=1)
make build-nocgo   # pure-Go fallback, no C++ toolchain needed
make docker-verify # prove both CGO modes build + pass tests on Linux (arm64 + amd64)
```

### Running

```bash
# Using default config
./build/vectordb

# Using custom config
./build/vectordb -config path/to/config.yaml
```

## Configuration

Configuration is handled through a YAML file. Here's an example configuration:

```yaml
server:
  host: localhost
  port: 8080

storage:
  path: data

index:
  type: hnsw
  dimensions: 128

database:
  max_vectors: 1000000
```

## Development
```bash
make build
go run cmd/vectordb/main.go -config config.yaml
```
### Running Tests

```bash
make test
```

### Linting

```bash
make lint
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
