# ADR: Compose API routes by capability group

- Status: Accepted
- Date: 2026-09-10

## Context

`Server.NewServer` registered every API route beside listener, auth, static
SPA, and security setup. That made transport growth harder to review and
encouraged treating the server as the owner of product domains.

## Decision

Keep the standard library `http.ServeMux` and move registration into cohesive
group functions in `internal/control/web/routes.go`. `Server` supplies explicit
dependencies and middleware; existing handlers remain responsible for request
validation and invoking the current Core services.

## Consequences

Route ownership is discoverable and independently testable without introducing
a framework or changing endpoint paths. Handler/business-logic extraction can
continue by domain later. The route file is still transport code and is not a
new application layer.

## Deferred / intentionally unchanged

No public response envelope migration, new provider, quota scraping,
orchestration engine, UI redesign, or automatic Maestro behavior is part of
this decision.
