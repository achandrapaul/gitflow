# Requirement coverage matrix

Status per `tests/SCENARIOS.md` "Coverage reporting format": **Verified** = automated test against real Git passes on macOS · **Implemented** = code exists, no dedicated automated test · **Blocked** / **Deferred** as noted. Linux is **pending** for every row (cross-compile and `go vet` for linux/amd64 pass; no tests were run on Linux).

Build and test: `go build -o bin/gitflow ./cmd/gitflow` · `go test ./...` · `go test -race ./...` · `go run ./tools/startbench -bin bin/gitflow`.

Modules: `internal/git` (Git data and commands), `internal/app` (TUI), `cmd/gitflow` (binary), `tools/startbench` (benchmark). Tests: `internal/git/*_test.go`, `internal/app/app_test.go`.

## Acceptance criteria (`tests/ACCEPTANCE.md`)

| ID | Module | Automated test | Status | Evidence / notes |
|---|---|---|---|---|
| K01 | app/changes.go, diffview.go, history.go | TestNavigationNeverMutatesGit | Verified | Snapshot of status, index, refs, sequencer and worktree bytes unchanged |
| K02 | app/changes.go, diffview.go, history.go, planners.go | TestNavigationNeverMutatesGit, TestStageShortcutAndMenuShareAction | Verified | Space toggles UI set only |
| K03 | app/app.go (`back`, overlays) | TestNavigationNeverMutatesGit, TestPausedCherryPickEscAbortAndQuit, TestRebaseEditStopAndPublishedWarning, TestDiscardRequiresConfirmation | Verified | Esc never reaches a Git handler |
| K04 | app/registry.go, actions.go | TestRegistryInvariants, TestStageShortcutAndMenuShareAction, TestCherryPickPlannerNoncontiguousOrder | Verified | Shortcut lookup returns the same `*Action` as the menu |
| K05 | app/overlays.go | TestMergeConflictResolveAndContinue, TestRebasePlannerValidationAndExecution, TestBranchInspectorActions | Verified (menu) / Implemented (`?` help) | Disabled rows show "unavailable: reason" |
| K06 | app/composer.go, overlays.go | TestComposerTypingAndCommit, TestNavigationNeverMutatesGit | Verified | Typing `s u a 1 c : q d A ?` in composer and palette |
| K07 | app/registry.go, app.go | TestRegistryInvariants (+ `c c`, `c p`, `r i`, `gg`, `g h`, `g l`, `g p`, `g f`, `c f` used across tests) | Verified | Duplicate key, prefix and navigation collisions rejected at startup; `S` reserved as global navigation |
| K08 | app/changes.go, history.go | TestNavigationNeverMutatesGit, TestHistoryPaginationAndSelectionPersistence | Verified (focus/filter/selection) / Implemented (scroll) | Screens are persistent objects |
| C01 | app/changes.go, app.go | TestStartupCleanRepository | Verified | |
| C02 | git/status.go | TestStatusFiltersAgainstRealRepository, TestParseStatusRecords | Verified | |
| C03 | git/status.go | same | Verified | |
| C04 | git/status.go | same | Verified | |
| C05 | git/status.go | same | Verified | |
| C06 | git/status.go, app/changes.go | TestParseStatusRecords | Verified | XY from `--porcelain=v2 -z` |
| C07 | app/diffview.go, changes.go | TestNavigationNeverMutatesGit, TestIndexOperationsOnRealRepository/T04 | Verified | |
| C08 | git/diff.go, app/actions.go | TestIndexOperationsOnRealRepository (T04, line staging, T06, T07), TestParseDiffAndBuildPatch, TestStageHunkFromDiffInspector | Verified | Patch revalidated against fresh diff + `git apply --cached --check` |
| C09 | app/changes.go | TestNavigationNeverMutatesGit, TestSelectAllVisibleFiles | Verified | Hidden count + menu target scope; global `S` selects all listed files (UI only) in S01 and S13 |
| C10 | git/history.go (CommitInvocation), app/composer.go | TestCommitOnlyIndex/T09, TestComposerTypingAndCommit, TestCommitOnlyIndex/T10 | Verified (index-only, hooks) / Implemented (signing) | Fixture disables signing; real runs use terminal handoff |
| C11 | app/composer.go | TestStartupCleanRepository, TestCommitHookFailureKeepsDraft | Verified | |
| C12 | git/status.go, diff.go, conflict.go | TestParseStatusRecords, TestPartialSupportReasons, TestIndexOperationsOnRealRepository/T26, TestStatusFiltersAgainstRealRepository | Verified (untracked, binary, rename, conflict) / Implemented (delete-only path) | |
| B01 | git/status.go, app/app.go | TestBranchIntelligence/T13, TestStartupBranchIntelligence | Verified | |
| B02 | git/status.go, app/app.go | TestBranchIntelligence/T11, TestStartupCleanRepository | Verified | "No upstream", never ↑0 ↓0 |
| B03 | git/branch.go | TestBranchIntelligence/T12, TestStartupBranchIntelligence | Verified | |
| B04 | git/branch.go, app/app.go | TestBranchIntelligence/T13, TestStartupCleanRepository | Verified | Freshness text is fixed "fetch time unknown" |
| B05 | app/app.go (`refresh`) | TestBranchInspectorActions (Fetch updates counts) | Verified | All loads asynchronous |
| B06 | app/app.go (`refresh`) | — | Implemented | Refresh runs only status, rev-parse, rev-list, operation detection; fetch only from S11 action |
| H01 | git/history.go, app/history.go | TestHistoryPaginationAndSelectionPersistence | Verified (100/page, prefetch) / Implemented (viewport virtualization) | |
| H02 | git/history.go | TestHistoryCursorOnMergeHeavyGraph | Verified | 1200 commits, 3 branches, merges; equals `rev-list --topo-order` |
| H03 | app/history.go | TestHistoryPaginationAndSelectionPersistence | Verified | |
| H04 | app/planners.go, git/planner.go | TestCherryPickPlannerNoncontiguousOrder, TestCherryPickScenarios/T15 | Verified | Selected on `side`, applied to `main` |
| H05 | app/planners.go, git/planner.go | TestCherryPickPlannerNoncontiguousOrder, TestCherryPickConflictRoutesToDashboard, TestPlannerInvalidatedWhenHeadMoves, TestCherryPickScenarios (T27, mainline) | Verified | |
| H06 | app/planners.go, git/planner.go | TestRebaseScenarios/T18, TestRebasePlannerValidationAndExecution, TestRebaseEditStopAndPublishedWarning | Verified (pick/fixup/edit/drop/reorder) / Implemented (reword/squash: Git editor) | |
| H07 | git/planner.go, app/planners.go | TestRebaseScenarios/T18 invalid, TestRebasePlannerValidationAndExecution, TestRebaseEditStopAndPublishedWarning | Verified | |
| H08 | app/planners.go | TestCherryPickPlannerNoncontiguousOrder | Verified | |
| O01 | git/operation.go | TestCherryPickScenarios/T16_T17, TestPausedCherryPickEscAbortAndQuit (reopen) | Verified | |
| O02 | app/actions.go (`global.operation`) | TestPausedCherryPickEscAbortAndQuit | Verified | |
| O03 | app/actions.go (`op.continue`) | TestMergeConflictResolveAndContinue, TestRebaseEditStopAndPublishedWarning, TestPausedCherryPickEscAbortAndQuit (disabled) | Verified | |
| O04 | git/operation.go, app/actions.go | TestMergeAndConflictResolution/T22, TestCherryPickScenarios/T27, TestMergeConflictResolveAndContinue | Verified (Git, merge UI) / Implemented (cherry-pick skip UI path) | |
| O05 | app/operations.go | TestPausedCherryPickEscAbortAndQuit, TestRebaseEditStopAndPublishedWarning | Verified | Abort result re-detected |
| O06 | app/operations.go, git/conflict.go | TestMergeConflictResolveAndContinue, TestMergeAndConflictResolution/T23 | Verified | |
| O07 | git/conflict.go, app/operations.go | same | Verified | |
| O08 | app/app.go, planners.go | TestPlannerInvalidatedWhenHeadMoves, TestCommitHookFailureKeepsDraft | Verified | |
| O09 | app/app.go (`openQuit`) | TestPausedCherryPickEscAbortAndQuit | Verified | |
| N01 | tools/startbench | — (benchmark) | Verified (macOS) | `docs/STARTUP_BENCHMARK.md`: warm p50 52.5 ms, p95 55.6 ms |
| N02 | app/app.go (`load`) | — | Implemented | Not measured under slow I/O |
| N03 | git/runner.go, app/app.go (`handoff`) | TestCommitOnlyIndex/T10, TestCommitHookFailureKeepsDraft | Verified (hooks) / Implemented (signing, SSH, credential helpers) | |
| N04 | git/runner.go | TestIndexOperationsOnRealRepository/T26 | Verified | See OPEN_QUESTIONS Q18 for the two shell-evaluated strings |
| N05 | — | — | macOS Verified / Linux **pending** | |
| N06 | internal/git vs internal/app | — | Implemented | No background service |
| N07 | — | — | Implemented | Deviations needing a decision listed in OPEN_QUESTIONS Q18–Q26 |

## Scenarios (`tests/SCENARIOS.md`)

| ID | Test | Status |
|---|---|---|
| T01 | TestStartupCleanRepository | Verified |
| T02, T03, T05 | TestStatusFiltersAgainstRealRepository | Verified |
| T04 | TestIndexOperationsOnRealRepository/T04, TestStageHunkFromDiffInspector | Verified |
| T06 | TestIndexOperationsOnRealRepository/T06 | Verified |
| T07 | TestIndexOperationsOnRealRepository/T07 | Verified |
| T08 | TestNavigationNeverMutatesGit | Verified |
| T09 | TestCommitOnlyIndex/T09, TestComposerTypingAndCommit | Verified |
| T10 | TestCommitOnlyIndex/T10, TestCommitHookFailureKeepsDraft | Verified |
| T11 | TestBranchIntelligence/T11 | Verified |
| T12 | TestBranchIntelligence/T12 | Verified |
| T13 | TestBranchIntelligence/T13, TestStartupCleanRepository | Verified |
| T14 | TestHistoryCursorOnMergeHeavyGraph, TestHistoryPaginationAndSelectionPersistence | Verified |
| T15 | TestCherryPickScenarios/T15, TestCherryPickPlannerNoncontiguousOrder | Verified |
| T16 | TestCherryPickScenarios/T16_T17, TestPausedCherryPickEscAbortAndQuit | Verified |
| T17 | TestPausedCherryPickEscAbortAndQuit (reopen) | Verified |
| T18 | TestRebaseScenarios/T18 (both), TestRebasePlannerValidationAndExecution | Verified |
| T19 | TestRebaseScenarios/T19 (both), TestRebaseEditStopAndPublishedWarning | Verified |
| T20 | TestRebaseEditStopAndPublishedWarning | Verified |
| T21 | TestRebaseScenarios/T19_T21 | Verified (Git layer; UI abort path verified with cherry-pick) |
| T22 | TestMergeAndConflictResolution/T22, TestMergeConflictResolveAndContinue | Verified |
| T23 | TestMergeAndConflictResolution/T23 (ours, theirs, both) | Verified |
| T24 | TestMergeAndConflictResolution/T24 | Verified |
| T25 | TestPlannerInvalidatedWhenHeadMoves | Verified |
| T26 | TestIndexOperationsOnRealRepository/T26 | Verified |
| T27 | TestCherryPickScenarios/T27 | Verified |
| T28 | TestPausedCherryPickEscAbortAndQuit | Verified |
| T29 | TestComposerTypingAndCommit | Verified |
| T30 | tools/startbench, docs/STARTUP_BENCHMARK.md | Verified (macOS) / Linux pending |

## v1.1 acceptance criteria (`spec/05_INTELLIJ_STYLE_GIT_WORKSPACE.md`)

| ID | Module | Automated test | Status | Evidence / notes |
|---|---|---|---|---|
| A01 | app/actions.go (`sync.*`), app/changes.go (header menu), app/branch.go (S11 menu) | TestRemoteSyncReview, TestBranchInspectorActions | Verified | `g l` and header `a` > Pull render the identical S15 screen; S11 Fetch/Pull/Push open the same S15 |
| A02 | app/branch.go (`syncScreen.view`) | TestRemoteSyncReview | Verified | Branch, remote, upstream, command, "from local refs; fetch freshness unknown", base labelled "independent metric, not the pull/push target" |
| A03 | git/branch.go (`SyncInvocation`) | TestRemoteSync/31, /32, TestRemoteSyncReview | Verified | `git pull --ff-only`; diverged pull refused, HEAD/index/worktree/local branches unchanged |
| A04 | git/branch.go | TestRemoteSync/33, /34, TestRemoteSyncReview | Verified | `git push <remote> HEAD:<upstream ref>`; no `-f`/`+`; non-FF leaves remote ref unchanged |
| A05 | app/branch.go (`executeUnavailable`), git/branch.go (`ErrNoUpstream`) | TestRemoteSync/35, TestRemoteSyncWithoutUpstream, TestBranchInspectorActions | Verified | Execute disabled with Set upstream guidance; repository byte-for-byte unchanged |
| A06 | app/history.go (Enter → S13) | TestCommitFileInspectorSelection | Verified | Whole-commit selection kept across S13 visits |
| A07 | app/history.go (`commitFilesScreen`, `orderedSelection[fileKey]`) | TestCommitFileInspectorSelection | Verified | "N selected in this commit" and S04 "Files: N selected in M commit(s)" persist |
| A08 | git/diff.go, app/planners.go | TestSelectiveApply/36, /37, TestSelectiveApplyFromInspector | Verified | Only selected paths change; staged; HEAD unchanged |
| A09 | git/diff.go | TestSelectiveApply/40, /gitlink | Verified | Add, delete, rename, binary, mode, symlink, typechange, unicode/space paths applied and checked; gitlink rejected |
| A10 | app/history.go (parent chooser), git/diff.go | TestMergeCommitRequiresParentChoice, TestSelectiveApply/41 | Verified | Merge commits require an explicit parent; files computed against that parent only |
| A11 | git/diff.go (`PreflightPatches`, `ApplyPatchSteps`), app/planners.go | TestSelectiveApply/38, /39, TestSelectiveApplyAcrossCommitsFailsClosed | Verified | Numbered order; preflight failure applies nothing; mid-run failure reports applied / failed / not attempted steps |
| A12 | app/history.go, planners.go, branch.go | TestCommitFileInspectorSelection, TestSelectiveApplyFromInspector, TestRemoteSyncReview, TestRemoteSyncWithoutUpstream | Verified | Snapshot (status, index, refs, operation state, worktree bytes) unchanged after Esc/Cancel |
| A13 | git/diff.go (`ValidateSelectiveApply`), app/planners.go | TestSelectiveApplyBlockedDuringOperation, TestSelectiveApplyAcrossCommitsFailsClosed | Verified | Paused cherry-pick blocks apply with its operation named; failed apply leaves "No active operation" |
| A14 | — | — | Implemented | Only S13–S15 and the documented keys added; new decisions in OPEN_QUESTIONS Q21–Q26 |

## v1.1 selective-apply decisions (user-mandated acceptance criteria)

| ID | Criterion | Module | Automated test | Status |
|---|---|---|---|---|
| U25 | Clean index and tracked worktree required; untracked allowed unless colliding with any affected path; operation in progress blocks; nothing stashed/cleaned | git/diff.go `ValidateSelectiveApply` | TestSelectiveApply/prerequisites, TestSelectiveApplyBlockedDuringOperation | Verified |
| U24.1 | Patch metadata preserved (rename source/destination, modes, binary) | git/diff.go `CommitFilePatch` (`--binary --full-index -M`, both rename paths, bytes passed unmodified) | TestSelectiveApply/40 | Verified |
| U24.2 | Every patch validated with `git apply --check` before application | git/diff.go `checkAndApply` (temp-index `--cached --check` for the whole queue, then `--index --check` per step) | TestSelectiveApply/38, /39 | Verified |
| U24.3 | Every supported category tested against real repositories | internal/git/diff_test.go | TestSelectiveApply/36–41 | Verified |
| U24.4 | Unsupported or ambiguous cases disabled with an actionable reason | git/diff.go `UnsupportedReason`, app/history.go parent chooser | TestSelectiveApply/gitlink, TestMergeCommitRequiresParentChoice | Verified |
| U24.5 | No silent omission or partial application of unsupported types | git/diff.go `ValidateSelectiveApply` (whole plan blocked) | TestSelectiveApply/gitlink | Verified |
| U24.6 | Submodule/gitlink changes rejected | git/diff.go | TestSelectiveApply/gitlink | Verified |
| U24.7 | No automatic commit; results staged | git/diff.go `ApplyPatchSteps` (`--index`) | TestSelectiveApply/36, TestSelectiveApplyFromInspector | Verified |
| U24.8 | Whole-commit cherry-pick kept separate | app/history.go (`orderedSelection[string]` vs `[fileKey]`) | TestCommitFileInspectorSelection (`c p` receives commits only) | Verified |
| U24.9 | Revalidation immediately before mutation; result re-read | app/planners.go (status, HEAD, operation, prerequisites, preflight inside the mutation) | TestSelectiveApplyAcrossCommitsFailsClosed | Verified |
| U24.10 | Keyboard conventions, review screens, safety kept | app/* | TestRegistryInvariants, A12 tests | Verified |

## v1.1 fixtures (`tests/SCENARIOS.md` 31–44)

| ID | Test | Status |
|---|---|---|
| 31 | TestRemoteSync/31, TestRemoteSyncReview | Verified |
| 32 | TestRemoteSync/32 | Verified |
| 33 | TestRemoteSync/33, TestRemoteSyncReview | Verified |
| 34 | TestRemoteSync/34 | Verified |
| 35 | TestRemoteSync/35, TestRemoteSyncWithoutUpstream | Verified |
| 36 | TestSelectiveApply/36 | Verified |
| 37 | TestSelectiveApply/37, TestSelectiveApplyFromInspector | Verified |
| 38 | TestSelectiveApply/38 | Verified |
| 39 | TestSelectiveApply/39, TestSelectiveApplyAcrossCommitsFailsClosed | Verified |
| 40 | TestSelectiveApply/40, /gitlink | Verified |
| 41 | TestSelectiveApply/41, TestMergeCommitRequiresParentChoice | Verified |
| 42 | TestCommitFileInspectorSelection, TestSelectiveApplyFromInspector, TestRemoteSyncReview | Verified |
| 43 | TestRemoteSyncReview, TestRemoteSync (upstream config) | Verified |
| 44 | TestCommitFileInspectorSelection (inspector round trips) | Verified (navigation) / Implemented (pagination: selection is held by the app, independent of history pages) |
