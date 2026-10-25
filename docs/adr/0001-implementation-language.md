# ADR 0001 — Implementation language: Go

Status: Accepted (2026-10-26)

## Context
`spec/04_SCOPE_AND_OPEN_DECISIONS.md` item 1 requires a documented choice between Go and Rust based on start latency, TUI library maturity, Git subprocess reliability, tests and packaging. CLAUDE.md rule 10 requires a short prototype or proof of suitability.

## Prototype
A throwaway Bubble Tea program (alt screen, one Lip Gloss bordered panel, `git --no-optional-locks status --porcelain=v2 --branch -z` loaded asynchronously from `Init`) was launched 21 times through `tools/startbench` in a 120×36 pty on a generated 500-commit fixture with 40 modified files.

| Measure | Result |
|---|---|
| Machine | darwin/arm64, Darwin 25.6.0, Apple M1, Go 1.27.1, git 2.56.0 |
| Cold (first launch, OS cache not purged) | 27.4 ms |
| Warm p50 / p95 (20 runs) | 22.3 ms / 24.1 ms |

Finding: Bubble Tea v1 (`tea_init.go`) queries the terminal background colour (OSC 11 + DSR) at package init and blocks up to `termenv.OSCTimeout` (5 s) when the terminal does not answer. Real terminals answer within one round trip; `tools/startbench` replies like a terminal so the measurement excludes that round trip. Terminals that never answer OSC 11 would stall startup; recorded in `OPEN_QUESTIONS.md` (Q9).

Rust was not prototyped: no Rust toolchain is installed on the reference machine, and rule 10 forbids expanding scope to compare stacks. The Go prototype already meets the 100 ms target with a wide margin.

## Decision
Go with Bubble Tea v1.3.10, Bubbles v1.0.0 (text input / textarea) and Lip Gloss v1.1.0.

| Criterion | Assessment |
|---|---|
| Start latency | ~22 ms warm to first frame in prototype; status loads asynchronously after the frame |
| TUI maturity | Bubble Tea/Lip Gloss/Bubbles are widely used (lazygit-class tools), Elm-style update loop maps directly onto the key-precedence contract |
| Git subprocess reliability | `os/exec` takes argv arrays (no shell), explicit `Dir`, separate stdout/stderr, context cancellation; `tea.ExecProcess` hands the terminal to Git for editors, hooks, pinentry and credential prompts |
| Tests | `go test` with real temporary repositories; the TUI model is a plain value driven by key messages, testable without a pty |
| Packaging | single static binary per GOOS/GOARCH (`darwin`/`linux`, `amd64`/`arm64`) |

## Consequences
- Git calls that may need the terminal (commit, amend, cherry-pick, rebase start/continue/skip, merge continue, fetch, pull, push, external editor) run through a terminal handoff so hooks, signing pinentry, credential helpers and `GIT_EDITOR` behave natively. Their stderr is also captured and shown in the TUI.
- Read-only calls use `git --no-optional-locks` so refreshing never takes `index.lock`.
- Only Go is used in product source; `prototype/index.html` remains a visual reference only.
