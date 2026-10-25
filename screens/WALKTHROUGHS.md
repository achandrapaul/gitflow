# Keyboard-by-keyboard journeys (normative acceptance flows)

**Important:** The app is keyboard-only, so “click-by-click” is defined below as exact focus/key/action/next-screen steps. Each action is reachable without a mouse. Focus restoration and selection are part of the contract.

## J01 — Open app, inspect and commit a file
1. Run `gitflow` from a Git working tree. S01 renders first interactive frame; branch metadata may load asynchronously.
2. Focus starts in Changed Files list on first visible file (if any), with All filter active.
3. Press `j/k` to move to a modified file. The adjacent diff preview updates but selection does not.
4. Press `Enter`. S02 Full Diff opens with separate staged/unstaged panes when file mixed.
5. Press `Esc`. Return S01 with same focused file, All filter, scroll, selections.
6. Press `Space` to UI-select the file; status bar increases selected count. Git index UNCHANGED.
7. Press `s`. Stage selected file(s), subject to fresh validation; refresh index and worktree statuses; clear or reconcile stale UI selection visibly.
8. Press `c c` (in navigation scope). S03 composer opens, showing staged-only summary.
9. Type subject and optional body; inspect review; invoke `Review commit` by action menu/keyboard focus and `Enter`.
10. Review exact staged content, choose Confirm with keyboard focus, press Enter. Native `git commit` runs, hooks/signing respected.
11. Return S01 with fresh status. `Esc` at any earlier stage NEVER commits/discards Git changes.

## J02 — Filter mixed/staged/not staged
1. On S01 press `4`: only files with BOTH staged index changes and additional unstaged worktree changes display.
2. Focus one mixed row; verify INDEX=M and WORKTREE=M (or native equivalent).
3. Press Enter S02: two distinct sections “HEAD → INDEX” and “INDEX → WORKTREE”.
4. Press Esc S01. Press `2`: Staged includes this mixed file.
5. Press `3`: Not staged excludes this mixed file, but includes truly unstaged and untracked paths.
6. Press `1`: All contains every changed path once.
7. Select file with Space in All; switch filter 3 where selected item hidden; hidden selection counter remains; `a` menu shows target scope before any bulk mutation.

## J03 — Stage one hunk only
1. S01 focus unstaged text file; Enter S02.
2. Tab to unstaged diff region; j/k onto desired hunk; Space selects hunk UI-only.
3. Press `s` to stage chosen hunk. App validates patch against current index/worktree and applies native index update.
4. Inspector refreshes: selected patch moves into staged section, remaining unstaged edits remain. If unsafe or stale, show error and make no partial claim.
5. Esc goes S01, counts/filters reflect actual index.

## J04 — Context action vs shortcut equivalence
1. S04 focus a commit, Space select; scroll/paginate and select a nonadjacent second commit.
2. Press `a` → S05 shows “Cherry-pick selected” and `c p`; Enter selects action → S06 planner.
3. Esc out (discard modified draft only if prompted) → S04 retains selected commits.
4. Press `c p` → S06. Assert SAME selected OIDs, same target, same validation and same planner as step 2.

## J05 — Cherry-pick unordered commits
1. S04 select several nonadjacent commits (possibly from multiple branches) using Space, paging as needed.
2. Press `c p` → S06 queue populated with explicit OIDs; target is checked-out branch.
3. Focus second queue row, Shift+K moves earlier; no Git mutation.
4. Press `p` to preview queue order and affected patches; return to plan.
5. Open `a` → Execute plan → review; confirm explicitly.
6. If Git pauses on conflict, S08 opens and shows actual sequencer status. No assumption all commits applied.
7. Esc S01 → paused-operation badge remains. Press `o` → S08 resumed UI only.

## J06 — Rebase plan and conflict
1. S04 choose explicit rebase base/ref via contextual action or S11; press `r i` → S07.
2. Inspect affected range; use j/k to focus row; `s` assigns Squash; Shift+J/K reorders; `v` previews.
3. Attempt invalid first-row squash → disabled Execute with explanation.
4. Fix plan, open `a` → Execute plan; confirm published-history warning if applicable.
5. Git pauses at conflict; S08 shows operation and real unresolved files; Enter path S09.
6. S09 choose side 1/2/3 for supported text region or `e` manual edit; review result; `a` > Mark resolved; repeat for all files.
7. Esc returns S08, where `c` Continue now becomes available if index unmerged entries gone.
8. Press `c`; Git attempts `git rebase --continue`, UI displays actual result or next pause.

## J07 — Abort protection
1. At S08 during paused rebase press Esc → S01 with paused badge. Confirm Git sequencer still exists.
2. Press `o` → S08. Press `A` → S10 Abort confirmation; Cancel focused by default.
3. Press Esc → S08. Verify ZERO Git abort executed.
4. Press A; Tab to Confirm; Enter → revalidate, execute `git rebase --abort`, inspect state, report actual success or failure.
5. If Git abort fails, stay in accurate paused/error state with stderr and no false success.

## J08 — Skip support differences
1. S08 with cherry-pick active: `s` opens skip warning, default Cancel; confirm executes native `git cherry-pick --skip` if valid.
2. S08 with merge active: `s` is unavailable; `a` menu labels Skip disabled “Merge does not support skip”.
3. S08 with an edit stop: state description must not lie that unresolved conflicts exist.

## J09 — Inspect branch intelligence
1. On S01 read current `feature/auth`, upstream `origin/feature/auth`, ahead/behind (e.g. 2/0), compare base `origin/main` (e.g. 7/3).
2. Focus branch header, Enter → S11. Verify two comparisons remain independent.
3. Execute Fetch explicitly in S11; updated remote-tracking data may change counts. No automatic fetch from rendering.
4. Esc returns S01, preserves changed-file state.
5. Simulate branch with no upstream → “No upstream” and no artificial zero count. If `origin/main` absent show “Base unavailable”.

## J10 — Escape and text safety
1. Enter S03 commit message input and type `s`, `u`, `a`, `1`, `c`. They become text, never Git commands.
2. Press Esc → editor exit/draft navigation, not commit, stage, discard, or abort.
3. From S05 menu press Esc → menu closes, selections remain.
4. From S09 resolver press Esc → S08, Git unchanged.
5. From S08 press Esc → S01, Git stays paused.
