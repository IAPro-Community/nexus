# Structured Harness Runtime — Preview Operating Contract

**Target branch:** `release/v1.0.0`  
**Status:** experimental / not certified, gated by `NEXUS_EXPERIMENTAL_HARNESSES=1`.  
**Last reviewed:** 2026-10-27

## What is actually implemented

The existing `SessionHost` now selects `structuredBackend` only for a runtime explicitly created with `agent_protocol=acp` or `agent_protocol=pi_rpc`. The host still owns the child process, PID/generation, IPC, attach, stop/terminate, ring buffer and Registry. These sessions are **not** PTY/ConPTY: they have a terminal-shaped, text-only Nexus command surface over the existing attach transport and a provider subprocess whose stdin/stdout are reserved for the binary JSONL protocol.

The shared `agentwire` library owns ACP JSON-RPC (initialization, session creation, prompt, cancel, permission callbacks) or Pi RPC (command acknowledgement, event stream, `agent_settled`, extension-ui dialogs). `AttentionDetector.ApplyProviderSignal` translates signals into existing registry and event bus state; PTY heuristics are disabled for these runtime instances. The browser Attention Card shows structured questions as `native_status`/terminal-only until an authenticated, version-fenced option API is available. No unrestricted filesystem/terminal ACP client capability is advertised.

**The adapter is version-sensitive and not verified against live authenticated tools.** ACP currently attempts protocol version 1. Check actual installed version; do not assume all ACP V2-only harnesses will connect. Pi/OMP UI messages are handled according to the documented `extension_ui_request` and `extension_ui_response` contracts.

## Supported preview launch pairs

| provider | agent_protocol | executed command | default |
| --- | --- | --- | --- |
| opencode | acp | `opencode acp` | existing PTY |
| omp | acp | `omp acp` | existing PTY |
| omp | pi_rpc | `omp --mode rpc` | existing PTY |
| pi | pi_rpc | `pi --mode rpc` | existing PTY |
| mimo | acp | `mimo acp` | existing PTY |
| dsh | acp | `dsh acp` | **no PTY fallback** |

Executable discovery and version probing are provided by the existing `ControlDriver` system. The four added preview drivers register only when the environment flag is active. They must not be labeled `SUPPORTED` for structured control in provider metadata before same-version E2E passes.

## Feature discovery decision

The experimental ACP/RPC selector is not promoted as a public feature
announcement or guided tour yet: this modal lacks an official feature-tour
surface, and the transport remains unverified end-to-end. Announcing it as
ready would misrepresent the release. After authenticated E2E certification,
register a real modal-scoped tour target and localize its announcement in all
supported languages, following `AGENTS.md` and the product discovery contract.

## Experimental agent creation in the UI

The existing **New Agent** modal now provides an **Agent protocol (experimental)**
selector inside *Advanced* for compatible installed providers. The selector
is populated exclusively from `GET /api/v1/providers[].structured_modes`, which
the Core returns only with `NEXUS_EXPERIMENTAL_HARNESSES=1` and a detected CLI.
Terminal/PTY remains the default; the user must choose a registered profile
for ACP or Pi RPC. The experimental selection pins the chosen provider and
profile (no scheduler fallback), and disables YOLO mode for structured runs.

These are *declared preview entrypoints*, not version-certified capabilities.
A provider with a recognized binary can still fail handshake or require auth.
When Core opt-in is off, this selector is not offered.

**Validation layering:** two supervised mock ACP subprocesses plus two
authenticated HTTP clients are now covered in
`internal/control/web/structured_multiclient_process_e2e_test.go`. That test
crosses the real SessionHost/IPC/JSON-RPC/Registry/HTTP boundaries and never
manipulates Registry attention state to pretend a reply. It does **not**
replace authenticated-provider testing or a genuine two-browser UI test.

`agentwire.Reducer` now participates in live structured event handling and
retains pending interventions across uncorrelated progress. Browser toasts are
also refreshed on changed request fingerprints rather than solely on runtime
identity.

## How to start a test runtime

1. Install and authenticate the provider CLI **outside** Nexus using its official flow.
2. Start the Nexus Core with `NEXUS_EXPERIMENTAL_HARNESSES=1`. Do not enable this by default for users.
3. Use the authenticated `POST /api/v1/runtimes` endpoint with the existing session/CSRF contract and a request like:

```json
{
  "provider": "opencode",
  "profile": "default",
  "workspace": "/absolute/existing/project/path",
  "agent_protocol": "acp"
}
```

Use `"pi_rpc"` for Pi. This API intentionally rejects unsupported provider/protocol combinations, custom command templates, unverified model override flags and non-empty raw CLI args. Agent input must be sent through the existing authenticated runtime/terminal endpoint; raw JSON protocol bytes must not be sent through xterm.

## Operator interaction

- Enter a normal text line: submit a new `session/prompt` or Pi `prompt`.
- `/cancel`: request provider cancellation using ACP `session/cancel` or Pi `abort`.
- `/permit req-1 allow_once`: select **only** an offered ACP option, or a Pi select/confirm option. It must exactly match the displayed interaction and option identifier; unrecognized/stale options are rejected.
- `/answer ui-1 text to send`: answer a Pi input/editor dialog. Terminal output is sanitized/redacted; these commands are still user-directed operations.

This interactive bridge does **not** implement high-confidence remote permission buttons yet. An OSC-only or structured attention signal alone does not authorize a generic `yes` write. UI cards must focus the correct terminal until the correlated authenticated RPC/ACP answer endpoint exists.

## Verification requirements before declaring supported

- `go test -race ./internal/control/agentwire` and host/driver/launcher/web tests with Go version from go.mod.
- Native provider sessions for OpenCode, Pi, Oh My Pi, MiMo and DeepSeek Harness, including a nontrivial prompt, completion, cancellation and actual permission/question.
- Cross-platform Windows ConPTY/pipe compatibility and cleanup of process trees; macOS/Linux cross-provider smoke.
- Two clients and two concurrent runtimes; two different sessions asking the same question must remain independent.
- Kill during permission, slow startup, invalid ACP method, JSONL overflow, stdout/stderr poisoning, malformed events and session restart.
- Verify that no runtime reports READY before ACP initialization, and no terminal or future provider session gets synthetic completion on silence.
- Attach same-SHA CI run, sanitized runtime traces, version matrix and screenshot proof to #97–#100.

Existing issues #97–#101 remain OPEN until these criteria are met; this code is intentionally experimental.

## OMP CLI supervised preview (opt-in, not certified)

This preview makes `nexus omp` available through the existing SessionHost
PTY lifecycle. It does **not** register an OMP account with the canonical Usage
scheduler, infer quota, promise remote approval actions or enable automatic
delegation. The existing experimental ACP and Pi RPC launchers are unchanged.

First install the official `omp` binary from upstream, then:

```bash
export NEXUS_EXPERIMENTAL_HARNESSES=1
nexus add omp default --no-login
nexus omp:default
# equivalent when only one profile exists:
nexus omp
```

The operator authenticates/configures OMP within the terminal for this isolated
profile. `nexus omp --help` and `nexus help omp` document these constraints.
With the feature flag disabled, OMP profile creation and execution fail closed.

**Isolation:** the supervised preview sets HOME and OMP's native
`PI_CONFIG_DIR=.omp`, `PI_CODING_AGENT_DIR=<nexus-profile-home>/.omp/agent`;
it clears inherited `OMP_PROFILE`/`PI_PROFILE` and foreign config/session/
worktree overrides. This intentionally differs from Pi's `.pi/agent` root.
User-supplied OMP flags remain native and the selected profile has no certified
credential/usage auto-switching. Launching `omp acp` or `omp --mode rpc`
inside PTY is rejected; use the explicit `agent_protocol` in Web for those
experimental modes.

**Not yet delivered:** OMP native authentication/identity projection in Usage,
CLI version handshake, real ACP/RPC question and approval round trips,
remote safe answer UI, two simultaneous real sessions, cross-platform E2E,
and verified native resume. Do not mark issue #98 as finished or announce
this as a production-ready provider.
