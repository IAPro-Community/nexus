# Nexus Browser — automatic multi-agent access

## Goal

Browser tools are a **Nexus capability**, not a Codex plugin. Interactive
supervised runtimes launched with Nexus—Codex, OpenCode, AGY, Claude, Gemini,
and Cursor—receive a scoped MCP connection when a local Nexus Core web server
is running. The Nexus Core owns the Browser session, video stream, permissions,
human takeover and project association. Maestro remains independent.

The integration is implemented at the common `internal/control/launcher`
boundary, not in bespoke Browser APIs for each model.

## What the user does

1. Open the Nexus Workspace; make sure the local Core is running.
2. Start an agent as usual, e.g. `nexus opencode`, `nexus agy` or
   via the Project's Agents UI. No `mcp add` or token copy is needed for
   a supported supervised runtime.
3. Ask the agent to open or validate the local app with Nexus Browser.
   The agent calls `browser_open` on the injected MCP service; Chromium
   starts **only then**. It returns the browser session ID and URL.
4. Open **Novo navegador** inside the same project: it attaches to an
   agent-owned browser when present and displays the live stream. It
   initially observes without stealing the agent's control lease.
5. The agent uses `browser_request_control`, `browser_navigate`,
   `browser_snapshot`, `browser_click`, `browser_fill`,
   `browser_wait` etc.
6. At login/MFA/parameter selection, the agent requests human intervention.
   The owner can take over on desktop or mobile, finish private input, then
   explicitly return control. The AI resumes the existing authenticated
   page without receiving passwords or OTPs.
7. The scoped runtime capability is revoked on stop. Abandoned sessions
   are rejected because the runtime must still be alive; credentials
   have a 24-hour maximum lifetime.

## Native provider adapters

| Provider | Nexus-managed integration | Configuration storage |
| --- | --- | --- |
| Codex | `-c mcp_servers.nexus-browser...` with bearer-token environment reference | Invocation arguments, token in child environment only |
| OpenCode | `OPENCODE_CONFIG_CONTENT` merged with current inline config; remote MCP with environment reference | Runtime environment only |
| Claude Code | `--mcp-config` with native stdio bridge | Owner-only runtime config file (no bearer token) |
| Gemini CLI | Runtime-specific `GEMINI_CLI_SYSTEM_SETTINGS_PATH` with native stdio bridge | Owner-only runtime config file (no bearer token) |
| AGY CLI | Merge the native `mcpServers` entry in its Nexus-isolated profile `~/.gemini/config/mcp_config.json` | Profile config containing only local `nexus __browser-mcp-stdio` command |
| Cursor CLI | Same native stdio bridge via its Nexus-isolated `.cursor/mcp.json` | Isolated profile config, no bearer token |

Nexus never edits the host user's global provider settings or project
MCP files. AGY/Cursor profile entries are static, carry no secrets, and do
not override an existing non-Nexus `nexus-browser` MCP server entry.

The local `nexus __browser-mcp-stdio` transport is a private internal
command. The supervised child inherits `NEXUS_BROWSER_TOKEN` and
`NEXUS_BROWSER_MCP_URL`, then relays JSON-RPC to the **same** browser
Core via loopback HTTP. A process outside Nexus cannot use those values
without a live, registered runtime.

## Isolation

- Each runtime gets its **own** short-lived `nbr_` token, stored on disk
  only as a SHA-256 digest in an owner-only directory.
- The broker checks the runtime's liveness and project identity on each
  request. Tokens are not general Nexus authentication credentials.
- A runtime's `browser_open` returns the already-owned Browser if present.
  It cannot attach to another runtime's browser without a separately
  authorized human-issued session grant.
- Creating a browser is lazy; session limit remains 4 until multi-context
  Chromium pooling is delivered.
- Native agent input is serialized by the Browser control lease. When
  a human takes over, MCP page observation and mutation are blocked.
- Noninteractive coding Missions with `allow_external_network=false`
  **do not** get browser tools, because Browser navigation can bypass
  network guards implemented only for CLI commands.
- Custom SSH, Docker, remote command-template providers are **not** injected:
  the local Core loopback address may be inaccessible in the target. Their
  integration needs a separately authenticated relay.

## Operational prerequisites and known limitations

- A Nexus Core web server must already be running; standalone direct
  provider invocation with no local Core does not inject Browser MCP.
- For AGY, Claude, Gemini and Cursor, the `nexus` (or `ai`) command
  must be resolvable by the host runtime; the stdio helper is not a
  separate package or paid plugin.
- A provider must support MCP and load the documented format. This is
  **automatic provisioning**, not proof of end-to-end interoperability with
  every installed CLI version. Validate native provider discovery.
- Web/desktop agents launched through the common Launcher get the
  capability. For bare `nexus opencode` with no preselected Project,
  Nexus resolves or creates the current workspace Project so the
  Browser appears in the normal Agent Workspace.
- Provider processes can inspect authenticated pages **after** human
  takeover ends; do not use it for high-risk production credentials
  without isolating remote spectators and network destinations.
- Bridge config injection should remain user-controllable via a runtime
  permission UI before remote or multi-user use. Mission contracts already
  guard network access, but granular project-scoped RBAC remains future work.
- Only the existing Chromium/CDP engine is required. Steel, Playwright MCP,
  and agent-browser remain optional reference technologies, not dependencies.

## Test plan

Run the Browser validation workflow, launcher unit tests and integration
smoke tests:

```sh
go test ./internal/browsercontrol ./internal/control/launcher ./internal/control/web -count=1
go test -race ./internal/browsercontrol -count=1
go vet ./internal/browsercontrol ./internal/control/launcher ./internal/control/web
cd web && bun run typecheck && bun run build
```

On an actual Linux host, run `nexus web` and `nexus opencode`, ask
OpenCode to open Browser, attach from desktop and mobile, request human
login, and confirm agent continuation. Repeat with AGY/Claude/Gemini
and the Mission runner under an explicit network authorization.

For previous manual session grants and handoff behavior, see
[Browser collaboration](./nexus-browser-collaboration.md).
