# 05 — APPROVED extension: IntelliJ-style Git workflows, GitFlow interaction (v1.1)

This document is a binding approved extension to v1.0, not a license to add other IntelliJ features. If an earlier document appears to prohibit *selective files from commits*, this extension explicitly approves only the behavior specified here. No other new feature may be inferred.

## Intent and scope
The user wants the IntelliJ Git tool-window capabilities presented in a discoverable, keyboard-only GitFlow TUI: pull, push, see/filter commits, select whole commits for cherry-pick, select individual FILES from a commit for selective application, and rebase. Reuse the current landing page (Changes Workspace), History S04, planner S06/S07, operation dashboard S08 and confirmations. Add **S13 Commit File Inspector**, **S14 Selective Apply Planner**, **S15 Remote Sync Review**. No PR, GitHub/GitLab provider integration, drag-and-drop, mouse, new Git data model, external account login, or automatic sync.

## Unchanging interaction contract
- `Enter` opens focused object or confirms focused dialog action; `Space` selects/deselects UI objects; `Esc` closes/back only, NEVER aborts/discards Git state; `a` opens contextual actions; `?` help; `:` palette; `j/k` navigate, `Tab` panel focus.
- A menu and direct shortcut invoke the SAME action ID, with identical guards, previews and Git handler.
- Whole-commit selections and individual-file-in-commit selections are DIFFERENT typed selection sets; NEVER silently mix or coerce them.
- Read-only navigation is safe during paused operations; mutations are guarded.
- Show command targets and current branch/upstream before running pull or push.

## Information architecture
S01 Changes Workspace: branch intelligence header (HEAD, tracking upstream, HEAD↔upstream ahead/behind, HEAD↔comparison-base ahead/behind, local fetch timestamp/unknown), changed file filters All/Staged/Not staged/Mixed, diff preview, stage/unstage, composer; top-level navigation to S04 History, S11 Branch, S08 Operations. Pull and Push via `g l` and `g p`, plus `a` on the branch header -> Pull/Push. Both open S15, not immediate execution.
S04 History: paginated commit graph/list, author/time/message, refs, commit OID, branch/base filters; selected commit counter; Enter opens S13, Space selects whole commit; `c p` opens S06 with selected commits; `r i` opens S07 with base selection; `a` menu shows only valid commands. History selection is not index staging.
S13 Commit File Inspector: header commit full/short OID, subject, author/time and parent count; left FILES CHANGED IN THIS COMMIT, right commit-scoped diff, footer help. `j/k` navigate files, `Space` toggles FILE-IN-COMMIT selection, `Enter` expands focused file diff, `Tab` toggles panels, `a` -> 'Apply selected files to working tree', `c f` direct shortcut for SAME action -> S14. `Esc` returns S04 preserving whole-commit and file selections independently. Include rename/delete/add types; binary file diffs show metadata not invented textual hunks. A no-selection `c f` applies focused file ONLY and explicitly says so in S14.
S06 Cherry-pick Planner: only whole commits; reorder, target, preview, apply with Git cherry-pick semantics. Individual files selected in S13 must NOT be silently included.
S14 Selective Apply Planner: list (source commit OID, path, change type) for each selected file; explicit current checkout destination (no silent checkout); ordered source commits if selections span commits; preview patch and target, warn on overlapping file patches, ask for final review, apply selected changes into index/worktree without automatically creating any commit. User later reviews Changes Workspace and creates a normal commit. Label UI 'Apply files from commits', explanatory subtitle 'Selective patch application — not git cherry-pick'. May name a sidebar entry 'Selective cherry-pick' ONLY if subtitle makes the distinction conspicuous. On conflict or failed apply, display truthful Git state and safe resolution/recovery instructions; do NOT pretend Git cherry-pick --continue/--abort applies to patch application. A failed patch cannot be claimed atomic or auto-recoverable. Do not perform unvalidated partial file overwrite or a whole-commit cherry-pick as a substitute.
S07 Rebase: retains exact existing base selection, ordered interactive todo/actions and safety. No single-file-in-commit rebase operation invented.
S15 Remote Sync Review: S15 shows operation Fetch/Pull/Push, local branch, remote, exact remote branch/upstream, ahead/behind based on local refs (staleness disclosed), selected strategy, expected command, what might change, Cancel (default) and explicit Execute. Pull supports fast-forward-only as default for MVP; if non-FF, explain and stop; no implicit merge/rebase policy guessed. Push defaults ordinary push to configured upstream; non-fast-forward rejected; NO force push and NO automatic upstream invention. If no upstream exists, show Set upstream as an explicit separate existing branch action and disable Pull/Push until configured. Network failure/auth error is shown verbatim safely with hints. Offer Fetch as a separate explicit action using existing branch/remote commands, not automatic background polling.

## Exact interaction journeys
J11 Pull: S01 focus branch header -> `a` -> Pull or `g l` -> S15 with target/strategy -> Enter on Review (no mutation) -> explicit Execute -> progress -> results -> Esc S01. No upstream: Pull disabled, explain. Diverged branch: FF-only pull fails with clear remedial next steps; do not silently rebase/merge.
J12 Push: S01 focus branch header -> `a` -> Push or `g p` -> S15 -> verify local/upstream refs -> explicit Execute -> Git push -> show outcome/ref refresh; non-FF leaves repo unchanged.
J13 Whole commits: S01 `h` (History view binding defined below) -> S04 `Space` select 2 noncontiguous commits -> `c p` or `a` > Cherry-pick -> S06 reorder/review/execute -> pause via S08 on conflict -> Continue/Skip/Abort as supported.
J14 Files from one commit: S04 focus commit -> Enter S13 -> Space on two files -> `c f` or `a` > Apply selected files -> S14 preview target and exactly the two files -> Execute -> S01 sees resulting index/worktree changes -> review and commit via `c c`. Do not create source commit or apply other files.
J15 Files from multiple commits: S13 select files on first commit, Esc S04, Enter second commit, select files; S14 shows commit-grouped ordered queue, overlapping path warnings, explicit execution order. On a failed step, stop and report exactly which steps completed. Do not imply cross-commit transactionality.
J16 Rebase: S04 `r i` -> S07 choose base, rewrite todo and validate -> preview/execute -> S08 on pause -> recover; Esc only leaves UI.

## Shortcuts added (non-conflicting in NORMAL contexts)
- `h`: focus/open History workspace from S01; may be implemented as top-level view switch (document if it conflicts with horizontal navigation). To avoid ambiguity, the normative global shortcut is **`g h`**, NOT single `h`. S01 header displays `g h` History.
- `g l`: open Pull review S15 (L = pull); `g p`: open Push review S15; `g f`: open Fetch review S15.
- `c p`: cherry-pick selected whole commits from S04.
- `c f`: apply selected FILES within commits from S13/S04 only when a typed file selection exists; otherwise disabled outside S13.
- `r i`: interactive rebase from S04.
- `a`: contextual actions, available in every main workspace with relevant options.
- S15: `Tab` move focus, `Enter` opens/focuses or activates selected REVIEW/EXECUTE action with explicit confirmation; `Esc` close and return (no Git mutation).
- In a modal/command input, scoped keys override prefix shortcuts; no keystroke may leak to the background screen.

## Technical correctness mandates
1. Git commit file changes are computed against the selected parent. A root commit uses the empty tree. For a merge commit, require an explicitly chosen mainline parent before computing the file patch; do not guess.
2. Selecting a file from commit C means apply the **change introduced in C for that path** (parent→C) onto current checked-out worktree/index. It does NOT mean blindly replace the destination file with its snapshot at C. Deletions, additions, renames and binary updates need Git-aware handling; if safe apply is unsupported, block with a clear reason. Respect renames and paths containing special characters (NUL-delimited plumbing output).
3. Use Git-produced patches and appropriate apply semantics, with preflight status and fail-closed strategy where safe application cannot be guaranteed. Do NOT blindly run `git restore --source=C -- PATH` as a cherry-pick substitute. Do NOT use `git cherry-pick C` when a file subset was requested.
4. A clean-worktree/index prerequisite is acceptable for the first Selective Apply implementation. Enforce it transparently, or implement robust overlap safeguards using real Git tests. Do NOT destroy preexisting work. Show if results land staged or unstaged; choose and document one consistent policy (recommended: **staged** if successful using safe `git apply --index` strategy; if robust 3-way fallback cannot be proven, stop rather than invent conflict resolver semantics). All resulting index/worktree state must be re-read from Git.
5. Remote HEAD comparison is to locally known refs, not guaranteed to represent server state until fetch. No automatic fetch. No automatic force push. Pull/push commands must respect credential helpers and noninteractive error feedback; no credential collection UI.
6. Branch and commit history must remain asynchronously paginated with bounded rendering. Never block 100 ms first interactive frame on remote connectivity or full history scan.

## Acceptance criteria (binding)
A01 Pull/Push both available by context menu and exact prefix shortcuts and produce equivalent review screens.
A02 Pull/Push show branch, upstream, command, ref freshness and clearly distinguish remote tracking from base `origin/main`.
A03 FF-only pull succeeds on linear upstream advancement and refuses divergence without implicit merge/rebase.
A04 Push succeeds when FF permitted; fails safely for remote non-FF without force.
A05 Missing upstream disables pull/push with intelligible setup guidance.
A06 S04 Enter opens commit file list and preserves unrelated whole-commit selections.
A07 S13 Space selects individual changed files from a specific commit and shows exact count, preserving this on return.
A08 Applying one selected file from a 3-file commit DOES NOT apply remaining 2 files or create a commit automatically.
A09 Applying a rename/delete/add is either demonstrably correct in test fixtures or explicitly unsupported/disabled for MVP; no silent replacement.
A10 File selection from multi-parent commit requires parent choice.
A11 Multiple commit-file selections execute in an explicitly visible order; partial failure accurately reports completed steps.
A12 Esc from S13/S14/S15 only navigates UI and never mutates Git.
A13 Active rebase/cherry-pick pause is not misidentified as selective patch application and vice versa.
A14 No new features outside this document and prior spec are introduced.
