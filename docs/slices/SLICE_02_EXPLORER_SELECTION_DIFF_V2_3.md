# Slice 02 — Changes Explorer, Branch Intelligence, Selection and Read-only Diff

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** B01–B06, C01–C07, C09, C13, C15–C20, UX-N03, UX-N07.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01.

**Existing screens only:** S01, S02; S12 navigation handoff.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- S01 groups each changed/untracked repo-relative path once, with native index and worktree columns; four filters `1`–`4` (All/Staged/Not staged/Mixed), including mixed paths in all four and untracked only in All/Not staged.
- Header shows current branch/detached HEAD, HEAD OID, real upstream and ahead/behind, **independent** `origin/main` else local `main` comparison and unknown ref freshness; never fetch automatically.
- S02 staged `HEAD→index` and unstaged `index→worktree` diffs separated; binary/read limitations truthfully displayed. Navigation/inspection is read-only.
- Focus `j/k` and arrows **never** select. Space toggles only focused file. Stable-anchored Shift+J/K range expands/contracts; independently Space-checked files survive contraction. `Shift+S` selects all FILES including filter-hidden and closed folders. Folder headers are not files.
- Filter changes and S01↔S02↔S12 navigation keep selection, scroll and focus. Overlay Esc closes overlay first; next root Esc clears all selected files and range anchor but not Git. Selection persists even if currently hidden, with total/hidden counts.
- S01 `v` toggles Flat/By folder immediately and saves global per-user preference atomically; no automatic mode change; show save failures. Minimum terminal size and stacked layout preserve all keyboard actions.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Parse `git status --porcelain=v2 --branch -z` with correct type-1/type-2/unmerged/untracked records, rename paths, special filenames and partial states. Build immutable file identities; never use line-oriented parsing for Git paths.
- [ ] Implement branch HEAD/upstream/base comparisons from local refs (including detached, missing upstream/base, divergent counts, unknown freshness); never substitute upstream for main.
- [ ] Build S01 unified list, status truth table, four filters, folder/flat view, persistence/error handling and reliable empty state.
- [ ] Implement separately owned focus, explicit checked set and anchored range; preserve checked identities through filter/scroll/collapse and return. Surface total/hidden counts in footer and future action preflight.
- [ ] Implement read-only S02 staged+unstaged diff, binaries and untracked display, stable focus return.
- [ ] Handle low terminal dimensions without losing approved controls; add snapshots and keyboard traversal tests.
- [ ] Implement read-only paused badge detection hook only (no S08 operations until Slice 09); render disabled commit reason in clean repo.
- [ ] Verify Git state snapshots before/after all read-only scenarios and persist truthful preference diagnostics.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T01–T05, T08, T11–T13, T30–T32, T51–T54, T106–T108, T111–T113

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Status truth table and mixed/untracked/rename semantics match real Git under all filters.
- No hidden selections disappear or silently join; Space never stages; Esc never mutates Git; `j/k` changes focus only.
- A mixed file presents two independent diff sections and independent ahead/behind comparisons are correct.
- Flat/Folder preference survives restarts/repositories or reports save failure honestly.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

File/hunk/line mutation, commit, sync, history or a recovery dashboard.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
