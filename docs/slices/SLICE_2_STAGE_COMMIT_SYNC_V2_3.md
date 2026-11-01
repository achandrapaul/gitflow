# Slice 2 — Stage + Unstage, Commit, Pull, Push

Contract: `spec/GITFLOW_REQUIREMENTS_V2_3.md` and the repo `CLAUDE.md`. Builds on slice 1.

## Goal

The everyday write loop: stage and unstage, commit with hooks honoured, then fetch, pull and push through a reviewed `g` menu.

## Functionality

- Stage and unstage at file, hunk and line level from S01 (`s`/`u`) and S02 (C08, C12).
- S03 Composer: one inline subject/body draft; `c` focuses it; review runs against the current staged index and goes stale when the index changes (C10, C11).
- Commit integrity (C22): record the reviewed staged-tree OID and message, compare with the actual commit after native hooks.
- `a v` (C23, K43): one-shot `--no-verify` through a warned review, never instant.
- Amend as an `a` action; never auto-pushes.
- S15 `g` menu with Review / Execute / Cancel, Cancel as the default:
  - `g e` fetch, `g p` configured pull, `g f` FF-only pull.
  - `g P` push, `g F` force-with-lease (never falls back to `--force`).
  - `g u` / `a u` upstream configuration (B09, K45).
- First push with no upstream reviews and uses `--set-upstream`; `origin/<branch>` is set only by that confirmed, successful push.
- No-upstream state shows the blocked reason instead of running.
- S10 Confirm for reviewed actions.
- After every mutation (U05, UX-N06): serialise, refresh real state, keep focus/scroll/filter/selection, show an inline outcome. Blocked actions show the reason, exact paths and next steps.

## Requirement IDs (primary)

| Area | IDs |
|---|---|
| Changes and commit | C08, C10–C12, C22, C23 |
| Keys | K30–K32, K43, K45 |
| Branches | B09 |
| Learning contract | U05, U06 |
| UX | UX-N02, UX-N06 |

## Tests

Primary: T06, T07, T09, T10, T84, T86, T87, T102–T104, T108–T110, T121–T126, T129, T130, T132, T134, T140–T143, T147, T150.

Partial:

| Test | Part asserted here | Primary slice |
|---|---|---|
| T139 | `g u` | 3 |
| T159, T160 | commit/sync part | 3 |

T110's paused-operation part is asserted in slice 3.

## Contract guards

- Hooks and signing are honoured; `--no-verify` only via a reviewed `a v`, one commit only.
- No implicit fetch, rebase, force push, checkout, stash, discard or delete.
- `g r` (pull --rebase) stays unavailable; S08 is not entered from this slice.
- Upstream is never set except by a confirmed, successful first push or explicit `g u`/`a u`.
- `main` is never updated by pulling it into a checked-out feature branch.
- Esc cancels the review without Git mutation.

## Out of scope

`g r`, history, cherry-pick, rebase, conflicts, branch creation and switching, stash, discard/delete (slice 3).
