# Slice 11 — Historical File Selection and Local Staged Patch Application

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** C24.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01–04, 07, 09–10.

**Existing screens only:** S13, S14, S01/S03 draft reuse.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- S13 file selection uses full source commit OID, selected parent, path/change identity; Enter on file inspects its diff. S14 shows ordered commit-parent-file groups, validates overlapping patches and previews exact result.
- **Strict clean tracked index and worktree** before selective application, regardless of ST03; untracked allowed only without path collision. Never silently stash, reset, discard or overwrite.
- Apply valid Git parent→commit patches to **BOTH index and worktree**, leaving staged local changes; **HEAD must not advance** and no commit created. Never replace files by historical snapshots or route through native whole-commit cherry-pick.
- Validate addition/deletion/rename/binary/merge-parent semantics; unsupported combinations block with precise reason; non-atomic partial results must be disclosed with actual completed/remaining actions.
- Generate accurate editable message identifying selected files and source OIDs in the existing S01/S03 composer; never overwrite nonempty existing draft without explicit choice; don't falsely claim full-commit cherry-pick.
- File and whole-commit selection remain independent. A single Space acts on one domain; Esc from S13 restores prior whole-commit mode with file selections remembered but inactive.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Build S13 file-change selection with exact commit/parent/path/change identity and read-only diff inspection.
- [ ] Implement S14 ordered plan and preview with supported-type matrix derived from normative v2.3 spec (do not assume all renames/binary types are supported).
- [ ] Implement preflight requiring clean tracked index/worktree and zero unmerged/sequencer state, plus untracked collision validation at Execute.
- [ ] Implement Git-native patch application with target correctness in index+worktree; test exact before/after HEAD/tree/stage bytes.
- [ ] On an unsupported or partial operation, show exact current repo state, applied/pending items and safe approved next steps; do not claim atomic rollback or cherry-pick `--abort` exists.
- [ ] Propose source-aware composer message, preserving existing draft until user explicitly accepts replacement.
- [ ] Verify the queue's active-mode separation and no accidental whole-commit operation.
- [ ] Never reuse ST03 dirty-state stash path for S14 even after Slice 15 is added.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T100, T115–T117

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- HEAD unchanged, new changes staged in index and present in working tree exactly as selected parent→commit patches.
- No silent overwrite of untracked paths or composer draft.
- Invalid/partial application yields truthful state, without a fictitious native cherry-pick sequencer.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Whole-commit ST03 recovery, arbitrary file checkout or snapshot replacement.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
