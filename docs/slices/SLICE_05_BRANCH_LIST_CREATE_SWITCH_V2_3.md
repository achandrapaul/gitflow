# Slice 05 — Branch Inspection, Local Branch Creation, Validated Switch and Delete

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** B07–B08, K34, K44, UX-N04.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01–04.

**Existing screens only:** S11 and S01 branch header; S05/S10.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- `g b` globally and S01 `b` alias open the **same** S11. Search/`Enter` inspect only; `s` is the deliberate switch action for eligible local branches after dirty/sequencer guards.
- `g n` creates a **local** branch from current HEAD; `a n` from S11/S01 branch context uses the explicit displayed starting ref. Creation validates name/start/existence, **never** switches, pushes, fetches or sets upstream.
- S11 shows real local and remote refs, tracking, comparison with main, HEAD/current status and valid contextual actions. Remote-tracking ref inspection never auto-creates local branch.
- Local branch deletion remains a distinct guarded approved contextual action; never delete current/protected/stale refs or bypass native safety. No custom destructive shortcut.
- ST02 stash-and-switch is **not** yet active; blocked switches must show exact tracked blocker and safe existing next step without auto-stash.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Build native refs/branch list with truthful local/remote differentiation, search and Enter inspection, and shared `g b`/S01 `b` dispatch.
- [ ] Implement local switch via approved `s` only after index/worktree/sequencer/stale-ref validation; refresh repo status/HEAD/upstream/base after Git completes.
- [ ] Implement explicit `g n` and contextual `a n` branch-create dialogs/review with starting full ref/OID, native name/existence validation and no checkout/upstream/push.
- [ ] Implement guarded local branch deletion using only the existing S05/S10 action; enumerate exact target and constraints, prevent deleting current branch, revalidate before mutation.
- [ ] Preserve/restore navigation context, report blocked switch when dirty; do not offer ST02 until Slice 14.
- [ ] Add bare repo/local fixture tests covering detached HEAD, stale branches, remote-tracking refs, deletion and concurrent repo modifications.
- [ ] **T128 is cross-functional**: assert only branch-create/switch/delete aspects here; explicit upstream and ST02 parts are tested again in later slices and traceability says so.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T55, T105, T128, T136–T138

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Enter/search never checks out; `s` switches only after preflight.
- New branch name exists at reviewed OID, while current HEAD/upstream/remote refs are unchanged; local branch deletion cannot bypass review.
- A dirty blocked switch never stashes by itself.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Create-from-main B10, switch-after-create B11, ST02 recovery, remote push/upstream assignment.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
