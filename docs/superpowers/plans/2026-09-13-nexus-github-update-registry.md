# Nexus GitHub Update Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Point Nexus update discovery at GitHub channel releases under `IAPRO/nexus`, keep Maestro on npm, and publish movable `channel-*` pointers from release.yml.

**Architecture:** Add registry URL helpers in `internal/update` (repo + channel → manifest URL). Wire `NewService` to use them unless `NEXUS_UPDATE_REGISTRY_URL` is set. Extend release workflow to recreate `channel-stable` / `channel-beta` pointer releases. Align installer default repo and docs.

**Tech Stack:** Go 1.25, GitHub Actions (`gh` CLI), existing Ed25519 update manifest signer.

## Global Constraints

- Canonical default repo: `IAPRO/nexus` (verbatim).
- Channel tags: `channel-stable`, `channel-beta` (verbatim).
- Manifest filenames: `update-manifest.json`, `update-manifest.sig`.
- Do not use `releases/latest` for installers.
- Do not merge Nexus and Maestro mutation endpoints.
- SCSS Modules / i18n rules apply only if UI copy changes; prefer API/docs-only for Maestro UX (already dual-status).

## File map

| File | Responsibility |
| --- | --- |
| `internal/update/registry.go` (new) | Repo/channel → default manifest URL |
| `internal/update/registry_test.go` (new) | URL construction + env precedence |
| `internal/update/service.go` | Call helper from `NewService` |
| `.github/workflows/release.yml` | Publish channel pointer after signed manifest |
| `install.sh` / `install.ps1` | Default `NEXUS_RELEASE_REPO` → `IAPRO/nexus` |
| `docs/reference/configuration.md` | Document defaults |
| `docs/getting-started/installation.md` | Mention canonical repo if needed |

---

### Task 1: Registry URL helpers (TDD)

**Files:**
- Create: `internal/update/registry.go`
- Create: `internal/update/registry_test.go`
- Modify: `internal/update/service.go`

**Interfaces:**
- Produces: `DefaultUpdateRegistryRepo = "IAPRO/nexus"`; `ChannelPointerTag(channel string) string`; `DefaultManifestURL(repo, channel string) string`; `ResolveRegistryURL(explicit, channel string) string` reading `NEXUS_UPDATE_REGISTRY_URL` then `NEXUS_RELEASE_REPO`.

- [ ] **Step 1: Write failing tests** for stable/beta URLs, env full override, repo override.
- [ ] **Step 2: Implement helpers + wire `NewService`.**
- [ ] **Step 3: `go test ./internal/update/ -count=1`**
- [ ] **Step 4: Commit** `feat(go): default Nexus update registry to IAPRO/nexus channel pointers`

### Task 2: Release channel pointer

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: After signed manifest + versioned publish, add step** that selects `channel-stable` vs `channel-beta` from version, recreates pointer release with only manifest+sig, `latest: false`.
- [ ] **Step 2: Commit** `ci(release): publish movable channel-stable/beta update pointers`

### Task 3: Installer + docs alignment

**Files:**
- Modify: `install.sh`, `install.ps1`, `docs/reference/configuration.md` (and installation.md if it hardcodes old repo)

- [ ] **Step 1: Default repo `IAPRO/nexus`; document `NEXUS_UPDATE_REGISTRY_URL` / `NEXUS_RELEASE_REPO`.**
- [ ] **Step 2: Commit** `docs(infra): point install/update defaults at IAPRO/nexus`

### Task 4: Verify Maestro boundary unchanged

**Files:**
- Test: `internal/control/web/handlers_nexus_test.go` (existing system updates / maestro update tests)

- [ ] **Step 1: Run focused web handler tests; no API contract change required if fields already dual.**
- [ ] **Step 2: Confirm help text still separates `update` vs `maestro update`.**
