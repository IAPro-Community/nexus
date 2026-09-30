# Nexus 1.0 Beta.7 — closure sweep / external certification handoff

Date: 2026-09-30  
Canonical branch: `release/v1.0.0`  
Version audited: `1.0.0-beta.7`  
Audited source SHA: `32b0cc3611850b744ba4c3191d35835d0744ce82`  
State: **READY_FOR_EXTERNAL_CERTIFICATION / NOT YET CERTIFIED**

## Purpose

This record closes everything that can be resolved from repository/source review
without claiming execution that did not happen. It intentionally separates:

- source/admin review evidence;
- execution that still must run on a real machine;
- GitHub Actions gates blocked by account credits.

The commit containing this report is documentation-only and therefore advances
the branch SHA after the audited source SHA above. Exact-SHA/native
certification must always target the then-current immutable
`release/v1.0.0` HEAD.

## Final repository sweep

### PASS — source/admin review

- `release/v1.0.0` remained the only canonical Nexus 1.0 release line during
  the sweep.
- `VERSION` remained `1.0.0-beta.7`; no beta increment was created merely
  because CI is unavailable.
- Issue #27 remains the single open release/homologation tracker.
- Historical `codex/windows-terminal-functional@d73d75a...` remains reference
  only; its valid Windows deltas are present or superseded and its ConPTY-default
  behavior must not be restored.
- Stale draft PRs #31, #33, #34 and #35 were closed without merge:
  - #31 conflicts with the current opaque-quota routing policy;
  - #33's OpenAI Admin/API usage collector is explicitly future/deferred work;
  - #34 contains stale convergence plus manifest re-anchoring without recapture;
  - #35 can promote DIRECT/provider preference into Mission and enables AGY
    AUTO delegation by default, conflicting with the beta.7 contract.
- Current interactive provider contract remains fail-closed:
  `delegation_mode=OFF` by default; Lead identity/continuity does not authorize
  Mission; provider-native TTY owns ordinary interaction.
- Current Codex hardening remains in source: keyboard enhancement disabled,
  SGR/DEC mouse noise filtering, CSI/SS3 coalescing, and conditional
  `--no-daemon` only for unsafe Unix socket path length on interactive TUI
  flows.
- Current Windows source includes supervised process-tree Job Object hardening
  and PID-reuse safeguards introduced after the previous native campaign.
- Voice-to-terminal source tests at the audited HEAD assert transcript insertion
  without implicit Enter/CR/LF.

These are source-review findings. They are not substitutes for runtime PASS.

## Known evidence blocker

`docs/assets/screenshots/manifest.json` is stale for the current visual source.

Top-level visual anchor:

`102213a5436f0f4f53aed5763949ce25460ec2b8`

The Composer UI changed after that anchor (including Interview depth), so the
manifest must not be re-anchored manually. Real synthetic recapture is required.

Required sequence on the final candidate:

```bash
cd web
bun install --frozen-lockfile
bun run docs:capture
cd ..
bun scripts/docs-verify.mjs
```

Only the generator may update the capture evidence/source anchors.

## Exact execution gates

### Linux amd64 — NOT RUN in this closure session

Requirements encoded by the repository homologator include Go 1.25.14,
Bun 1.3.9, clean exact SHA, documentation verification, frontend quality,
Go formatting/vet/lint/security/tests and release/runtime gates.

Run from a clean clone/worktree of the final SHA:

```bash
NEXUS_EXPECTED_SHA=<FINAL_SHA> ./scripts/homologate-beta.sh
```

Authenticated provider smoke remains separate:

```bash
./scripts/smoke-providers.sh
nexus codex --yolo
```

The interactive Codex smoke must explicitly verify arrows, mouse, Ctrl+C,
selectors, absence of literal CSI fragments, ordinary prompt behavior without
Mission promotion, and normal resume/fork behavior.

### Windows amd64 — NOT RUN in this closure session

Run on native Windows AMD64 with Go 1.25.14 and Bun 1.3.9:

```powershell
$env:NEXUS_EXPECTED_SHA="<FINAL_SHA>"
./scripts/homologate-beta.ps1
```

Also perform authenticated `nexus codex --yolo`, AGY and OpenCode smokes.
ConPTY remains opt-in; do not change the source contract merely to make a
particular terminal appear green.

### macOS arm64 — NOT RUN

If macOS remains in the announced platform matrix, native same-SHA Desktop/HUD,
provider and packaging evidence is still mandatory before support is claimed.

## GitHub Actions

Current exact-SHA run observed during the sweep:

- run: `36747107698`
- audited head: `32b0cc3611850b744ba4c3191d35835d0744ce82`
- state: `queued`
- first self-hosted sanitize job: `queued`
- Windows native certification: `skipped`
- macOS native certification: `skipped`

The user confirmed Actions credits are unavailable. Classification:

**BLOCKED_EXTERNAL — GitHub Actions credits / runner execution unavailable.**

Do not convert this state to PASS. When credits/runners return, execute the
normal exact-SHA workflow against the final immutable release HEAD.

## Final checklist

- [x] Canonical release/version revalidated.
- [x] Open release PR residue reviewed and closed unmerged.
- [x] Historical Windows branch reviewed; no port required.
- [x] Codex/Lead/Mission source contracts rechecked.
- [x] Usage/routing stale branches classified against current authority.
- [x] CI credit limitation classified as BLOCKED_EXTERNAL.
- [ ] Regenerate all visual captures on the final SHA.
- [ ] `bun scripts/docs-verify.mjs` PASS after recapture.
- [ ] Linux exact-SHA homologation PASS.
- [ ] Windows native exact-SHA homologation PASS.
- [ ] Authenticated Codex/AGY/OpenCode provider smokes PASS.
- [ ] macOS native exact-SHA certification PASS if macOS is advertised.
- [ ] GitHub Actions exact-SHA PASS after credits/runners are restored.
- [ ] Stable certification/signing/soak PASS before `v1.0.0` stable publication.

## Promotion rule

Beta.7 is now **ready to enter external/native certification**, not certified.

Do not create beta.8 solely because CI is blocked. Create a new beta identifier
only if subsequent functional/source changes intentionally create a new
candidate. Any commit after the final certification target invalidates
same-SHA evidence for that target.
