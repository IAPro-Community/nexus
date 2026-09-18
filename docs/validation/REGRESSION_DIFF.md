# Regression Diff — Phase A Initial Audit

Date: 2026-09-11
Branch: `feat/nexus-maximum-delivery`
HEAD: `bf0caad103499450564d670dfaffd302b745c147`
Remote comparison: `origin/feat/nexus-maximum-delivery` at `ab056f558fb2886b21db60ca7951cfb92deb46c7`

## Scope

The local branch contains two commits not present in the remote reference:
`cc1ebf0` and `bf0caad`. The functional delta is concentrated in launch-mode
resolution and colon-prefix terminal routing, with tests and validation docs.

## Public-surface review

| Surface | Observed delta | Initial status |
|---|---|---|
| Provider launch | TTY now resolves to supervised unless explicitly direct/headless | IMPLEMENTED_UNVERIFIED |
| Launch flags | `--direct`, `--supervised`, `--print` are stripped/resolved centrally | IMPLEMENTED_UNVERIFIED |
| Terminal control | `:nexus`/`:ai` and `::` escapes added alongside slash aliases | IMPLEMENTED_UNVERIFIED |
| API routes/DTOs | No route or migration change in the two local commits | NOT_APPLICABLE to local delta |
| Provider registrations | No change in the two local commits | NOT_APPLICABLE to local delta |
| Persistence | No migration change; runtime/agent tests added | IMPLEMENTED_UNVERIFIED |
| Desktop/installer/update | No change in the two local commits | NOT_APPLICABLE to local delta |
| TUI/completion | No implementation delta proven by these commits | PARTIAL |

## Missing evidence

No fresh end-to-end provider PTY run, terminal follow-handoff run, completion
matrix, browser E2E, native Windows/macOS run, or full baseline compatibility
matrix was completed in Phase A. No removal was observed in the local commit
delta, but that does not prove behavior compatibility.

## Corrective closure delta (working tree)

After the frozen Phase A report, focused changes added persistent AgentSpec
presets and compilation metadata; canonical AgentMatcher integration for
Flow/Mission AUTO; natural-language TaskRequirements and
`nexus run "<goal>"`; deterministic resource tie-breaking; structured
`REQUIRED_RESOURCE_SELECTION` conflicts; and truthful handoff identity and
`NATIVE_RESUME_UNVERIFIED` classification. No historical CLI/API/database
migration was removed or edited retroactively.
