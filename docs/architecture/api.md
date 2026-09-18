# Nexus API v1

The REST API is versioned under `/api/v1`. Web and Desktop use the same Core
server contract; the CLI may call Core services directly for local commands.

## Composition

`internal/control/web/server.go` owns listener, middleware, and SPA lifecycle.
`internal/control/web/routes.go` registers cohesive groups:

- control: health, session, workspaces, runtimes, providers, profiles, events;
- Nexus: projects, agents, resources;
- planning: intelligence, composer, artifacts, flows, plans;
- missions: runs, schedules, missions;
- system: info, doctor, updates;
- filesystem: browse, scan, inspect, mkdir;
- Maestro: status, advice, catalog, sync, update.

## Metadata contract

Authenticated `GET /api/v1/system/info` returns semantic metadata:

```json
{
  "apiVersion": "v1",
  "serverVersion": "dev",
  "build": { "version": "dev", "commit": "unknown" },
  "providers": ["agy", "claude"],
  "capabilities": { "agy": { "terminal": true } }
}
```

The actual capability object is derived from the existing control-driver
registry. Provider ordering is deterministic. Existing endpoint payloads are
not wrapped or changed by this metadata addition.

## Web and Desktop client boundary

`web/src/api.ts` owns the shared HTTP transport: desktop authentication/base
URL, CSRF headers, session-expiration notification, response parsing and the
typed `NexusRequestError`. `web/src/nexus/api.ts` is the typed domain facade
for Projects, Agents, Planning, Missions and system operations; it delegates
requests to that transport instead of maintaining a second `fetch` pipeline.
The public `NexusAPIError` export remains an alias for compatibility.

## Errors and compatibility

Handlers preserve the legacy `error` string and status code, and now add a
machine-readable `code` field without removing the old field:

```json
{
  "error": "project not found",
  "code": "PROJECT_NOT_FOUND"
}
```

The central mapper covers common domain errors (`PROJECT_NOT_FOUND`,
`SESSION_NOT_FOUND`, `RUNTIME_NOT_RUNNING`, `PROVIDER_UNAVAILABLE`,
`QUOTA_UNKNOWN`, `RATE_LIMITED`, `WORKSPACE_INVALID`, and
`MISSION_NOT_FOUND`) plus transport fallbacks such as `INVALID_REQUEST`,
`AUTH_REQUIRED`, `CONFLICT`, and `INTERNAL_ERROR`. New endpoints must use the
same `error` field, redact sensitive text, and return predictable `405`/`4xx`
responses. Exact endpoint-specific mapping remains incremental where a
handler currently emits a domain error without a stable semantic phrase.

Runtime/event records may include `correlation_id` to group lifecycle events
for a MissionRun or handoff without changing the existing event ID or runtime
filter contract.
