# 02 — Git data semantics and native commands

## Repository status authority
Use native Git commands, structured/parseable output, no screen-scraping. `git status --porcelain=v2 --branch -z` where applicable for working-tree and branch tracking. Account for rename, delete, conflict, binary, untracked, spaces/newlines in paths. Paths require NUL-safe processing. Git index/worktree can change independently.

## Changes filter truth table
Let I = index differs from HEAD; W = worktree differs from index; U = untracked. U is *not* a staged index diff.
| I | W | U | All | Staged | Not staged | Mixed |
|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | No | No | No | No |
| 1 | 0 | 0 | Yes | Yes | No | No |
| 0 | 1 | 0 | Yes | No | Yes | No |
| 1 | 1 | 0 | Yes | Yes | No | Yes |
| 0 | 0 | 1 | Yes | No | Yes | No |
This categorization includes deletes, renames, type changes and conflicts as statuses; unresolved conflicts get a clear special indicator. If an unresolved conflict makes I/W classification nontrivial, show conflict marker and surface it in All and the appropriate native state; do not misrepresent it as safely staged.

## File list and diffs
- All list: each changed/untracked path represented ONCE; INDEX and WORKTREE columns show Git XY status. Separate green staged and amber dirty indicators. Untracked `??` is visible.
- Staged diff: `HEAD → index` (`git diff --cached`, with safe pathspecs).
- Unstaged diff: `index → working tree` (`git diff`, with safe pathspecs). For untracked, show file as untracked content rather than fabricate tracked diff.
- Mixed inspector shows both staged and unstaged diffs with section headings. Commit review only summarizes the actual index, not the working tree.
- File stage/unstage use native Git index operations with pathspec-safe execution. Hunk/line partial staging must validate patch applicability and preserve unaffected working-tree and index content; use well-tested patch application methods. No arbitrary line selection guaranteed stageable. For binary files, file-level action only.
- Changes refresh after stage/unstage/commit/checkout/fetch-related ref updates. UI never infers stage status by toggling a Boolean.
- Discard is destructive, must preview scope and confirm. Never automatically use `git clean -fd` or discard untracked files in MVP.

## Branch intelligence
- Current branch, HEAD short object ID, configured tracking upstream, ahead/behind versus upstream, independently chosen comparison base.
- Default comparison base = `refs/remotes/origin/main` if available, otherwise `refs/heads/main` if available; if neither exists show `Base unavailable` and no ahead/behind count. Do NOT guess `master` or network-call automatically.
- Upstream is read via Git branch tracking configuration; if missing display `No upstream`, not 0/0.
- `git rev-list --left-right --count HEAD...<ref>` yields left-only (ahead) and right-only (behind) commit counts. Avoid relying on abbreviated hashes alone.
- Remote-tracking refs are locally cached; show “Local refs · last fetched unknown” unless trustworthy fetch time metadata exists. The app must not claim remote real-time freshness. Fetch is user-invoked.
- After switching branch, re-evaluate HEAD, upstream, base, and selected path state. If detached HEAD, show `DETACHED @ <hash>` and warning; do not invent branch label.
- A branch can be both ahead and behind. Do not equate diverged with broken; do not automatically rebase.
- Branch inspector (S11) shows remote, upstream, comparison base, counts, worktree status, and explicit fetch/pull/push and upstream configuration actions.

## Commit history pagination
- Read bounded chunks, initial target 100 commits; prefetch next before end; visible rows only + overscan.
- Pagination must maintain a stable traversal definition including options/filters/reference snapshot; refs changing can invalidate cursor and require a non-destructive refresh/notice.
- Git history is a graph, not a flat offset database. Cursor strategy must account for topo/order, merges and repeated commits; no missing/duplicate records from page boundaries. Test large DAG fixtures.
- Search/filter scope must be labeled (loaded-only versus full repo); do not misleadingly claim repo-wide search with loaded-only results.
- Preserve focus and selection across page loads, return visits and filters; hidden selection count surfaced.

## Git operation semantics
- Native operations: `git cherry-pick`, `git rebase -i`/sequencer, `git merge` as relevant to conflict dashboard; continue/skip/abort available per Git operation.
- Supported matrix:
  - Rebase: `git rebase --continue`, `--skip`, `--abort`.
  - Cherry-pick: `git cherry-pick --continue`, `--skip`, `--abort`.
  - Merge: `git merge --continue`, `git merge --abort`, **NO skip**.
- Detect state from repository metadata and Git plumbing as appropriate, not UI saved flags. Before each mutating action refresh status and validate preconditions.
- “Resume dashboard” = show S08, no Git execution; “Continue” = explicit Git mutation.
- Cherry-pick planner takes explicitly selected OIDs and order, executes sequentially, handles merge commits only with explicitly chosen mainline parent (otherwise block and explain).
- Interactive rebase requires explicit base/range and affected commits. No arbitrary noncontiguous set silently converted into a rebase range. Validate squash/fixup preceding commit and other todo constraints.
- Conflict resolver must describe ours/theirs in context; rebase semantics can be counterintuitive. For unsupported conflict types offer external editor/manually supported native resolution, not a misleading auto-resolve.
- Continue enabled only when known prerequisites satisfied. Git can still fail; always handle errors.
- Abort is never promised as universal undo, especially if unrelated worktree edits exist. Revalidate and confirm.

## Commit
- Composer displays subject/body editing (multiline) and currently staged paths/hunks only.
- Commit via native Git while honoring hooks, signing and message config; report failures.
- Amend is separate action and warns when a commit may be published; no auto-force-push.
- Nothing staged → Commit unavailable with explanation; staged only with message → review → explicit execute.
- When Git status has unmerged index entries, block unsafe commit and route to conflict dashboard.

## Command process safety
- Execute binary+argv without shell expansion, repo cwd explicit.
- Never store or print credentials in logs; respect Git config and credential helper.
- Don't run parallel conflicting index/ref mutations within TUI; external process changes require revalidation.
- Errors show Git stderr and next safe action, not “success” based on anticipated output.
