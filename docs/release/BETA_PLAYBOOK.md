# Nexus Beta Playbook — v1.0.0-beta.1

How to cut, sign, publish, roll out, and roll back a closed beta without
ever exposing source.

## 1. Prerequisites (external gates)

- `NEXUS_UPDATE_PRIVATE_KEY` (Ed25519, release environment only — never in
  repo, binary, or logs) and `NEXUS_UPDATE_PUBLIC_KEY` (injected at build
  via `.goreleaser.yaml` ldflags into the single trust-root source).
- Distribution storage/CDN + DNS (`updates.<dominio>/beta/manifest.json`
  target) OR a separate binaries-only GitHub repo with Releases access.
- Windows Authenticode certificate (`EXTERNAL_SIGNING_GATE` until wired).
- Apple Developer ID + notarization credentials
  (`EXTERNAL_NOTARIZATION_GATE` until wired).
- GitHub admin token for branch protection
  (`EXTERNAL_GITHUB_ADMIN_GATE` until applied).

## 2. Cut the version

1. Freeze SHA: record `git rev-parse HEAD`, ensure `git status --short`
   shows only intended files.
2. Set `VERSION` + `web/package.json` version identically.
3. Regenerate `web/bun.lock` with `bun 1.3.9` (`bun install`, never hand-edit)
   and confirm `bun install --frozen-lockfile` passes.
4. Regenerate `docs/legal/GO_MODULES.txt` (`go list -m all > ...`).

## 3. Certify the SHA (all green on the SAME SHA)

- `go build ./...`, `go test ./...`, `go test -race` on critical paths,
  `go vet`, `golangci-lint run`, `govulncheck ./...` (zero known).
- Web: `format:check`, `lint`, `lint:styles`, `check:styles`,
  `typecheck`, `test`, `build`, `verify`, Playwright + axe + visual smoke.
- Desktop smoke per OS; installer smoke (NSIS/DEB/RPM); update E2E
  (A→B happy path + hash/signature/downgrade negatives).
- SBOM (CycloneDX) + license scan artifacts attached to the release.

## 4. Sign and publish

1. Build via the release workflow (`.github/workflows/release.yml`):
   artifacts + SHA-256 + signed `update-manifest.json` (Ed25519,
   keyring supports current+successor during rotation).
2. Verify signature with a clean checkout using only the public trust root.
3. Publish manifest + artifacts to the channel (`beta`), keeping the
   movable channel pointer (`channel-beta`) separate from immutable
   version tags. Tag `vX.Y.Z-beta.N` must point at the certified SHA —
   never reuse a stale tag.

## 5. Rollout (progressive channels)

`internal` → `alpha` → `beta-closed` → `beta-expanded`. The updater
supports channels; move the pointer only after smoke on the previous ring.
Kill-switches (`tunnel`, `autonomous_mission`, `remote_access`) can disable
a dangerous surface without an emergency release once remote config lands;
until then they are env-driven (`NEXUS_DISABLE_<NAME>=1`).

## 6. Rollback

- Move the channel pointer back to the last good manifest; clients refuse
  automatic downgrade, so publish the previous version as a new
  higher-priority manifest entry OR instruct explicit admin downgrade.
- `Updater.Rollback()` restores the pre-update backup on the client when
  apply fails mid-flight; never leaves a half-installed tree.

## 7. Kill switch and incidents

- Suspected compromise of the signing key: rotate (add successor, publish,
  revoke predecessor per the rotation test), re-sign, move pointer.
- Malicious artifact: revoke manifest, publish clean pointer, notify beta
  users to verify `nexus update --check` trust chain.
- Support: users run the diagnostics bundle flow (explicit preview,
  redacted by construction); never ask for raw tokens or project files.

## 8. Telemetry during beta

Analytics stays opt-in; monitor only aggregate allowlisted events plus
update check/install counters. No prompt/code/file/token content, ever.
