# GitFlow — v2.3 Acceptance Criteria (current only)

These are checks against `spec/GITFLOW_REQUIREMENTS_V2_3.md`. The checklist contains only current shortcut and filter behaviors. All checked states require actual test evidence.

## Keyboard and navigation
- [ ] K01 Enter opens item; never stages/commits/runs an operation on a repository row.
- [ ] K02 Space toggles UI selection; no Git mutation.
- [ ] K03 Esc dismisses one UI layer/back; never invokes Git abort, reset, discard.
- [ ] K04 Menus and shortcuts dispatch identical action registry ID and validation.
- [ ] K05 Context help and action menus show enabled/disabled reasons.
- [ ] K06 Text editor/input never fires normal navigation or Git mutation shortcuts.
- [ ] K07 Prefix keys and modal trapping work; no double invocation.
- [ ] K08 Focus/filter/selection/scroll restore on navigating back where item still exists.
- [ ] K09 `Shift+S` in S01 Changes list selects every changed/untracked file (including filter-hidden paths) without Git mutation; repeated invocation is idempotent and does not trigger in editors, modals, or other workspaces.

## Changes and commit
- [ ] C01 S01 default shows branch metadata, unified changed files, diff, composer.
- [ ] C02 All contains each changed/untracked file once.
- [ ] C03 Staged contains index-changed files INCLUDING mixed.
- [ ] C04 Not staged contains every file with unstaged worktree changes, INCLUDING mixed, plus untracked files.
- [ ] C05 Mixed iff both index and worktree changed.
- [ ] C06 File row has accurate INDEX and WORKTREE Git XY statuses, not mutable UI booleans.
- [ ] C07 Mixed diff has distinct HEAD→INDEX and INDEX→WORKTREE sections.
- [ ] C08 File, hunk and supported line staging preserve untouched content; patch errors safe.
- [ ] C09 UI selection survives filters; hidden count and target scope accurate.
- [ ] C10 Normal Commit uses actual reviewed staged index, honors Git hooks/signing, and verifies actual committed content; optional C23 `a v` bypass is separately warned and still honors signing and safety.
- [ ] C11 Empty-stage commit disabled; failed commit keeps message and true status, including a hook-modified index, and requires new review if changed.
- [ ] C12 Untracked, binary, rename, delete, merge-conflict paths handled safely.
- [ ] C13 Select-all count and hidden-selected count reflect complete unified Changes file set; later Stage/Unstage remains a separate explicit action with stale-state validation.

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
- [ ] O06 Conflict file Enter opens resolver; 1/2/3 choose the correctly labeled actual native side or available base for supported text cases; no invented Both/base equivalence.
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



## Additional current file and branch acceptance
- [ ] **C14** Deleting one eligible file removes only the reviewed on-disk path, never auto-stages; Cancel leaves state unchanged; multi-file/folder/unmerged/submodule/stale targets blocked.
- [ ] **C15** Flat and By folder contain identical unique changed-file identities with distinct staged/unstaged diffs; folder nodes are not file-action targets.
- [ ] **C16** Last explicitly selected display mode persists globally per user across restarts/repositories, and never changes automatically; failures are reported.
- [ ] **C17** Collapsed/filtered changed files remain eligible for Shift+S and retain correct selected/hidden counts.
- [ ] **C18** `j/k` changes only focus with or without existing checked files; Space changes only focused membership; Shift+J/K changes deliberate range while preserving independent checked members; Esc clears explicit selections in S01 (overlay-first) without Git mutation.
- [ ] **B07** `g b` (S01 `b` alias) opens branch list, search and Enter inspection never checkout; `s` only validated local switch.
- [ ] **U01–U03** Contextual learning is task-first with no onboarding wizard, and the 4-of-5 five-minute novice target is measured rather than assumed.

## Consolidated current UX acceptance (v2.3)
- [ ] **U04** Across S01/S04/S06/S11/S08, screen/branch/focus, active selected domain/count/hidden count, and impending action target/consequence are evident without leaving the screen.
- [ ] **U05** Routine successes refresh native Git state and show concise non-blocking results; focus/scroll/filter/selection are preserved when valid. Blocked/partial errors identify impacted paths/refs and real safe next steps; no auto-alternative commands.
- [ ] **U06** `c` focuses the sole composer; S03 shares draft, preserves it on Esc/failure, refuses stale-index review, verifies committed tree/message after hooks, commits only explicitly and clears draft on verified success; distinct `a v` `--no-verify` action warns/requests confirmation once; Amend via `a` warns on publication.
- [ ] **U07** Native paused operation drives a persistent badge; `o` opens dashboard; Esc never continues/skips/aborts Git.
- [ ] **C19** Mixed (I=1,W=1) appears in Staged, Not staged, Mixed and All; same path only once in each view; untracked appears in Not staged.
- [ ] **C20** File focus is implicit action target ONLY when no checked selection exists; ordinary `j/k` never changes checked membership; Shift range is explicit and contractible; Space toggles only focused membership; Esc clears all S01 checked members and range anchor, hidden count accurate; UI selection never stages by itself.
- [ ] **C21** S06 browsing, selection and preview permit dirty tracked index/worktree. At whole-commit Execute, clean tracked state runs directly **with zero stash operations**; dirty tracked state offers ST03 `s` stash-and-proceed or Esc Cancel. Selective-file Execute still requires clean tracked state. Both modes reject unmerged/incompatible sequencer states and untracked collisions; source browsing never switches target checkout.
- [ ] **K30** `g p/r/f/P/F/e/c/h/b/n/s/u` map to exactly specified global actions; no nested operation-choice popups; `gg` unbound; `G` last loaded row preserved; case-sensitive labels and exact target display.
- [ ] **K31** Text inputs, overlay precedence and action validators prevent shortcut leakage; routine actions don't require additional generic confirmations; guarded force requires review + `c` and strict lease validation.
- [ ] **K32** `c` in S01 focuses the existing composer; Amend via contextual action, without collision with dashboard `c`.
- [ ] **K33** `g h` opens paginated S04; the old `H` binding is retired; selection and navigation return intact.
- [ ] **K34** S11 `s` explicitly requests validated local switch; Enter only inspects; remote refs cannot silently switch; S01 `s` stages only.
- [ ] **K35** S01 `v` directly toggles Flat/Folder and persists across restart/repo; S07 `v` still Preview; no intermediate selector.
- [ ] **K36/K37** S01 `j/k` focus-only (no selection changes), Space focused toggle, Shift+J/K anchored range, Shift+S hidden select-all, overlay-first Esc clear and checked-only mutation targeting all pass.
- [ ] **K38** Checked/hidden/active mode counts and mutation target displayed; no silent fallback on empty or invalid batch; the action target matches checked full path IDs.
- [ ] **K39** `g c` from S01, S04 and branch-specific commit lists preserves source/selected OIDs and allows dirty tracked read-only entry; whole-commit execution either verifies clean tracked state or follows explicitly authorized ST03; selective-file execution stays clean-only; current target never switches.
- [ ] **K40** S06 whole commits vs S13/S14 file selection use separate preserved typed sets; Enter commit enters file selection, Esc restores whole mode; `p` previews active mode, `a` exposes only correct Execute; whole-commit Execute runs native cherry-pick; file Execute stages local changes and no commit; never mixed execution.
- [ ] **K41** After staging or status refresh focus stays on valid path or nearest survivor, minimal footer and inline outcomes reflect real Git, paused indicator visible and actionable.

## Newly approved v2.3 correctness and feature acceptance
- [ ] **C22** Reviewed staged-tree OID and message are recorded; if hooks modify index before rejection, the changed real index is shown and review invalidated; if successful commit tree or final message differs, the actual committed content/message are disclosed without automatic undo/amend.
- [ ] **C23** Normal Commit executes hooks; a separate contextual `Commit without verification hooks` uses native `--no-verify` only after one-time explicit warning; it bypasses only supported hooks, does not persist, automatically retry, skip signing, or bypass staged-index/sequencer safety preflight.
- [ ] **C24** Applying selected historical file changes applies supported patches to BOTH local worktree and index, with staged status verified; no whole-commit cherry-pick or automatic commit occurs; composer prefill accurately identifies selected files/source commits and protects existing draft; partial failures disclosed.
- [ ] **O10** Conflict sides show truthful branch/ref + operation role or exact commit OID when no reliable branch exists; displayed options map correctly to native index sides in merge, rebase and cherry-pick; base only where supported.
- [ ] **C18/C20 UX check** `j/k` never changes selections, before or after selecting multiple files; `Space` changes only focused file, Shift+J/K extends/contracts explicit range without dropping unrelated checked members, Esc clears selected set and retains focus.
- [ ] **H05 recovery** Interrupted multi-commit queue reconstructs completed/active/pending items from native Git state after restart or explicitly reports an unreconcilable queue, with no duplicate application.
- [ ] **Coverage gates** Branch create/delete/upstream, amend, supported line staging and minimum-terminal navigation have focused integration fixtures in addition to broad acceptance checks.
- [ ] **Traceability** Each active stable requirement ID is mapped to at least one acceptance criterion and a concrete fixture in `tests/TRACEABILITY_V2_3.md`. K29 remains reserved/inactive; no IDs silently reused.

## v2.3 newly approved global/contextual shortcuts and branch/stash lifecycle
- [ ] **K42** `g` global and `a` contextual prefixes dispatch only their own approved scoped commands. `a e` is unbound; normal editor text, overlays and paused state take precedence; no implicit Git mutation at prefix entry.
- [ ] **K43** `a v` in commit context opens one-shot `git commit --no-verify` review/confirmation, never immediate commit; next normal commit runs hooks; ordinary `v` retains S01/S07 behavior.
- [ ] **K44** `g n` begins create-local-branch review from HEAD; `a n` begins from explicitly displayed branch context; neither auto-checkouts, pushes, sets upstream or changes remote state; successful create presents explicit `s`/Enter switch or Esc stay prompt.
- [ ] **K45** `g u` configures upstream only for current local branch; `a u` configures upstream only for contextual local branch. Both review exact target and do not auto-push/fetch/check out. On first normal `g P` without upstream, show confirmed `origin/<branch>` destination and use native `-u` with push; never preconfigure or claim success after failure.
- [ ] **K46** `g h` History replaces `H`; `g b` opens Branches (S01 `b` alias retained); `g s` opens read-only S16 Stash List. No collision with local `s`/`v` keys or `g` menu case sensitivity.
- [ ] **B08** Local branch creation checks name/start ref, creates only the local ref, and leaves current checkout/upstream unchanged **until** user explicitly switches using `s`/Enter on post-create prompt; `Esc` does not delete created branch.
- [ ] **B09** First push on no-upstream local branch reviews exact `origin/<branch>` and remote-ref status (including unknown), uses `git push -u origin <branch>` only after approval; subsequent push respects configured upstream; missing `origin`, rejected or failed push never falsely configures/reports upstream.
- [ ] **ST01** S16 lists actual stash refs and metadata read-only (including empty/malformed lists), with return navigation; no create/apply/pop/drop/clear action from S16. ST02 branch-switch and ST03 whole-commit cherry-pick are the only approved scoped stash workflows.
- [ ] **C09/K38/T109 clarification** Switching filters retains ALL selected files; hidden selection count and actual complete action target (e.g. `3 selected · 2 hidden`) are shown before execution, and no selected file is silently omitted. Eligible/ineligible results are truthful.

## v2.3 branch, cherry-pick and continuity acceptance
- [ ] **B10/K47** Contextual `a m` always targets **local `main`**, irrespective of selected branch; when behind cached `origin/main`, `p` offers FF-only update of actual local `main` then create, `c` creates directly from unchanged local `main`, `Esc` cancels; command keys are prompt-scoped and text inputs consume letters.
- [ ] **B10** Updating `main` never pulls into another current branch, never merges/rebases or overwrites dirty state, and never creates the branch after a failed update without an explicit direct-create choice. Missing origin/main, divergence, stale refs, failed fetch and a branch creation failure show truthful exact state.
- [ ] **B11/K47** Following successful `g n`, `a n` or `a m`, message is `Created <new> · Still on <current>`; post-creation `s`/Enter explicitly switches and `Esc` keeps current checkout and created ref; ordinary S11 Enter still inspects, S11 `s` still switches.
- [ ] **ST02/B11** Dirty tracked switch block shows impacted paths and `s` to open a *reviewed*, S10 `c`-confirmed **Stash and Switch** versus Esc manual resolution. Stash tracked changes only, validate stash success before retry; no automatic stash, apply/pop/drop, or stash manager commands; failed checkout retains stash, original branch and true error.
- [ ] **C09/C20/K38** Footer unmistakably shows total and hidden selected target counts. Existing action review lists **every** affected repo-relative path and intended operation, marks hidden items, and shows ineligible reasons without changing selection semantics; no mandatory modal on every ordinary stage/unstage.
- [ ] **H04/C21/K39/UX-N05** Dirty tracked index/worktree does not block S06 entry, inspection, selection, or preview. Whole-commit Execute offers deliberate ST03 stash workflow if dirty; clean whole-commit Execute uses no stash. Selective-file Execute is clean-only; both respect sequencer, unmerged index and path-collision checks.
- [ ] **ST03** In whole-commit mode, dirty Execute shows actual tracked paths and selected ordered commits with `s` Stash and Proceed / `Esc` Cancel. Cancel does not mutate Git or selections; `s` authorizes only one tracked-only stash, successful preflight then native cherry-pick. Stash failure means no cherry-pick, no untracked stash.
- [ ] **ST03** All cherry-pick commits finish successfully before automatic stash restoration; pending/conflicted/aborted cherry-pick does **not** trigger restoration. Restore exact workflow stash OID, preserve prior staged state when supported, never pop/remove a stash before verified apply; no accidental cross-branch checkout.
- [ ] **ST03/O07** Stash-restore conflicts display `c` Continue unstash (`S09`/manual conflict resolution, no auto-commit) or `a` Abort unstash (explicitly confirm safe rollback only of restore-created tracked edits). Esc is navigation-only; cherry-picked commits survive either choice; original stash preserved on conflicts or abort.
- [ ] **ST03/O08** Rollback to post-cherry-pick baseline is permitted only when verified safe; refuse destructive automatic rollback on external edits/untracked collateral. Successful conflict-free restoration cleans up only verified workflow stash; failed cleanup reports retained stash; S16 remains read-only.
- [ ] **UX-N07/U05** Screens retain focus, scroll, filters, checked identities, source/mode/queue, composer draft and return points where valid; mutation or external ref changes reconcile and explain stale targets rather than silently resetting or re-targeting.
- [ ] **Traceability** All v2.3 changes link to numbered integration fixtures with execution status clearly unverified until tested; previously absent T129 Amend mapping is included.
