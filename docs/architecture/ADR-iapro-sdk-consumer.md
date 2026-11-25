# ADR: Consuming shared IAPro-SDK contracts

- Status: Proposed; implementation is local and awaits independent review
- Date: 2026-11-24
- Related: Nexus #235, #237–#242; SDK #1–#6; `DEV/SPECS/IAPRO_SDK_INTEGRATION.md`

## Context

Nexus and Calango share provider-neutral contracts, but Nexus also owns mature
runtime, quota evidence, resource scheduling, handoff transactions, durable
Mission coordination, cost ledgers, user interface, and Wails lifecycle. The
SDK is a Go 1.23 pre-alpha library. Its current `main` SHA has green CI, but no
tag; some packages remain narrower than Nexus behavior.

## Decision proposed

1. Consumers pin immutable module revisions with `go.sum`; no release uses a
   floating branch or local `replace`.
2. Share neutral value contracts and pure helpers only after their serialized
   and semantic behavior matches. Current direct aliases cover provider/account,
   failure/usage/capability types, routing reason values, and coordination
   envelope identity/equality/cloning.
3. Keep product orchestration and state in the host: ResourceScheduler,
   MissionRunner, storage, authorization, effective budgets/reservations,
   credential custody, Maestro integration, Wails, and product UI.
4. Use host adapters where Nexus behavior is ahead of the SDK. Do not remove
   quota freshness/evidence rules, provider isolation, ledger semantics, or
   continuity behavior until the SDK has parity tests and the consumer passes
   regression checks.
5. A Go library lease interface and in-memory reference store do not establish
   cross-process or cross-product exclusion. That claim requires a shared
   durable linearizable store and enforcement of monotonic fencing at host write
   boundaries.
6. No code change activates optional Nexus features or changes commercial,
   authorization, sandbox, or provider permission defaults.

## Consequences

- SDK updates can be consumed independently from Nexus releases and rolled back
  by reverting the module pin and adapter changes.
- Temporary host compatibility adapters remain only for behavior absent from
  the pinned SDK; each is listed with a removal condition in the integration
  audit.
- Provider runtimes, quotas, routing/failover, handoff, Capacity Monitor,
  Cost Intelligence, React AI Control/HUD and interprocess leases remain partial
  integration gates.
- A green SDK CI run is evidence for the library SHA only, not for Nexus or
  Calango compatibility.

## Validation and approval gates

Confirm provenance and notices; pass SDK CI; pass Nexus `go test`, race, vet,
lint, frontend typecheck/test/build and Wails build; validate exact-SHA
Windows/Linux/macOS; run provider conformance and authorized real-provider
homologation; validate Calango without modifying its repository; receive
independent review; update issues/Project only through authorized GitHub
access. Until those gates pass, this ADR remains proposed and the integration
is NO_GO for release.
