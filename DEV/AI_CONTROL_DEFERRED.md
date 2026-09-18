# AI Control — Deferred Backlog & Non-Goals

The following capabilities have been deliberately deferred from the current milestone (`v0.4.0`) to protect architecture simplicity, security boundaries, and core runtime reliability:

1. **Public Internet Hosting & Cloud SaaS**:
   - *Rationale*: Exposing developer workstations and CLI subprocesses directly to the public internet requires complex multi-tenant IAM, authentication proxies, WAFs, and DDoS protections. Access is strictly scoped to loopback, private VPNs, and SSH port forwarding tunnels.

2. **Monaco / Full IDE Code Editor**:
   - *Rationale*: AI Control is a local control plane for coding agents and terminals, answering *"what are my agents doing and how do I control them?"* It does not compete with or replace VS Code, Cursor, or JetBrains IDEs.

3. **Autonomous Task Planner / Multi-Agent Orchestrator**:
   - *Rationale*: Agent planning and reasoning belong inside the coding agents themselves (e.g. Codex, Claude Code, Gemini CLI, AGY). AI Control acts as the runtime supervisor, account rotator, and terminal multiplexer.

4. **Multi-User Enterprise RBAC**:
   - *Rationale*: AI Control is designed primarily for developer pair-programming and individual local workstation supervision. Multi-user role-based access control is deferred until multi-node Hub topologies are stabilized.

5. **Direct Public Cloud Token Proxy**:
   - *Rationale*: Storing or proxying third-party API tokens centrally introduces credential harvesting attack vectors. Credentials remain isolated in developer keyrings and official provider credential stores.

6. **Codex app-server / structured attention events** (next attention milestone):
   - *Status*: Deferred by design **for control**. Drivers still report `StructuredEvents` / `Approvals` as UNSUPPORTED and `ControlLevel=TERMINAL`.
   - *Already integrated (not deferred)*: the **quota-only** slice of the app-server. `internal/core/provider/adapters/codex/app_server_usage.go` spawns a short-lived `codex app-server --stdio` per profile and calls `account/rateLimits/read`, producing `OFFICIAL_API` snapshots without a session, a prompt or quota consumption. It is read-only, identity-gated (the reported `accountId` and the echoed `codexHome` must match the profile), memoized for 60s, and disabled by `NEXUS_CODEX_APP_SERVER=0`. This is deliberately *not* the control adapter: it starts no conversation, subscribes to no events, and grants no approval surface.
   - *Current honest fallback*: `AttentionDetector` scrapes PTY stdout **only** when `ControlLevel=TERMINAL`, provider is not `shell`, and no structured adapter is enabled. Attach WebSockets are display-only; browser push comes from the focused-project registry poll.
   - *Target*: Enable a Codex app-server (JSON-RPC) adapter that publishes domain events (`APPROVAL_REQUIRED`, `AGENT_WAITING`, turn state) with `runtime_id` + `agent_id`. When that lands, set `StructuredEvents=SUPPORTED` (and/or `ControlLevel=EVENTS`) so PTY heuristics turn off automatically via `SetControlPolicy`.
   - *Non-goal of the fallback*: Guessing agent turns from tool logs (`error:`, apt `[Y/n]`) is not a product contract — only a TERMINAL-mode safety net until the adapter exists.
