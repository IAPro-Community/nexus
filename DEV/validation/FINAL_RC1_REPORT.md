# FINAL RC1 REPORT (WIP)

STATUS: CURRENT  
Starting SHA: `3985908244cc01c9e08c5ad4da0854622673a3d7`  
Branch: `fix/nexus-rc1-closure`  
Final SHA: see `git rev-parse HEAD`

## RC-SECURITY

| Item | Status |
|------|--------|
| EffectiveExposure + tunnel PUBLIC_REMOTE | PASS (local) |
| Remote bootstrap one-time + TTL + Secure cookies | PASS (local) |
| cloudflared pin + SHA256 | PASS (local) |
| Embedded Ed25519 trust root | PASS (local) |
| Private signing key in CI (`NEXUS_UPDATE_PRIVATE_KEY`) | BLOCKED_EXTERNAL |

## RC-STABILITY

| Item | Status |
|------|--------|
| TempDir fixtures / non-Windows `/tmp` gate | PASS (local) |
| SessionHost QA readiness (no fixed sleep) | PASS (local) |
| SQLite Close() error checks | PASS (local) |
| VERSION ↔ package.json ↔ wails in `make quality` | PASS |
| E2E storage reset + waitForFunction | PASS (code) |
| macOS plist version/XML CI assert | PASS (workflow) |
| Native Windows/macOS CI green on this SHA | PENDING |

## RC-RELEASE

| Item | Status |
|------|--------|
| Promote native → then signed manifest | PASS (workflow) |
| Manifest keys `linux_amd64` + desktop/NSIS + `target` | PASS (local tests) |
| Apply verifies archive SHA before extract | PASS (local) |
| Package-manager markers (NSIS/deb/rpm) + path heuristics | PASS (code) |
| Windows in-use replace via MoveFileEx delay | PASS (code; native PENDING) |
| Authenticode / notarization | BLOCKED_EXTERNAL |
| Same-SHA multiplatform CI green | PENDING |

## Verdict (interim)

**NO_GO** — local RC-SECURITY/STABILITY/RELEASE slices advanced; still blocked on CI green across platforms, signing secret in Actions, and Authenticode/notarization.
