# Slice 3 — History, Cherry-pick, Rebase, Branches, Stash and the rest

Contract: `spec/GITFLOW_REQUIREMENTS_V2_3.md` and the repo `CLAUDE.md`. Builds on slices 1 and 2.

## Goal

Everything left in the v2.3 contract, including the ST03 dirty cherry-pick recovery, and populating traceability with real results.

## Functionality

- History S04 / S13 (`g h`, H01–H08, U07): initial 100 commits with prefetch, pagination, search scope, compare, merge-parent choice, commit file inspector.
- S05 actions overlay (K39, K40).
- Cherry-pick Explorer S06 (`g c`, C21, UX-N05): read-only entry even when dirty; two mutually exclusive modes, whole commits or apply files.
- Selective apply S13/S14 (C24): clean-only; validated parent→commit patches applied to index and worktree, staged, no auto-commit.
- ST03 whole-commit dirty recovery:
  - Clean tracked state executes directly; no stash.
  - Dirty shows exact paths and ordered commits; `s` Stash and Proceed, Esc cancels with no mutation.
  - `git stash push` tracked-only (no `-u`/`-a`), stash OID captured, clean state verified before cherry-pick.
  - On failure or pause keep the stash and use S08.
  - After all commits succeed, no sequencer and safe tracked state: `stash apply --index` of that exact stash.
  - Drop only the verified GitFlow stash; if the drop fails, say it was retained.
  - Restore conflict: `c` continues into S09, `a` rolls back the restore attempt only when safe, otherwise blocked.
- S07 interactive rebase todo; S08 paused-operation dashboard; S09 conflict resolver with 3-way view and ref+role labels (O10, U04); `g r` pull --rebase.
- Branches S11 (`g b`, S01 `b` alias): search, Enter inspects, `s` switch (B07, K34, UX-N04).
- `g n` / `a n` local branch creation (B08, K44); `a m` create from local `main` with `p` FF-only update-and-create, `c` direct create, Esc cancel (B10, K47).
- Switch prompt after creation: `s`/Enter switch, Esc stay (B11).
- ST02 Stash and Switch when a switch is blocked by tracked changes: confirmed, tracked-only, never restored automatically.
- S16 read-only stash list (`g s`, ST01, K46).
- C14 Delete file from disk: S01 `a` action for exactly one eligible file, reviewed.
- Correct T90, T148, T149; add T161–T170.
- Populate `tests/TRACEABILITY_V2_3.md` with real modules and results only.

## Requirement IDs (primary)

| Area | IDs |
|---|---|
| History and cherry-pick | H01–H08, U07, K33, C21, C24, K39, K40, UX-N05 |
| Stash | ST01–ST03 |
| Rebase, paused operations, conflicts | Section "Interactive rebase, paused operations and conflict resolution", O10, U04 |
| Branches | B07, B08, B10, B11, K34, K44, K46, K47, UX-N04 |
| Changes | C14 |

## Tests

Primary: T14–T25, T27, T28, T47–T50, T55, T56, T85, T89–T101, T105, T106, T114–T120, T127, T128, T133, T135–T139, T144–T146, T148, T149, T151–T170.

Also asserted here: T110 paused-operation part (primary slice 2).

## Contract guards

- ST03 applies only to whole-commit cherry-picks. S13/S14 stays clean-only and staged-without-commit.
- Never stash from preview, silently, or to fix unmerged, sequencer or untracked-collision blockers.
- Never pop before validating the restore; restore never auto-commits and is not a sequencer.
- ST02 never auto-restores. S16 never applies, pops or drops.
- No arbitrary stash apply/pop/drop commands and no new screen beyond S01–S16.
- `a m` targets local `main`; never pulls `main` into a feature branch.
- Branch creation never switches, sets upstream or pushes on its own.
- Native rebase only; conflict labels show the real ref and role.
- Esc never resets or aborts Git.

## Out of scope

Anything not in the v2.3 contract: PR/hosting APIs, CI, issue trackers, general stash management.
