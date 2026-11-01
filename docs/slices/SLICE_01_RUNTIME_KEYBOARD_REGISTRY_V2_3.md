# Slice 01 — Runtime, Native Git Read Layer, Keyboard and Action Registry

## 1. CONTRACT AND REQUIREMENTS — READ BEFORE WORK

**Normative authority (mandatory in Claude's repository):** `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repository `CLAUDE.md`. **Status of this document:** implementation delivery instructions; it is NOT a replacement normative product specification.

**Primary requirement ownership:** K01–K09, K29, K35–K38, K41–K42, U01–U03, UX-N01.

**Related cross-slice obligations:** Re-evaluate exact v2.3 clauses affecting S01–S16, global/contextual dispatch, Esc safety, native Git data authority, confirmations and test evidence. **Prerequisites:** None.

**Existing screens only:** S01 shell; S05/S12/S15 infrastructure only.

## Non-negotiable contract enforcement

- **Normative source:** the repository's actual `spec/GITFLOW_REQUIREMENTS_V2_3.md` and its applicable `CLAUDE.md`. Acceptance/scenarios support verification; older archives and Claude's former three-slice summaries are **not** an override. Do not infer missing clauses from these slice instructions.
- **Before editing:** read the full normative v2.3 requirement clauses and exact v2.3 test cases; verify each referenced requirement/test ID and all impacted cross-cutting clauses. If a file, ID, clause or test definition is missing, inconsistent or conflicts with this slice, **STOP the affected implementation**, record the exact conflict, and request a contract decision. Never fill the gap by guessing.
- **Scope:** implement only approved GitFlow S01–S16 functionality. No new screens, shortcuts, modes, hosting APIs, speculative workflows or 'quality-of-life' Git behavior. Preserve approved IDs; `K29` remains reserved.
- **Keyboard invariant:** Enter inspects/opens, Space selects the current typed object, Esc navigates/cancels UI only. `a` contextual, `g` global, `a e`, `gg`, retired `H` and deferred `r i` unbound. Editor keystrokes never dispatch Git shortcuts. Preserve all existing bindings when adding a slice.
- **Git invariant:** native Git semantics, explicit repository cwd and argv (no shell interpolation), NUL-safe paths, safe guards, no silent checkout/fetch/push/stash/discard/delete/overwrite. Git state is authoritative; no navigation event mutates repository state. Revalidate at execution; serialize incompatible mutations.
- **Evidence invariant:** tests use isolated temporary repositories and local bare remotes. Record actual index/worktree/HEAD/refs/sequencer before and after. Do not claim tests pass until they are run. `tests/TRACEABILITY_V2_3.md` must record requirement → test → real source module → result/evidence; no fabricated paths or green status.


## 2. APPROVED BEHAVIOR REQUIRED IN THIS SLICE

- macOS/Linux keyboard-first TUI, S01 as immediate first screen; first **interactive** frame targets 100 ms, cold/warm p50/p95 with machine/method documented.
- Native Git read runner takes argv and explicit repository cwd; Git stdout/stderr/status and cancellation handled without shell interpolation. No network or mutation needed to draw first frame.
- All approved actions exist exactly once in a shared registry: ID, title, scope, target/availability with reason, preview/confirmation, handler and refresh policy. Unimplemented actions are disabled ("Not available yet") and must never run a placeholder Git command.
- Dispatch precedence: focused editor → top overlay → active planner → focused panel → global navigation. `g` and `a` second-key dispatch is case-sensitive and single-consumption. S05/S12/S15 are infrastructure here, **not** authorization to activate operations.
- `Enter`, Space and Esc retain typed semantics; overlay-first Esc and root no-op; `q`, `:`, `/`, `?`, Tab, j/k, arrows, Ctrl+D/U, G as approved; no `gg`, `a e`, `H`, `r i`.

## 3. STRICT IMPLEMENTATION ACTIONS FOR CLAUDE — IN ORDER

- [ ] Write a one-page ADR selecting **Go or Rust** based on startup, terminal library, Git subprocesses, portability, packaging and testability; do not create a new feature decision.
- [ ] Implement repository discovery, native Git process adapter, structured errors and NUL-safe path primitives; add unit fixtures for unusual paths and failure responses.
- [ ] Implement S01 terminal shell, responsive focus model, read-only banner and active prompt; do not run mutating Git commands.
- [ ] Build the single typed action registry with all v2.3-approved action IDs/mnemonics verified against the normative registry, initially disabled outside this slice.
- [ ] Implement a deterministic key-state machine for g/a prefixes, key precedence, single consumption, modal/menu Escape and editor capture.
- [ ] Render S05 contextual list, S12 help/palette and compact S15 global menu from the same registry; do **not** invent Review/Execute/Cancel for routine sync operations.
- [ ] Build reusable nonmodal result/error channel and persistent navigation-state primitives for later slices; keep every future operation unavailable.
- [ ] Instrument first interactive frame and create reproducible cold/warm startup measurement harness; report failure if >100 ms without pretending target met.

## 4. TEST OWNERSHIP AND EVIDENCE

**Primary tests whose meanings can be cross-checked against supplied v2.1 scenarios:** T26, T29, T46, T88, T131, T144

**Important:** These are planning assignments, not assertions of passing tests. When a test was amended in v2.3, **replace the old v2.1 expectation with the actual normative v2.3 scenario**. For v2.3-only T148–T170, do not infer semantics from their numbers. Use `TEST_OWNERSHIP_AND_GAPS.md` and the actual repository's `tests/SCENARIOS_V2_3.md` / `tests/ACCEPTANCE_V2_3.md` before marking them implemented. Every end-to-end scenario may require supporting assertions from multiple slices even when it has one primary owner.

**Required evidence:** Git fixture setup, action invoked (keyboard/menu/palette same handler), actual command/process outcome, pre/post snapshots of relevant index/worktree/HEAD/refs/sequencer, source module and test result; benchmark/usability evidence where applicable.

## 5. DEFINITION OF DONE — ALL MUST HOLD

- No key, navigation, startup, help or palette operation changes HEAD, index, worktree, refs or sequencer.
- Every unimplemented action displays a truthful disabled reason, and no unapproved key triggers anything.
- Measured startup report includes platform, hardware, sample sizes, cold/warm p50/p95 and interactive-frame definition.
- Next slice may rely on a verified safe Git read adapter and exact shared registry.

- Every primary requirement is traced to real code and real execution evidence, or clearly documented as Blocked/Not implemented. All regressions from earlier completed slices pass.
- No unrelated feature, screen, shortcut or mutation path was introduced.

## 6. OUT OF SCOPE / REJECTED SHORTCUTS

Mutating Git; functional status/diff, stage, commit, branch, history, network, stash, cherry-pick, rebase or delete.

## 7. CLAUDE HANDOFF FORMAT

At completion provide: (1) requirement IDs verified against exact v2.3 clauses; (2) implementation files and commit/diff summary; (3) exact commands and Git state changes; (4) automated tests run and actual outcomes; (5) unresolved blockers or coverage gaps; (6) changes to `tests/TRACEABILITY_V2_3.md`. **Do not label the slice complete until the evidence exists.**
