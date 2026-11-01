# Slice 14 — ST02 Confirmed Tracked-only Stash-and-Switch Recovery

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** ST02.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 05, 08–09.

**Existing screens only:** S11/S10/S08 as permitted by v2.3 contract.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- Per uploaded v2.3 slice summary: ST02 **Stash and Switch** is offered when a branch switch is blocked by tracked changes, only after explicit confirmation; stash tracked changes only, never automatically restore after switching.
- S16 remains read-only and must not become a stash manager. Never stash untracked files, unmerged entries or incompatible sequencer state merely to make switch succeed.
- Exact confirmation keys, ownership/recovery data, failure/cancel semantics, path-collision checks and v2.3 test IDs are **not present in the supplied summaries**; obtain normative ST02 before implementation. No guesswork.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Read normative ST02, B07 and all v2.3 ST02 test scenarios; if missing, stop this slice as BLOCKED.
- [ ] Wire the ST02 option **only** from the validated dirty-switch blocker; no generic stash command anywhere.
- [ ] Show tracked paths, exact target branch and contents to be stashed, preserve untracked files and require explicit confirmation exactly as normative contract states.
- [ ] Revalidate index/worktree, target ref and sequencer immediately before stash and switch; track actual mutation outcomes and native stash identity as required by contract.
- [ ] Ensure no automatic stash apply/pop/drop after switch, no side effects on Esc/cancel and no S16 mutation affordance.
- [ ] On stash or checkout failure, inspect and report actual current branch, stash ref/OID and index/worktree; do not claim rollback unless proved.
- [ ] Add real repository fixture tests for clean/dirty/unmerged/untracked collision, failure and interrupted execution.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** No specific v2.3-only test description verifiable from supplied sources; see the requirement gate below.

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Implementation remains blocked until normative ST02 details and tests are verified.
- Only explicit ST02 path may stash tracked changes; successful switch leaves stash retained and not automatically restored.
- Cancel/Esc leaves index/worktree/refs/stash unchanged.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Generic stash management, ST03 cherry-pick recovery and auto-restore.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
