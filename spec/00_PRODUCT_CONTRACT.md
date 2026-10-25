# 00 — Frozen product contract

## Core idea
GitFlow is an extremely user-friendly, fun-but-professional **terminal-native Git application** for macOS and Linux. It combines Lazygit-like fast keyboard navigation with IntelliJ-like cherry-pick, rebase and conflict guidance. Color provides information rather than decoration. A user should be able to complete the common Git commit loop on the FIRST screen: identify current branch and remote relationship, inspect changed files, stage intended content, and create a commit.

## Must-respect decisions
- TUI only; no mouse requirement or GUI shell. Keyboard-only for all P0 functionality.
- Start at **Changes Workspace**, not a landing splash, welcome tour, separate metrics page, or empty dashboard.
- Always-visible compact **branch intelligence header**: current branch or detached HEAD, HEAD abbreviated ID, configured upstream and local ahead/behind, independently configured comparison base and local ahead/behind, freshness label for remote-tracking data, active Git operation warning, working tree status.
- Single unified changed-file list, each path appears once, with 4 views: `[1] All`, `[2] Staged`, `[3] Not staged`, `[4] Mixed`.
- Two status columns: `INDEX` (HEAD → index) and `WORKTREE` (index → working tree). Show an untracked marker as applicable. Mixed state is visually obvious.
- Focused file opens a two-sided staged/unstaged inspector, showing distinct diffs when mixed. File/hunk and supported line-level stage/unstage are P0.
- Commit composer remains accessible directly from the home workspace; commit includes only actual index content; allow amend with published-history warning.
- History loads incrementally using pagination and virtualized visible rows; preserve selection across navigation/filter/pagination.
- Both contextual action menus and direct keyboard shortcuts mandatory; same registry and validations.
- Cherry-pick planner: multi-source/non-contiguous selection, reorder, preview, execute ordered commits, pause/resume/skip/abort using Git sequencer state.
- Interactive rebase planner: select base/range, pick/reword/edit/squash/fixup/drop, reorder, validate, preview; preserve actual Git rebase behavior.
- Conflict dashboard: conflict files, three-way for supported text conflicts, external editor fallback, continue/skip/abort as supported, non-destructive Esc.
- 100 ms to first usable frame **target**, not promise. Loading status/branches/history can continue after frame. Define and measure cold/warm p50/p95 on reference equipment.
- No product implementation language frozen. Write ADR to choose Go or Rust.

## Screen IDs
S01 Changes Workspace (startup), S02 Full Diff Inspector, S03 Commit Composer/Review, S04 Commit History, S05 Contextual Action Menu, S06 Cherry-pick Planner/Review, S07 Rebase Planner/Review, S08 Paused Operation Dashboard, S09 Conflict Resolver, S10 Abort/Skip confirmation, S11 Branch Inspector, S12 Command Palette/Help. Do not create other product screens except inline Git error/empty states and basic configuration diagnostics strictly necessary to implement listed screens.

## Primary actions (P0)
- Repository discovery/read-only state, basic status / diff / stage / unstage / commit / amend.
- Create/switch/delete local branch; fetch/pull/push and set upstream (explicit actions, not auto-sync).
- Commit graph with incremental history, inspect commit/diff, compare selected commits.
- Cherry-pick and interactive rebase plans.
- Active rebase/cherry-pick/merge visibility, supported continue/skip/abort, conflict resolution.
- Basic search/filter, command palette and contextual help.

## Interaction tone
- Polished and efficient; use color and clean panel borders for clarity.
- Plain language: “2 conflicts need resolution”, “2 commits ahead of upstream”, “3 commits behind main”; do not label commits as already merged merely from ahead/behind.
- No gamification, achievement systems, mascots, confetti, emojis in commit data, or mandatory onboarding. Avoid whimsical copy during errors or destructive actions.

## Hard release boundaries
P1/P2 excluded from MVP: PR/hosting API, CI, issue tracker, worktrees GUI, submodule management GUI, stash manager, bisect UI, Git notes UI, tag manager, remote hosting auth UX, full recovery center, custom macros, marketplace, native GUI, themes system, telemetry, AI-generated commits. Repository state that incidentally contains these concepts may be displayed safely, but do not build dedicated features.
