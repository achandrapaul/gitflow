# GitFlow v2.3 — Approved changes (29 October 2026)

## Supersedes v2.2
1. **ST03 — No stash when clean:** whole-commit cherry-pick in clean tracked state runs directly; no stash/restore action or prompt. Noncolliding untracked paths alone do not trigger stash.
2. **ST03/C21/H04/K39/UX-N05 — Dirty whole-commit recovery:** S06 stays read-only/browsable when dirty. At deliberate whole-commit Execute, show exact tracked blockers and `s` Stash and Proceed or Esc Cancel. `s` creates a single explicitly authorized tracked-only stash, verifies it, then executes selected whole commits via native Git.
3. **ST03 — Auto-restoration after successful commits:** when the entire cherry-pick queue has completed as real commits (and no sequencer/dirty-state blocker remains), automatically restore the exact workflow-created stash and staged state where supported. Do not auto-restore after incomplete/aborted cherry-picks. Preserve the stash until apply success is verified; clean up only its validated identity.
4. **ST03/O07/O08 — Unstash conflicts:** `c` Continue unstash proceeds through existing conflict resolver; `a` Abort unstash explicitly and safely undoes only restoration changes back to the recorded post-cherry-pick state, preserving committed cherry-picks and original stash. If rollback cannot be proven safe, refuse it and retain state. `Esc` always navigates; no native `stash --abort` or forced destructive reset.
5. **C24 scope preserved:** selective historical file application is not a committed cherry-pick; it stays clean-tracked-only with no stash-and-proceed choice.
6. **Tests/traceability:** corrected stale T90, updated T148/T149, added T161–T170 and mappings for ST03/affected existing stable IDs. No test execution or performance measurement is claimed.

## Preserved decisions
- ST02 Stash and Switch is separate and never auto-restores; S16 stash list remains read-only.
- `a m` create from local main with optional safe fast-forward update, explicit post-creation switch, global/contextual shortcuts, exact hidden selections, persisted navigation, all existing rebase/commit/upstream workflows and S01–S16 remain unchanged.
- General stash management, new screens, unrelated feature additions remain excluded.
- This archive is the normative contract plus verification plan, not application source or evidence of passing tests.
