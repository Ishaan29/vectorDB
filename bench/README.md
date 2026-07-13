# vectorDB benchmark harness

`cmd/bench` is an end-to-end test bench for comparing versions of the DB on
identical, deterministic data. Run it, change the code, run it again with a
new `-label`, and diff the JSON files in `bench/results/`.

```bash
make bench-e2e          # full 100k-vector baseline (several minutes)
make bench-e2e-quick    # 10k-vector smoke run (~30s)

# or directly, with flags:
go run ./cmd/bench -n 100000 -dim 128 -label my-change
```

(For micro-benchmarks of the math kernels and raw HNSW ops, use `make bench`,
which runs the `go test -bench` suites instead.)

## What it measures

The bench runs at two levels so you can tell *where* time goes:

| Level | Path exercised | What it isolates |
|---|---|---|
| **index** | `internal/index.HNSWIndex` directly | pure ANN performance (graph build + traversal + distance math) |
| **engine** | `internal/engine.Engine` | the user-visible path: locks, BadgerDB writes on ingest, per-result store hydration on search |

Metrics per level:

- **Build / ingest** — total wall time, vectors/sec, per-insert latency
  percentiles (index level), heap growth and bytes/vector (index level).
- **Search latency** — p50/p90/p95/p99/max per query, single-threaded, for
  each `k` in `-k` (default 1,10,100) at the default `efSearch=50`.
- **Recall@k (accuracy)** — fraction of the exact top-k (brute-force cosine
  ground truth) that the ANN search returns, averaged over all queries.
- **efSearch sweep** — recall vs latency trade-off curve at k=10 for each
  value in `-ef` (default 16,32,64,128,256). This is the canonical ANN
  quality/speed curve.
- **Throughput (QPS)** — single-threaded QPS per run, plus a concurrent
  phase (`-workers`, default NumCPU) that measures aggregate QPS and tail
  latency under lock contention.
- **Restart time** (`-restart`) — engine stop + Badger reopen + full HNSW
  rebuild from storage; the operational "how long until searchable after a
  crash" number.

## Current baseline (v0.2 — SIMD core, fogfish/hnsw)

Captured 2026-07-13 at commit `19ec07a` on an Apple M1 Pro (10 cores,
go1.25.0, `cgo/neon` backend). Dataset: 100,000 × 128-dim clustered vectors,
1,000 held-out queries, seed 42. Raw data:
[`results/baseline_20260713-173909.json`](results/baseline_20260713-173909.json).

### Build / ingest

| Metric | Index level (pure HNSW) | Engine level (+BadgerDB) |
|---|---|---|
| Total time (100k vectors) | 49.3 s | 58.8 s |
| Throughput | 2,027 vec/s | 1,700 vec/s |
| Insert latency p50 / p99 | 0.44 / 1.46 ms | — |
| Index heap overhead | 32.8 MB (344 B/vec, excl. ~512 B/vec raw data) | — |

Build throughput decays as the graph grows (3,068 vec/s at 10k → 2,027 at
100k) — each insert costs ~`efConstruction * log(n)` distance evaluations.
The Badger write path adds only ~16% on top of HNSW insertion.

### Search — k sweep at default efSearch=50, single-threaded

| k | level | recall@k | QPS | p50 | p95 | p99 |
|---|---|---|---|---|---|---|
| 1 | index | 0.690 | 4,411 | 0.23 ms | 0.35 ms | 0.55 ms |
| 1 | engine | 0.669 | 3,266 | 0.29 ms | 0.53 ms | 0.89 ms |
| 10 | index | 0.681 | 4,252 | 0.23 ms | 0.41 ms | 0.66 ms |
| 10 | engine | 0.655 | 1,666 | 0.57 ms | 0.86 ms | 1.24 ms |
| 100 | index | 0.400 | 4,107 | 0.24 ms | 0.40 ms | 0.73 ms |
| 100 | engine | 0.398 | 559 | 1.71 ms | 2.32 ms | 3.24 ms |

Index vs engine at the same k isolates the hydration tax: one Badger `Get` +
JSON unmarshal per returned result (2.5x QPS loss at k=10, 7x at k=100).
Recall@100 collapses because `efSearch=50 < k` (see notes below).

### Search — efSearch sweep at k=10, index level

| efSearch | recall@10 | QPS | p50 | p99 |
|---|---|---|---|---|
| 16 | 0.594 | 7,619 | 0.13 ms | 0.42 ms |
| 32 | 0.656 | 5,495 | 0.19 ms | 0.45 ms |
| 50 | 0.681 | 4,252 | 0.23 ms | 0.66 ms |
| 64 | 0.692 | 3,585 | 0.27 ms | 0.80 ms |
| 128 | 0.711 | 2,384 | 0.38 ms | 1.30 ms |
| 256 | 0.724 | 1,648 | 0.52 ms | 1.72 ms |

**This curve is the baseline's headline finding.** A well-built HNSW graph at
M=16 / efConstruction=200 typically reaches recall@10 ≥ 0.95 on data like
this; here, 16x more search effort buys only 0.59 → 0.72. Recall is limited
by graph construction quality, not search effort — the number to beat when
changing the index implementation.

### Concurrent search — k=10, efSearch=50, 10 workers

| level | QPS | scaling vs 1 thread | p50 | p99 |
|---|---|---|---|---|
| index | 19,063 | 4.5x | 0.23 ms | 6.04 ms |
| engine | 6,555 | 3.9x | 0.67 ms | 9.60 ms |

Throughput scales sub-linearly (~4.5x on 10 cores) and p99 inflates ~10x
under load — `RWMutex` contention plus memory bandwidth.

## Tracking versions against the baseline

Add a row here for each meaningful change, using the same fixed-seed run
(`go run ./cmd/bench -label <version-name>`). Key columns to watch, all at
k=10 / ef=50 / index level unless noted:

| version (label) | commit | build vec/s | recall@10 | QPS 1-thread | p50 | p99 | conc. QPS ×10 | engine QPS |
|---|---|---|---|---|---|---|---|---|
| `baseline` | `19ec07a` | 2,027 | 0.681 | 4,252 | 0.23 ms | 0.66 ms | 19,063 | 1,666 |

## Comparing versions fairly

- Keep `-seed`, `-n`, `-dim`, `-dist` fixed across runs — the dataset and
  queries are then bit-identical.
- HNSW level assignment is random *inside* the library, so recall can move
  ±0.01–0.02 between runs of identical code. Treat small recall deltas as
  noise; rerun before believing them.
- The result JSON records the git commit, math backend (`cgo/neon` vs
  `go/scalar`), CPU, and all parameters — so a result file is self-describing.
- Run on an idle machine, plugged in; laptop thermal throttling skews tails.

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `-n` | 100000 | base vectors |
| `-dim` | 128 | dimensions (must match engine config) |
| `-queries` | 1000 | held-out queries (never inserted) |
| `-k` | `1,10,100` | k values benched at default efSearch |
| `-ef` | `16,32,64,128,256` | efSearch sweep at k=10 |
| `-dist` | `clustered` | `clustered` (100 Gaussian clusters, realistic) or `uniform` (adversarially hard for ANN) |
| `-seed` | 42 | dataset RNG seed |
| `-mode` | `all` | `index`, `engine`, or `all` |
| `-workers` | NumCPU | concurrent-phase goroutines |
| `-concurrent-ops` | 5000 | total searches in concurrent phase |
| `-restart` | false | measure engine restart/rebuild time |
| `-label` | `baseline` | name recorded in the result file |
| `-out` | `bench/results` | result directory |
| `-data` | system temp | parent dir for the engine's throwaway Badger data |

## Notes on interpreting the baseline

- Engine-level search is expected to be much slower than index-level at the
  same k: `Engine.Search` does one Badger `Get` + JSON unmarshal per result
  to hydrate it. That gap is the cost of hydration, not of the index.
- Recall at k=100 with the default `efSearch=50` is structurally poor: HNSW
  needs `ef >= k` to fill the candidate list. The index wrapper currently
  does not clamp `efSearch` up to `k`.
- Index build is single-threaded through a mutex; build throughput drops as
  the graph grows (each insert is ~`efConstruction * log(n)` distance calls).
