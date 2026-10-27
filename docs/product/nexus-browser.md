# Nexus Browser — Workspace browser (v1 experimental)

## Scope

Nexus Browser is a managed Chromium session inside the existing Terminals workspace.
It is presented through the same DESKTOP / MOSAIC / SPLIT / TABS window manager as
PTYs, but is not itself a PTY. It uses the existing Core API and remote tunnel.

## Prerequisites

Install Chrome, Chromium, or Microsoft Edge on the Nexus host. If it is not on
PATH, set NEXUS_CHROMIUM_PATH to the trusted executable path. The browser runs
in headless mode with a temporary, private profile and dynamically chosen
loopback DevTools port. No new Node or Electron runtime is required.

## Usage

1. Open a Nexus project and select Terminals.
2. Select **Novo navegador**. A Browser window appears in the same workspace.
3. Enter an HTTP(S) URL (for example http://localhost:3000).
4. Click the rendered viewport to interact with the underlying page.
5. Use the text input to type into a focused page element; Enter and Tab are
   available as explicit actions. Mouse wheel / mobile swipe scroll the page.
6. Use **Inspect** to read current visible page text. This is exposed for
   further agent integrations, but does not authorize independent agents yet.
7. Arrange, minimize or maximize the Browser with the existing workspace
   layout controls. Sessions remain alive while the panel is hidden.

## API

All routes are behind the existing Nexus session / CSRF middleware and support
same-origin access over the existing HTTPS remote tunnel.

- GET /api/v1/browser/sessions?project_id=...
- POST /api/v1/browser/sessions — { "project_id": "...", "url": "https://..." }
- GET /api/v1/browser/sessions/:id — metadata
- GET /api/v1/browser/sessions/:id/frame — authenticated JPEG frame
- GET /api/v1/browser/sessions/:id/snapshot — current URL, title and text/elements
- POST /api/v1/browser/sessions/:id/action — type: navigate/click/type/scroll/key
- DELETE /api/v1/browser/sessions/:id — terminates Chrome session and deletes profile

Only HTTP(S) navigation is permitted. Arbitrary JavaScript execution,
DevTools WebSocket URLs and Chromium debugging endpoints are not exposed via
the application API. Chromium crashes return errors rather than fabricated success.

## Security and operational bounds

- The Core is the only CDP controller; DevTools binds to loopback.
- Maximum four Chromium instances; each has an isolated temporary profile.
- Closing a session deletes its temporary profile; exiting the Core terminates
  managed Chromium processes.
- Sessions do NOT persist across a Core restart; reopening a saved workspace
  will create a new isolated session.
- The existing tunnel invitation currently grants broad workspace access.
  Browser-only read/control grants and human-vs-agent control leases are
  **not yet implemented**. Do not advertise scoped remote access.
- HTML DOM text may contain personal or confidential information. Avoid
  logging/storing snapshots; never submit sensitive results to models without
  user consent and applicable policy checks.
- Agent-owned autonomous browser use, mission evidence and Maestro integration
  are **future work**: HTTP endpoints are a foundation, not authorization
  for the agent to act on the user's behalf.
- UI uses repeated JPEG captures; it is not WebRTC or low-latency video
  streaming. Sites needing protected content, media DRM or advanced browser
  features are not guaranteed to work.

## Minimum verification

Run Go tests for internal/browsercontrol and internal/control/web,
then web typecheck, unit tests, stylesheet checks and build. Exercise
real Chrome on Linux, WebView2 on Windows and WebKitGTK/WKWebView
on Linux/macOS respectively. Check navigation, click, text, scroll,
project isolation, CSRF, remote cookie auth, session deletion,
process cleanup and browser absence diagnostics.
