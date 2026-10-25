# Startup benchmark (N01, T30)

Target: 100 ms to first interactive frame (CLAUDE.md rule 9). This is a measurement on one machine, not a guarantee.

## Method
- Tool: `go run ./tools/startbench -bin bin/gitflow -runs 31`
- Launches the binary in a 120×36 pty with `TERM=xterm-256color`, answers the terminal colour query the way a terminal does (see ADR 0001), and stops the clock when the first frame containing `CHANGES` is written. That frame is interactive; status, branch, base and operation data fill in asynchronously afterwards.
- Fixture: generated repository with 500 commits (fast-import) and 40 modified files.
- Cold: first launch in the fresh fixture. The OS page cache is **not** purged (needs root); the binary had already been executed once.
- Warm: the following 30 launches, 300 ms apart so processes from the previous launch can settle.

## Results (2026-10-26)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.1 / git 2.56.0 |
| Cold | 31.8 ms |
| Warm p50 | 52.5 ms |
| Warm p95 | 55.6 ms |
| Warm max | 59.4 ms |
| First-ever execution of a freshly built binary | ~760 ms before any output (macOS first-execution security assessment of the new executable; observed once per build, outside the app's control) |

Observed timeline on a warm launch (Python pty trace): terminal setup bytes at ~24 ms, first frame with placeholders (`Loading status…`) at ~41 ms, status filled at ~57 ms.

## Not measured
- Linux: pending (no Linux machine or running container available in this session).
- True cold start with purged caches.
- Terminals that never answer OSC 11 (would stall up to 5 s, see OPEN_QUESTIONS Q9).
