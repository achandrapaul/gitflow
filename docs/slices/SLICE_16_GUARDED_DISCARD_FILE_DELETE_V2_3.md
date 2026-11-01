# Slice 16 — Guarded Discard and Exact Single-file Delete from Disk

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** C14.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 02–05, 09.

**Existing screens only:** S01/S05/S10.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- S01 `d` is **Discard changes**, distinct from S01 contextual `a` → **Delete file from disk** and palette entry. Neither action runs on Space/Enter/navigation.
- Delete-file action targets **exactly one** eligible file, with visible exact path, tracked/staged/untracked status and irreversible-loss warning; require explicit confirmation. Never delete folders/submodules/conflicted/stale paths or multi-file selected batches; never follow symlinks.
- Deleting on-disk file must **not** stage, unstage, reset, restore or change branch/ref. For a staged path the index stays staged, and Git status subsequently truthfully reports worktree deletion.
- Discard has its own approved scope preview/confirmation. Do not invoke `git clean -fd` or silently discard untracked content; native state must be revalidated before executing.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Verify normative v2.3 C14 and discard requirements, retaining existing action IDs and exact S01 `d` behavior.
- [ ] Implement two separate registry handlers and reviews; context and palette must route to the same individual handler.
- [ ] Implement single-file typed path validation with repo-relative normalized identifier, no directory traversal/recursive delete, no following symlinks, and stale-state checks.
- [ ] Implement explicit Cancel-default confirmation and pre-execution snapshot revalidation; no command can execute from Enter-inspection or Esc.
- [ ] After file delete, reread true index/worktree/HEAD/ref status; never auto-stage deletion.
- [ ] Implement approved Discard operation separately, with exact staged/unstaged scope and no implicit untracked cleanup.
- [ ] Test cancellation, tracked/unstaged/staged/untracked, multi-select, folder/submodule/conflict, symlinks and path races in disposable repos.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T47–T50, T56

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Cancel leaves all Git/filesystem bytes unchanged.
- Confirmed Delete removes only one eligible on-disk item while index and HEAD stay unchanged; symlinks aren't followed.
- Discard and Delete remain visibly different actions with different consequences.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Recursive `git clean`, automatic deletion, new delete keybindings or implicit stage/unstage.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
