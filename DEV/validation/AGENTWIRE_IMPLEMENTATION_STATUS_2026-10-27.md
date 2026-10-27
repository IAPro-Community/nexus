# Validation: universal harness/attention implementation slice (2026-10-27)

> **RELEASE GATE: NOT READY.** This is a partial engineering implementation, not certification of issues #97–#101. Authoritative source of truth remains the GitHub issues and `docs/design/universal-agent-harness-support.md`. Target branch: `release/v1.0.0`.

## Delivered in this pass
- AGY: native OSC 7501 parser and runtime attention integration; provider-selected `AGY_CLI_OSC7501=on` unless disabled by environment; mapped question / approval / auth / working / error / done. Program-status signal supports **terminal focus only**, not remote permission approval.
- AGY regression tests: split status frames; provider blocker precedence over PTY narration; separate native status; no clearing a native blocker from Enter alone.
- `internal/control/agentwire`: bounded (4 MiB) newline JSON transport, ACP bidirectional JSON-RPC base, Pi/OMP JSONL request+event client, per-runtime epoch/sequence reducer and independent attention requests.
- `internal/control/driver/harness_preview.go`: opt-in preview drivers for `pi`, `omp`, `mimo`, `dsh` via `NEXUS_EXPERIMENTAL_HARNESSES=1`. DSH refuses unsafe pretend-interactive launch; other drivers advertise unsupported structured events and partial resume.
- Frontend native-status prompts are informational and allow **open terminal**, not arbitrary text replies; runtime types extended.

## Validation performed
- An isolated, mirrored `agentwire` module containing the same transport semantics was run with local Go 1.23: `go test -race -count=1 ./internal/control/agentwire` **PASS**. This does **not** establish full repository build or same-SHA certification: `go.mod` requires Go 1.25.
- OSC parser-only isolated unit command: `go test -count=1 ./internal/control/host/osc7501.go ./internal/control/host/osc7501_test.go` **PASS**.
- **NOT RUN:** real authenticated providers, end-to-end terminal / API flows, cross-platform Windows ConPTY, full Go 1.25 build / lint / typecheck, frontend typecheck / E2E, same-SHA CI, real browser+mobile concurrent clients.
- The existing `AttentionDetector` integration tests were added to the release but **not executed in the repo**.

## 27 October addendum: SessionHost integration implemented (not yet certified)

The initial blocker recorded below has advanced in the release. The code now selects a protocol-specific backend inside the existing `SessionHost` for explicitly created `agent_protocol=acp|pi_rpc` sessions, gated by `NEXUS_EXPERIMENTAL_HARNESSES=1`. The host owns the process/IPC/session lifecycle and waits for the structured handshake before marking startup ready; it does not automatically convert any existing PTY session.

- `internal/control/host/structured_backend.go`: supervised stdin/stdout process with bounded UI projection, terminal-shaped text input, explicit `/cancel`, `/permit`, `/answer`; no uncorrelated approval.
- `internal/control/agentwire/bridge.go`: ACP initialize/new/prompt/cancel + permission options; Pi/OMP RPC prompt, `agent_settled` and extension UI select/confirm/input/editor requests.
- `internal/control/launcher/launcher.go` + `internal/control/driver/structured_command.go`: capability-scoped command path; default PTY path unchanged.
- `internal/control/web/handlers_api.go`: explicit API request `agent_protocol`.
- `internal/control/host/host.go`: registry/event bridge and stale-attention cleanup on process exit.
- New fake protocol server tests under `agentwire/bridge_test.go` and `host/structured_backend_test.go`.
- GitHub workflow `.github/workflows/agentwire-verification.yml` was added; **its status has not been confirmed**. Adding a CI workflow is not evidence that it passed.

**Remaining critical work:** compiler/full test verification at release HEAD; Go formatting; real authenticated E2E for each harness; authenticated option-selection HTTP contract (currently terminal-only); version negotiation beyond ACP v1; validated process-tree termination on Windows; complete provider-picking UX; integration of the standalone normalized reducer with the global attention projection. These are NOT certified.

### Revised blocker interpretation

The historical blockers in the next section reflect the situation *before* the addendum. `ACP not wired into SessionHost`, `Pi client has no launcher` and `MiMo/DSH ACP launch not integrated` are no longer true of the source implementation, but they remain **unvalidated runtime capabilities**. Do not close their GitHub issues or advertise them as supported until their release criteria pass.

## Remaining blockers (issues are intentionally open)
- **#97:** ACP transport is **not wired into SessionHost**. Need owned subprocess pipes, startup handshake, advertised capability negotiation, request/permission policy, cancellation/cleanup, domain event publication and trace verification. Until then Codex/OpenCode remain terminal-only.
- **#98:** Pi/OMP RPC clients exist as libraries, but no structured launcher or persisted session lifecycle integration; preview PTY paths need actual authenticated validation.
- **#99:** MiMo/DSH ACP launch not integrated; DSH is deliberately not launchable in preview PTY.
- **#100:** Universal state reducer not subscribed to Nexus event bus/registry. AGY OSC path is a targeted improvement; entire Centre of Attention is not yet universal. Existing PTY heuristic behavior remains for other providers.
- **#101:** Competitive matrix/selection documented, but no Goose/Qwen/Hermes/Cline/Kimi/Kilo/Copilot adapters installed or verified.
- **Provider architecture:** preview drivers are opt-in and not an integration guarantee. Any user-visible provider picker must label preview status and must not infer `SUPPORTED` from a detected executable.
- **Security:** no automatic approvals; no silent PTY → headless migration; no remote answer until provider-origin interaction IDs, session ownership and response policy are verified.

## Next validation commands (with Go 1.25 and checked-out release SHA)
```bash
go test ./internal/control/agentwire ./internal/control/host ./internal/control/driver
go test -race ./internal/control/agentwire ./internal/control/host
go vet ./internal/control/agentwire ./internal/control/host ./internal/control/driver
make format-check
cd web && yarn typecheck
```
Then run real authenticated AGY/OpenCode/Pi/OMP/MiMo/DSH behavior per capability and Windows/Linux/macOS. Attach same-commit CI, terminal traces with secrets redacted, two-client concurrency and security checks to the issues before closing any of them.

## Change-control
Keep implementation on `release/v1.0.0` per issue #27, avoid a parallel beta. Do not confuse a library/test passing with a functional, user-facing provider adapter. Promote each provider independently only after measured capability and native E2E evidence.
