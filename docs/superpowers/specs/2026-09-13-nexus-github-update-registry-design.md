# Nexus GitHub Update Registry — Design

**Date:** 2026-09-13  
**Status:** Approved  
**Scope:** Nexus update discovery via GitHub channel pointers (`IAPRO/nexus`) + unified Nexus/Maestro update UX with separate mutation channels.

## Problem

The Update Service defaults to `https://updates.iapro.dev/v1/nexus/manifest.json`, a host that does not exist for the community/IAPRO path. Artifacts are already published to GitHub Releases with Ed25519-signed manifests; only **discovery** depended on a missing CDN.

Maestro already updates via npm. Product UX must show both components without merging mutation paths.

## Goals

1. Nexus discovers updates from GitHub Releases under **`IAPRO/nexus`**, using movable channel tags — no `updates.iapro.dev` required.
2. Unified **status** surface (CLI/Web): Nexus + Maestro versions and availability.
3. Separate **actions**: Nexus self-update (GitHub) vs Maestro update (npm).
4. Keep Ed25519 trust root and detached `.sig` verification unchanged.
5. Preserve installer rule: no `releases/latest` for install scripts.

## Non-goals

- Creating/transferring the GitHub org/repo `IAPRO/nexus` (operational).
- Standing up `updates.iapro.dev`.
- Merging `nexus update` and `maestro update` into one mutating command.
- Frente C / soak / overnight certification.

## Design

### Discovery URLs

| Channel | Default manifest URL |
| --- | --- |
| `stable` | `https://github.com/IAPRO/nexus/releases/download/channel-stable/update-manifest.json` |
| `beta` | `https://github.com/IAPRO/nexus/releases/download/channel-beta/update-manifest.json` |

Detached signature: sibling `update-manifest.sig` (existing `detachedSignatureURL` behavior).

Channel selection (unchanged): version string containing `beta` → `beta`; else `stable`.

### Repo resolution order

1. Explicit `NEXUS_UPDATE_REGISTRY_URL` (full manifest URL) — highest priority.
2. Else build URL from repo + channel tag:
   - Repo: `NEXUS_RELEASE_REPO` if set, else **`IAPRO/nexus`**.
   - Tag: `channel-stable` or `channel-beta`.

### Release workflow

On each authorized release:

1. Build/sign `update-manifest.json` + `.sig` with `--base-url` pointing at the **immutable** versioned release assets (`…/releases/download/<tag>/…`).
2. Attach those files to the versioned GitHub Release (existing).
3. Publish/replace the movable pointer release `channel-stable` or `channel-beta` containing **only** the current `update-manifest.json` and `update-manifest.sig` (not full binary set). Do **not** mark channel pointers as GitHub “Latest”.

`GITHUB_REPOSITORY` drives where pointers are published. After the repo lives at `IAPRO/nexus`, publish target and client default align. Until transfer, forks may set `NEXUS_RELEASE_REPO` / `NEXUS_UPDATE_REGISTRY_URL` for local verification.

### Maestro boundary

- Status: `GET /api/v1/system/updates` continues to report Nexus (Update Service) + Maestro (npm view).
- Mutate Maestro only via `POST /api/v1/maestro/update` and `nexus maestro update`.
- Mutate Nexus only via Update Service / `nexus update`.

### Installers

- Keep forbidding `releases/latest` / `@latest` floating install downloads.
- Align default `NEXUS_RELEASE_REPO` fallback to `IAPRO/nexus` so install and update share the same canonical product identity.

## Success criteria

- [ ] Default (no env): Check uses `IAPRO/nexus` `channel-*` URL for the active channel.
- [ ] `NEXUS_UPDATE_REGISTRY_URL` still overrides fully.
- [ ] `NEXUS_RELEASE_REPO=owner/name` retargets channel URL without full override.
- [ ] Release workflow updates the matching `channel-*` pointer after signing.
- [ ] System updates API still exposes Nexus + Maestro fields; mutations remain separate.
- [ ] Unit tests cover URL construction; existing signature fetch path unchanged.
- [ ] Docs (`configuration.md` / getting-started) describe GitHub defaults.

## Risks

| Risk | Mitigation |
| --- | --- |
| `IAPRO/nexus` not yet transferred | Overrides; document; fail closed on HTTP errors (existing). |
| Channel pointer force-update races | Single release job owns the pointer; `--clobber`/recreate release. |
| Accidental “Latest” on channel tag | Explicitly disable latest on pointer releases. |
