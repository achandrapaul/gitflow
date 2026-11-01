# Slice 1 — App Launch + Commit Explorer

Contract: `spec/GITFLOW_REQUIREMENTS_V2_3.md` and the repo `CLAUDE.md`. This slice adds no Git mutation path.

## Goal

Launch into a real repository and explore its working state: the S01 changed-file list, S02 diff, branch header, navigation, palette and help. Every later key is registered but unavailable with a stated reason.

## Functionality

- Native Git runner with explicit repo cwd and safe argv; porcelain v2 status parsing.
- Branch header (B06): branch or detached HEAD, upstream, ahead/behind from local refs only. No implicit fetch.
- S01 Changes Explorer: rows with XY status, binary marker, filters `1`–`4` (All / Staged / Not staged / Mixed), Flat/Folder view with saved preference.
- Focus and selection (C18/C20, UX-N03): `j`/`k`/arrows move focus only; Space toggles; Shift+S selects all; `G` last row; `gg` unbound.
- S02 Diff: staged and unstaged diff of the focused file, binary placeholder, read-only.
- Shared action registry, dispatch precedence (text input → overlay → screen → global), `g` global and `a` contextual prefixes, case-sensitive second key.
- Esc is navigation-only. `a e`, `gg` and `H` stay unbound.
- S12 palette and help; footer with counts and available keys.
- Clean repo: commit shown disabled with its reason.
- Small-terminal stacked layout with a defined minimum size.
- Paused-operation badge, read-only, never steals focus (the S08 dashboard lands in slice 3).
- Navigation continuity read path (UX-N07): focus, scroll, filter and selection survive S01 ↔ S02 ↔ S12.
- First interactive frame measured against the 100ms target, with hardware and method recorded.

## Requirement IDs (primary)

| Area | IDs |
|---|---|
| Branches (header, read-only) | B01–B06 |
| Changes Explorer | C01–C07, C09, C13, C15–C20 |
| Keys and dispatch | K01–K09, K35–K38, K41, K42 |
| Learning contract | U01–U03 |
| UX | UX-N01, UX-N03, UX-N07 (read path) |

Partial here: K33 (read-only key registration only; primary in slice 3).

## Tests

Primary: T01–T05, T08, T11–T13, T26, T29–T32, T46, T51–T54, T57, T58, T88, T107, T111–T113, T131.

Partial (only the slice-1 part is asserted; traceability says so):

| Test | Part asserted here | Primary slice |
|---|---|---|
| T84, T86, T87 | key registered, unavailable with reason | 2 |
| T106 | S01 `v` behaviour | 3 |
| T108, T109, T147, T150 | selection and footer | 2 |
| T144 | read-only part | 3 |
| T159, T160 | navigation part | 3 |

## Contract guards

- No code path in this slice runs a mutating Git command.
- B06 header never fetches; no network call just to draw the UI.
- A key whose slice has not landed is registered with `Available=false` and the reason "Not available yet". It never binds to other behaviour.
- `a e` stays unassigned; `H` stays retired; `gg` stays unbound.
- Esc never resets, aborts or discards.
- The 100ms first frame is reported only from a real measurement.

## Out of scope

Stage/unstage, commit, fetch/pull/push (slice 2). History, cherry-pick, rebase, conflicts, branches, stash, discard/delete (slice 3).
