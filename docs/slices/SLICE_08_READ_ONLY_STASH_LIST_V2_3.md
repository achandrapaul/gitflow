# Slice 08 — S16 Native Read-only Stash Listing

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** ST01, K46.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01, 07.

**Existing screens only:** S16.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- `g s` opens repository-local S16 displaying native stash refs/OIDs/messages/dates, source branch only when Git reliably exposes it.
- List, scroll, search/filter if specifically approved by normative spec, inspect and exit are **read-only**; no stash create, push, pop, apply, drop, clear or manager action from S16, context menus or palette.
- Stash ordering/identity must be actual current Git values; show empty and parse-error states without inventing entries. Return retains prior navigation.
- ST02/ST03 later allow narrow, specifically approved workflow-managed stash mutations. They do **not** turn S16 into a stash manager.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Implement `g s` registry binding to S16 only and ensure it cannot reach S01 Stage or S08 Skip.
- [ ] Read stash metadata from native Git safely, distinguishing stash position refs from immutable commit OIDs and handling missing/ambiguous metadata.
- [ ] Display truthful read-only entries and empty/error cases; restore previous focus on return.
- [ ] Explicitly deny stash mutation actions in S16, S05 and palette; add mutation-free state snapshot regression test.
- [ ] Keep ST02 and ST03 unavailable with reason until implemented in their own slices.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T145, T146

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Stash refs/OIDs, index, worktree and HEAD are byte-for-byte unchanged after any S16 user action.
- No generic stash apply/pop/drop action is present anywhere.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

ST02 stash-and-switch, ST03 stash-and-proceed and arbitrary stash management.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
