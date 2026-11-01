# GitFlow v2.3 — Contract-locked, Incremental Implementation Slices

**Purpose:** replace Claude's three oversized delivery slices with **16 narrowly scoped, dependency-ordered implementation slices**, each as a separate Markdown file with normative contract/requirement IDs FIRST, then explicit actions, tests, acceptance, and exclusions.

## Mandatory source gate (do not skip)

The files provided for this planning exercise included Claude's v2.3 slice summaries and an earlier **complete v2.1** handoff archive. They did **not** include the actual `spec/GITFLOW_REQUIREMENTS_V2_3.md`, repository `CLAUDE.md`, nor the exact v2.3 acceptance/scenario files. Accordingly, this packet is **not a substitute for the normative v2.3 specification**. Claude must read and confirm the repository's actual v2.3 normative documents before coding. If absent, contradicting this plan, or missing referenced IDs, **stop affected work and report a precise contract blocker**. Do not silently use v2.1 or this packet as a fallback product contract.

The supplied v2.3 summaries explicitly add B10/B11, K47, ST02, ST03 and read-only entry to S06 when ordinary tracked state is dirty; they describe changes to tests T90/T148/T149 and new T161–T170. Those summaries are **incomplete**; full semantics and v2.3 test descriptions must be verified from actual v2.3 sources. T89 also needs mandatory reconciliation, because its v2.1 version blocked read-only S06 entry.

## Delivery order

| Seq | Slice Markdown file | Feature boundary | Dependencies |
|---:|---|---|---|
| 01 | [`SLICE_01_RUNTIME_KEYBOARD_REGISTRY_V2_3.md`](SLICE_01_RUNTIME_KEYBOARD_REGISTRY_V2_3.md) | Runtime, Native Git Read Layer, Keyboard and Action Registry | None |
| 02 | [`SLICE_02_EXPLORER_SELECTION_DIFF_V2_3.md`](SLICE_02_EXPLORER_SELECTION_DIFF_V2_3.md) | Changes Explorer, Branch Intelligence, Selection and Read-only Diff | 01 |
| 03 | [`SLICE_03_STAGE_UNSTAGE_V2_3.md`](SLICE_03_STAGE_UNSTAGE_V2_3.md) | File, Hunk and Supported Line Staging/Unstaging | 01–02 |
| 04 | [`SLICE_04_COMMIT_AMEND_VERIFY_V2_3.md`](SLICE_04_COMMIT_AMEND_VERIFY_V2_3.md) | Single Composer, Commit Review, Hooks, Amend and One-shot No-Verify | 01–03 |
| 05 | [`SLICE_05_BRANCH_LIST_CREATE_SWITCH_V2_3.md`](SLICE_05_BRANCH_LIST_CREATE_SWITCH_V2_3.md) | Branch Inspection, Local Branch Creation, Validated Switch and Delete | 01–04 |
| 06 | [`SLICE_06_FETCH_PULL_PUSH_UPSTREAM_V2_3.md`](SLICE_06_FETCH_PULL_PUSH_UPSTREAM_V2_3.md) | Fetch, Configured Pull, FF-only Pull, Push, Explicit Upstream and Lease | 01–05 |
| 07 | [`SLICE_07_COMMIT_HISTORY_INSPECTION_V2_3.md`](SLICE_07_COMMIT_HISTORY_INSPECTION_V2_3.md) | Paginated History, Commit Inspection and Compare | 01–02, 05 |
| 08 | [`SLICE_08_READ_ONLY_STASH_LIST_V2_3.md`](SLICE_08_READ_ONLY_STASH_LIST_V2_3.md) | S16 Native Read-only Stash Listing | 01, 07 |
| 09 | [`SLICE_09_PAUSED_OPERATIONS_CONFLICTS_V2_3.md`](SLICE_09_PAUSED_OPERATIONS_CONFLICTS_V2_3.md) | Native Paused-operation Dashboard and Truthful Conflict Resolution | 01–04 |
| 10 | [`SLICE_10_CHERRY_PICK_WHOLE_COMMITS_V2_3.md`](SLICE_10_CHERRY_PICK_WHOLE_COMMITS_V2_3.md) | Cherry-pick Explorer, Mode Separation and Native Whole-commit Execution | 01–02, 07, 09 |
| 11 | [`SLICE_11_SELECTIVE_HISTORY_FILE_APPLY_V2_3.md`](SLICE_11_SELECTIVE_HISTORY_FILE_APPLY_V2_3.md) | Historical File Selection and Local Staged Patch Application | 01–04, 07, 09–10 |
| 12 | [`SLICE_12_INTERACTIVE_REBASE_AND_REBASE_PULL_V2_3.md`](SLICE_12_INTERACTIVE_REBASE_AND_REBASE_PULL_V2_3.md) | Visual Interactive Rebase Planner and `g r` Pull-rebase | 01–07, 09 |
| 13 | [`SLICE_13_CREATE_FROM_MAIN_AND_POST_CREATE_SWITCH_V2_3.md`](SLICE_13_CREATE_FROM_MAIN_AND_POST_CREATE_SWITCH_V2_3.md) | Approved Create-from-main Flow and Post-create Switch Prompt | 05–06, 09 |
| 14 | [`SLICE_14_ST02_STASH_AND_SWITCH_V2_3.md`](SLICE_14_ST02_STASH_AND_SWITCH_V2_3.md) | ST02 Confirmed Tracked-only Stash-and-Switch Recovery | 05, 08–09 |
| 15 | [`SLICE_15_ST03_DIRTY_CHERRY_PICK_RECOVERY_V2_3.md`](SLICE_15_ST03_DIRTY_CHERRY_PICK_RECOVERY_V2_3.md) | ST03 Tracked-only Stash-and-Proceed for Whole-commit Cherry-pick | 08–11, 14 |
| 16 | [`SLICE_16_GUARDED_DISCARD_FILE_DELETE_V2_3.md`](SLICE_16_GUARDED_DISCARD_FILE_DELETE_V2_3.md) | Guarded Discard and Exact Single-file Delete from Disk | 02–05, 09 |

## Execution discipline

For each slice, Claude MUST (1) verify exact normative v2.3 clauses and test IDs, (2) identify changed and cross-cutting requirement IDs, (3) confirm no contradiction with earlier completed slices, (4) implement only approved functionality, (5) run unit/integration/real-Git fixtures, (6) update v2.3 traceability with module+evidence, (7) stop on unsolved design ambiguity rather than invent a resolution.

**One primary ownership per requirement ID:** see `REQUIREMENT_OWNERSHIP.md`. Implementation of shared invariants in other slices is **supporting**, never reassignment of contract ownership. **One primary test owner where the scenario's meaning is known:** see `TEST_OWNERSHIP_AND_GAPS.md`. Exact v2.3-only T148–T170 meaning/ownership remains unverified until source is available, and must not be fabricated.

**Special release gates:** Slice 01 action registry/keyboard semantics; Slice 02 selection model; Slice 04 commit hooks; Slice 09 native sequencer/conflicts; Slice 10 whole-commit cherry-pick; Slice 11 clean-only selective apply; Slice 13 B10/B11 normative verification; Slice 14 ST02 normative verification; Slice 15 ST03 normative verification. A feature cannot be enabled before its preflight, recovery and tests land.

**No new product functionality:** no S17, no new shortcut, no `a e`, no `gg`, no revived `H`, no generic stash manager, no auto-rebase/checkout/stash/force push, no arbitrary file checkout, no implicit state mutation on navigation. Routine `g e/p/f/P` must not acquire a generic extra Execute popup. `g r` only when rebase recovery is ready.

## Required response from Claude after each slice

1. Verified normative IDs/clauses and any unresolved contradiction.
2. Actual files changed and code diff summary.
3. Real automated tests run with pass/fail/blocked and exact Git fixture effects.
4. `tests/TRACEABILITY_V2_3.md` updates containing **real** module paths/evidence (never pretend placeholders passed).
5. What remains explicitly disabled and which next slice can begin.

**Do not rename, delete, or rewrite the authoritative v2.3 spec from this planning packet.**
