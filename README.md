# GitFlow — Clean v2.3 Build Handoff

**Current single source of truth:** `spec/GITFLOW_REQUIREMENTS_V2_3.md` (approved decisions, 29 October 2026). Older normative versions are excluded. This package contains one active spec and matching current verification artifacts.

## Start here
1. `spec/GITFLOW_REQUIREMENTS_V2_3.md` — sole normative product, keyboard, UX and Git contract.
2. `CHANGELOG_V2_3.md` — approved changes from v2.2, with preserved decisions.
3. `tests/ACCEPTANCE_V2_3.md` and `tests/SCENARIOS_V2_3.md` — checks and integration fixtures.
4. `tests/TRACEABILITY_V2_3.md` — stable IDs to tests, including evidence status.
5. `CLAUDE.md` — implementation instructions.

## v2.3 cherry-pick behavior
- **Clean tracked state:** Whole-commit cherry-pick executes directly, without any stash/unstash operations.
- **Dirty tracked state:** `g c` browsing stays read-only. At whole-commit Execute show affected paths and `s` **Stash and Proceed** or `Esc` **Cancel**. Only explicit `s` authorizes tracked-only stash creation and the reviewed cherry-pick sequence.
- **After all commits succeed:** Automatically restore original tracked changes and staged state where supported, using the exact workflow-created stash. No restore while native cherry-pick is paused/aborted.
- **Restore conflicts:** `c` **Continue unstash** to S09/manual conflict resolution, or `a` **Abort unstash** to confirmed, safely verified rollback of only attempted restoration. Preserve original stash and committed cherry-picks; Esc only navigates.
- **Selective historical-file application:** Never uses the new stash-and-proceed flow; still requires a clean tracked index/worktree before staging selected patches.
- **No extra screens or general stash management:** S16 remains read-only; branch-switch stash recovery (ST02) stays unchanged and never auto-restores.

## Unchanged v2.2 behavior
`a m` create from main and FF-update/direct choice, `g n`/`a n` local creation, explicit `s`/Enter switch prompt, branch-switch ST02 recovery, exact total/hidden selection review, navigation continuity, global `g`/contextual `a` shortcuts, native staging/commit/rebase/paused-op contracts, 100ms first-frame target, and all S01–S16 screen IDs remain in force.

## What this package is not
This is a **contract and test-plan handoff**, not executable app source, executed integration tests or a validated performance benchmark. Only approved operations may be implemented.
