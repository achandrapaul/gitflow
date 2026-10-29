# GitFlow — v2.3 Integration and UX Scenarios (current only)

Run in temporary Git repositories and local bare remotes. Never mutate a user's repository. Assert actual Git index, worktree, HEAD, branch refs, upstream, and sequencer state. Where relevant, capture command stdout/stderr and reproduce the test fixture. These are the only active test scenarios; no old-version shortcut requirements are included. Existing IDs are retained for traceability of still-valid behavior.

## Core Git integrity, staging, history, safety and startup
| ID | Fixture and action | Assert |
|---|---|---|
| T01 | Clean repo, launch | Interactive S01, no changed rows, commit disabled, branch info present |
| T02 | Modified tracked file, no staging | All+Not staged, not Staged/Mixed |
| T03 | Staged tracked file, unchanged after | All+Staged, not Not staged/Mixed |
| T04 | Stage hunk A then edit hunk B | Exactly one Mixed row; staged/unstaged diffs contain correct portions |
| T05 | Untracked file | All+Not staged, never Staged unless explicitly `git add` |
| T06 | Stage mixed file remaining worktree changes | Index updated, file becomes Staged only, untouched patches preserved |
| T07 | Unstage staged hunk | Working copy preserved, index correctly updated |
| T08 | Focus/filter/selection navigation through S02 | Git byte-for-byte unchanged; hidden selections retained |
| T09 | Commit staged portion of mixed file | Commit only index; worktree retains extra unstaged changes |
| T10 | Commit hook rejects | No false success; input draft retained; accurate status |
| T11 | No upstream branch | “No upstream” and no computed upstream zero counts |
| T12 | Feature upstream differs from origin/main | Two independent ahead/behind pairs correct |
| T13 | Detached HEAD, missing base | Clear detached/base unavailable states |
| T14 | >1000 commits and merge graph | Incremental pages stable, no duplicate IDs, selection persists |
| T15 | Noncontiguous commits cherry-picked/reordered | Result reflects explicit queue order or paused conflict accurately |
| T16 | Cherry-pick conflict then Esc | Sequencer state intact after UI navigation |
| T17 | Cherry-pick interrupted, close/reopen app | S08 detects actual state; continue/skip/abort available as permitted |
| T18 | Interactive rebase reorder/fixup | New Git history reflects plan; invalid first-row fixup rejected |
| T19 | Interactive rebase conflict/edit stop | Correct pause reason, no fake conflict at edit stop |
| T20 | Rebase abort confirmation Esc | No Git ref/worktree mutation from Esc; sequencer still exists |
| T21 | Rebase abort confirmed | Native abort attempted; outcome checked and disclosed |
| T22 | Merge conflict active | Continue/Abort shown, Skip disabled |
| T23 | Resolution choose each supported conflict side labeled by actual branch/ref+role or commit OID, and valid base where available | Correct stage/content mapping; no fictitious `Both` option, file content and index verified before explicit Continue |
| T24 | Binary conflict | No invalid three-way text side selection |
| T25 | Other process changes HEAD during planner | Execute invalidates stale plan, no mutation until revalidation |
| T26 | Filenames with spaces, unicode, unusual characters | No broken path parsing or unsafe argv |
| T27 | Empty cherry-pick | Proper paused/explanation/skip behavior, no bogus success |
| T28 | Root Esc and quit while paused | Quit UI leaves Git operation untouched |
| T29 | Keyboard typing in commit message | `s`, `u`, `a`, `c`, digits typed as chars, no actions |
| T30 | S01 filter 3 with staged/mixed/untracked paths; press `Shift+S` twice | All unified changed paths selected, including hidden; idempotent; no index/worktree/HEAD/ref changes; counts accurate |
| T31 | After S01 `Shift+S`, switch filters and deselect one with Space | All other selections persist; focused row only toggles; no Git mutation |
| T32 | Input editor, modal, S04 history, S13 commit-file inspector; type/press `Shift+S` | Input receives uppercase S where appropriate, no S01 bulk-selection action outside permitted scope |
| T46 | Launch performance | Measured first interactive frame; benchmark results and machine details retained |

## Current file deletion, folder persistence and learnability
| ID | Fixture and action | Assert |
|---|---|---|
| T47 | Tracked modified file → Delete file action, cancel then explicitly confirm | Cancel no mutation; confirm deletes only working file, INDEX unchanged, real Git status reports worktree deletion |
| T48 | Untracked file → Delete file action | Warn unrecoverable contents; cancel no loss; confirm removes file, no staging or ref change |
| T49 | Staged modified/addition file → Delete file | Popup explains index still staged; after confirmation index byte-for-byte unchanged and worktree status truthful |
| T50 | Multiple selected files, focused folder, conflicted file, submodule, stale/moved path | Delete disabled or fails closed; no recursive/unintended removal; no shell/path escape; symlinks never followed |
| T51 | Same fixture under Flat and By folder, including mixed/renamed/untracked, switch status filters | Same unique file identity/status sets; folder counts reflect displayed files; no Git mutation |
| T52 | By folder, collapse directory and filter out selected files, `Shift+S`, `Space` on folder | All changed FILES selected across collapsed/hidden paths; counts correct; folder Space does nothing; Git unchanged |
| T53 | Press `v` once (Flat→Folder), restart, change repo, press `v` once (Folder→Flat), restart | Persist exact last explicitly chosen mode across restarts/repos; no automatic change |
| T54 | Config file unavailable/write fails/corrupted preference | Graceful fallback and visible diagnostic, no silent claim of persistent choice, app still navigable |
| T55 | `g b` Branches list, `/` search, Enter inspect, Esc return, type b in commit composer | No unexpected checkout; branch refs/index/worktree unchanged; input scope respected |
| T56 | From S01 `a` offers both “Discard changes” and “Delete file from disk” | Distinct actions, correct validations and confirmation; `d` retains discard meaning |
| T57 | Normal user flow launch → inspect → Space → stage → commit; use `?`/`a` | Context labels and shortcuts visible; no hidden required shortcut or multi-screen pre-wizard |
| T58 | Five first-time GitFlow users on same fixture, no shortcut documentation | At least 4 of 5 reach commit review after inspect/stage within 5 min; record elapsed time, friction and failures honestly |

## Current two-key operations, cherry-pick, selection and feedback
| ID | Fixture and user interaction | Required assertion |
|---|---|---|
| T84 | In Changes, `g` followed by `p` | One configured-policy normal pull after preflight; no nested popup; accurate failure if no strategy/upstream. |
| T85 | `g` then `r` / `f` | Rebase pull / ff-only pull respectively; divergence/conflict follows native semantics. |
| T86 | `g P`, then `g e` | Normal push and explicit fetch; real refs updated; no force, extra choice or auto-fetch. |
| T87 | `g F`, then Esc vs `c` | Force lease always guarded; Esc unchanged; `c` validates destination+expected OID; stale lease rejected. |
| T88 | `g g`; uppercase `G`; `g` while text editing | No action/jump from `gg`, `G` last row, text never triggers Git menu. |
| T89 | `g c` with tracked staged file | S06 read-only browsing opens and permits inspection/selection/preview; Execute is disabled with specific dirty-path reason until clean, index and worktree unchanged by browsing. |
| T90 | `g c` with tracked unstaged edits; separately with unmerged entries | Unstaged edits allow read-only browsing/selection/preview; deliberate whole-commit Execute offers `s` Stash and Proceed or Esc Cancel with affected paths; unmerged index blocks execution without offering stash as a fix; ordinary S06 navigation does not mutate Git. |
| T91 | `g c` with only non-colliding untracked file | Explorer opens; file is preserved. |
| T92 | Untracked destination collides with source-cherry-pick file creation | Revalidate before execution; refuse overwrite, untracked bytes unchanged. |
| T93 | Existing paused rebase/cherry-pick/merge then `g c` | New plan blocked; paused state preserved; `o` opens accurate dashboard. |
| T94 | `g c`, change source branch; inspect commits | HEAD/target/current checkout unchanged; only source commit list changes. |
| T95 | History `g h`, select two commits, `g c` | S06 opens with exact full selected OIDs, selection counts and source branch preserved. |
| T96 | S06 whole commits, Enter focused commit | Opens S13 file mode; old whole-commit selection stored but inactive; Space selects only files. |
| T97 | S13 file selections, Esc then return | S06 whole-commit selection restored; file selection remembered but inactive, no Git mutation. |
| T98 | Select commits across two browsed source branches; paginate | Full OIDs, counts, order and source context preserved; no checkout. |
| T99 | S06 preview whole plan `p` then `a` execute | Only whole commits cherry-picked in reviewed order; selected files not applied separately. |
| T100 | S13/S14 file mode `p` preview then `a` Execute after clean-tracked/collision checks | Only selected parent→commit file patches apply to BOTH local index and worktree as STAGED changes; HEAD unchanged, no auto-commit; source-aware composer message prepared without overwriting an existing draft; unsupported cases guarded. |
| T101 | Two selected commits cause cherry-pick conflict partway | Completed count correct, sequencer paused, visible badge; `o` opens S08; Esc never aborts. |
| T102 | S01 `c`, edit draft, leave and return, Commit review | Same single draft and staged-only index snapshot, no second composer; commit explicit. |
| T103 | Modify staged index after composer review | Stale review refused/re-run; no unintended commit or draft loss. |
| T104 | Successful commit versus failing hook/signature, including hook that modifies index | Success verified from actual commit; failure retains draft and displays real index/worktree state (which hooks may have changed), invalidates stale review, reports error. |
| T105 | `g h` then Esc; S11 `g b` then Enter and `s` | History context returns; Enter only inspects; `s` switches eligible local branch with guards, remote cannot auto-switch. |
| T106 | S01 `v`, restart, switch repos; S07 `v` | S01 immediately toggles and saves Flat/Folder, survives restart; S07 still previews. |
| T107 | Mixed tracked file and untracked paths through filters 1–4 | Mixed appears in Staged, Not staged, Mixed, All; untracked in Not staged; one row per file. |
| T108 | S01 check A and B with Space; navigate using `j/k` across C and D; Space toggles C only; revisit after navigation; Esc | `j/k` never changes checked set; toggles only C; stage acts on checked targets only; Esc clears selections, keeps focus and does not mutate Git. |
| T109 | Check 3 changed paths across All; switch to Not staged so 2 checked paths become hidden and one visible focused path is unchecked; invoke Stage after seeing `3 selected · 2 hidden` | All 3 checked paths remain the declared targets (including hidden); focused unchecked path is not included; eligible changes apply, already-staged/ineligible selections are reported truthfully, no silent target substitution; Esc clears all including hidden. |
| T110 | Disabled pull, stale ref, failed network push, paused operation | Each reports exact real blocker/partial state and safe next step; success has no mandatory popup; persistent paused badge visible, `o` recoverable. |

## Execution evidence
For every criterion in `tests/ACCEPTANCE_V2_3.md`, record: Not implemented / Implemented-unverified / Passed / Failed / Blocked; corresponding test ID, source module, Git fixture and actual output. Never count visual prototype behavior as verified Git execution.

## Preserved earlier contract regression fixtures
| ID | Fixture and user interaction | Required assertion |
|---|---|---|
| T111 | Check file A with Space; navigate `j/k` across B, C and back, then check C | A remains checked throughout navigation; B never auto-selected; Space checks only focused C; Git state unchanged until explicit action. |
| T112 | Independently Space-check A, focus C, use Shift+J/K to expand then contract visible range across D/E | Anchor remains stable; only range-derived rows removed on contraction; A remains checked; folder headers never checked; ordinary `j/k` doesn't mutate selection. |
| T113 | Mixed checked/unchecked S01 list with selected file hidden by filter; open a menu, press Esc twice | First Esc closes overlay, retains checked set; second Esc clears ALL S01 file checks and range anchor while retaining focus/filter; no Git mutation. |
| T114 | In S06 Space-check one or more whole commits, inspect with Enter, select explicit whole-commit Execute | Neither Space nor Enter executes; deliberate reviewed Execute uses native cherry-pick into current target, creates expected commits without applying individual-file queue. |
| T115 | Clean tracked repo with noncolliding untracked file; select one safe historical changed file and execute S14 | Exact parent→commit patch applied to both worktree and index, appears staged, untracked preserved, HEAD unchanged; file not merely replaced with historic snapshot. |
| T116 | Select files from multiple historical commits; existing composer draft is nonempty | Accurate editable source-OID/file list provenance is offered; existing draft unchanged until user deliberately chooses replacement; no auto-commit. |
| T117 | Multi-file apply with unsupported rename/binary/overlap or path collision; induce failure after one supported step | Unsupported paths blocked; collision bytes preserved; if partial application occurs report exact completed/remaining items and actual index/worktree; never claim atomic rollback or native cherry-pick abort. |
| T118 | Merge feature/payments into main with a textual conflict | Visible labels `main (current)` and `feature/payments (incoming)` reflect actual Git stage-2/stage-3 content; valid base separately labeled; selected side content verified before mark-resolved. |
| T119 | Rebase feature/payments onto main with conflict | User-facing `main (rebase destination)` and replayed source commit identity match the actual native sides (including reversed intuitive ours/theirs); chosen content correct. |
| T120 | Cherry-pick detached or branch-ambiguous source commit causing conflict | Source labeled by full/short unambiguous commit OID + replayed role; no invented branch; valid side mapping and manual fallback. |
| T121 | Pre-commit hook changes staged index then exits nonzero | Commit failure reported, draft retained, real modified index reread and stale review invalidated; no silent reset/discard or claim index unchanged. |
| T122 | Pre-commit hook stages additional changes and returns zero | If commit succeeds, compare reviewed tree OID and actual new HEAD tree, flag difference and offer inspection of actual created commit, not a false normal-content success. |
| T123 | Commit-msg hook alters message and succeeds | Actual committed message compared to reviewed draft, difference disclosed; no fabricated message equivalence. |
| T124 | Normal commit vs explicit `--no-verify` in repo with pre-commit/commit-msg/prepare-commit-msg hooks | Normal attempt runs normal hooks; single bypass attempt skips only Git-supported pre-commit and commit-msg, still executes prepare-commit-msg, honors signing and other Git protections. |
| T125 | Open `Commit without verification hooks`, cancel/confirm, then attempt a second normal commit | Cancellation never commits; confirmation is required, bypass applies once and is not sticky; next commit defaults to running hooks. |
| T126 | Explicit verification bypass with invalid/stale index, active sequencer or signing failure | GitFlow's existing preflight still blocks invalid action; failed signing is reported, not implicitly disabled by bypass; draft kept. |
| T127 | Cherry-pick ordered commits A,B,C; A succeeds, B pauses; terminate and restart before resolving B | Real Git sequencer reconciles completed A/active B/pending C without applying A twice; unreconcilable app queue is clearly blocked, never guessed. |
| T128 | Create and delete local branches; set/change upstream; dirty and protected conditions | Expected refs/upstream only; native validation and explicit warnings protect current/unmerged work and prevent implicit switch. |
| T129 | Use contextual Amend on local commit and on potentially published commit; cancel and confirm | Correct reviewed index and new HEAD, published-history warning and explicit confirmation, no automatic force push. |
| T130 | Stage supported individual line in multi-hunk file; attempt unsupported partial patch | Only intended index content staged; unrelated working tree untouched; invalid patch blocked without partial silent staging. |
| T131 | Render near minimum practical terminal dimensions and resize during S01/S06/S08 | All approved actions remain keyboard accessible, no hidden irreversible operation or undocumented shortcut; actual min width/height recorded. |
| T132 | Hook initiates or coincides with unexpected HEAD/reference change during commit | Do not attribute an unrelated commit to current action; report actual HEAD/trees, preserve draft if outcome uncertain, require safe revalidation. |
| T133 | S06 keep two whole commits and three individual file selections; alternate active modes and source branches | Both total/hidden counts displayed, inactive set visibly labeled, displayed consequence matches ONLY the active mode, no mixed execution. |
| T134 | After hook failure, offer explicit bypass action but user does not invoke it; then intentionally invokes it | No automatic reattempt; default normal Commit still runs hooks; bypass runs only after guarded user choice. |

## Preserved global/contextual shortcut, read-only stash and first-push regression fixtures
| ID | Fixture and interaction | Required assertion |
|---|---|---|
| T135 | From S01 and S04, press `g h`; separately press retired `H` in normal navigation | `g h` opens same S04 with pagination and restores context on Esc; `H` does not open History; no repository mutation. |
| T136 | Use `g b` across normal panels and existing S01 `b` | Both open exactly the same S11 action/state; `b` outside its existing S01 scope does not gain a global binding; `Enter` inspects and never checks out. |
| T137 | Create local branch with `g n` from current HEAD while `origin/<branch>` absent, then with same-name remote tracking ref present | New local ref created from reviewed HEAD; current checkout remains unchanged until post-create switch prompt choice; no upstream in either fixture; no push/network or remote-ref creation. |
| T138 | In S11 focus branch A, `a n` with reviewed starting ref A, versus `g n` from current checked-out B | Contextual creation starts at displayed A while global creation defaults to B's HEAD; user chooses name, no implicit switch, push or upstream; post-create switch is optional, missing/stale/invalid ref blocked. |
| T139 | Current branch and selected different local branch in S11, use `g u` and `a u` separately | `g u` targets only checked-out branch; `a u` targets only selected contextual local branch; both review exact remote and merge ref, never push/fetch/switch. |
| T140 | Fresh local branch without upstream, `g P`, confirm target `origin/<branch>` and perform first push to available origin bare repo | No remote/upstream is created before approval; native `git push -u origin <branch>` succeeds; Git config and tracking ref confirmed after success, future push honors upstream without resetting it. |
| T141 | First push no upstream, target `origin/<branch>` already exists but is divergent; also missing origin, credential/auth or rejected push | Exact known/unknown destination status shown; no force; missing origin blocks; rejected/failed/uncertain push never reports successful tracking configuration; no silent checkout or GitFlow preconfiguration. |
| T142 | Branch tracks a non-origin upstream; invoke `g P` | Existing upstream honored; no implicit replacement with origin, no forced `-u` or new tracking relation; native non-fast-forward behavior preserved. |
| T143 | In S03 commit context invoke `a v`, cancel, invoke again and explicitly approve; then make normal commit | First path mutates nothing; approved path uses one-shot native `--no-verify` with warnings and signing/preflight, and ordinary later commit runs hooks; text editor `a`/`v` inserts letters, not commands. |
| T144 | Across S01, S11, S06 and S03, inspect `a` menu, `g` menu and type same keys in a focused editor; attempt unbound `a e` | Contextual/global action domains never leak; disabled reason shown for invalid context; `a e` never executes, creates or changes Git state; `g s` read-only does not trigger S08 Skip or S01 Stage. |
| T145 | Populate stash via native fixture, open `g s`, scroll and return; verify stash refs OIDs/messages before and after | S16 reads true stash list, no stash mutation/ref removal/working-tree modification; no apply/pop/drop/clear action in context or palette; restores previous navigation state. |
| T146 | Empty stash or malformed Git stash-list output, invoke `g s` | Truthful empty/error state, keyboard return works, no fabricated entries or automatic stash actions. |
| T147 | Multiple filters and collapsed folder with 3 explicitly selected files, 2 hidden; enter Stage via direct shortcut/menu | Displays total 3 targets and 2 hidden *before* action; includes all selected IDs, never silently substitutes focused unchecked path; eligible paths processed and ineligible ones accurately reported, no invalid Git operation attempted. |

## v2.3 approved UX and branch-flow regression fixtures
| ID | Fixture and user interaction | Required assertion |
|---|---|---|
| T148 | From feature/old with tracked staged/unstaged edits, `g c` and inspect multiple source commits and file diffs; preview whole-commit and file modes | S06 opens read-only despite dirty tracked state; can inspect/select/preview with no Git mutation; whole-commit Execute offers scoped ST03 `s` Stash and Proceed or Esc Cancel, while selective-file Execute stays blocked until tracked-clean. |
| T149 | Dirty tracked S06 plan selected, an external Git process changes index before execution; include incompatible sequencer and noncolliding untracked case | Revalidate before presenting/performing ST03: stale tracked state invalidates prior review; unmerged/sequencer never bypassed by stash; noncolliding untracked allowed, collision fails closed; selected OIDs/target preserved. |
| T150 | S01 select three changed files, filter or collapse to hide two, check Stage footer; then open an existing guarded batch-action review (for example Discard) with one ineligible path | Footer says `3 selected · 2 hidden`; existing review lists all three full repo-relative paths with intended operation and hidden marker plus blocker; direct Stage remains safe without a mandatory modal; focus-only row not silently added. |
| T151 | Create local branch with `g n`, press Esc at Switch?; repeat create via `a n` then press `s`; create via `a m` then press Enter | Each creation leaves old checkout until explicit switch; Esc preserves created ref and stays; s/Enter checks out correct newly created ref, no upstream/push, no misleading S11 Enter behavior. |
| T152 | In S11 use Enter on focused local branch then `s` to request switch; try `s` while switch would overwrite staged/unstaged tracked changes | Enter inspects only; S11 s requests switch; when blocked it reports exact paths and offers Stash and Switch vs Esc/manual; no mutation until review confirmed. |
| T153 | Blocked tracked switch: choose `s` Stash and Switch then cancel S10; repeat and confirm `c` | Cancellation preserves tracked changes/checkout/stash refs; confirmation makes one real tracked-only stash then switches safely, respects selected destination, leaves stash saved and does not apply/pop after switch. |
| T154 | Stash succeeds but checkout fails (external state change), and separate stash failure; include untracked colliding path and unmerged sequencer | Stash failure means no checkout; if checkout fails after stash, stash kept, real identity/current branch shown, no automatic restore; untracked collision/unmerged operation not misrepresented as solvable by tracked stash. |
| T155 | From feature/old, local main four commits behind origin/main, `a m`, select `c` direct | New branch starts at old local-main OID; main unchanged, no network, feature checkout/index/worktree unaffected; post-create switch prompt shown. |
| T156 | From feature/old with staged and unstaged tracked changes, local main FF-behind origin/main, `a m`, choose `p` update | Explicit native fast-forward-only update advances actual local `main` ref, then new branch points to updated main; feature branch checkout, index and worktree unchanged; no merge/pull into feature branch or implicit stash. |
| T157 | Currently on main with clean state, behind origin/main; `a m` choose `p`; then currently on main with unsafe dirty update | Safe native FF-only pull updates main and creates branch; unsafe case is blocked with truthful state and direct-create option; no hidden stash/discard or switch before user choice. |
| T158 | `a m` when main and origin/main diverge; missing local main, missing origin, failed network/auth; also update succeeds but create fails due to duplicate name | No merge/rebase/reset or fabricated freshness; direct create offered only with existing local main; when update already succeeded but creation fails, report actual updated main and no created branch, no automatic rollback. |
| T159 | Navigate S01→S02→S04→S06→S13→S06→S01 with file checks, filters, scroll, queues, source, draft; no mutation | Return focus and scroll restored; checked IDs and hidden counts stable, inactive selection set preserved, source and draft intact; Esc dismisses UI levels only. |
| T160 | Same navigation as T159 with external Git mutation deleting a focused file and moving a referenced branch | Native refresh prunes stale target only, explains it, restores nearest valid focus and revalidates affected planner; unrelated checks/draft retained; no silent retarget or Git mutation on navigation. |

## v2.3 approved whole-commit cherry-pick stash/restore fixtures (ST03)
| ID | Fixture and user interaction | Required assertion |
|---|---|---|
| T161 | Clean tracked index/worktree, select and execute two whole commits; observe stash refs and Git command log | Native commits created in reviewed order; no stash push/apply/pop/drop or restore prompt; existing stash entries untouched. |
| T162 | Staged + unstaged tracked edits exist; whole-commit Execute → `s` Stash and Proceed; cherry-pick two nonconflicting commits | Exact changed paths shown before `s`; tracked-only stash created after authorization, untracked ignored; both cherry-picks committed before automatic apply of exact workflow stash; original edits/staging accurately restored; verified stash cleanup does not touch unrelated stashes. |
| T163 | Dirty whole-commit Execute blocker → `Esc` | Cancels pending execution, original index/worktree and stash list unchanged, no commits applied; returns with plan selections/order intact. |
| T164 | Induce stash push failure, or a dirty/unmerged state appearing immediately after `s` before execution | No cherry-pick begins when stashing/preflight fails; no untracked stash, no unsupported repair command, truthful error and native state shown. |
| T165 | Dirty stash-and-proceed multi-commit queue pauses with cherry-pick conflict; resolve and Continue to finish; independently Abort the cherry-pick | No stash restoration while sequencer is active; on fully successful completion, attempt restoration once; if cherry-pick aborted, retain original stash and report actual committed/aborted state without auto-restore. |
| T166 | Dirty stash-and-proceed commits succeed but original tracked changes conflict during automatic restoration; press `c` | Original cherry-picked commit HEADs remain; conflicting paths and stash identity shown; partial restore retained, S09/manual conflict resolution offered with correct native side labels, no `cherry-pick --continue` or automatic commit; backup stash preserved. |
| T167 | Same conflicting restore, choose `a` Abort unstash then explicitly confirm when baseline is still safe | Revert only restoration-introduced tracked/index edits to the recorded post-cherry-pick baseline, preserve completed commits/HEAD, preserve stash exact OID and unrelated untracked files; no generic sequencer abort. |
| T168 | External process changes tracked file, HEAD or index after restoration begins; request `a` Abort unstash; press Esc in conflict overlay | Automatic rollback refused when provenance/baseline unsafe, external edits preserved; stash and actual conflict state reported. Esc never resets/discards/aborts Git. |
| T169 | Dirty whole-commit cherry-pick succeeds and stash applies cleanly but workflow-stash cleanup fails or another stash is inserted concurrently | Original changes restored; only exact recorded stash identity considered for cleanup; failed cleanup reports retained stash, never removes unrelated entry or claims cherry-pick rollback. |
| T170 | In S14 Apply selected files with dirty tracked index/worktree, attempt Execute; separately exercise untracked destination collision in whole-commit ST03 | Selective-file Execute remains clean-only with **no** ST03 `s` option; untracked collisions block whole-commit execution without stashing, preserve colliding bytes; no selection-domain mixing. |
