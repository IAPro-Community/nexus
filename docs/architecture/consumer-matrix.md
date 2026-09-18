# Nexus consumer matrix

Nexus has one Core and multiple surfaces. A surface may choose a local Core
service or the versioned HTTP API, but it must not reimplement domain rules.

| Surface | Transport boundary | Owns | Must not own |
| --- | --- | --- | --- |
| CLI | `internal/app` → Core services | argument parsing, rendering, exit codes | provider lifecycle, quota calculation, persistence |
| Web | `web/src/nexus/api.ts` → `/api/v1` | presentation, interaction state, typed request facade | runtime lifecycle, planning rules, direct scattered `fetch` |
| Desktop | embedded Web → same `/api/v1` Core server | native bridge and window integration | a second runtime, API contract, provider policy |
| HTTP server | `internal/control/web` → application/Core services | auth, CSRF, DTO/HTTP status mapping, route registration | durable business rules |

## Current client contract

`web/src/api.ts` is the single HTTP transport for Web and Desktop. It owns
desktop auth/base URL, CSRF, session-expiration signaling, response parsing and
`NexusRequestError`. `web/src/nexus/api.ts` is the domain facade and preserves
the `NexusAPIError` alias for existing consumers.

The CLI intentionally does not wrap local Core calls in an HTTP client: its
local command path is already a direct Core consumer. A future remote CLI mode
must reuse the same `/api/v1` contract rather than inventing provider-specific
HTTP calls.

## Compatibility rule

New surface behavior starts at the nearest existing application/Core boundary.
If a handler or component needs a rule that is not present there, add a
focused Core contract and test before adding transport-specific behavior.
