# Mission Teams testing

Deterministic tests cover:

- DIRECT selection for atomic work and explainable preset selection for feature,
  bug, refactor, architecture and release goals;
- preset/facts and Flow/WorkPlan round trips, invalid preset fail-closed and
  backend/frontend fan-out with dependency gates;
- structured finding identity, deduplication, resolution and regression;
- explicit preset/runtime override precedence, provider-preference projection
  and DelegationMode OFF anti-overprocessing;
- planning → implementation → review → verification phase lifecycle, Lead
  identity preservation and no write dispatch from review/verifier phases;
- bounded remediation, restart persistence, provider failover and existing
  AUTO/ASK/OFF contracts through the canonical runner tests.

The full validation matrix remains separate from deterministic fixtures. Fake
providers prove state-machine behavior only. Authenticated provider evidence,
native Windows/macOS evidence and overnight certification must be recorded as
their actual status and never inferred from these tests.
