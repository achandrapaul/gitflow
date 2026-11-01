# Slice 04 — Single Composer, Commit Review, Hooks, Amend and One-shot No-Verify

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** C10–C11, C22–C23, K43, UX-N06.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** 01–03.

**Existing screens only:** S01/S03, S05 contextual, S10 guarded confirmation.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- Exactly one S01/S03 subject/body draft: S01 `c` focuses it; S03 reviews the **same** draft, staged index only; no staged content disables commit. Editing or navigation doesn't lose draft.
- Commit is explicit after review. Capture reviewed index tree OID/message and HEAD; revalidate before execution. Normal native `git commit` runs hooks and signing. Inspect actual HEAD/tree/message afterward; disclose hook-induced differences, never claim reviewed and committed contents identical if changed.
- On hook/signing failure or uncertain unexpected HEAD movement, preserve draft, reread real index/HEAD, invalidate stale review, never auto-reset, auto-retry, auto-amend or silently disable signing.
- `a v` only in commit context opens one-shot **warned** `git commit --no-verify` review, never instantaneous; Git-supported pre-commit and commit-msg bypass only, not prepare-commit-msg or signing. Future normal commits return to hooks.
- Contextual Amend is a distinct reviewed action with published-history rewrite warning and no auto-push. S10 Cancel default; Esc cancels UI, not Git.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Connect one draft model to S01 inline composer and S03 review; make typing take priority over action shortcuts.
- [ ] Build staged-only review with index tree identity, message and precondition snapshot; reject changed index/HEAD before executing.
- [ ] Invoke native commit through shared registry and validate actual created commit via Git (HEAD parent/identity, tree/message); present differences and inspection route.
- [ ] Support hooks that fail, mutate index, mutate message, succeed, or unexpectedly change refs; preserve draft and expose changed reality.
- [ ] Add contextual guarded Amend; never silently push rewritten commits.
- [ ] Add contextual `a v` with warning, explicit review/confirm and one-use flag that resets on every outcome; never skip GitFlow preflight or signing.
- [ ] Update nonmodal outcome and blocked-action feedback without routine success popup.
- [ ] Run hook/signing fixtures and the measured learning study only if actually conducted; do not fabricate usability results.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T09, T10, T57, T58, T102–T104, T121–T126, T129, T132, T134, T143

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- Commit contains exactly the actual intended index snapshot, with discrepancies disclosed if hooks modify it; unstaged portions stay unstaged.
- No stale review is executable; failed hook retains draft and leaves actual Git state intact.
- Normal commit hooks remain default, one-time bypass requires deliberate confirmation, Amend never triggers a push.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Remote sync, branch mutation, cherry-pick or rebase.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
