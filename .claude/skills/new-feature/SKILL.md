---
name: new-feature
description: Spec-first workflow for any new feature or significant design change. Use when the user asks to add a feature, change architecture, or start something bigger than a bug fix — write and approve a spec before writing code.
---

# New feature (spec-first)

No significant feature lands without an approved spec. Bug fixes and
mechanical refactors are exempt; anything that adds API surface, changes
data semantics, or touches the kernel dispatch design is not.

## Steps

1. **Draft the spec.** Copy `docs/specs/TEMPLATE.md` to
   `docs/specs/YYYY-MM-<slug>.md` (date = today, slug = short kebab-case
   feature name). Fill every section. For design-heavy features, launch
   the `spec-architect` subagent to explore the codebase and draft the
   Design / API impact / Performance expectations sections.

2. **Stop for approval.** Present the spec to the user and wait. Do not
   start implementing an unapproved spec. Record decisions from the
   discussion in the spec, then set Status: Approved.

3. **Implement against the spec.** If reality diverges from the spec
   mid-implementation, update the spec first (it is the source of truth),
   flagging the change to the user if it alters goals or API.

4. **Verify.** If the change touched `pkg/vectormath`, the C++ core, or
   `internal/index`, finish with the `verify-kernel-change` skill.
   Otherwise `make test-all` at minimum.

5. **Close the loop.**
   - Set the spec's Status to Implemented.
   - Add/update the row in `docs/specs/README.md`.
   - Update `CONTEXT.md` (in-flight work, recent decisions, next up).
   - If conventions changed, update `CLAUDE.md`.
