# Integration scenario / fixture specification

Run tests in temporary repositories initialized by test harness. Use real Git and local bare remotes, never user repositories. Git identity set only within fixtures. Assert actual `git status --porcelain=v2`, index diffs, refs and sequencer state before/after actions. Capture test output and failure reproduction commands.

| ID | Fixture and action | Assert |
|---|---|---|
| T01 | Clean repo, launch | Interactive S01, no changed rows, commit disabled, branch info present |
| T02 | Modified tracked file, no staging | All+Not staged, not Staged/Mixed |
| T03 | Staged tracked file, unchanged after | All+Staged, not Not staged/Mixed |
| T04 | Stage hunk A then edit hunk B | Exactly one Mixed row; staged/unstaged diffs contain correct portions |
| T05 | Untracked file | All+Not staged, never Staged unless explicitly `git add` |
| T06 | Stage mixed file remaining worktree changes | Index updated, file becomes Staged only, untouched patches preserved |
| T07 | Unstage staged hunk | Working copy preserved, index correctly updated |
| T08 | Focus/filter/selection navigation through S02 | Git byte-for-byte unchanged; hidden selections retained |
| T09 | Commit staged portion of mixed file | Commit only index; worktree retains extra unstaged changes |
| T10 | Commit hook rejects | No false success; input draft retained; accurate status |
| T11 | No upstream branch | “No upstream” and no computed upstream zero counts |
| T12 | Feature upstream differs from origin/main | Two independent ahead/behind pairs correct |
| T13 | Detached HEAD, missing base | Clear detached/base unavailable states |
| T14 | >1000 commits and merge graph | Incremental pages stable, no duplicate IDs, selection persists |
| T15 | Noncontiguous commits cherry-picked/reordered | Result reflects explicit queue order or paused conflict accurately |
| T16 | Cherry-pick conflict then Esc | Sequencer state intact after UI navigation |
| T17 | Cherry-pick interrupted, close/reopen app | S08 detects actual state; continue/skip/abort available as permitted |
| T18 | Interactive rebase reorder/fixup | New Git history reflects plan; invalid first-row fixup rejected |
| T19 | Interactive rebase conflict/edit stop | Correct pause reason, no fake conflict at edit stop |
| T20 | Rebase abort confirmation Esc | No Git ref/worktree mutation from Esc; sequencer still exists |
| T21 | Rebase abort confirmed | Native abort attempted; outcome checked and disclosed |
| T22 | Merge conflict active | Continue/Abort shown, Skip disabled |
| T23 | Resolution choose ours/theirs/both | File content and index correctness verified separately before Continue |
| T24 | Binary conflict | No invalid three-way text side selection |
| T25 | Other process changes HEAD during planner | Execute invalidates stale plan, no mutation until revalidation |
| T26 | Filenames with spaces, unicode, unusual characters | No broken path parsing or unsafe argv |
| T27 | Empty cherry-pick | Proper paused/explanation/skip behavior, no bogus success |
| T28 | Root Esc and quit while paused | Quit UI leaves Git operation untouched |
| T29 | Keyboard typing in commit message | `s`, `u`, `a`, `c`, digits typed as chars, no actions |
| T30 | Launch performance | Measured first interactive frame; benchmark results and machine details retained |

## Coverage reporting format
For every `tests/ACCEPTANCE.md` ID report: Implemented / Verified / Blocked / Deferred, linked source module, automated test ID, measured evidence. Do not use “Complete” for mock-only UI.
