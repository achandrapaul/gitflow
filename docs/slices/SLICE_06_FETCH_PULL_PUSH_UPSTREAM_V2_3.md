# Slice 06 — Fetch, Configured Pull, FF-only Pull, Push, Explicit Upstream and Lease

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** B09, K30–K32, K45.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01–05.

**Existing screens only:** S15 global; S05 contextual upstream; S10 guarded confirmation.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- `g e` fetch; `g p` configured pull; `g f` fast-forward-only pull; `g P` normal push. **No generic S15 Review/Execute/Cancel wizard for routine commands.** Operation-specific preflight/reviews only where normative contract requires them.
- Pull honors Git's configured strategy and actual upstream; unknown strategy/upstream is a blocker, not a guessed rebase/merge. Do not automatically merge/pull `main` into checked-out feature branch.
- On first push without upstream, review destination `origin/<current-local-branch>`; after explicit approval invoke native `git push -u origin <local-branch>`, and only a successful verified push may establish tracking. If origin missing or push rejected/uncertain, do not claim tracking.
- `g u` explicitly reviews upstream for **checked-out** local branch; contextual `a u` for the **selected** local branch. Neither fetches, pushes nor switches.
- `g F` force-with-lease has distinct guarded review of actual remote/ref, expected OID and rewrite risk; `c` confirms after revalidation; stale/unknown lease fails closed; never unrestricted force.
- `g r` pull-rebase remains unavailable until S08/rebase support is ready (Slice 12).

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Implement fetch, configured pull, FF-only pull and ordinary push via native Git and the existing registry; no generic extra execution dialog.
- [ ] Preflight upstream, strategy, dirty/sequencer conflicts, destination existence and changed refs; show target/reason before invoking.
- [ ] Implement explicit `g u`/`a u` configuration flows with validated local/remote/ref identities; no mutation before approval.
- [ ] Implement no-upstream first-push review and `push -u` success verification; handle existing/divergent remote, absent origin, auth failures and uncertainty.
- [ ] Implement separate `g F` guarded force-with-lease expected-OID review and revalidation; never downgrade to `--force`.
- [ ] Stream long-running Git command progress without freezing UI; serialize incompatible operations; refresh actual refs/status and retain focus.
- [ ] Keep `g r` unavailable with reason; enable it only with Slice 12's native rebase pause recovery.
- [ ] Test against temporary local bare remotes and diverse Git configs, never live network credentials.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T84, T86, T87, T110, T139–T142

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- No auto-fetch/checkout/rebase and no generic nested action popup for routine g commands.
- Failed/uncertain first push does not fabricate upstream; existing non-origin upstream isn't overwritten.
- Force-with-lease cannot run without distinct explicit confirmation and safe lease validation.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Pull --rebase, branch creation from main, cherry-picking, automatic stash.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
