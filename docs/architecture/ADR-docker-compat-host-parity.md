# ADR: Docker compatibility layer with host-parity

## Status

Accepted — 2026-09-10

## Context

Users asked for a Docker path on Windows/Linux without replacing native Desktop. Provider CLIs (Claude, Gemini, Codex, OpenCode) store auth and history on the host. Partial mounts of `~/.config` break absolute paths (lesson from vpn-dev-workspace). Nexus already isolates credentials per profile via env dirs under `~/.local/share/ai-manager/profiles/...`.

## Decision

1. Ship Docker as an **additive** compatibility layer: Linux `nexus` + embedded web only. No Wails/desktop.
2. Default compose profile is **host-parity**: bind-mount `HOST_HOME_DIR` and `WORKSPACE_DIR` at the same absolute paths (RW), with matching `HOST_UID`/`HOST_GID`.
3. Require explicit `I_ACCEPT_HOST_PARITY=1`. Document blast radius as equivalent to a user shell.
4. Keep SSH agent opt-in; never mount `docker.sock`; bind web to `127.0.0.1`.
5. Optionally install provider **binaries without credentials** in the image; host PATH takes precedence. Auth remains on mounted profile homes.
6. `nexus doctor` reports desktop/WebView2/ConPTY as SKIPPED/N/A inside containers (`NEXUS_DOCKER=1`).

## Consequences

### Positive

- Same login/history/sessions for providers inside and outside Docker.
- Windows users keep native Desktop via NSIS/`install.ps1`.
- Clear trust boundary documentation.

### Negative

- Home RW is powerful; a compromised agent process can read the whole home.
- Windows-native-only CLI installs are invisible to the Linux container (WSL required).
- Image may lag third-party CLI package names; host-parity is the durable path.

## Documentation (kept separate)

- User guide: [`docs/getting-started/docker-compat.md`](../getting-started/docker-compat.md)
- Native install (not this ADR): [`docs/getting-started/installation.md`](../getting-started/installation.md)
- Packages: [`docs/getting-started/packages.md`](../getting-started/packages.md)

## Alternatives considered

- **Selective mounts only** — rejected as default: breaks CLI absolute paths; allowed later via ADR if needed.
- **Docker as sole Windows distribution** — rejected: cannot deliver Wails Desktop/ConPTY parity.
- **Bake secrets into the image** — rejected: insecure and non-portable across `docker pull`.
