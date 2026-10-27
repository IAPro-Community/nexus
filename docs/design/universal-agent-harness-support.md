# Universal Harness Support & Agent Attention — Nexus 1.0

**Status:** PROPOSED / NOT IMPLEMENTED / NOT HOMOLOGATED  
**Date:** 2026-10-27  
**Canonical target:** `release/v1.0.0`  
**Owner:** Nexus Control Plane. Maestro remains independent and owns planning, orchestration, reasoning and memory.

## Product decision

Nexus supervises **harness runtimes**, not model names. Models such as Xiaomi MiMo or DeepSeek can be configured through harness providers without inventing a new runtime integration per model. A harness such as `mimo` or `dsh` needs its own launcher/provider registration, even when its wire protocol is shared.

Prioritize a shared **ACP client transport** (negotiated protocol version + capabilities), then reuse it for OpenCode, MiMo Code, Oh My Pi, DeepSeek Harness and future compatible harnesses. Keep an **RPC JSONL transport** for Pi and OMP where it offers more complete structured events; native provider adapters map both transports into the existing Nexus Registry and event bus. Preserve interactive PTY/ConPTY as a separate valid mode, **never transparently switch a user's existing TUI to a headless process**.

"Commander" needs identity resolution: https://thecommander.app/docs is a macOS agent control/review app, and https://github.com/mblua/AgentsCommander is another multi-CLI supervisor. Treat these as **competitor/UX references**, not execution providers. The npm `commander` package is an argument parser, not an AI coding harness. Do not create `CommanderDriver` without identifying a concrete executable and session protocol.

## Baseline already in Nexus (inspected on release/v1.0.0)

- `internal/control/driver/{agy,codex,opencode,claude,gemini}_driver.go`: all currently implement terminal-only supervised execution. Existing drivers declare `StructuredEvents=UNSUPPORTED`, including Codex and OpenCode despite upstream protocols.
- `internal/control/driver/driver.go`: `EffectiveCapabilities` with evidence and `ControlLevel`. **Extend optional capabilities**, do not add a parallel provider registry or blindly change booleans.
- `internal/control/host/attention.go`: PTY fallback recognizer, OSC title, prompt detection and `ObserveOperatorInput`.
- `internal/control/registry` and `internal/control/events`: runtime state and event bus should remain canonical.
- `web/src/app/attentionRadarModel.ts`, `web/src/workspace/surfaceAttention.ts`, `web/src/notifications/attentionDelivery.ts`: consumer UI; attach WS is display-only; only one notification producer.
- `internal/nexus/runner/attention_center.go`: MissionRun attention read model; must **not** be replaced or conflated with Direct runtime attention.
- Existing work: #78 PTY/ConPTY reliability; #81 two-agent runtime QA.

## Provider/harness inventory and integration strategy

| Runtime ID | Command / protocol to probe | Primary integration | Fallback | Tier | Notes |
|---|---|---|---|---|---|
| opencode | `opencode acp`; version-dependent `serve`/SSE | ACP when supported and verified; HTTP events for installed legacy version | existing PTY | P0 upgrade | Existing driver; keep existing session semantics and auth profiles |
| omp | `omp acp`; `omp --mode rpc` / `rpc-ui` | ACP common path; RPC if required for rich native UI and settled status | PTY | P0 new | OMP's JSONL RPC protocol is *not* JSON-RPC; match prompts by request ID; `--no-ui` changes semantics |
| pi | `pi --mode rpc` | native JSONL RPC for event + command correlation; ACP bridge only when actually available | PTY | P0 new | Success of prompt ACK != settled turn; avoid assuming builtin approval prompts |
| mimo | `mimo acp` when installed version advertises it | ACP | PTY / verified server path | P0 new | MiMo model != MiMo Code CLI; do not alias `opencode` registration to `mimo` |
| dsh | `dsh acp`; `dsh sdk` | ACP; SDK JSON-RPC as later native enhancement | `dsh headless --json` only with explicit headless mode | P0 new | Developer preview; breaking changes expected; profile sandbox/trust policy respected |
| codex | app-server | native JSON-RPC where explicitly verified | existing PTY | P0 upgrade | Existing app-server usage path is quota-only, **not** permission/turn control |
| agy | OSC7501 when installed version verified; PTY | OSC native status + PTY contextual fallback | existing PTY | P0 upgrade | OSC status signal alone is not a structured approval API |
| claude | native hooks / official bridge when supported | event/hook adapter with verified ownership | existing PTY | P1 upgrade | Hook notifications are not a safe remote approval mechanism by themselves |
| gemini | hook/notification protocols | event/hook adapter | existing PTY | P1 upgrade | approval-mode flag is not permission-response transport |
| goose | official CLI/API/ACP if probed | evaluated adapter | PTY | P1 new | Do not invent capability |
| qwen | CLI / structured mode if probed | evaluated adapter | PTY | P1 new | Favor consistent onboarding |
| cline | CLI/SDK after concrete qualification | evaluated adapter | PTY | P1 new | IDE plugin vs CLI distinction |
| hermes | CLI/API after version verification | evaluated adapter | PTY | P1 new | Agent may be long-lived/offline/multi-channel |
| kimi, kilo, copilot | official CLI or ACP when verified | evaluated adapter | PTY | P2 candidate | Defer until P0 evidence and adoption checks |
| Oh My OpenAgent (OMO) | OpenCode/Codex plugin installations | compatibility metadata / plugin detection | underlying harness | P1 compatibility | Avoid duplicate provider for a layer without separate runtime lifecycle |
| Commander / AgentsCommander | app-level supervisor | competitive research (review/diffs, terminal navigation, onboarding) | n/a | UX research | They are not interchangeable with coding agent CLIs |

Adoption ordering factors: active maintenance, download trends, protocol observability, authenticated CLI workflows, cross-platform installability, capability evidence, burden of tests. Stars are *signals*, not user counts.

Sources checked: 
- ACP protocol: https://agentclientprotocol.com/ and https://github.com/agentclientprotocol/agent-client-protocol/blob/main/docs/protocol/v2/overview.mdx
- OpenCode ACP: https://dev.opencode.ai/docs/acp/
- OMP RPC: https://github.com/can1357/oh-my-pi/blob/main/docs/rpc.md ; OMP ACP: https://github.com/can1357/oh-my-pi/blob/main/README.md
- Pi RPC: https://pi.dev/docs/latest/rpc ; https://pi.dev/docs/latest/cli-integration
- Xiaomi MiMo Code: https://github.com/XiaomiMiMo/MiMo-Code ; https://github.com/XiaomiMiMo/awesome-mimo-agent/blob/main/docs/mimo_code.md
- DeepSeek Harness: https://github.com/deepseek-ai/deepseek-harness ; https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/cli-help.md
- Commander: https://thecommander.app/docs ; AgentsCommander: https://github.com/mblua/AgentsCommander
- Cross-provider reference (not contract authority): https://github.com/giuliastro/harness-remote

## Architecture: transport != provider != model

```
                Nexus UX (Web / Desktop / Mobile)
                               |
               Attention read model & commands
                               |
        Existing Runtime Registry + existing Event Bus
                               |
     Per-runtime reducer + durable pending interventions
                               |
          Normalized observation/response contracts
                  /           |             \
            ACP Client     JSONL RPC       PTY/OSC/hooks
                  \           |             /
                  Provider-specific profile/adapters
                               |
               User-installed, authenticated CLI
```

Implement shared transport clients beneath provider-specific optional adapters, not a second provider registry. Driver metadata should report `installedVersion`, `executable`, `executionMode`, `protocol`, `verifiedCapabilities`, `evidence` and `lastCheckedAt`. Source of truth for credentials stays at provider's isolated home/auth store. Respect user-selected protocol mode.

### Normalized semantics

Do **not** collapse process, turn and intervention into a single string:

- Process: `STARTING | ALIVE | EXITED | CRASHED | UNREACHABLE | UNKNOWN` (verified supervisor).
- Turn: `UNKNOWN | IDLE | RUNNING | SETTLED | ERROR | CANCELLED` (provider-verified where possible).
- Attention: `NONE | QUESTION | APPROVAL | AUTH | FAILED`, with durable `PENDING | SUBMITTING | RESOLVED | EXPIRED | CANCELLED`.
- Evidence: `AUTHORITATIVE | SIGNALLED | INFERRED | UNKNOWN`; event source is `ACP | RPC | PROVIDER_HTTP | SDK | HOOK | OSC | PTY | PROCESS`.
- Action capabilities per request: `FOCUS_TERMINAL | REPLY_TEXT | CHOOSE_OPTION | APPROVE | REJECT | RETRY`. A displayed question is NOT automatically remotely actionable.

Suggested envelope:

```json
{
  "schemaVersion": 1,
  "eventId": "uuid",
  "runtimeId": "rt-123",
  "agentId": "agent-123",
  "projectId": "project-123",
  "providerId": "omp",
  "providerSessionId": "provider-session",
  "turnId": "turn-1",
  "runtimeEpoch": 2,
  "sequence": 17,
  "type": "INTERVENTION_REQUESTED",
  "source": "ACP",
  "evidence": "AUTHORITATIVE",
  "observedAt": "2026-10-27T15:00:00Z",
  "payload": {
    "interactionId": "provider-or-runtime-scoped-id",
    "kind": "APPROVAL",
    "summary": "Run proposed command?",
    "options": [],
    "responseCapabilities": ["FOCUS_TERMINAL"]
  }
}
```

Dedup **per runtime + epoch + provider-interaction ID**, not solely by text (two agents asking the same question remain two independent blockers). Application alerts may group by visual summary, but never merge underlying actions.

Provider event/ACP correlation should drive resolution; user input sent to PTY only marks `SUBMITTING`, not `RESOLVED` until verified. On detach/reconnect, re-fetch runtime state and pending prompts; expire old interactions after process exit or confirmed turn change. Handle provider session restart with new epoch and fencing; missing heartbeat is `UNKNOWN`, not "completed".

**Security:** no automatic `yes`/permission grants; no secret echo in notification/persistent events; authenticate caller, bind session ID and project to authorization checks, reject stale interaction versions, use one-time idempotency keys and explicit user intent. Fallback CLI launch never upgrades permissions by accident. ACP agent requests to read/write local files and execute commands must pass Nexus workspace policy; do not implement ACP as an unrestricted filesystem/terminal proxy.

**Delivery:** a single server-side attention projection reconciles events. Avoid per-tab pushes and duplicate toasts; notifications must degrade safely when browser/tab is offline. Mobile response only when the protocol exposes a secure, correlated option, otherwise deep-link to remote terminal.

## Acceptance criteria by phase

### Phase A — shared observation foundation [P0]
- [ ] Protocol mode detection by binary/version and probe, with evidence truth.
- [ ] Shared ACP client: spawn, initialize, session/new/load/resume when advertised, updates, permissions, cancellation, shutdown, stderr isolation, backpressure, timeout.
- [ ] Shared JSONL RPC client: strict LF framing, request ID correlation, bounded frames, reconnect/restart strategy where possible.
- [ ] Existing PTY behavior retained without breaking interactive sessions.
- [ ] Unit + protocol contract tests with truncated JSON, malformed frames, old versions, races, duplicate events.
- [ ] Release documentation of supported/partial/unsupported controls.

### Phase B — harness adapters [P0]
- [ ] OpenCode structured channel, without regression of terminal mode.
- [ ] Pi and OMP adapter: detect, authenticate, launch, prompt, state, question, cancel, resume where official API permits.
- [ ] MiMo and DSH adapters: isolated profiles, structured events, session control and capability-aware approvals.
- [ ] Integrated terminal and new structured mode clearly selectable/observable.
- [ ] Linux/macOS/Windows provider smoke matrix; real authenticated E2E proof for each claimed capability.

### Phase C — universal attention [P0]
- [ ] Unified Attention Center for every supported runtime and mission projection, without duplicating source of truth.
- [ ] Correct question/approval/auth/status transitions, server-side dedup, per-runtime identity.
- [ ] Safe correlated response, retry and terminal focus; busy/sending/error UI state.
- [ ] No false "completed" from idle, echoed log text or PTY silence.
- [ ] Two agents/clients (desktop+mobile/browser) simultaneously, detach, reconnect, crash, restart, stale response.
- [ ] No automatic permissions; no duplicated push notifications.
- [ ] Accessibility and localization of attention states.

### Phase D — discoverability and remaining harnesses [P1/P2]
- [ ] Provider picker categorized by **installed/runnable**, **available to install**, **unsupported mode**, model choice independent.
- [ ] One-click probe/doctor: CLI version, login state, compatible transport, feature gaps, troubleshooting.
- [ ] Compatibility discovery for Oh My OpenAgent; *not* a duplicate provider unless separately executable.
- [ ] Evaluate Goose, Qwen, Hermes, Cline, Kimi, Kilo, Copilot using publicly versioned evidence and representative E2E.
- [ ] Competitive UX notes from Commander and AgentsCommander without embedding their product layers.

## Non-goals

- Reimplement providers' agent loops and their own permission policy.
- Recreate Maestro's orchestration or memory.
- Force all harnesses into a single lowest-common-denominator protocol.
- Hide capability gaps or claim a provider is supported merely because its executable launches.
- Treat screenshots of TUI or arbitrary regex text as high-confidence approval APIs.

## Definition of Done

A provider is **SUPPORTED** only after test evidence demonstrates: native install/detection, authenticated launch, prompt/turn lifecycle, two distinct concurrent sessions, accurate attention requests, safe interaction or clearly view-only UI, detach/reconnect, cancellation, restart/stale-state behavior, and no regression on existing terminals. Report any missing capability explicitly as **PARTIAL**, **UNSUPPORTED** or **NOT_TESTED**. No production release claim until real E2E evidence has been attached to release validation.
