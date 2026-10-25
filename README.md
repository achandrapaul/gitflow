# GitFlow — Claude build handoff

**Deliverable:** strict requirements, keyboard contract, screen designs, step-by-step interaction walkthroughs, Git semantics, release scope, acceptance tests, and a standalone clickable/keyboard-preview HTML file.

## Start here
- `CLAUDE.md` — instructions intended for Claude Code, including prohibition on additional features.
- `spec/00_PRODUCT_CONTRACT.md` — baseline product decisions and P0 scope.
- `spec/01_KEYBOARD_AND_ACTIONS.md` — exact keys, conflicts, focus/selection semantics.
- `spec/02_GIT_SEMANTICS.md` — index/worktree, upstream/base, native Git operations.
- `spec/03_WORKFLOWS.md` — explicit entry/exit paths and Git command lifecycle.
- `spec/04_SCOPE_AND_OPEN_DECISIONS.md` — frozen scope, intentionally unresolved details and implementation selection rules.
- `screens/SCREEN_SPECS.md` — terminal wireframes for all screens, including empty/error states.
- `screens/WALKTHROUGHS.md` — precise keystroke-by-keystroke journeys; no mouse is part of the product.
- `tests/ACCEPTANCE.md` — individual acceptance criteria.
- `tests/SCENARIOS.md` — fixture/integration test scenarios.
- `prototype/index.html` — open locally in a browser to explore representative screens; **not** implementation code, **not** a real Git tool.

## Vocabulary
- **Focused**: one row currently highlighted by keyboard navigation. Not selected, staged, or executed.
- **Selected**: UI-only membership toggled via Space; may include hidden-by-filter rows.
- **Staged**: actual index difference from HEAD. Git-native status.
- **Not staged**: file with changes or untracked content, but **no** index change.
- **Mixed**: file has index change AND additional worktree change.
- **Upstream**: tracking branch configured for the current local branch.
- **Comparison base**: separately configured local ref, default `refs/remotes/origin/main` when available.
- **Resume dashboard**: reopen the UI, no Git mutation.
- **Continue**: invoke the active operation's native `--continue` when valid.
- **Skip/Abort**: explicit Git mutations requiring review/confirmation.

## How to give this ZIP to Claude
Extract the ZIP into a project working directory and tell Claude Code: **“Read `CLAUDE.md` and implement exactly the bundled GitFlow specification. Do not introduce extra features. Maintain a requirement-to-test coverage matrix and report decisions or blockers.”**

Sample filenames and counts across the screen prototypes are illustrative only. Product screens must draw values from the real repo.


## Approved v1.1 update
**Read `spec/05_INTELLIJ_STYLE_GIT_WORKSPACE.md`**. This adds IntelIiJ-inspired history-to-file navigation, reviewed pull/push, and file-selective commit patch application. The latter is intentionally **not** the same Git operation as cherry-picking a whole commit. The added screens S13–S15 are binding.
