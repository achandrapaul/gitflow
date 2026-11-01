# Slice 09 — Native Paused-operation Dashboard and Truthful Conflict Resolution

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** U04, O10.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01–04.

**Existing screens only:** S08, S09, S10.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- Detect real native merge/rebase/cherry-pick state at startup and after commands; S08 shows actual pause reason, commit/progress, conflicted paths and **only supported** Continue, Skip, Abort, Refresh controls.
- S08 `c` Continue subject to preflight, `s` guarded Skip only for rebase/cherry-pick, `A` guarded Abort, `r` Refresh; S10 Cancel default with exact operation and consequences; `o` opens S08 from normal navigation when paused.
- Esc, `q` and UI navigation leave sequencer untouched; app restart reloads real sequencer state instead of trusting stale UI memory.
- S09 displays real branch/ref+role, or actual commit OID+role when ref cannot be reliably identified. Index stage mapping is operation-specific, particularly reversed intuitive sides in rebase. Only valid base variant can be shown; no fictional 'Both'.
- 1/2/3 choose only valid text conflict variants; n/N region navigation and `e` configured editor; explicit mark-resolved/index action, **never auto-continue**. Binary/rename/delete use safe fallback.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Implement Git-state detectors for merge/rebase/cherry-pick pause and filesystem/index conflict statuses, reading sequencer metadata each refresh.
- [ ] Implement S08 dashboard and persistent compact paused badge with supported-native-operation control matrix; disable unsupported Skip for merge.
- [ ] Implement S10 guarded confirmation with Cancel default, `c` only after preflight and honest stderr/status after action.
- [ ] Implement S09 conflict region inspection, actual ref/role stage mapping, explicit mark-resolved action and manual editor fallback.
- [ ] Build native temp-repo fixtures for merge, rebase and cherry-pick text conflicts and binary/delete conflicts; assert actual index stages/contents.
- [ ] Test restart, external changes, Esc and quit leaving Git state unchanged; test Continue/Skip/Abort only when valid.
- [ ] Keep starting new rebase/cherry-pick workflows gated until their own slices; use external Git to prepare fixtures.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T16, T17, T22–T24, T28, T118–T120

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Source labels and stage-2/3 contents are correct for all three operations; unsupported conflict types never present false choices.
- Esc/quit never abort, skip or reset; restart reflects actual Git operation state.
- Each enabled recovery control issues only its supported native command after explicit review/guards.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Starting cherry-picks or interactive rebase; ST02/ST03 managed stash recovery.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
