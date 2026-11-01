# Slice 07 — Paginated History, Commit Inspection and Compare

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** H01–H08, K33, U07.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01–02, 05.

**Existing screens only:** S04, S13 read-only inspector, S05 contextual actions.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- `g h` opens S04 from normal navigation; retired `H` unbound. History initially loads ~100 commits, prefetches before end, virtualizes rows and preserves full-OID focus/checks across pages and filters.
- History is a Git DAG: stable traversal/refs snapshot, no duplicate/missing records at page boundaries; ref movement requires truthful invalidation/refresh.
- Enter inspects commit/diff; Space selects *whole commit* identity in history; exactly two checked commits may be compared; arbitrary selection is not a rebase range.
- Search scope is labeled loaded-only vs whole-repository. Branch source browsing never switches current target or HEAD.
- Context handoff by `g c` to S06 retains exact OIDs and browsing branch when Slice 10 lands; before then it is registered and unavailable with reason.
- Historical commit file inspector S13 is **read-only here**; per-file selecting/execution belongs to Slice 11.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Implement native bounded commit graph traversal, stable cursor/snapshot and incremental page loading; test merge DAG >1000 commits.
- [ ] Render S04 rows, graph, commit details and diffs using full-OID identities; retain focus/scroll/selection over pagination.
- [ ] Add source branch browse/search, truthful search scope, two-commit compare and independent global history `g h` action.
- [ ] Implement read-only inspection path for commit files in S13, with no per-file selection/execution yet.
- [ ] Register read-only S04→S06 handoff with disabled reason until Slice 10; never switch HEAD to browse commits.
- [ ] Recheck no index/worktree/ref mutation on pagination, preview or Esc.
- [ ] Add test cases for merges, renamed files, ref updates during pagination, selection persistence and detached state.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T14, T95, T135

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Stable history pages without duplicates or omissions under fixed traversal, and truthful refresh on ref change.
- `g h` works; retired `H` does not; inspection and source changing are read-only.
- Two-commit compare does not start a rebase or checkout.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Cherry-pick execution, apply historical files, rebase.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
