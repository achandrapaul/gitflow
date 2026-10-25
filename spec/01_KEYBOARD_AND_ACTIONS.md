# 01 — Keyboard, focus, selection, action system

## Normative key rules
1. `Enter`: open focused row/item; inside a menu choose highlighted action; inside explicit confirmation choose highlighted button. Enter on file/commit NEVER stages, commits, cherry-picks or aborts.
2. `Space`: toggle UI selection of focused selectable file, commit, hunk, or line. NEVER mutate Git index. Nonselectable locations show a hint, no hidden mutation.
3. `Esc`: dismiss top overlay; if none, back to prior view; if at root, no-op. NEVER invoke Git abort/reset/discard. During editing, Esc exits edit mode without mutation to Git (draft handling below).
4. `a`: contextual action menu for **selected set** if valid selections exist in current object domain; otherwise focused object. A selection count and hidden-selected count must be shown when applicable.
5. `:` command palette; `/` scoped search/filter; `?` shortcut help.
6. `j/k` or arrows move focus; `h/l` or left/right collapse/expand hierarchical items where meaningful; `Tab` and `Shift+Tab` cycle panels; `gg` first loaded row, `G` last loaded row; `Ctrl+D/U` half-page scroll.
7. All multi-key sequences are scoped to normal navigation, with prefix hints. Escape cancels pending prefix. Text input consumes normal characters and disables action keys.
8. Screen-level `[1..4]` switches Changes view (All, Staged, Not staged, Mixed), only when Changes Workspace focused and not typing.
9. `q` returns from detail to parent; on root, opens explicit Quit control, never Git abort.
10. Escape from dirty planner asks “Discard this unexecuted plan?” with Cancel default. This is an application draft operation, not a Git abort.

## Keymap by scope
| Scope | Key | Command |
|---|---|---|
| Global navigation | `Enter` | Open item |
| Global navigation | `Space` | Toggle selection |
| Global navigation | `S` | Select/Deselect all files listed |
| Global navigation | `Esc` | Dismiss/back |
| Global navigation | `j/k`, arrows | Navigate |
| Global navigation | `Tab` / `Shift+Tab` | Next/previous panel |
| Global navigation | `a` | Contextual menu |
| Global navigation | `:` | Palette |
| Global navigation | `?` | Help |
| Global navigation | `gg`, `G` | Start/end loaded |
| Global navigation | `Ctrl+D/U` | Half-page scroll |
| Changes | `1/2/3/4` | All/Staged/Not staged/Mixed |
| Changes | `s` | Stage focused-or-selected change scope |
| Changes | `u` | Unstage focused-or-selected change scope |
| Changes | `d` | Open discard confirmation; do not execute until explicit confirmation |
| Changes | `c c` | Commit composer |
| Changes | `c a` | Amend composer |
| History | `c p` | Cherry-pick planner |
| History/branch | `r i` | Interactive rebase planner |
| History | `d c` | Compare selected commit pair (valid pair only) |
| Normal | `o` | Open operation dashboard if operation exists; otherwise show “No active operation” |
| Operation dashboard | `c` | Continue (validation-gated) |
| Operation dashboard | `s` | Skip (where supported, confirmation) |
| Operation dashboard | `A` (Shift+a) | Abort (confirmation) |
| Operation dashboard | `r` | Refresh operation state |
| Cherry-pick planner | `Shift+J / Shift+K` | Move focused queue item down/up |
| Cherry-pick planner | `p` | Open plan preview |
| Rebase planner | `Shift+J / Shift+K` | Move focused todo row down/up |
| Rebase planner | `p/r/e/s/f/d` | Pick/Reword/Edit/Squash/Fixup/Drop on focused todo |
| Rebase planner | `v` | Preview plan |
| Conflict resolver | `1/2/3` | Choose ours/theirs/both for supported text region |
| Conflict resolver | `n/N` | Next/previous conflict region |
| Conflict resolver | `e` | Open configured external editor |

## Exact focus and actions
- Focus moves independently of selection; Enter does not deselect.
- `Space` toggles membership for current item only; does not stage/unstage.
- In a selectable list with selection, `s/u` target selected items of the proper domain; when no selection, focused item. Show target count before bulk mutation if destructive or large.
- Hidden selected paths/commits survive filtering and pagination, and UI shows hidden count and target scope before mutation.
- Multi-selection represents full object IDs, never display row index/abbreviated hash as identity. For files use repo-relative safe path plus index/worktree status version context.
- Cross-repository selection is isolated; invalid/stale targets block execution until refreshed/reselected.
- A planner captures an explicit immutable selection snapshot on entry. Subsequent navigation cannot silently alter an active plan.
- Up/down reordering changes the queue/todo only, never Git until execution.

## Inputs, overlays, and precedence
Key event order: focused text input/editor → topmost overlay → active planner/workflow → focused panel → global. Modal dialogs trap focus. Never dispatch a key to an underlying screen after modal consumes it. Menus show shortcut hints and disabled reasons.

## Action registry contract
Each action has ID, title, valid scope, selection predicate, shortcut(s), availability predicate with reason, review/confirmation policy, handler, and post-action refresh policy. Menu and shortcut dispatch SAME ID. No duplicate behavior paths.

## Direct shortcut / menu pair examples
- selected history commits → `c p` OR `a` > “Cherry-pick selected” → S06.
- choose history base → `r i` OR `a` > “Interactive rebase” → S07.
- conflict dashboard → `A` OR `a` > “Abort operation” → S10.
- Changes file → `s` OR `a` > “Stage” → index update.
- Changes file → `u` OR `a` > “Unstage” → index update.

## Key ambiguity rules
- `c p` only resolves in History normal mode, not within composer or operation dashboard.
- `p` on Rebase planner = pick; `p` on Cherry-pick planner = preview.
- `a` always opens context menu in normal navigation; `A` means Abort only inside Operation dashboard.
- `1–4` filters are only in Changes normal mode; `1–3` choices only in Conflict resolver.
- `Shift+J/K` used for queue reordering in planners; when selection extension is implemented in lists, do not introduce shortcut collisions. **MVP supports Space toggle; range selection is not required.**

## Acceptance invariants
- No key can invoke two handlers in one scope.
- Every P0 action is accessible via context menu or palette even when shortcuts are unknown.
- Input fields never trigger Git actions while typing.
- Returning from S02/S04/S05 preserves focus/selection/filter/scroll unless repository mutation invalidated the row.
