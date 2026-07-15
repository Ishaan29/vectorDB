# CONTEXT.md — current state of work

Living document, unlike CLAUDE.md (timeless conventions). Read this at
the start of a session; update it at the end of any session that changes
project state (the `new-feature` and `pr-prep` skills enforce this).

_Last updated: 2026-07-14_

## Active branches & worktrees

| Branch | Worktree | What it's for |
|---|---|---|
| `main` | `vectorDB/` (primary) | Stable; PRs merge here |
| `dev` | — | Integration branch |

`banchmark` (PRs #6, #8) and `chore/ai-workflow` (PR #7) are merged and
can be deleted.

## In-flight work

None — working tree clean, all branches merged as of 2026-07-14.

## Recent decisions

- **SIMD NEON core shipped** — see
  [docs/specs/2026-07-simd-neon-core.md](docs/specs/2026-07-simd-neon-core.md)
  and [docs/simd-benchmark-report.md](docs/simd-benchmark-report.md).
  NEON-only first iteration; fused + batched kernels; `pkg/vectormath`
  API frozen.
- **Spec-first workflow adopted** — features start from
  [docs/specs/TEMPLATE.md](docs/specs/TEMPLATE.md) via the `new-feature`
  skill; specs are the design record.
- **E2E benchmark baseline captured** (PR #6) — `cmd/bench` harness;
  100k-vector baseline in
  [bench/results/](bench/results/baseline_20260713-173909.json), tables
  and version-tracking log in [bench/README.md](bench/README.md).
  Headline: recall@10 = 0.681 at efSearch=50, flat recall curve points
  at fogfish/hnsw graph-construction quality as the ceiling.
- **Maintenance landed via PR #8** — GitHub Actions CI (cgo + nocgo),
  portable relative paths in the default config, and removal of the
  legacy `db/`/`storage/` brute-force scaffold (CLAUDE.md updated in
  the same merge).

## Next up

- amd64 AVX2/FMA kernels with runtime dispatch — draft spec at
  [docs/specs/2026-07-avx2-dispatch.md](docs/specs/2026-07-avx2-dispatch.md);
  needs real amd64 hardware for benchmarks.
- Consider insert-time vector normalization (cosine → dot product; halves
  FLOPs, changes stored-data semantics) — needs a spec.
- Benchmark baseline flagged two index issues worth specs: `efSearch`
  not clamped to `k` in `HNSWIndex.Search` (recall@100 = 0.40), and the
  per-result Badger `Get` hydration tax in `Engine.Search` (7x QPS loss
  at k=100).
