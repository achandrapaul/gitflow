# Slice 03 — File, Hunk and Supported Line Staging/Unstaging

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** C08, C12, U05–U06, UX-N02.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01–02.

**Existing screens only:** S01, S02.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- S01 `s`/`u` and S02 contextual actions stage or unstage selected file/hunk/supported line. Partial patches must preserve non-target changes and fail closed on invalid application; binary only file-level.
- Checked file targets take precedence even if hidden by filters/collapsed folders. Show total selected and hidden count **before** mutation; never silently swap with focused unchecked row. With no checked set, focused eligible file is sole default target. Explicit stale/invalid gestures must not retarget silently.
- Actual native index state after a mutation (not UI toggles) determines status, diffs, counts and selection reconciliation. Serialize mutations; recheck HEAD/index/path status at execution.
- Disabled/ineligible members retain exact reason; on partial outcomes disclose which paths/hunks succeeded or failed. Focus/scroll/filter and meaningful selections survive refresh.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Implement a typed action target resolver consuming the Slice 02 checked-set model, including hidden selections, ineligible paths and stale identities.
- [ ] Implement native file stage/unstage with pathspec-safe argv and NUL-safe path handling; never auto-add every path in repo.
- [ ] Implement validated hunk and supported-line patch staging/unstaging, with correct index vs worktree rules and safe fallbacks for binary/unsupported cases.
- [ ] Add pre-execution path/index revalidation, serialized operation dispatch and honest per-target partial result reporting.
- [ ] Refresh full status/diffs from Git after mutation and reconcile focus/selected identities; maintain footer action counts.
- [ ] Create temp Git fixtures for mixed, rename/delete, untracked, overlapping edits, invalid line patch, binary and spaces/newline filenames.
- [ ] Keep every commit and network action disabled until its own slice.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T06, T07, T109, T130, T147

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Targeted index contents exactly equal expected staged changes while unaffected working-tree content stays unchanged.
- Hidden checked files remain targets; focused unchecked file is never silently included.
- Unsupported line/hunk patch reports a blocker without corrupting the index.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Commit creation, discard or file deletion, cherry-pick and other Git mutations.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
