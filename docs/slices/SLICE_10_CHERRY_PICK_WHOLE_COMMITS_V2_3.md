# Slice 10 — Cherry-pick Explorer, Mode Separation and Native Whole-commit Execution

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** C21, K39–K40, UX-N05.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01–02, 07, 09.

**Existing screens only:** S06; S13 read-only/selection handoff; S08 on pause.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- `g c` opens S06 **read-only even when ordinary tracked changes exist** per supplied v2.3 slice summary; dirty state is evaluated for Execute, not silently mutated on entry. Whether an unmerged/sequencer state blocks entry vs only Execute must be confirmed against the actual v2.3 normative contract.
- S06 target is fixed to checked-out branch; source selector changes **only browse data**. Whole-commit mode defaults with full-OID Space selections, explicit queue and scoped Shift+J/K reorder, `p` preview and active-mode `a` Execute. Enter inspection never executes.
- Entire commit is applied using native `git cherry-pick`, with explicit merge parent when required. Never route a whole commit through selective-file patches.
- Whole-commit and selected-file domains are stored independently, visibly active/inactive, and never combined. S13 Esc restores whole-commit selection. Different source branches may contribute full OIDs to ordered queue.
- On clean tracked state, execute only after exact target/OID/queue/collision/sequencer review and revalidation; untracked colliding paths block. On dirty tracked state, **do not execute** until ST03 has landed and its approved confirmation is used.
- S08 handles sequencer pause; report completed/current/pending accurately; do not replay already-created commits after restart.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Implement S06 source selector and commit/file panels using Slice 07 history loader, with immutable checked-out target and no checkout on source navigation.
- [ ] Build independent selected whole-commit OID queue vs typed selected-file set, with accurate counts and inactive labeling. Scoped Shift+J/K reorders queue only.
- [ ] Implement `p` operation-specific preview and `a` active-mode Execute entry (never `a e`); preview includes target, origin OIDs, order, commit consequences, collisions and state preconditions.
- [ ] Implement native `git cherry-pick` whole-commit execution on **clean tracked state only**, including guarded merge-commit parent and untracked collision handling.
- [ ] On conflict/pause, hand off to Slice 09 S08, reconcile native sequencer with owned queue after restart and never repeat completed commits.
- [ ] Implement S04→S06 source/selection handoff; keep file-application Execute disabled until Slice 11.
- [ ] **Critical stale-test gate:** T89/T90 previously required blocking `g c` on dirty state in v2.1; the v2.3 slice summary allows read-only entry. Verify actual v2.3 T89/T90 wording; if obsolete, correct tests **before** implementing. Never pass obsolete test by reverting the approved new behavior.
- [ ] Implement blocked dirty Execute guidance without creating a stash or launching ST03 early.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T15, T27, T89–T94, T96–T99, T101, T114, T127, T133

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Navigation, Space, Enter, source browsing and plan preview do not mutate Git; target never changes until user deliberately switches elsewhere.
- Clean whole-commit Execute invokes native cherry-pick in reviewed order; file queue remains untouched.
- Dirty tracked state never triggers implicit stash/checkout/cherry-pick and reports blocker while ST03 is unavailable.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Selective file patches and ST03 automatic workflow-managed stash.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
