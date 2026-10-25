# 03 — Workflow contracts and state machines

## Shared workflow state machine
NORMAL → (contextual `a` or direct shortcut) → VALIDATE → PLANNER (if necessary) → PREVIEW/REVIEW → CONFIRM EXECUTION → RUNNING → COMPLETED / PAUSED / FAILED.
PAUSED → INSPECT/RESOLVE → VALIDATE → CONTINUE or SKIP or ABORT → RUNNING / COMPLETED / PAUSED / FAILED.
NAVIGATE BACK from PAUSED simply changes UI view, does NOT change Git operation. Quit similarly does not abort.

### Uniform ESC policy
- S05 menu Esc → invoking screen.
- S02 diff Esc → S01 (preserving filter and selection).
- S03 composer Esc → prior screen; dirty input triggers “Keep draft / Discard draft / Cancel” UI-only dialog. No commit.
- S06/S07 planner Esc → previous view, but if plan modified, ask whether to discard unexecuted plan; default Cancel.
- S09 conflict resolver Esc → S08 paused dashboard, Git remains paused.
- S08 paused dashboard Esc → previous repo screen with visible paused banner.
- S10 destructive confirmation Esc → cancel dialog.
- S01 root Esc → no-op; `q` explicit quit.

## Cherry-pick
Entry from S04: `Space` select arbitrary noncontiguous commits → `a` > Cherry-pick OR `c p`. If none selected, focused commit is target. If selected commits come from multiple branches their full OIDs are retained; do not infer branch membership from one filter.
S06 target branch explicitly displayed (default currently checked-out HEAD; if target not checked out, offer explicit switch/cancel rather than silently switching). Queue order is explicit and moveable with Shift+J/K; show each commit hash/message and order.
`p` preview displays target, queue, files affected, duplicate patch warnings where detectable, and potential conflict uncertainty. Explicit “Execute plan” is required. After starting, track actual native sequencer state. No pretend atomic single transaction for multiple commits. On conflicts go S08. On empty commit show cause and valid next actions. On success show applied resulting commit mappings and status. No manual `Esc` abort.

## Interactive rebase
Entry S04/S11 `r i` or context menu > Interactive rebase. Must choose explicit base/ref and show computed affected range; verify rebase can be started in current repository state. Plan consists of ordered sequence, each row one of pick/reword/edit/squash/fixup/drop. Invalid first-row squash/fixup must be blocked and explained. Reorder via Shift+J/K. `v` preview; explicit Execute. If published commits affected, warn and require acknowledgement; no automatic force push. At edit stop show “Amend then Continue”; at conflicts go S08. Track rewritten history faithfully; no blanket promise that commit IDs remain same.

## Paused operations
On startup or repository state change, detect rebase/cherry-pick/merge operation. Show persistent operation badge on S01. `o` opens S08.
S08 displays operation type, branch/current HEAD, target/base if available, progress if Git data supports it, paused reason, active commit if available, conflicted file list; current Git state is truth. Missing data show `Unknown`, not fabricated numbers.
- `Enter` on conflicted file → S09.
- `c` → revalidate; if unmet prerequisites show reason; if valid call operation-specific `--continue` and refresh state.
- `s` → if rebase/cherry-pick, open S10 Skip confirmation, default Cancel; if merge, show disabled with “Merge has no skip operation”.
- `A` → S10 Abort confirmation with operation, branch, progress, known impact, underlying command; default Cancel. Confirmation runs Git and verifies result.
- `r` refresh Git status; `Esc` back only; `q` quit UI without abort.

## Conflict resolver
S09 focuses one conflicted path. Display base/ours/theirs where present and editable result; keyboard n/N iterates conflict regions, 1/2/3 select ours/theirs/both for a supported text conflict; e opens configured external editor; an explicit “Mark resolved” action applies/stages result after checking unresolved index and remaining conflict markers where applicable. No immediate automatic mark-resolved from choosing a side; choosing either or both can require manual editing. Return to S08 when done. For binary and complex rename/delete conflicts, present supported safe manual guidance and state, not fake three-way text.

## Commit workflow
S01 list focus and optional UI-only selection → `s/u` mutate index for selected/focused scope (file, hunk, supported lines), with fresh status check; status filter dynamically updates. `c c` → S03 composer, subject/body; staged-only summary; Review → explicit Commit; successful commit returns S01 and refreshes. `c a` opens amend composer separately; show published-history risk. `Esc` never commits or discards working files.

## Branch workflow
S11 invoked by opening branch header or via palette. Display two independent comparisons: HEAD vs its upstream; HEAD vs comparison base. Branch actions via context menu: create, switch, delete local branch, fetch, pull, push, set upstream; risky choices explicit confirmation. No automatically running network operations on landing page.
