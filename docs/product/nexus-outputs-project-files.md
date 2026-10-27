# Nexus Outputs / Project Files

> Implementation: GitHub #94 — project-scoped generated media and files, shared by providers.

## User experience

Within an open project, choose **Outputs · Arquivos** from the left rail or command palette. The **Galeria** tab scans that project's authorized directory, displaying supported images, PDFs, media, and text files. **Arquivos** browses folders using project-relative paths; hidden directories and sensitive paths are not listed. Previews and downloads work through the same authenticated Nexus Web origin, on desktop or mobile.

An image generated on the host by OpenCode, AGY, Codex, or another provider is discoverable as soon as it is committed to the project's filesystem. Nothing is automatically uploaded to a cloud service. The user may download the original and verify its SHA-256 when computed.

## Architecture

- `internal/nexus/artifacts`: constrained scanning, path validation, MIME sniffing, secure open, bounded thumbnail generation, hash calculation.
- `GET /api/v1/projects/{id}/outputs?kind=all&offset=0&limit=48`: discover recent resources, with deterministic pagination, hard scan depth/count limits and truthful truncation indicators.
- `GET /api/v1/projects/{id}/files?path=relative/path`: list directory entries (maximum 500 entries; never enumerate host bookmarks).
- `GET /api/v1/projects/{id}/files?path=relative/path&view=1`: authenticated inline viewing of supported content; HTML is always served as inert text; SVG is never inlined.
- `GET /api/v1/projects/{id}/files?path=relative/path&download=1`: stream exact original bytes, no screenshots or image re-encoding.
- `GET /api/v1/projects/{id}/files?path=relative/path&thumbnail=1`: 512px JPEG thumbnail, bounded image dimensions, decode semaphore, Core-owned cache; original preserved.
- `web/src/features/outputs/OutputsSurface.tsx`: responsive gallery/files window integrated into the existing Workspace surface lifecycle and project navigation.

All Web endpoints use the **existing Nexus project session and authentication boundary** including remote tunnels, instead of broadening the loopback-only `/api/v1/fs` endpoints. Paths are relative to the project, checked for blocked segments, and opened under Go's `os.Root` confinement (Go 1.25), including symlinks. Hidden/credential-like filenames and large unsafe MIME payloads are excluded. `nosniff`, no-store and sandbox CSP guard file responses.

## Provider independence

The filesystem scanner is provider agnostic and does not require agents to install plugins or register a new output. A file is attributed to `origin=filesystem` until there is verifiable runtime-specific authorship evidence: **never fabricate `agent_id` or `runtime_id`**. Nexus Browser screenshots are not misrepresented as original exports; Browser browsing remains a separate capability.

## Current limitations and certification gates

This is the functional base for issue #94, **not the closing criterion of the full epic**. The following features require dedicated implementation or live validation and must not be treated as shipped:

1. Durable provenance by `agent_id`/`runtime_id` linked to mission history, explicit agent publishing, version snapshots, true realtime subscriptions and revisions (currently periodic polling).
2. Browser download/export events attached to Outputs with verified origin, as distinct from screenshots.
3. Fine-grained multi-user project RBAC; current Nexus authenticated project sessions are not a per-member authorization system. Do not deploy to untrusted multi-tenant users yet.
4. High-volume archive streaming, audio/video previews with platform-specific codecs, and real-device mobile validation.
5. Native application Remote View (out of MVP; requires separate RFC/security review).

## Minimal Linux + mobile smoke test

1. Start Nexus Core in a new checkout; open the same project from two **separately authenticated** client browsers.
2. Run OpenCode and AGY simultaneously; create `outputs/a.png`, `outputs/b.png`, `outputs/report.pdf` and observe them in the Outputs gallery.
3. Preview, download original and compare SHA-256 to the host file; change a file and confirm refresh catches it.
4. Attempt to read `.env`, `../outside`, `node_modules`, symlinks outside the workspace and executable/HTML disguised as PNG. All must be rejected.
5. Reconnect the mobile client; ensure the same authorized resources appear and no original is automatically copied outside the project.
6. Run `go test ./internal/nexus/artifacts ./internal/control/web` and `cd web && bun run typecheck && bun run build`.

Release readiness must additionally satisfy all the acceptance criteria in [issue #94](https://github.com/IAPro-Community/nexus/issues/94).
