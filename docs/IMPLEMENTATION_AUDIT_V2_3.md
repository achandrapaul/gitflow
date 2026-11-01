# Go stub audit against GitFlow v2.3

Audit date: 30 October 2026. Authority: `spec/GITFLOW_REQUIREMENTS_V2_3.md`. Scope: all 29 Go files in the current working tree, module configuration, current acceptance/scenario/traceability documents, and supporting documentation. The Go source is currently untracked; this assessment concerns the working-tree contents, not an implementation recorded in a commit.

**Overall status: scaffolding only; not a functional GitFlow application and not ready for acceptance.** No end-to-end product workflow can be marked Passed. Type declarations, action names and screen identifiers establish intended structure, but do not implement the corresponding requirements.

Observed inventory: 1,922 Go source lines, 82 return statements containing `errNotImplemented`, 73 action declarations with no assigned handler, target predicate or availability predicate, and zero `_test.go` files. Many additional placeholders return an empty string, nil, false or an unchanged value without an error.

## Executed validation

| Check | Result | Meaning |
|---|---|---|
| Initial `go test ./...` | Environment failure | `go` was 1.27.3 while GOROOT pointed to mise's 1.27.1 installation. This is not a source compilation defect. |
| `go test ./...` with matching GOROOT | Exit 0; all six packages report `[no test files]` | Compilation succeeded; no product tests ran. |
| `go build` with matching GOROOT | Exit 0; binary produced | Source compiles. Build also emitted a nonfatal module-cache write-permission warning. |
| `go vet ./...` with matching GOROOT | Exit 0 | Static analyzer found no diagnostics; this does not validate behavior. |
| `gofmt -l cmd internal` | Empty output | Source formatting passes. |
| Launch built binary in ordinary repository | Exit 1: `start gitflow: app: not implemented` | No usable S01 frame or workflow. Reproduced in an isolated temporary repository. |
| Launch in repository whose directory ends in a space | Exit 1: `resolve git directory: exit status 128` | Additional concrete path-handling defect before app construction. |
| Ten targeted temporary audit tests | Nine fail, one passes | Confirms placeholder behavior; only a basic real-Git runner read succeeds. |
| Startup benchmark, Linux runtime, novice usability | Not executed | There is no usable frame to benchmark; no current evidence for these acceptance gates. |

Audit commands used command-local environment overrides; no global Go settings were changed:

```sh
env GOROOT=/opt/homebrew/Cellar/go/1.27.3/libexec \
  GOCACHE=/private/tmp/gitflow-audit-go-cache go test ./...
env GOROOT=/opt/homebrew/Cellar/go/1.27.3/libexec \
  GOCACHE=/private/tmp/gitflow-audit-go-cache go vet ./...
env GOROOT=/opt/homebrew/Cellar/go/1.27.3/libexec \
  GOCACHE=/private/tmp/gitflow-audit-go-cache \
  go build -o /private/tmp/gitflow-audit-bin ./cmd/gitflow
```

The extra audit tests were written only into `/private/tmp/gitflow-audit-oyf_8zm4`, a copy of the module. They are diagnostic probes, not completion of the specified integration suite. Temporary paths are session artifacts, not durable test infrastructure. No implementation files were modified, and mutating Git probes ran only in temporary repositories.

## Findings, ordered by impact

1. **Release blocker: the application cannot start.** `internal/app/app.go:90` always returns an error from `NewApp`. `Init` schedules no load, `Update` does nothing, `View` is empty, and navigation/dispatch/refresh/mutation/handoff methods are empty. Every concrete screen and overlay also has an empty renderer and no-op update. C01, K01–K08, N01–N03, U04–U07 and UX-N01–UX-N07 are unavailable, not merely unverified.

2. **Release blocker: all product Git operations are unimplemented.** Status reading, diffs, staging, commits, branches, history, remote synchronization, rebase, cherry-pick, conflicts, deletion and stash operations return `git: not implemented`. The working process runner cannot substitute for these domain semantics. See `internal/git/status.go:58`, `index.go:29`, `diff.go:48`, and the corresponding public methods in the other Git files.

3. **Safety blocker for future wiring: collision detection silently reports no collisions.** `internal/git/status.go:46` returns nil regardless of untracked paths and intended changes. A probe with the same path in both sets fails. This violates the required preflight behavior for C21/C24/ST02/ST03. No overwrite was observed because execution is currently stubbed; this becomes dangerous if mutation handlers are connected before the guard is implemented.

4. **Status and selection helpers silently return incorrect results.** `FileChange.Mixed` always returns false; `Status.Matches` excludes every path from every filter (`internal/git/status.go:22,42`). `orderedSelection.Toggle` returns the original selection, `Contains` returns false, and `Ordered` returns nil (`internal/app/history.go:17`). Mixed membership, explicit selection, hidden targets, range selection and typed queue persistence therefore do not work. Probes confirmed failures for a mixed file in all four filters and a single selection toggle.

5. **The action registry is declarations only.** `internal/app/registry.go:70` rejects construction; all lookups return no result. All 73 entries in `defaultActions` omit `Run`, `Target` and `Available`. Key precedence, contextual targeting, disabled reasons, single dispatch and menu/palette equivalence cannot function. Eight IDs appear in multiple declarations across scopes; the future implementation must deliberately unify these into shared actions/bindings instead of accidentally overwriting `byID` or rejecting legitimate aliases. Repeated IDs alone are not proof of an executed duplicate-handler bug.

6. **Both stash recovery workflows are entirely absent.** All ST03 preflight, execution, resume, restore, drift and rollback methods return errors (`internal/recovery/restore.go:54` onward). ST02 review/execute and recovery-state load/save/clear are also stubs. Capturing `StashOID`, `Baseline` and `Phase` fields does not establish exact-stash restoration, durable restart recovery, verified cleanup or safe rollback. T151–T170 cannot pass through the app.

7. **Repository discovery corrupts valid whitespace in paths.** `cmd/gitflow/main.go:33,39` calls `strings.TrimSpace` on Git's directory output. A temporary repository named `trailing-space ` fails directory discovery, while an ordinary name gets as far as the expected app-construction error. Remove only the output record terminator; preserve actual pathname bytes. N04/T26 require safe path handling.

8. **No acceptance tests or usable fixture implementation exist.** There are no `_test.go` files. `internal/gittest/fixture.go:21` and the write/commit/remote/dirty helpers call `t.Fatal(errNotImplemented)`. Creating a temporary directory is the only implemented fixture operation. A green package compilation must not be reported as verified feature coverage.

9. **Existing evidence documents do not describe this source.** `COVERAGE.md` claims numerous Verified results and names nonexistent tests/modules. `docs/STARTUP_BENCHMARK.md` names an absent `tools/startbench` and historical results that cannot validate this non-starting binary. Current v2.3 traceability correctly avoids Passed claims, but its “source not supplied” wording is now stale because untracked stubs exist. Historical evidence must not be carried forward as current acceptance evidence.

## Requirement status by area

“Not implemented” below includes declared-but-empty functions. “Partial foundation” means some supporting code exists, without satisfying the entire criterion. This table covers the active stable-ID families; K29 remains reserved and inactive.

| Requirements | Required behavior | Current status and evidence |
|---|---|---|
| K01–K09 | Inspect/select/back, input trapping, single dispatch, navigation continuity, scoped select-all | Not implemented. Empty event loop and screen updates; broken selection helpers. |
| K30–K35, K42–K47 | Current global/contextual keys, branch/upstream prompts, composer focus and view toggle | Key metadata mostly present; execution, scope validation and overlays absent. Reserved-key map exists but is not enforced. |
| K36–K38 | Focus vs checked state, stable range selection, hidden counts and exact targets | Not implemented. One selection container and range-anchor field do not implement the required membership rules or review. |
| K39–K41 | Dirty read-only explorer entry, separate execution domains, refresh/focus feedback | Not implemented. No active-mode transition, source handoff, execution guard or reconciliation. |
| C01–C07, C12, C19 | Unified native status and accurate filters/diffs | Not implemented; mixed/filter predicates demonstrably wrong. |
| C08 | Safe file/hunk/supported-line staging | Not implemented. Index and patch methods return errors; no real patch applicability or preservation tests. |
| C09, C13, C17, C18, C20 | Stable explicit/hidden selection and exact action targeting | Not implemented; selection toggle probe fails. No independent Space/range membership or stale-target behavior. |
| C10, C11, C22, C23 | Single staged-index commit review, hook integrity, draft retention, one-shot bypass | Request/result fields exist; commit and composer behavior absent. No actual tree/message verification or hook fixtures. |
| C14 | Confirmed safe single-file unlink without staging | Identity and deleter APIs are stubs. No filesystem validation, confirmation or unlink implementation. |
| C15, C16 | Flat/folder identity equivalence and atomic global preference persistence | Layout enum/field exist; rendering, toggle, load and save absent. |
| C21, H04, H05 | Read-only cherry-pick explorer, ordered native commits and truthful recovery | Queue/outcome types exist; browsing, selection, mainline handling, execution and recovery absent. |
| C24 | Apply selected parent-to-commit patches to index/worktree, preserve draft and explain partial outcomes | Patch/selection/report types exist; all operations and composer handoff absent. |
| B01–B07 | Real branch intelligence, cached freshness, separate base, explicit safe switching | All branch readers/mutations and branch screen are stubs. No fabricated metadata is rendered only because nothing is rendered. |
| B08, B09, K44, K45 | Create without checkout/upstream, explicit switch, first push and tracking | Names/fields/prompts present; no validation, creation, switching, destination review or tracking verification. |
| B10, B11, K47 | Local-main FF/direct choice, post-create switch and dirty recovery | Prompt types and API names exist; required native behavior and safeguards absent. |
| H01–H03 | Stable paginated DAG history, virtualized viewport and preserved OID selections | Cursor, page size 100 and summary types exist; every pager method returns an error. Offset field alone neither proves nor disproves eventual correct pagination. |
| H06–H08 | Explicit rebase range, todo validation/reorder, publication warning and UI-only plan cancellation | Todo enum and planner fields exist; validation, execution and navigation absent. |
| O01–O10 | Native paused-operation detection, guarded recovery and truthful conflict side mapping | Operation enum and conflict data types exist; detector/controllers/resolver/labels are unimplemented. `Operation.Active` only checks an already populated enum. |
| ST01 | Real read-only stash list | List API and screen exist as stubs; no metadata reading or rendering. |
| ST02 | Explicit tracked-only stash-and-switch, never automatic restore | All workflow and native operations absent. |
| ST03 | Clean direct execution; authorized dirty stash; post-success restore; safe conflict rollback | All workflow methods and persistence absent; no validated baseline/provenance or restart tests. |
| Global Git operations / K30 | Configured pull, rebase/FF-only variants, push/first push, strict force-with-lease, fetch | Enums/action metadata exist; remote methods return errors. No native operation or lease validation. |
| U01–U07, UX-N01–UX-N07 | Learnability, orientation, accurate feedback, one draft, continuous workspace | Not implemented and usability not measured. Empty views cannot establish these criteria. |
| N01 | 100 ms first interactive frame, documented cold/warm p50/p95 | Not measurable for this app; startup fails. Historical benchmark is not current evidence. |
| N02, N03 | Responsive I/O, native hooks/signing/credential/editor handoff | Partial process support only; app mutation scheduling and terminal handoff are no-ops. No responsiveness or interactive-process verification. |
| N04 | Safe argv/cwd, paths and diagnostics | Partial foundation: runner uses `exec.CommandContext`, explicit cwd and separate stderr. Basic native read probe passes. Path discovery fails on trailing whitespace; N04 as a whole is not passed. |
| N05 | macOS and Linux validation | macOS compilation/vet checked and startup failure reproduced. No Linux runtime test. |
| N06 | Separation of UI and Git workflows, no daemon | Structural foundation present: separate app/git/recovery/prefs packages, no daemon implementation. Runtime separation/serialization still unverified. |
| N07 | Approved scope complete; no opportunistic features | Required product scope incomplete. No implemented excluded product features found; an unused enum is not evidence of a new user-facing feature. |

## Scaffold alignment concerns to resolve before filling methods

These are design gaps or stale declarations, not claims that an unreachable UI already performs an unsafe operation.

| Concern | Source | Required adjustment |
|---|---|---|
| S15 still modeled as routine remote-sync review/execute | `internal/app/branch.go:26`; `app.go:31`; routine remote actions have `ReviewPlan` | Implement the v2.3 compact global menu without mandatory nested routine operation-choice popups. Preserve only the specified contextual/guarded reviews. |
| Conflict action titles say “Choose ours/theirs” | `internal/app/actions.go:142` | Present actual branch/ref or OID plus operation role, with a valid separately identified base. Internal side enums are fine; bare user-facing side names are not O10-compliant. |
| Stash-and-proceed `s` declared for general cherry-pick scope | `internal/app/actions.go:118` | Restrict it to the explicit dirty-execution prompt. No normal explorer shortcut may silently authorize stashing. |
| Amend has a struct flag but no registered action | `internal/app/changes.go:59`; `actions.go` | Add the approved distinct guarded contextual action through the shared registry; do not invent a shortcut. |
| Single draft ownership and branch-header focus not represented in working behavior | `internal/app/changes.go:13`; `app.go:73` | Establish one durable draft shared by S01/S03 and keyboard focus for branch header/files/diff/composer. Current changesFocus only names files/composer. |
| Dual-mode explorer state incomplete | `internal/app/planners.go:10`; `history.go:48` | Preserve source selector, fixed target, independent commit/file sets, active mode, queue and return context. Merely placing selections on separate screens does not establish continuity. |
| File-selection identity omits change type | `internal/git/diff.go:16` | Carry or reliably resolve the full required OID/parent/path/change-type identity, including rename endpoints and refreshed eligibility. |
| Recovery provenance not yet designed in executable code | `internal/recovery/store.go:15`; `restore.go:74` | Prove which edits restoration introduced, detect intervening changes and refuse rollback when attribution is uncertain. Baseline fields alone are not a safe rollback implementation. |
| Runner is not a complete process safety layer | `internal/git/runner.go:24,63`; `internal/app/app.go:114` | Implement mutation serialization and terminal handoff, and avoid exposing credentials from argv/stderr in diagnostics. Raw errors currently retain/format argv and stderr without redaction; no actual credential disclosure was tested or observed. |

## Targeted probe outcomes

| Probe | Result |
|---|---|
| Mixed path reports Mixed | Failed: always false. |
| Mixed path belongs to All/Staged/Not staged/Mixed | Failed: absent from all four. |
| Exact untracked destination collision reported | Failed: nil result. |
| Status snapshot from real temporary Git repository | Failed: not implemented. |
| Runner initializes temporary Git repo and reads native worktree status flag | Passed; limited runner evidence only. |
| Construct app | Failed: not implemented. |
| Render first app frame | Failed: empty string. |
| Toggle file selection | Failed: selected set remains empty. |
| Construct default registry | Failed: not implemented. |
| All declared actions have handlers/predicates | Failed: all 73 lack wiring. |

## Documentation conflicts affecting future verification

`tests/SCENARIOS_V2_3.md:66` T89 still says staged dirt disables whole-commit execution until clean. ST03 requires the explicit stash-and-proceed choice for staged and/or unstaged dirt; T89 needs correction before it becomes an executable test. The normative specification takes precedence.

The older screen specs, walkthroughs, coverage matrix, manifest and open questions contain superseded keys, Mixed-filter behavior and S15/conflict semantics. They must not override v2.3. In particular, historic claims involving `c c`, `c p`, `r i`, routine S15 sync review, bare ours/theirs choices, or exclusion of Mixed from Not staged are unsuitable implementation guidance.

## Recommended implementation order

1. Build real temporary-repository fixtures and status/path/safety tests; replace silent successful-looking placeholders before connecting mutations.
2. Make S01 start, implement action registry/input precedence/navigation and real asynchronous status loading.
3. Complete the core inspect → select → stage → review → commit flow, including hook verification and draft persistence.
4. Add branches/upstream/remote operations and history, then the independent cherry-pick/file-apply modes and native rebase/conflict recovery.
5. Implement ST02/ST03 with durable identity, restart reconciliation, conflict provenance and refusal of unsafe rollback; run the real T151–T170 fixtures.
6. Update current traceability from actual results; measure startup, small-terminal behavior, Linux runtime and novice usability. Retire stale evidence claims.

No product requirement should be marked Passed solely because its type, method or shortcut appears in this stub.
