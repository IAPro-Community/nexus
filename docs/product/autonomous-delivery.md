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

## Promotion from a supervised terminal

While typing in `nexus <provider>`, Nexus keeps a byte-transparent shadow of
the current line. Escape/CSI/OSC control sequences and bracketed paste never
enter the logical prompt. `AUTO` promotes eligible compound requests to a
Mission, `ASK` emits a confirmation boundary, and `OFF` is pure passthrough.
The Enter byte is held only until that decision; an accepted prompt clears the
provider line and the same Lead receives the Mission synthesis when it ends.
Multiple runtimes may use the same provider profile concurrently: promotion
is a routing choice, not an account, profile, terminal or Agent singleton.

Review findings are returned to the Builder automatically. Stable finding IDs
prevent duplicate Attention and allow Nexus to detect progress. The loop is
bounded by the existing autonomy contract; exhausted progress ends as
`FAILED_NO_PROGRESS` or a genuine typed human blocker.
