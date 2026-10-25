# 04 — Strict boundaries, defaults, and unresolved decisions

## Absolutely out of MVP
Do not implement custom Git hosting API, PR reviews, CI, WIP automation, stash manager, worktree manager, bisect, submodule manager, release/tag builder, reflog recovery center, dark/light theme editor, plugins, mouse handlers, AI commit generation, cloud login, persistent background daemon, command suggestions learned from usage, or telemetry. Display native Git data relevant to listed workflows but do not create new flows for these topics.

## Frozen defaults
- App name in docs: `GitFlow` (binary name `gitflow`).
- Startup screen: S01 Changes Workspace.
- Base ref: origin/main if present, else local main, else unavailable; user can explicitly change through S11, but do not infer another base.
- Commit history first fetch batch: 100; prefetch before bottom; virtualized viewport.
- Four filters exactly: All, Staged, Not staged, Mixed. Staged includes Mixed. Not staged excludes Mixed. Mixed requires both index and worktree changes.
- TUI only, keyboard only, macOS/Linux first, no mouse and no desktop GUI.
- Default shortcut labels and contexts are in spec/01.
- No automatic `git fetch`, no auto-rebase, no auto-force-push.
- Settings storage for comparison ref and keymap only if necessary; customizable keymap is P1. Avoid a settings screen in MVP.

## Items implementation team must resolve without adding visible features
1. **Go vs Rust**: decide with ADR citing start latency, TUI library maturity, Git subprocess reliability, tests, packaging. Both allowed; no product-language mandate.
2. **100ms hardware target**: record CPU/OS/terminal and warm/cold test methodology; report failure rather than hide.
3. **Partial line staging**: specify supported forms and test; show disabled with reason if patch cannot safely apply.
4. **Native interactive rebase adapter**: use Git-supported editor/sequence mechanisms safely and portably; never silently change user's global git config.
5. **Multi-commit cherry-pick atomicity**: Git operations are not transactionally atomic; report step-by-step state and real recoverability.
6. **Merge conflicts**: display merge operation and continue/abort; skip unsupported. New merge planning UI is out of scope.
7. **Tool fallback**: conflict external editor uses user's configured Git editor; if missing, present actionable error, not an invented built-in full IDE editor.
8. **Branch “freshness”**: if fetched-at timestamp cannot be trusted, show “Local refs, fetch time unknown.” Never invent last-fetch value.
9. **Terminal sizing**: document and test minimum; degrade to a stacked view without removing actions or inventing a new navigation tree.

## Decision/change management
If Claude believes a requirement cannot be implemented as written, put the issue into `OPEN_QUESTIONS.md` with exact reference IDs, alternatives, implications, and its current blocked status; do NOT introduce an unsolicited feature. For any proposed feature not in P0, implement NOTHING until user expressly approves it. Existing baseline requirements beat convenience.
