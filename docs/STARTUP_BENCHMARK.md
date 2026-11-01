# Startup benchmark (T46)

Target: 100 ms to first interactive frame (spec: Product, scope, and outcomes). This is a measurement on one machine, not a guarantee.

## Interactive frame definition
The first frame written to the terminal that contains `GitFlow · S01`. Bubble Tea reads keys from the moment that frame is drawn. The frame needs no Git status, history or network work. The only Git process before it is `git rev-parse` for repository discovery. Since Slice 02, status, base, freshness and paused-operation reads start from `Init` after the frame and fill the header and file list when they finish.

## Method
- Build: `go build -o bin/gitflow ./cmd/gitflow` and `go build -o bin/startbench ./tools/startbench`
- Tool: `bin/startbench -bin bin/gitflow -runs 31`, invoked 10 times
- Each invocation generates a fresh fixture repository with 500 commits (fast-import) and 40 modified files. It then launches the binary 31 times in a 120×36 pty with `TERM=xterm-256color`, answers the terminal colour and cursor queries, and stops the clock when the frame marker appears.
- Cold: the first launch in each fresh fixture. 10 samples. The OS page cache is **not** purged (that needs root), and the binary had already been executed once.
- Warm: the remaining 30 launches per invocation, 300 ms apart. 300 samples in total. The tool reports p50/p95 per invocation, not pooled.

## Results (2026-10-30, Slice 01)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.3 / git 2.56.0 |
| Cold samples (ms) | 31.9, 32.4, 32.8, 32.9, 33.0, 33.2, 33.3, 33.8, 34.1, 34.4 |
| Cold p50 / p95 | 33.1 ms / 34.4 ms |
| Warm p50, per invocation | 61.3 – 68.8 ms |
| Warm p95, per invocation | 69.1 – 78.4 ms |
| Warm max, all 300 samples | 83.4 ms |
| Result | Every sample is below 100 ms |

Warm launches are consistently slower than the first launch in a fixture. This repeats the pattern of the earlier prototype and is not explained yet.

## Results (2026-10-30, Slice 02)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.3 / git 2.56.0 |
| Cold samples (ms) | 31.4, 31.8, 32.1, 32.1, 32.2, 32.4, 32.4, 32.4, 32.8, 33.9 |
| Cold p50 / p95 | 32.3 ms / 33.9 ms |
| Warm p50, per invocation | 56.0 – 68.0 ms |
| Warm p95, per invocation | 69.6 – 75.2 ms |
| Warm max, all 300 samples | 88.8 ms |
| Result | Every sample is below 100 ms |

The frame marker is the S01 shell, so these numbers do not include the time until status rows appear.

## Results (2026-10-30, Slice 03)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.3 / git 2.56.0 |
| Cold samples (ms) | 31.8, 32.1, 32.3, 32.4, 32.6, 32.8, 33.6, 33.9, 34.5, 34.7 |
| Cold p50 / p95 | 32.7 ms / 34.7 ms |
| Warm p50, per invocation | 63.7 – 66.8 ms |
| Warm p95, per invocation | 69.5 – 74.7 ms |
| Warm max, all 300 samples | 83.2 ms |
| Result | Every sample is below 100 ms |

## Results (2026-10-30, Slice 04)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.3 / git 2.56.0 |
| Cold samples (ms) | 31.8, 31.8, 31.9, 31.9, 31.9, 32.4, 32.6, 33.0, 33.1, 33.2 |
| Cold p50 / p95 | 32.2 ms / 33.2 ms |
| Warm p50, per invocation | 55.7 – 67.3 ms |
| Warm p95, per invocation | 69.1 – 79.5 ms |
| Warm max, all 300 samples | 93.8 ms |
| Result | Every sample is below 100 ms |

## Results (2026-10-30, Slice 09)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.3 / git 2.56.0 |
| Cold samples (ms) | 32.0, 32.0, 32.0, 32.1, 32.4, 32.5, 32.6, 32.8, 33.3, 33.5 |
| Cold p50 / p95 | 32.5 ms / 33.5 ms |
| Warm p95, per invocation | 71.7 – 76.7 ms |
| Result | Every reported cold and warm p95 value is below 100 ms |
| Excluded | One extra run straight after the rebuild measured 664.8 ms cold: the first execution of a new binary (see Not measured) |

## Results (2026-10-31, Slice 10)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.3 / git 2.56.0 |
| Cold samples (ms) | 32.1, 32.3, 32.3, 32.7, 32.7, 33.0, 33.1, 33.2, 34.3, 34.6 |
| Cold p50 / p95 | 32.9 ms / 34.6 ms |
| Warm p95, per invocation | 71.2 – 73.9 ms |
| Result | Every reported cold and warm p95 value is below 100 ms |

## Results (2026-11-01, Slice 11)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.3 / git 2.56.0 |
| Cold samples (ms) | 31.4, 32.0, 32.0, 32.4, 32.9, 33.0, 33.4, 33.7, 34.8, 35.1 |
| Cold p50 / p95 | 33.0 ms / 35.1 ms |
| Warm p95, per invocation | 72.0 – 75.4 ms |
| Result | Every reported cold and warm p95 value is below 100 ms |

## Results (2026-11-01, Slice 12)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.3 / git 2.56.0 |
| Cold samples (ms) | 32.1, 32.4, 32.8, 33.0, 33.1, 33.5, 33.5, 34.2, 34.4, 43.7 |
| Cold p50 / p95 | 33.3 ms / 43.7 ms |
| Warm p95, per invocation | 64.0 – 74.5 ms |
| Result | Every reported cold and warm p95 value is below 100 ms |
| Excluded | A first set of 10 runs printed only the warm max (64.9 – 80.9 ms), so its cold and p95 values were not recorded |

## Results (2026-11-01, Slice 13)
| Item | Value |
|---|---|
| Machine | Apple M1, darwin/arm64, Darwin 25.6.0 |
| Go / Git | go1.27.3 / git 2.56.0 |
| Cold samples (ms) | 610.8, 32.6, 31.6, 32.5, 32.0, 32.4, 31.8, 32.5, 32.2, 31.2 |
| Cold p50 / max, runs 2–10 | 32.2 ms / 32.6 ms |
| Warm p95, per invocation | 68.7 – 74.0 ms |
| Result | Every warm p95 and every cold value from runs 2–10 is below 100 ms |
| Excluded | Run 1 cold (610.8 ms) was the first launch of the freshly built binary, which includes the macOS security assessment listed under Not measured; its warm p95 was 71.0 ms |

## Not measured
- Linux: no Linux machine or container was available.
- True cold start with purged caches.
- The first execution of a freshly built binary. macOS runs a security assessment of a new executable (about 760 ms in an earlier session) that is outside the app's control.
- Terminals that never answer the colour query.
- Time until the status, header and file list are loaded after the first frame.
