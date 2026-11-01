# Test → Primary Slice Ownership and Provenance Gaps

**Exact scenario authority:** actual repository `tests/SCENARIOS_V2_3.md` and `tests/ACCEPTANCE_V2_3.md`. Below, T01–T147 are assigned **by the supplied complete v2.1 scenario meanings**; their meanings MUST be rechecked against v2.3, especially dirty cherry-pick entry T89/T90. This map does not claim any test passed.

**Known absent ranges in uploaded v2.3 delivery summary:** T33–T45, T59–T83. Do not fabricate these tests.

| Test ID | Provisional primary slice | Provenance / required check |
|---|---:|---|
| T01 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T02 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T03 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T04 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T05 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T06 | 03 | v2.1 scenario known; verify exact v2.3 wording |
| T07 | 03 | v2.1 scenario known; verify exact v2.3 wording |
| T08 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T09 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T10 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T11 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T12 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T13 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T14 | 07 | v2.1 scenario known; verify exact v2.3 wording |
| T15 | 10 | Implemented: noncontiguous commits applied in queue order (TestCherryPickerPick, TestCherryPickPreviewAndExecute) |
| T16 | 09 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 09 |
| T17 | 09 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 09 |
| T18 | 12 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 12 |
| T19 | 12 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 12 |
| T20 | 12 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 12 |
| T21 | 12 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 12 |
| T22 | 09 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 09 |
| T23 | 09 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 09 |
| T24 | 09 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 09 |
| T25 | 12 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 12 |
| T26 | 01 | v2.1 scenario known; verify exact v2.3 wording |
| T27 | 10 | Implemented: empty pick pauses natively into the Paused Operation Dashboard; no `--allow-empty` (TestCherryPickEmptyPickPauses) |
| T28 | 09 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 09 |
| T29 | 01 | v2.1 scenario known; verify exact v2.3 wording |
| T30 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T31 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T32 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T46 | 01 | v2.1 scenario known; verify exact v2.3 wording |
| T47 | 16 | v2.1 scenario known; verify exact v2.3 wording |
| T48 | 16 | v2.1 scenario known; verify exact v2.3 wording |
| T49 | 16 | v2.1 scenario known; verify exact v2.3 wording |
| T50 | 16 | v2.1 scenario known; verify exact v2.3 wording |
| T51 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T52 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T53 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T54 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T55 | 05 | v2.1 scenario known; verify exact v2.3 wording |
| T56 | 16 | v2.1 scenario known; verify exact v2.3 wording |
| T57 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T58 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T84 | 06 | v2.1 scenario known; verify exact v2.3 wording |
| T85 | 12 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 12 |
| T86 | 06 | v2.1 scenario known; verify exact v2.3 wording |
| T87 | 06 | v2.1 scenario known; verify exact v2.3 wording |
| T88 | 01 | v2.1 scenario known; verify exact v2.3 wording |
| T89 | 10 | Reconciled: v2.3 T89 says Execute disabled with dirty paths. Implemented as disabled, paths listed, no stash (TestCherryPickGuards). Conflicts with C21/T90 ST03 offer; see OPEN_QUESTIONS Q33 |
| T90 | 10 | Split: preview with tracked edits implemented here (disabled with paths); `s` Stash and Proceed offer owned by Slice 15 |
| T91 | 10 | Implemented: browsing another source never checks it out (TestCherryPickExplorerBrowsesWithoutCheckout) |
| T92 | 10 | Implemented: untracked collision blocks Execute without stash (TestCherryPickGuards) |
| T93 | 10 | Implemented: paused operation blocks a new plan; `o` opens the Paused Operation Dashboard (TestCherryPickGuards, TestCherryPickRestorePreflight) |
| T94 | 10 | Implemented: Source/Target header, local then cached remote-tracking refs, no fetch (TestCherryPickExplorerBrowsesWithoutCheckout) |
| T95 | 10 | Covered in Slice 10: `g c` from History carries source and selected OIDs (TestCherryPickHandoffFromHistory) |
| T96 | 10 | Implemented: Execute unavailable until the current queue is previewed (TestCherryPickPreviewAndExecute) |
| T97 | 10 | Implemented: queue change after preview requires a new preview (TestCherryPickPreviewAndExecute) |
| T98 | 10 | Implemented: selection survives handoff and source change by full OID (TestCherryPickHandoffFromHistory) |
| T99 | 10 | Implemented: reviewed whole commits land in reviewed order (TestCherryPickPreviewAndExecute, TestCherryPickRestoreExecute) |
| T100 | 11 | Pass — TestSelectiveApplyStagesFiles (2026-11-01) |
| T101 | 10 | Implemented: conflict pause opens the Paused Operation Dashboard (TestCherryPickPausesIntoDashboard) |
| T102 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T103 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T104 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T105 | 05 | v2.1 scenario known; verify exact v2.3 wording |
| T106 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T107 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T108 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T109 | 03 | v2.1 scenario known; verify exact v2.3 wording |
| T110 | 06 | v2.1 scenario known; verify exact v2.3 wording |
| T111 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T112 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T113 | 02 | v2.1 scenario known; verify exact v2.3 wording |
| T114 | 10 | Implemented: J/K reorder changes queue and preview order (TestCherryPickPreviewAndExecute) |
| T115 | 11 | Pass — TestSelectiveApplyStagesFiles, TestApplyQueue, TestSelectiveApplyExecute (2026-11-01) |
| T116 | 11 | Pass — TestSelectiveApplyKeepsDraft, TestSelectiveApplyOverlapWarning (2026-11-01) |
| T117 | 11 | Pass — TestSelectiveApplyBlockers, TestSelectiveApplyPartialFailure, TestCommitFilesSupport (2026-11-01) |
| T118 | 09 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 09 |
| T119 | 09 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 09 |
| T120 | 09 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 09 |
| T121 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T122 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T123 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T124 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T125 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T126 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T127 | 10 | Implemented: dashboard shows completed count from the sequencer after restart (TestCherryPickerPick, TestCherryPickPausesIntoDashboard) |
| T128 | 05 | v2.1 scenario known; verify exact v2.3 wording |
| T129 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T130 | 03 | v2.1 scenario known; verify exact v2.3 wording |
| T131 | 01 | v2.1 scenario known; verify exact v2.3 wording |
| T132 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T133 | 10 | Implemented: both counts shown, inactive set labelled and not applied (TestCherryPickModeSeparation) |
| T134 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T135 | 07 | v2.1 scenario known; verify exact v2.3 wording |
| T136 | 05 | v2.1 scenario known; verify exact v2.3 wording |
| T137 | 05 | v2.1 scenario known; verify exact v2.3 wording |
| T138 | 05 | v2.1 scenario known; verify exact v2.3 wording |
| T139 | 06 | v2.1 scenario known; verify exact v2.3 wording |
| T140 | 06 | v2.1 scenario known; verify exact v2.3 wording |
| T141 | 06 | v2.1 scenario known; verify exact v2.3 wording |
| T142 | 06 | v2.1 scenario known; verify exact v2.3 wording |
| T143 | 04 | v2.1 scenario known; verify exact v2.3 wording |
| T144 | 01 | v2.1 scenario known; verify exact v2.3 wording |
| T145 | 08 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 08 |
| T146 | 08 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 08 |
| T147 | 03 | v2.1 scenario known; verify exact v2.3 wording |
| T148 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation. Slice 11 supporting part only: Execute in the Apply Files Planner is blocked by dirty or staged tracked changes and offers no stash (TestSelectiveApplyBlockers) |
| T149 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T150 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T151 | 13 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 13 |
| T152 | 14 | ST02 Stash and Switch; v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 13 |
| T153 | 14 | ST02 Stash and Switch; v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 13 |
| T154 | 14 | ST02 Stash and Switch; v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 13 |
| T155 | 13 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 13 |
| T156 | 13 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 13 |
| T157 | 13 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 13 |
| T158 | 13 | v2.3 wording verified against tests/SCENARIOS_V2_3.md in Slice 13 |
| T159 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T160 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T161 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T162 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T163 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T164 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T165 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T166 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T167 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T168 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T169 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation |
| T170 | **UNASSIGNED — CONTRACT GATE** | v2.3 scenario body unavailable; do not infer test from ID or old 3-slice allocation. Slice 11 supporting part only: Execute in the Apply Files Planner is blocked by dirty or staged tracked changes and offers no stash (TestSelectiveApplyBlockers) |

## Cross-slice assertions (not second primary owners)

- T57/T58 full end-to-end learning path spans Slice 01–04; primary evidence collected in Slice 04 only if test actually run.
- T85 `g f` existed in Slice 06; native rebase-pull `g r` lands in Slice 12, which owns the cross-functional scenario's primary completion.
- T95 history selection starts in Slice 07 and handoff is completed in Slice 10.
- T106 S01 `v` lands in Slice 02; S07 `v` preview asserted again in Slice 12.
- T110 mutation feedback starts with staging/commit; remote cases verified in Slice 06 and paused recovery in Slice 09.
- T128 branch creation/switch/delete lands Slice 05; upstream assertions finish in Slice 06; ST02 is a separate v2.3-only contract.
- T144 registry/prefix test spans S01/S03/S06/S11/S16; primary action registry proof in Slice 01 and revisit each new surface.
- T147 selection model in Slice 02, action-target Stage enforcement in Slice 03.
- T89/T90 MUST be updated from normative v2.3 before Slice 10: don't preserve older entry-blocker assumptions against dirty read-only entry.
- T148–T170 require exact v2.3 scenario definitions. Original Claude v2.3 summary says T148/T149 should be corrected and T161–T170 added; this is insufficient to assign assertion meanings or claim pass.

## Test evidence statuses (use exact ones supported by project traceability)

`Not implemented` / `Implemented-unverified` / `Passed` / `Failed` / `Blocked`. Record module, fixture, command outcome, index/worktree/HEAD/refs/sequencer postconditions and stdout/stderr where relevant. A Markdown planning assignment does not count as a passing test.
