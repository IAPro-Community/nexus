# Nexus Capability Preservation Ledger

Baseline: `c96885218b3cff319f481aed4423dc5fd7b52a30`

| Capability | Baseline | Final | Status | Evidence | Regression |
|---|---|---|---|---|---|
| Projects/filesystem/Browser/Scan/Inspect/mkdir | Present | Preserved | KEEP | filesystem and web handler tests | None found |
| Project Manager / OS file manager/editor/terminal | Present | Preserved | KEEP | project UI/API and existing validation artifacts | Native OS effect unverified |
| Branch display / BranchSwitcher / checkout | Present | Preserved | KEEP | project Git handlers/UI | None found |
| Providers/profiles/accounts/auth/capabilities/config | Present | Preserved | KEEP | provider/profile/config/driver tests | Real credentials not asserted |
| Usage/quota/tokens/cache/reset/status semantics | Present | Preserved | KEEP | profile/quota tests and Resources UI | Real provider data unavailable |
| Resource discovery/health/recommendation/MANUAL/AUTO | Present | Preserved | KEEP | scheduler/recommendation tests | None found |
| Persistent Agents/create/start/stop/recover/config/Safe Apply | Present | Preserved | KEEP | Nexus/config/runner tests | Real Safe Apply unverified |
| Runtime generations/reconnect/continuity/host lifecycle | Present | Preserved | KEEP | runtime/host/registry/handoff tests | Windows/macOS runtime unverified |
| AgentTerminal/TerminalPane/ANSI/I-O/resize/lease | Present | Preserved and fixed | FIX | frontend 30 tests plus Go terminal/host tests | TerminalPane parser-consumer mismatch fixed |
| Unix PTY / Windows ConPTY / named pipes | Present | Preserved | REVIEW_REQUIRED | source/tests and Windows cross-compile | `UNVERIFIED_ON_WINDOWS` |
| Work/Intelligence/API/CLI | Present | Preserved | KEEP | intelligence and API tests | Real intelligence provider unverified |
| WorkPlan/revisions/diff/restore/clarifications/DAG | Present | Preserved | KEEP | plan/store/plangraph tests | Browser editor flow incomplete |
| Mission run/pause/resume/Take Control/cancel/recovery/verification | Present | Preserved | KEEP | runner durability/verifier tests | Real Mission E2E unverified |
| Schedules/after-run/resource scheduling | Present | Preserved | KEEP | scheduler tests/API | Full UI flow unverified |
| Maestro detect/status/update/skills | Present | Preserved | KEEP | Maestro tests/UI/API | External installation unverified |
| Nexus/Maestro updates | Present | Preserved | KEEP | update handlers/UI | External update not executed |
| CSRF/origin/secrets/isolation/launch env/Safe Apply | Present | Preserved | KEEP | security/origin/session/race tests | scanners unavailable |
| Workspace V2 tabs/desktop/presentation | Partial | Added in canonical worktree | ADAPT | presentation/model/state tests | Browser persistence incomplete |
| Ask existing Agent | Partial | Added in canonical worktree | ADAPT | API validation and UI model tests | Real provider interaction unverified |
| Project Shell | Absent | Dedicated shell surface | ADAPT | `ShellDriver`, `ProjectShellSurface`, launcher and frontend build/tests | Native PTY/browser lifecycle unverified |
| Context Readiness contract | Absent | Five-state contract | ADAPT | `internal/nexus/context_readiness.go` and state/drift tests | Persistence/Composer gate not wired |
| Composer/Flow V0 surfaces | Partial/Absent | WorkSurface/PlanBuilder with readiness model, FlowCanvas and FlowRunSurface | REVIEW_REQUIRED | frontend suite plus WorkPlan/Mission/graph/capsule tests | readiness persistence, full inspector/refinement and browser flow incomplete |

No baseline tests or functional files were removed. The committed baseline range has no diff because the official HEAD equals the baseline; recovery changes are now committed on the canonical branch.
