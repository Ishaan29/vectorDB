# CONTEXT.md — current state of work

Living document, unlike CLAUDE.md (timeless conventions). Read this at
the start of a session; update it at the end of any session that changes
project state (the `new-feature` and `pr-prep` skills enforce this).

_Last updated: 2026-07-13_

## Active branches & worktrees

| Branch | Worktree | What it's for |
|---|---|---|
| `main` | `vectorDB/` (primary) | Stable; PRs merge here |
| `dev` | — | Integration branch |
| `banchmark` | — | Benchmark work (merged to main via PR #6) |
| `chore/ai-workflow` | `vectorDB/` (current) | This AI-workflow setup |

## In-flight work

- **AI workflow setup** (`chore/ai-workflow`): skills, agents, specs,
  this file — pending PR.

## Recent decisions

- **SIMD NEON core shipped** — see
  [docs/specs/2026-07-simd-neon-core.md](docs/specs/2026-07-simd-neon-core.md)
  and [docs/simd-benchmark-report.md](docs/simd-benchmark-report.md).
  NEON-only first iteration; fused + batched kernels; `pkg/vectormath`
  API frozen.
- **Spec-first workflow adopted** — features start from
  [docs/specs/TEMPLATE.md](docs/specs/TEMPLATE.md) via the `new-feature`
  skill; specs are the design record.

## Next up

- amd64 AVX2/FMA kernels with runtime dispatch — draft spec at
  [docs/specs/2026-07-avx2-dispatch.md](docs/specs/2026-07-avx2-dispatch.md);
  needs real amd64 hardware for benchmarks.
- Consider insert-time vector normalization (cosine → dot product; halves
  FLOPs, changes stored-data semantics) — needs a spec.
- Planned maintenance (branches not yet created): GitHub Actions CI
  (cgo + nocgo, amd64 + arm64); remove the legacy `db/`/`storage/`
  brute-force scaffold (update CLAUDE.md when it lands); portable
  relative paths in the default config (`config.yaml` has absolute
  macOS paths).
