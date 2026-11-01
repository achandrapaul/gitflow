# Slice 12 — Visual Interactive Rebase Planner and `g r` Pull-rebase

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** No new ID; owns execution of the existing interactive rebase/paused/conflict normative section.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01–07, 09.

**Existing screens only:** S07/S08/S09/S10.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- S07 planner has explicit base/onto and **contiguous affected range**, with approved Pick/Reword/Edit/Squash/Fixup/Drop, scoped Shift+J/K reorder, `v` preview and contextual `a` Execute. Never bind deferred `r i`.
- Validate fixup/squash ordering, unsupported merge-containing ranges and stale source refs; show rewritten-commit consequences and warn/confirm published-history rewrite. No implicit push.
- Use native interactive Git rebase/sequencer semantics, not manually manufactured fake commits; on edit/conflict pause S08/S09 show actual Git reason and valid controls.
- Enable `g r` native `pull --rebase` only when S08/S09 and the required preflight are available; normal `g p` remains configured-policy and `g f` FF-only.
- Never update local `main` by merging it into checked-out feature branch as a side effect; never confuse user-selected commits with a valid contiguous rebase range.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Implement S07 base/onto/range picker using exact Git refs and an explicit ordered todo plan.
- [ ] Implement todo edits, constrained reorders, `v` preview and preflight for invalid fixup/squash, merge graph and ref staleness.
- [ ] Implement native interactive rebase plan dispatch with guarded published-history warning; store only application-level context needed for UI, never supplant native sequencer state.
- [ ] Connect real pauses and restart recovery to S08/S09; distinguish edit stops from actual conflicts.
- [ ] Enable `g r` through existing registry with approved native pull --rebase semantics and real dirty/upstream/strategy guards.
- [ ] Exercise real temporary history/rewrite fixtures, recovery states, concurrent HEAD changes and abort cancellation without Git mutation.
- [ ] Keep `r i` unbound and preserve S01 `v` folder toggle while S07 `v` previews.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T18–T21, T25, T85

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Rebase yields exact native history for valid plans; invalid plans never start; no automatic force push.
- Pauses and aborts reflect actual sequencer and expose correct conflict roles.
- `g r` works only after validated preflight; routine `g p` and `g f` semantics are unchanged.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

ST02/ST03 managed stash recovery or unapproved history rewrite actions.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
