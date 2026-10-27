# Nexus Browser Collaboration — complete human/agent cycle

## Product requirement

One Nexus workspace is accessible from desktop or mobile while the browser and
application under test run on the Nexus Core host. A terminal coding agent
uses the **same browser session** through a scoped, explicitly authorized MCP
connection. When login, MFA or manual parameters are needed, the agent pauses,
the user completes the sensitive steps in the **existing Nexus live viewer**,
and the agent resumes only after explicit completion.

The live viewer, Chrome DevTools Protocol (CDP) and Nexus Workspace remain
managed by the same Go Core. Maestro is not replaced. Browser capabilities do
not grant filesystem, mission or Nexus administration access.

## Open-source implementation decision (October 2026)

| Project | License | Reusable idea | Integration decision |
| --- | --- | --- | --- |
| [Vercel agent-browser](https://github.com/vercel-labs/agent-browser) | Apache-2.0 | Session-aware CLI, WS frame delivery, keyboard/touch input, agent skills and CDP connection | Optional future provider; do not spawn an unmanaged parallel browser in the default path |
| [Microsoft Playwright MCP](https://github.com/microsoft/playwright-mcp) | Apache-2.0 | Semantic navigation, accessibility snapshots, MCP tools, isolated profiles | Model the tool contract after MCP; use Playwright as an optional richer execution provider, not a second permission system |
| [Steel Browser](https://github.com/steel-dev/steel-browser) | Apache-2.0 | Session manager, live debugger, profile management | Optional provider candidate, not bundled in the Go release; **Steel Cloud features must not be assumed available when self-hosting** |
| [browser-use](https://github.com/browser-use/browser-use) | MIT | Task-focused agent browsing | Do not introduce another mandatory agent/orchestrator |
| [MCP specification](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports) | Protocol | Stateless Streamable HTTP via JSON-RPC | Minimal native implementation, scoped to one session |

No third-party browser source code has been copied or vendored into Nexus.
Before any future vendoring preserve the upstream license, attribution,
NOTICE obligations and track upstream security updates.

## What works

1. Open a Browser surface inside a Nexus project.
2. Select **Conectar Codex / Claude / outro agente via MCP** and
   **Gerar acesso temporário**. This creates a **one-hour browser-only
   bearer token** valid only for that browser instance.
3. Copy the setup commands from the UI. The MCP endpoint is always bound
   to the same Nexus Core host's **loopback** interface. No separate
   localhost forwarding is needed for the agent running on that machine.
   The user interface can be reached through the existing Nexus tunnel.
4. Give the coding agent permission to use the Browser MCP, then instruct it
   to call `browser_request_control`, `browser_navigate`, `browser_snapshot`
   and `browser_click` / `browser_fill`. An existing human controller
   must release control before an agent can acquire it.
5. When a logged-in page is required, the agent calls:
   `browser_request_human({"reason":"Sign in and select organization"})`.
   Its input lease is released, and subsequent agent snapshots/screenshots
   and mutations are rejected while the handoff is pending.
6. The Nexus Browser shows the intervention banner on desktop or mobile:
   **Assumir para autenticar**. The user clicks or taps the page, enters
   login/password/MFA and selects required parameters. These values remain
   inside the browser and are **not passed to the MCP tool response**.
   The UI password-mode input masks characters while the handoff is active.
7. The user presses **Concluído · devolver ao agente**. The Core records the
   completed handoff, releases human control, and permits the agent to
   reacquire its own lease. The agent can poll `browser_wait_for_human`
   (up to 9 seconds per call) and then `browser_request_control`.
8. The agent can inspect the authenticated result without being given the
   password or MFA code. Closing the Browser session revokes all grants.
   Tokens can also be manually revoked from the same Browser surface.

## Codex terminal setup

The UI provides a one-time, scoped token and the actual Core loopback URL,
including its configured port:

```sh
# On the machine that runs Nexus Core (not your mobile browser):
export NEXUS_BROWSER_TOKEN='<temporary-token-from-Nexus>'
codex mcp add nexus-browser \
  --url http://127.0.0.1:13000/api/v1/browser/mcp \
  --bearer-token-env-var NEXUS_BROWSER_TOKEN
```

The token must remain available to the Codex process. Replace port 13000 with
the actual port. Use a secure mechanism for environment variables; avoid
committing secrets to project files or shell history.

A generic Streamable HTTP MCP client can set
`Authorization: Bearer <temporary-token>`. This endpoint is intentionally
not public; remote ChatGPT/cloud agents cannot connect directly through the
Nexus tunnel. An authenticated remote bridge would require its own
granular permissions and safety audit.

## MCP tools available

- `browser_state` — project/browser identity, controller and handoff status
- `browser_request_control`, `browser_release_control` — 25-second input lease
- `browser_navigate`, `browser_click`, `browser_fill`, `browser_wait`,
  `browser_key` — serialized mutations on the same page
- `browser_snapshot`, `browser_screenshot` — read-only observation,
  **disabled for agent capabilities during a human intervention**
- `browser_request_human`, `browser_wait_for_human` — safe human
  collaboration without copy/pasting user credentials into AI chats

The MCP implementation uses stateless POST JSON-RPC and does not spawn
a new Chrome process. It requires a separate Nexus Browser session and
explicit grant. The Core itself is the source of truth for input ownership.

## Session state machine

```text
Agent controls -> browser_request_human -> REQUESTED
REQUESTED -> human begins -> HUMAN_ACTIVE
HUMAN_ACTIVE -> user completes -> COMPLETED
COMPLETED -> agent reacquires -> Agent controls
REQUESTED / HUMAN_ACTIVE -> user cancels -> CANCELLED
```

**Important:** a control lease expires after 25 seconds; a human intervention
does not. The agent cannot silently resume when the phone loses signal.
Control changes and CDP mutations are serialized by the same session mutex.

## Tests and release gates

```sh
go test ./internal/browsercontrol ./internal/control/web -count=1
NEXUS_BROWSER_INTEGRATION=1 go test ./internal/browsercontrol -run TestChromiumSmokeOptional -count=1
cd web && bun run typecheck && bunx vitest run src/app/browserSurfaces.test.ts && bun run build
```

End-to-end manual demonstration on a real Nexus Linux host:

1. Start web server in a terminal, create Browser and open localhost.
2. Connect a Codex agent via an explicit browser-only MCP grant.
3. Agent requests control and navigates to a login-protected page.
4. Agent requests human, and the phone shows the intervention banner.
5. Verify agent cannot call screenshot/snapshot/click while waiting.
6. User types credentials and completes MFA from the Nexus mobile viewer.
7. User selects parameters and presses **Concluído**.
8. Agent reacquires control and validates the authenticated screen.
9. Revoke token and confirm further MCP calls return HTTP 401.
10. Close Browser, check Chromium child process and temporary profile cleanup.

## Boundaries still requiring independent work

- Automatic MCP grant injection into every terminal runtime, mission and
  worker is **not implemented**. Manual authorized connection works for
  compatible terminal clients. MissionRunner AutonomyContract integration
  remains a separate security boundary, not something the Browser MCP can
  bypass automatically.
- The browser runtime currently has a maximum of **four isolated Chromium
  sessions**. Multi-context Chromium pooling, persistent profiles and
  background session suspension are not included.
- Nexus tunnel sessions have broad Nexus permissions. Do not grant remote
  Nexus access to untrusted users, and do not treat an MCP browser grant as a
  project/tenant RBAC substitute.
- During human intervention, agent MCP observation is blocked, but ordinary
  authorized Nexus viewers can still see the browser stream. Private
  multi-user viewer isolation requires further access control.
- Saved cookies exist only in the temporary Chromium profile and disappear
  when the browser session is closed. Credentials and OTPs should never be
  included in prompts, logs, screenshots saved as mission evidence or
  browser automation scripts.
- The Nexus viewer is JPEG streaming, not a full OS remote desktop. Test
  touch gestures and keyboard focus on real mobile devices.
