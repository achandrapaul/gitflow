# Slice 13 — Approved Create-from-main Flow and Post-create Switch Prompt

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** B10–B11, K47.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 05–06, 09.

**Existing screens only:** S01/S11 existing branch context and existing confirmation UI.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- Per uploaded v2.3 slice summary: contextual `a m` creates local branch from **local `main`**, offering `p` FF-only update-and-create, `c` direct create, Esc cancel; never pull main into checked-out feature branch.
- After local branch creation, separately offer approved switch prompt: `s`/Enter intentionally switch, Esc stays. Creation itself never switches/sets upstream/pushes.
- This summary is **not** the full B10/B11/K47 contract: precise applicability, validation, branch-name flow, remote/main preconditions and exact test IDs MUST be checked in `GITFLOW_REQUIREMENTS_V2_3.md` and v2.3 scenarios before implementation.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Read full B10, B11, K47 and related exact v2.3 acceptance scenarios; if absent or inconsistent, record BLOCKED and **do not code this slice**.
- [ ] Confirm `a m` contexts and `p`/`c` meanings against normative contract; route through existing registry and screens without adding a new screen/mnemonic.
- [ ] Implement separate, reviewed local-main startpoint and optional FF-only update flow, with no feature-branch merge and no implicit network/checkout unless explicitly permitted.
- [ ] Implement post-create switch prompt strictly after successful branch creation; Esc keeps current checkout; Enter/s switch uses Slice 05 guarded switch.
- [ ] Verify actual local main OID, created ref, checked-out HEAD, index/worktree/upstream and remote before/after every cancellation/failure/success.
- [ ] Test missing local main, divergent main, detached HEAD, dirty repo, failed FF-only update and user declining switch per full v2.3 contract; do not assume outcomes not specified there.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** No specific v2.3-only test description verifiable from supplied sources; see the requirement gate below.

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Implementation remains blocked until full normative v2.3 clause and scenarios are verified.
- `a m` never rewrites a feature branch or silently checks out/pushes; Esc leaves approved existing Git state intact.
- Only a separately approved and guarded switch changes checkout after creation.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

ST02 or ST03 stash recovery; generic create-from-remote branching.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
