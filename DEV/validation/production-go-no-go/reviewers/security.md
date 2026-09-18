# Security Red-Team — Production Go/No-Go

**Audited executable candidate:** `e53f8352e4c3647632ebac7877cf0a6a3bd62647`  
**Reviewer:** independent security-review subagent (retry after first launch failed prompt contract)  
**Posture:** reject-oriented. No code changed. No secrets.

## Verdict

| Dimension | Verdict |
| --- | --- |
| SECURITY (production) | **FAIL** |
| SECURITY (local loopback) | **PARTIAL** |

The updater *engine* is fail-closed. The *distribution chain* (CLI + desktop + NSIS + same-SHA CI + provisioned trust root) is not closed for this SHA.

## Updater: fail-closed vs insecure

| Path | Behavior | Verified / inferred |
| --- | --- | --- |
| Empty `ProductionTrustRootHex` | No production key in default keyring | Verified `internal/update/keyring.go:24-36` |
| Unsigned manifest | `ErrManifestUnsigned` | Verified `internal/update/service.go:277-294` |
| Tampered / untrusted key | Ed25519 reject | Verified negative tests |
| Release workflow requires hex public key + fixed key ID | Release cannot publish without vars | Verified `.github/workflows/release.yml:143-169` |
| Desktop / NSIS outside signed manifest | Desktop uses unsigned SHA256 sidecar | Verified `scripts/sign-update-manifest/main.go:109-142`, `install.sh:480-493` |
| Launcher handshake fail after spawn | Detached process may survive | Verified `internal/control/launcher/launcher.go:194-224` |

## Findings

### SEC-01 BLOCKER — Production auto-update chain not closed for this SHA

`ProductionTrustRootHex` is source-empty and only injected via ldflags/CI. No published signed release for `v0.5.0-beta.23` / `e53f835`. Fail-closed is correct; it is not a certified production updater.

### SEC-02 BLOCKER — Same-SHA hosted CI absent

`gh run list --commit e53f835…` was empty. Local smoke binary linked as `37f970b`. Release `same-sha-gate` cannot have run.

### SEC-03 HIGH — Desktop and NSIS outside the Ed25519 manifest

`releaseArtifactMetadata` covers `nexus_*` CLI archives. Desktop install verifies `desktop-checksums.txt` (SHA256) without Ed25519. A compromised GitHub Release asset plus matching checksum can substitute `nexus-desktop` without breaking CLI signature checks.

### SEC-04 HIGH — Orphan SessionHost after handshake failure

Post-`SpawnDetachedHost` errors mark `FAILED` without Stop/kill. Aligns with runtime-recovery RR-02.

### SEC-05 HIGH — WebSocket query token on loopback

Accepted on loopback; rejected under tunnel / private `--remote` (tests). Same-user local process threat remains.

### SEC-06 MEDIUM — Auth state on disk

`sessions.json` / `listen.json` mode 0600 on loopback. Expected for a local control plane; not multi-user hardened.

### SEC-07 MEDIUM — Docker host-parity blast radius

Opt-in `I_ACCEPT_HOST_PARITY=1` mounts `$HOME` RW. Compromised container equals host-user credentials.

### SEC-08 MEDIUM — Docker image curls `https://opencode.ai/install \| bash`

Unpinned install script during image build.

### SEC-09 MEDIUM — macOS desktop CI injects empty trust root by default

`.github/workflows/ci.yml` uses `${NEXUS_UPDATE_PUBLIC_KEY:-}`. Missing var embeds empty keyring into the macOS desktop artifact.

### SEC-10 LOW — CSP `style-src 'unsafe-inline'`

Partial XSS mitigation; not a demonstrated sink.

## Positive controls (do not justify PASS)

govulncheck job; Ed25519 negative tests; tunnel auth armed before WaitForTunnel; Origin is not an auth factor; default bind 127.0.0.1; CLI installer rejects unsigned manifest; CSP/nosniff/DENY/no-referrer observed on isolated web smoke.

## Not claimed

No exploit PoC, no credential dump, no production key invention.
