# Agent routing

Agent selection is provider-independent. `internal/nexus/agent_matching.go`
owns persistent-Agent eligibility, taxonomy normalization, deterministic score,
confidence, explainability, and rejection reasons. Resource/provider/profile
selection remains in `ResourceScheduler` and `RecommendResources`.

Legacy role-only Agents remain readable through `NormalizeAgentSpec`; the
normalizer copies only the stored role and does not invent behavior.

Current evidence: unit tests and Flow/Mission AUTO integration through the
matcher; natural-language `nexus run "<goal>"` also materializes the same
requirements. Provider-backed runtime E2E provenance remains explicitly
pending.
