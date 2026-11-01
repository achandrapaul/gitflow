# Normative v2.3 contract availability and reconciliation gate

## Available materials and their limits

- **Available:** full approved v2.1 archive (historical comparison only); four uploaded Claude v2.3 delivery Markdown summaries; GitFlow established project decisions.
- **Not available in uploaded material:** the v2.3 normative `spec/GITFLOW_REQUIREMENTS_V2_3.md`, repository `CLAUDE.md` current version, `tests/SCENARIOS_V2_3.md`, `tests/ACCEPTANCE_V2_3.md`, `tests/TRACEABILITY_V2_3.md`, or actual implementation code.
- **Implication:** requirement ID coverage can be organized using the v2.3 summaries, but exact v2.3 acceptance meaning cannot be certified from the uploaded summaries alone. The packet **must not claim compliance** as a result.

## Claude must do FIRST (before editing any source code)

1. Read actual normative `spec/GITFLOW_REQUIREMENTS_V2_3.md` and repo `CLAUDE.md` in full. Verify which one is superior if the repo's instructions specify hierarchy; never invent one.
2. Read all v2.3 acceptance/scenarios and reconcile each primary requirement/test to `REQUIREMENT_OWNERSHIP.md` and `TEST_OWNERSHIP_AND_GAPS.md`.
3. Confirm the v2.3 behavioral changes B10/B11/K47, ST02/ST03, dirty read-only S06 and T89/T90/T148/T149/T151–T170 exactly; replace any outdated test predicates.
4. Validate that all approved keys/actions, S01–S16 and exclusion list are unchanged from the normative spec. If conflicts exist, list exact IDs and source quotations; **do not alter spec or implement guessed behavior**.
5. Only then implement Slice 01 and proceed in order; keep not-yet-implemented actions unavailable with an honest reason.

## Critical apparent contradiction to reconcile

- v2.1 tests T89 and T90 expected `g c` to block before Explorer when tracked files were dirty. The uploaded v2.3 Slice 3 says `g c` **read-only entry even when dirty** and separately authorizes ST03 guarded whole-commit dirty recovery. Therefore v2.1 T89/T90 expectations cannot automatically be reused. The uploaded slice specifically calls for T90 correction but omits T89. **Do not reintroduce the obsolete entry guard to make an old test green.** Verify both actual v2.3 scenarios.
- Uploaded v2.3 Slice 2 describes a generic S15 Review/Execute/Cancel menu. The complete v2.1 contract explicitly says S15 is **not a routine nested popup**. Unless the actual v2.3 normative document changed that rule explicitly, do not add an extra generic execution step.
- Uploaded v2.3 summaries describe ST02 and ST03 stash mutations. These are narrow exceptions to prior v2.1 excluded generic stash commands, **not** permission to introduce a stash manager or general stash operations.

## Missing test-definition protocol

When v2.3 scenarios are not available, T148–T170 cannot be assigned meaningful assertions or marked Passed. Keep status `Blocked: v2.3 scenario definition not available`. Once read, assign each ID to exactly one primary slice by the scenario's actual feature; update `TEST_OWNERSHIP_AND_GAPS.md`. Do not map by test number adjacency or by a guessed feature.
