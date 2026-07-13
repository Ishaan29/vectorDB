---
name: pr-prep
description: Pre-PR ritual — lint, full tests, subagent code review, docs/context updates, and a clean commit message. Use when the user says a change is ready, asks to open a PR, or asks to commit substantial work.
---

# Prepare a PR

## Steps

1. **Quality gates:**
   ```bash
   make lint        # needs golangci-lint; CGO_ENABLED=1 to typecheck cgo
   make test-all    # both CGO modes
   ```
   If the diff touches `pkg/vectormath` or `internal/index`, run the
   full `verify-kernel-change` skill instead of bare test-all.

2. **Review pass:** launch the `code-reviewer` subagent on the branch
   diff (`git diff main...HEAD`). Fix every Critical and Major finding;
   use judgment on Minor/Suggestion and tell the user what was skipped.

3. **Docs and context:**
   - Update any spec whose Status changed (`docs/specs/`).
   - Update `CONTEXT.md` — in-flight work, decisions, next up.
   - Update `CLAUDE.md` / `README.md` if commands or architecture changed.

4. **Commit and PR:**
   - Never commit directly to `main` — work on a feature branch.
   - Commit message: one imperative summary line, then a short body
     saying *why*, and benchmark deltas when performance changed.
   - PR description: link the spec if one exists, list verification
     actually performed (which of test-all / parity / docker-verify /
     bench ran), and call out anything intentionally deferred.

## Rules

- Report test/lint output honestly — a red gate blocks the PR, it does
  not get a footnote.
- No drive-by refactors in a PR branch; spin them off.
