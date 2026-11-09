# AGY balance quota recheck — 2026-11-09

- Command: `NEXUS_REAL_HOME=/home/desenvolvedor nexus usage --refresh --json`; refresh only, no prompt or provider generation.
- Two AGY profiles returned recent `CACHED` snapshots with source `CLI_OUTPUT` and attributable account scope.
- Sanitized account 1: `Gemini Models` exposes weekly remaining 100 only; `Claude & GPT Models` exposes weekly remaining 99 only. No numeric 5h window was returned for either observed group.
- Sanitized account 2: `Gemini Models` exposes weekly remaining 100 and 5h remaining 100; `Claude & GPT Models` exposes weekly remaining 1 and 5h remaining 100.
- Strict `:balance` gate result: 1/2 eligible. Generic resource `eligibility.eligible` is not the AGY 5h balance gate and must not be used as proof of 5h evidence.
- Existing regressions: `TestBalancedAGYSelectionRejectsWeeklyOnlyQuota`, `TestBalancedAGYSelectionRequiresFreshAttributableEvidenceForEveryKnownGroup`, and persistent rotation tests in `internal/core/scheduler/scheduler_test.go`.
- Rotation with two live eligible accounts is still unverified because the first account has no 5h evidence. Do not generate a model call to force eligibility; retry the read-only quota refresh after the CLI/provider exposes 5h for that account.
