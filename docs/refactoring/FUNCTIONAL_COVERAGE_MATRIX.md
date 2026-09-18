# Functional coverage matrix

Status vocabulary: VERIFIED means fresh executable evidence exists;
PARTIAL means only part of the path is proven; UNVERIFIED means implementation
or unit evidence exists without the required integration/E2E proof.

| Capability | Unit | Integration | E2E | Current status |
|---|---:|---:|---:|---|
| TTY launch mode and control prefixes | yes | partial | no | PARTIAL |
| Persistent AgentSpec specialty | yes | Flow/Mission auto partial | no | PARTIAL |
| Canonical AgentMatcher | yes | Flow/Mission/`nexus run` wiring | no | PARTIAL |
| Provider/profile ResourceScheduler | yes | partial | no | PARTIAL |
| Same-provider account handoff | yes | partial | no | PARTIAL |
| Cross-provider context handoff | yes | partial | no | PARTIAL |
| Terminal follow-handoff | no | no | no | UNVERIFIED |
| Automatic failover safe point | partial | partial | no | UNVERIFIED |
| Linux build and ordinary test suite | yes | yes | applicable | VERIFIED |
| Windows/macOS native runtime | no | no | no runner | UNVERIFIED |

The authoritative current verdict is
[`docs/validation/EVOLUTION_FINAL_VALIDATION.md`](../validation/EVOLUTION_FINAL_VALIDATION.md).
