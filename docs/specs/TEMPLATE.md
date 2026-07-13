# Spec: <feature name>

- **Status:** Draft | Approved | Implemented | Superseded
- **Date:** YYYY-MM-DD
- **Author(s):** <who> (with AI assistance where applicable)
- **Supersedes / superseded by:** <spec link or —>

## Problem

What is broken, missing, or too slow, and why it matters now. One or two
paragraphs; link issues/benchmarks as evidence.

## Goals

- Bullet list of what this change must achieve — each one testable.

## Non-goals

- What is deliberately out of scope (and, briefly, why).

## Design

The concrete approach: packages touched, new types/functions, data flow.
Include at least one considered-and-rejected alternative and the reason it
lost. Diagrams welcome if the flow is non-obvious.

## API impact

New or changed public surface. Note: `pkg/vectormath` public signatures
are frozen — a change there is a blocker to escalate, not a design detail.

## Performance expectations

Expected effect on the hot path (HNSW → CosineDistance), cgo crossing
count, allocations. Cite current baselines from
`docs/simd-benchmark-report.md`. After implementation, replace estimates
with measured numbers.

## Test plan

Unit / parity / integration / benchmark coverage. Float comparisons are
tolerance-based, never exact. Kernel-adjacent changes must pass
`make test-all` and `make docker-verify`.

## Open questions

Decisions pending user input. Resolve (and record the resolution) before
Status moves to Approved.
