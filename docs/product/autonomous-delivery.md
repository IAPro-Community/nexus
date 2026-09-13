# Autonomous delivery

Nexus can assemble a virtual engineering team around a user goal. The user
defines the outcome; Nexus keeps the Lead Agent, specialists, routing,
handoffs, review loop and verification in one durable Mission.

```text
Lead
  -> Plan
  -> Build
  -> Review
  -> Fix findings when needed
  -> Verify
  -> COMPLETED_VERIFIED
```

Agents are persistent professionals. Models and providers are replaceable
resources. A configurable example may prefer Codex for planning, AGY for
implementation and a Claude-compatible resource for review, but those names
are not hardcoded and are never claimed unless the resource actually ran.

Small requests remain fast and direct. A typo such as “Corrija este typo no
botão Login” does not receive four phases. A substantial feature can use
`FEATURE_DELIVERY`; architecture and release-risk signals use the matching
specialized preset.

Review findings are returned to the Builder automatically. Stable finding IDs
prevent duplicate Attention and allow Nexus to detect progress. The loop is
bounded by the existing autonomy contract; exhausted progress ends as
`FAILED_NO_PROGRESS` or a genuine typed human blocker.
