# Nexus Browser — Agent Workspace (experimental)

The Browser is a first-class `WorkspaceSurface` of type `browser`.
It shares the existing **Tabs, Split, Desktop and Mosaic** presentation
with terminal/agent surfaces; there is no second window manager or agent
orchestrator. Maestro remains independent.

## Usage

1. Install Chrome, Chromium or Edge on the machine running the Nexus Core.
   Set `NEXUS_CHROMIUM_PATH` if the executable is not discoverable.
2. Start Nexus, open a project and go to **Terminais → Novo navegador**.
3. The Browser starts with `http://localhost:3000` to support the
   project's local development server. An unavailable URL can show Chrome's
   network error page without destroying the session; enter another URL.
4. Use the address bar, Back/Forward, Reload, click-to-control, scroll,
   text insertion, Enter/Tab/Backspace and Inspect. Toggle desktop/mobile
   viewport presets (1280×800 / 390×844) to test responsive layouts.
   The zoom control enlarges
   the remote viewport without changing the underlying Chromium dimensions.
5. The viewport receives JPEG frames via an authenticated Nexus WebSocket
   backed by Chromium `Page.startScreencast`. It falls back to
   authenticated JPEG requests if streaming disconnects.
6. The same project Workspace and browser session can be viewed through the
   existing remote access tunnel. Use **Assumir controle** / **Liberar controle**
   to hand off keyboard/mouse control between authenticated devices.

## Architecture

```
Existing Agent Workspace / terminal window manager
     ├── terminal surfaces (PTY and agent runtime)
     └── browser surfaces
              │
Nexus Core authenticated browser REST/WebSocket API
              │
browsercontrol.Manager (max 4 isolated sessions)
              │
Chrome / Chromium / Edge (headless, temporary user profile)
              └── local-only Chrome DevTools Protocol
```

The control lease is tied to the authenticated Nexus session, lasts 25
seconds, and can only be renewed by its holder while active. A different
device can view the stream but cannot send input until the lease is
released or expires. The frontend releases control when its surface
becomes invisible. A lease does **not** confer access to a project:
authentication and any broader Nexus project authorization still apply.

## Core API

All REST routes run behind existing Nexus authentication / CSRF checks.

| Endpoint | Purpose |
| --- | --- |
| GET /api/v1/browser/sessions | List managed browser sessions |
| POST /api/v1/browser/sessions | Create session (`project_id`, `url`) |
| GET /api/v1/browser/sessions/:id | Session information |
| POST /api/v1/browser/sessions/:id/control | `acquire`, `renew`, `release` |
| GET /api/v1/browser/sessions/:id/stream | WebSocket, `{"type":"frame","data":"<base64-jpeg>"}` |
| GET /api/v1/browser/sessions/:id/frame | JPEG screenshot (fallback) |
| GET /api/v1/browser/sessions/:id/snapshot | Read-only title/URL/text/element summary |
| POST /api/v1/browser/sessions/:id/action | `navigate`, `back`, `forward`, `reload`, `click`, `scroll`, `type`, `key`, `click_selector`, `fill_selector`, `wait_selector`, `viewport` |
| DELETE /api/v1/browser/sessions/:id | Stop browser and delete temp profile |

REST actions require a **control lease**; screenshots and snapshots
do not. Semantic DOM actions take `selector` (CSS string),
`text` for filling, and optional `timeout_ms` (maximum 5 seconds)
for waiting. The `viewport` action accepts `width` and `height` within
server-enforced limits. Clicks use Chrome's pointer dispatch after scrolling
the selected element into view. The Inspect panel shows candidate
selectors from the current DOM; it never reads password input values.
Selectors and text are treated as data, and the HTTP API exposes
no generic JavaScript execution method. The streamed frames
have no cookies or direct DevTools access.
No arbitrary `Runtime.evaluate` operation is exposed over Nexus HTTP.

## Known constraints / not yet available

- **Remote invitations are still Nexus-wide**. Browser-only share tokens,
  project-level access grants and separate remote identities are **not yet
  implemented**. Do not share the general Nexus tunnel with untrusted users.
- A Nexus session is one control identity. Browser-only ACL and hardware
  device identity are future work.
- The semantic API makes selectors available for future agents, but agent
  tool discovery, MissionRunner `AutonomyContract` authorization and
  auditable action receipts are **not yet wired**. Automated write actions
  must not bypass the existing orchestration/authorization contract.
  Read-only snapshots are untrusted page content, not instructions.
- Live JPEG delivery is capped to 10fps to bound remote resource usage.
- Screenshot media is JPEG and not DRM/AV streaming. Higher-fidelity
  streaming (WebRTC, audio, camera, download handling) is not included.
- Temporary profiles are private and removed at close. Logins do **not**
  survive Core restarts. Never store session cookies or passwords in
  Mission evidence. URL allow/block rules are not a network sandbox:
  a page may trigger arbitrary HTTP requests. Use Chrome's OS sandbox.
- Real Windows WebView2, macOS and Linux desktop UX still require
  owner/device homologation.
- The docs screenshot manifest may be stale after unrelated HUD work;
  refresh the visual evidence rather than bypassing the capture gate.

## Validation

`go test ./internal/browsercontrol ./internal/control/web`

`NEXUS_BROWSER_INTEGRATION=1 go test ./internal/browsercontrol -run TestChromiumSmokeOptional -count=1`

`cd web && bun run typecheck && bunx vitest run src/app/browserSurfaces.test.ts && bun run build`

Manual cases: open with no local development server, navigate to running
server, inspect, screenshot, resize, minimize, switch product tabs, remote
viewer, lease conflict, release and takeover, restore layout, close and
verify process cleanup.
