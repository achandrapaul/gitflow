# Slice 15 — ST03 Tracked-only Stash-and-Proceed for Whole-commit Cherry-pick

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** ST03.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 08–11, 14.

**Existing screens only:** S06/S08/S09/S10; no new screen.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- Per supplied v2.3 summary: dirty tracked state may enter S06 read-only; on WHOLE-commit Execute show exact paths and ordered commits; explicit `s` **Stash and Proceed** (Esc cancels); never stash on browsing/preview.
- `git stash push` **tracked-only** (no `-u`, no `-a`) and capture exact stash OID, then verify clean tracked state before native whole-commit cherry-pick. Unmerged, active sequencer and untracked collisions are blockers, not stash opportunities.
- On cherry-pick failure/pause, **keep** owned stash and expose S08; no premature restore. After ALL commits succeed, no sequencer and safe tracked state, `stash apply --index` **of exactly owned stash**; drop only verified GitFlow-owned stash after successful verified restore, report retained stash if drop fails.
- On restore conflict, `c` continues into S09 and `a` rolls back restore attempt **only when safe**, otherwise blocked; precise guards/commit ownership and crash recovery **must** come from actual v2.3 normative contract and tests.
- ST03 **never** applies to S13/S14 selective file application: that workflow remains strictly clean-tracked, staged-without-commit and never stashes.
- ST03's exact action keys/state machine/test cases **cannot be fully verified from supplied summaries**; this is an implementation blocker until normative v2.3 spec and test cases are supplied in repository.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] First read actual ST03 normative section and T161–T170 verbatim, and reconcile with C21, S08, S09; if unavailable/inconsistent, mark BLOCKED and do not implement.
- [ ] Implement a native-state-driven state machine with explicit phases (review; stash-created; clean-verified; cherry-pick progress/pause; pre-restore validation; apply--index; restore verification; optional verified drop; retained/blocked). Do not invent extra user actions or screen IDs.
- [ ] Require exact owned stash OID identity; never rely on `stash@{0}` position alone, never operate on an unrelated existing stash or silently pop.
- [ ] Revalidate paths/index/worktree/sequencer and untracked collisions before stash, each cherry-pick execution, restore and drop. If external Git modified state, block and report instead of guessing.
- [ ] Drive whole commits through native cherry-pick only; preserve selected commit order and reconcile completed/current/remaining against native sequencer after restarts.
- [ ] On any stash, cherry-pick, apply, rollback or drop failure, record actual stash OID/ref, HEAD, index/worktree, and sequencer; do not claim clean rollback if unproved.
- [ ] Use S08/S09 for pauses/conflicts with their true semantics; Esc never aborts; restore rollback only if safely proven per normative rules.
- [ ] Run real multi-repo Git fixtures with preexisting stashes, another process adding a stash, concurrent edits, untracked paths/collision, conflicts, restart between every phase and drop failures.
- [ ] Confirm S14 still blocks on dirty tracked state **even when ST03 is now available**; S16 stays read-only.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** No specific v2.3-only test description verifiable from supplied sources; see the requirement gate below.

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Implementation remains blocked until full normative ST03 and exact T161–T170 cases are verified.
- No unrelated stash is applied/dropped; no user content or untracked file overwritten; partial states reported accurately.
- Whole commits only, no auto-stash in preview, no auto-commit of restored changes, no fake sequencer semantics for selective apply.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Arbitrary stash apply/pop/drop, automatic cleanup of unrelated changes, generic dirty selective apply.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
