# Screen-by-screen exact terminal design

All wireframes assume a usable terminal around 120×36; adapt proportionally. Sample paths/counts/hashes are fictitious. **No clickable mouse target is required in the product**; “open/click through” means focus + Enter or a direct keyboard sequence. Common footer always exposes relevant keys and `?` help.

## S01 — Changes Workspace (default launch)
```text
┌ gitflow · ~/repo ───────────────────────────────────────────────────────────────┐
│ ● feature/auth @ 7ac91f2  → origin/feature/auth  ↑2 ↓0  [local refs]           │
│   vs origin/main ↑7 ↓3  · working tree: 6 changed  · operation: none            │
├ CHANGES ────────────────────────────────┬ DIFF INSPECTOR ──────────────────────┤
│ [1 All:6] [2 Staged:4] [3 Not:2] [4 Mixed:2]│ src/auth/login.ts                 │
│  SPACE   PATH                 IDX WT    │ ┌ STAGED · HEAD → INDEX ──────────┐ │
│ > [ ] M  src/auth/login.ts      M   M    │ │+ fix validation                 │ │
│   [ ] M  src/auth/session.ts    M   -    │ └─────────────────────────────────┘ │
│   [ ] M  src/ui/Login.tsx       -   M    │ ┌ UNSTAGED · INDEX → WORKTREE ───┐ │
│   [ ] A  tests/auth.test.ts     A   -    │ │+ adjust token expiration       │ │
│   [ ] M  src/utils/token.ts     M   M    │ └─────────────────────────────────┘ │
│   [ ] ?  docs/auth.md           -   ??   │ Enter: full diff  s Stage  u Unstage │
│ 2 selected (1 hidden by filter)         │                                      │
├ COMMIT COMPOSER ────────────────────────┴──────────────────────────────────────┤
│ Subject: feat(auth): improve session handling                                  │
│ Staged: 4 files · Only index content will be committed                          │
│ [c c Compose] [c a Amend] [a Actions]                                           │
├────────────────────────────────────────────────────────────────────────────────┤
│ j/k Move  Enter Open  Space Select  1-4 Filter  s/u Stage  a Actions  ? Help  │
└────────────────────────────────────────────────────────────────────────────────┘
```
Branch header opens S11 via Enter when focused; `o` opens S08 if paused. Focus order branch header → file list → diff → composer. The file list is single unified list, no three independent staging trees. File row shows both XY and contextual status; no duplicate row for mixed files. The footer and header remain visible. If branch metadata still loading, show placeholders, not fake zeros.

## S02 — Full Diff Inspector
Shows path, index and worktree state, staged diff separate from unstaged diff, file/hunk/line focus, `Space` UI-select, `s` stage (unstaged), `u` unstage (staged), `Tab` pane cycle, `Esc` return S01. Binary/untracked/rename/unmerged paths show appropriate status/unsupported action guidance. Selection displayed. No commit action from Enter on diff line.

## S03 — Commit Composer and Review
Composer includes subject, optional body multiline input, staged summary (real index only), Cancel/Esc. Review screen/overlay shows staged paths and hunk summary and the commit message. Only explicit Confirm Commit mutates Git. Nothing staged disables Review with reason. Amend path is distinct and warns on publication risks. Commit errors return to composer with draft retained.

## S04 — Commit History
```
┌ HISTORY · feature/auth ────────────────────┬ COMMIT DETAILS ──────────────────┐
│ [ ] ● 7ac91f2 Fix auth refresh             │ commit 7ac91f2                   │
│ >[x] ● b23a5c1 Fix login validation        │ author · timestamp · parents     │
│  [x] ● 64ca2d1 Add middleware              │ changed files · +/− · diff       │
│         (... loads next page on scroll)    │                                 │
│ Selected: 2 (0 hidden) · 100 loaded         │ Enter opens focused commit      │
├ j/k Move · Space Select · a Actions · c p Cherry-pick · r i Rebase · Esc Back ┤
└───────────────────────────────────────────────────────────────────────────────┘
```
Provide branch/ref and filter context; pagination cursor and load-in-progress indicator. `Enter` opens commit detail. `Space` selects. `a` opens S05. `c p` S06; `r i` S07 after base selection.

## S05 — Contextual Actions
Overlay centered on invoking screen; title “Actions for N selected commits” or “Actions for <focused item>”. Entries from shared action registry with disabled reasons and shortcut hints. j/k focus; Enter activate; Esc dismiss, never clear selections. For history commits: Cherry-pick, Interactive rebase (base must be set), Compare (exact 2), Copy hashes (optional non-mutating; if not built no row). For changes: Stage/Unstage/Discard (guarded). Don't add unrelated items.

## S06 — Cherry-pick Planner
Header: source selection (may be multi-branch), target checked-out branch, total. Left ordered queue of full-OID-backed commits with original short hashes/messages; right preview of selected commit changes and warnings. `Shift+J/K` reorder focused queue item; `Enter` inspect; `Space` include/exclude; `p` preview. Explicit Execute plan action via context action (`a`) opens review with Cancel default. On Git pause S08. Esc while plan modified prompts discard of plan only.

## S07 — Rebase Planner
Header: current branch, explicit chosen onto/base, exact affected range, published-history risk label. Ordered todo list rows with action `pick/reword/edit/squash/fixup/drop`, `Shift+J/K` reorder; `p/r/e/s/f/d` set action on focused row; `v` preview; `a` action menu for Execute. Invalid first-row squash/fixup shows inline error and disables execution. Distinct from selecting random commits. Esc never invokes rebase abort.

## S08 — Paused Operation Dashboard
```
┌ OPERATIONS · REBASE PAUSED ───────────────────────────────────────────────────┐
│ feature/auth → main   step 15/20 (if available)   commit 7ac91f2            │
│ Reason: 2 unresolved files. Git operation still active.                       │
│ > src/auth/session.ts      UNRESOLVED    [Enter resolve]                      │
│   src/auth/token.ts        UNRESOLVED    [Enter resolve]                      │
│                                                                              │
│ c Continue [disabled: 2 conflicts]  s Skip  A Abort (confirm)                │
│ r Refresh · a Actions · Esc Back (operation remains paused)                  │
└──────────────────────────────────────────────────────────────────────────────┘
```
If intentionally paused for edit, show edit instructions and avoid inventing conflict files. If merge is active, Skip disabled with specific reason. `Esc` returns previous view and leaves paused badge visible on S01. UI Resume = reopen S08 with `o`, no Git execution.

## S09 — Conflict Resolver
Show filepath, conflict region N of M, 3-way base/ours/theirs panes where supported, result preview, meaning of ours/theirs for active operation. `1` Accept ours, `2` Accept theirs, `3` Accept both (when meaningful), `n/N` next/previous, `e` external editor, `a` Actions > Mark resolved (validate and stage), `Esc` S08. Choosing a side alone does NOT automatically run continue or mark file resolved.

## S10 — Skip / Abort Confirmation
Title directly names effect: “Abort interactive rebase?” or “Skip current commit?”. Display branch, operation, progress if available, underlying native Git command, consequences and recovery limitations. `Cancel` default focus. `Esc` Cancel. Tab focus Confirm. Enter on Confirm executes only after validation, then refresh Git status. A failed command shows stderr and retains accurate state. No destructive commands on Esc.

## S11 — Branch Inspector
Current branch, HEAD full/short OID, upstream or “No upstream”, ahead/behind against upstream, independently chosen compare base, ahead/behind base, detached HEAD state, local refs freshness notice, working tree, active Git operation. Actions available via contextual menu: explicit fetch/pull/push/upstream config plus create/switch/delete local branch (confirmation where risky). No automatic networking. Return to S01 via Esc.

## S12 — Palette and Help
Palette entered with `:` and fuzzy filters only documented commands from shared registry; unavailable actions are disabled with reasons. `?` help displays shortcuts for the active scope; no edit side effects. Esc closes. While palette text input focused, typing `s`, `u`, etc. does not call Git.

## Empty, loading, error states
- Clean repo: S01 file list says “Working tree clean” with 0 files; staged commit disabled; still show branch intelligence and history access.
- Unknown upstream: `No upstream`, no counts; offer explicit configuration in S11.
- Unknown base: `Base unavailable`, no fabricated comparison counts.
- Detached HEAD: prominent name/status; no made-up branch.
- Loading: UI frame interactive, status/history placeholders then asynchronously filled.
- External repository mutation: present refreshed Git state and invalidate stale plan with clear message.
- Git error: show command context, stderr excerpt, retry/return as applicable, no bogus “success.”
