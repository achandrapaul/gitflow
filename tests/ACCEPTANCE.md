# Acceptance criteria — all P0 unless noted

## Keyboard and navigation
- [ ] K01 Enter opens item; never stages/commits/runs an operation on a repository row.
- [ ] K02 Space toggles UI selection; no Git mutation.
- [ ] K03 Esc dismisses one UI layer/back; never invokes Git abort, reset, discard.
- [ ] K04 Menus and shortcuts dispatch identical action registry ID and validation.
- [ ] K05 Context help and action menus show enabled/disabled reasons.
- [ ] K06 Text editor/input never fires normal navigation or Git mutation shortcuts.
- [ ] K07 Prefix keys and modal trapping work; no double invocation.
- [ ] K08 Focus/filter/selection/scroll restore on navigating back where item still exists.

## Changes and commit
- [ ] C01 S01 default shows branch metadata, unified changed files, diff, composer.
- [ ] C02 All contains each changed/untracked file once.
- [ ] C03 Staged contains index-changed files INCLUDING mixed.
- [ ] C04 Not staged contains files with no index changes and worktree/untracked changes, EXCLUDES mixed.
- [ ] C05 Mixed iff both index and worktree changed.
- [ ] C06 File row has accurate INDEX and WORKTREE Git XY statuses, not mutable UI booleans.
- [ ] C07 Mixed diff has distinct HEAD→INDEX and INDEX→WORKTREE sections.
- [ ] C08 File, hunk and supported line staging preserve untouched content; patch errors safe.
- [ ] C09 UI selection survives filters; hidden count and target scope accurate.
- [ ] C10 Commit includes ONLY real staged/index changes and respects hooks/signing.
- [ ] C11 Empty-stage commit disabled; failed commit keeps message and true status.
- [ ] C12 Untracked, binary, rename, delete, merge-conflict paths handled safely.

## Branch status
- [ ] B01 Current branch + HEAD short OID shown, detached HEAD is explicit.
- [ ] B02 Upstream and ahead/behind correct; no upstream means unknown counts, never 0/0.
- [ ] B03 Separate base comparison correct; do not mix upstream with main.
- [ ] B04 Base missing shows unavailable; fetched timestamp never invented.
- [ ] B05 Counts refresh after relevant Git ref changes; status UI nonblocking.
- [ ] B06 No fetch is initiated by dashboard painting.

## History and planners
- [ ] H01 Initial history batch bounded (100 default), incremental pagination and virtualized viewport.
- [ ] H02 No duplicate/missing history boundary rows on merge-heavy fixture.
- [ ] H03 Commit selections persist through page load, filters, inspector return.
- [ ] H04 Cherry-pick accepts noncontiguous cross-branch selected OIDs; queue reorder explicit.
- [ ] H05 Cherry-pick preview/validation/execute/recovery truthful; merge commits require mainline.
- [ ] H06 Rebase scope/base explicit, `pick/reword/edit/squash/fixup/drop` and reorder work.
- [ ] H07 Invalid first squash/fixup plan blocked; published-history risk warned.
- [ ] H08 Draft planner Esc discards only unexecuted plan after UI-only confirmation, never Git.

## Paused operations / conflicts
- [ ] O01 Detect active rebase/cherry-pick/merge after app restart from native Git state.
- [ ] O02 `o` reopens dashboard without issuing `--continue`.
- [ ] O03 `c` runs operation-specific continue only with validated prerequisites.
- [ ] O04 Skip is available for rebase/cherry-pick, unavailable for merge; confirmation required.
- [ ] O05 Abort requires distinct confirmation, Cancel default, Esc cancels, success checked.
- [ ] O06 Conflict file Enter opens resolver; 1/2/3 choose side for valid text case.
- [ ] O07 Resolution requires separate mark-resolved/stage; no automatic continue.
- [ ] O08 Errors and external Git mutation cause rescan, never fabricated completion.
- [ ] O09 Leaving TUI with paused operation leaves Git operation intact.

## Non-functional
- [ ] N01 100ms first usable frame target measured/report p50/p95 cold/warm + machine specs.
- [ ] N02 UI navigable during slower Git I/O.
- [ ] N03 Native Git config/hooks/signing/SSH helpers respected.
- [ ] N04 Git argv execution and paths safe, no shell-interpolated arbitrary values.
- [ ] N05 macOS and Linux tested; untested platform explicitly marked pending.
- [ ] N06 UI and Git workflow layers are separated; no required background service.
- [ ] N07 P0 scope complete; out-of-scope features not introduced.


## v1.1 additional mandatory acceptance criteria
Implement A01–A14 from `spec/05_INTELLIJ_STYLE_GIT_WORKSPACE.md` as automated test cases wherever Git backend semantics are involved.
