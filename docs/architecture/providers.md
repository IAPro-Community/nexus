# Provider registry and capabilities

Launch args, env overlay, binary discovery and quota source class are frozen in
one [`ProviderSpec`](provider-contract.md) per provider
(`internal/core/provider/spec`). Adapters and supervised drivers must derive
from that Spec instead of maintaining parallel tables.

Nexus has two complementary registries:

- `internal/core/provider.Registry` integrates provider adapters for detection,
  auth, usage, conversations, and execution.
- `internal/control/driver.Registry` integrates supervised runtime drivers for
  process, terminal, attach, resume, prompt, and control capabilities.

Interfaces are deliberately segregated (`AuthProvider`, `UsageProvider`, and
`ConversationProvider`) rather than forcing every provider to implement one
large contract. CLI aliases (`codex`, `agy`, `claude`, `opencode`, `gemini`,
and `cursor`) resolve through the registry while preserving direct commands.

Capability responses include evidence-bearing statuses for supervised control.
The API metadata endpoint derives its provider capability map from this
registry, so surfaces do not need provider-name switches to decide whether a
feature is available.

Provider IDs are normalized to trimmed lowercase at registration and lookup,
so adapter casing cannot create an unreachable registry entry.

Nexus-owned application instances may inject a `driver.Registry`. Resource
discovery, continuity/resume checks, autonomous execution and CLI intelligence
resolve drivers through that instance. Direct legacy constructors retain a
process-global fallback, but transport/application wiring no longer requires
the global registry.

Adding a provider requires adapter/driver registration, capability evidence,
tests, and documentation of unsupported operations. It does not require a new
HTTP route or frontend business rule.

### Provider-specific policy boundaries

The driver registry owns provider discovery, capabilities, command construction,
resume and kickoff contracts. A small set of explicit switches remains in
autonomous execution for policy decisions that are not provider discovery:
approval mode, sandbox/write authorization, read-only review mode and verified
headless CLI syntax. These switches are intentionally reviewed as security
policy and must not be replaced by capability booleans without an equivalent
authorization contract and tests.
