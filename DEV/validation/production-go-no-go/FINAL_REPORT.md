# Nexus Production Certification

Generated: 2026-09-13T04:40:00Z  
Auditor: independent Release Certification Red Team  
Rule applied: evidence > documentation > intention

## Candidate

| Field | Value |
| --- | --- |
| Branch | `feat/nexus-maximum-delivery` |
| Executable candidate SHA | `e53f8352e4c3647632ebac7877cf0a6a3bd62647` |
| HEAD at audit start | `e53f8352e4c3647632ebac7877cf0a6a3bd62647` |
| HEAD at audit close | `019dc585f0d587a61f7ff5594ef154bb8c4f5b29` |
| Origin feat | `d15aa712dd4da433e5d0bad01c169205197865c6` (candidate **not pushed**) |
| Merge-base `origin/main` | `1899ca6334576d859056d48a394e51d03758f313` |
| vs `origin/main` | **91 ahead / 1 behind** at close (finalization report inverted this) |
| Version | `0.5.0-beta.23` |
| Working tree | Dirty throughout (validation docs, untracked evidence) |

```text
CERTIFICATION_INVALID_FOR_CURRENT_HEAD
```

HEAD moved during the audit (`e53f835` → `37f970b` → `d22c293` → `61a9531` → `49af5d3` → `019dc58`). Those later commits are documentation/evidence, not a frozen production tag. No immutable release object was certified. Overnight evidence was written against a missing finalization report, then finalization appeared **after** overnight aborted.

Required sources:

| Artifact | At close | Verdict inside |
| --- | --- | --- |
| `DEV/validation/production-finalization/FINAL_REPORT.md` | EXISTS | `NOT_READY_FOR_OVERNIGHT_CERTIFICATION` |
| `DEV/validation/overnight-production-certification/FINAL_REPORT.md` | EXISTS (untracked at start) | `ABORTED_PRECONDITION_FAILED` |

## Executive Verdict

```text
PRODUCTION_NO_GO
```

`PRODUCTION_GO` and `PRODUCTION_GO_WITH_ACCEPTED_DEBT` are rejected.

The product has substantial local Linux implementation and many passing unit tests. That is not production readiness. Overnight was not run. Native Windows/macOS were not run for this SHA. Hosted CI does not know this SHA. The last completed CI on the branch failed Linux, Windows, macOS E2E and Browser. Authenticated Mission completion was not proven on the public surface. Several false-success and hidden-coupling defects are in the code path a production user would hit.

One demonstrable blocker defeats several positive local gates. There are several.

## Product Promise

What was **actually proven** in this audit (Linux host, executable SHA `e53f835`, dirty tree):

- `make quality` exit 0 (this session).
- `go vet ./...` exit 0.
- `make security` / govulncheck: no Go vulnerabilities.
- `make docs-verify` exit 0 (134 Markdown files).
- `make build` exit 0; binary reported commit `37f970b` because HEAD had already moved while linking.
- `go test -race ./...` exit 0 — 1079 tests / 65 packages.
- `make overnight-smoke` exit 0 in **0.243s** — this is a bounded unit harness, not an 8h soak.
- Isolated CLI: `nexus run --help` works; no project → explicit error; project add works; `nexus run "<goal>"` exits 1 in 0.2s.
- Isolated web: HTML 200 + CSP; API 401 without session.
- Host `providers status`: Codex `REGISTERED_AUTHENTICATED`; several others unregistered / pending auth.

What was **not proven**:

- 8h overnight operation.
- Restart/recovery of a live Mission.
- Maestro OFF as a normal install (this host still discovers Maestro via hardcoded home paths).
- Authenticated Mission → verified DoD on the public surface.
- Native Windows ConPTY/Named Pipe/NSIS/Wails runtime.
- Native macOS.
- Hosted same-SHA CI.
- Production signed updater.
- Interactive Web UI (certification browser loaded HTML/JS; `#root` stayed empty — harness limitation not promoted to a product blank-screen P0).

## Platform Matrix

| Surface | Status | Evidence |
| --- | --- | --- |
| Windows | **FAIL / NOT_VERIFIED** | No same-SHA native run. `gh run list --commit e53f835…` = []. Latest completed ancestor CI `b142171` **failed** Windows E2E (ConPTY/Named Pipe/PowerShell), including host/launcher 600s timeouts. Linux cross-compile is not Windows certification. |
| Linux | **PARTIAL** | Local quality/vet/race/security/build PASS on this machine. Hosted Linux E2E on `b142171` **FAILED** (`TestDesktopBootstrapRequiresDesktopOrigin`). Same-SHA hosted Linux: **NOT_VERIFIED**. |
| macOS | **NOT_VERIFIED** | No native runner. Ancestor CI macOS E2E **FAILED**. Darwin cloudflared pin still absent. |
| Web | **PARTIAL** | Local finalization claims hermetic E2E/a11y/visual PASS. Hosted Browser job on `b142171` **FAILED** (`page.waitForSelector` timeout). This audit: HTML+auth gate observed; interactive UI not confirmed. |
| Desktop | **PARTIAL** | Linux Wails claimed PASS by finalization locally. Windows/macOS desktop native smoke **NOT_VERIFIED**. NSIS job skipped on last CI. |

## Functional Matrix

| Capability | Status | Evidence |
| --- | --- | --- |
| Direct | **PARTIAL** | Host has authenticated Codex. Isolated `nexus run` never reached a provider session. UX reviewer: first-run provider setup is a dead end. |
| Autopilot | **FAIL** | Public `nexus run "<goal>"` uses `StartMissionRun(..., autonomous=false)` and **returns on first `stepErr`**. Smoke: exit 1 in 0.2s. Overnight autopilot campaigns **NOT_STARTED**. |
| Mission | **FAIL** | No authenticated completed Mission/DoD for this SHA. Public Mission PATCH can persist `COMPLETED` without runner/evidence (`handlers_missions.go`). |
| Overnight | **FAIL** | `ABORTED_PRECONDITION_FAILED`. Duration **0h**. Campaigns A–F not started. `make overnight-smoke` ≠ 8h. |
| Recovery | **FAIL** | Unit persistence exists. Live restart/process-kill/orphan cleanup **NOT_VERIFIED**. Launcher can orphan detached hosts. Watchdog is not sampled during a blocked provider call. |
| Multi-agent | **FAIL** | Worktrees are isolated; global DoD runs on `run.Workspace` = canonical checkout. Integration of agent trees is not proven. Overnight campaign E not started. |
| Provider routing | **FAIL** | `UNKNOWN` remains allocatable (`Available=true`, LRU). Isolated run reported “quota/rate limit failure” with **unregistered** profiles. |
| Quota | **FAIL** | Missing percents can become `0`. Claude/Gemini/OpenCode `Usage:true` from local `usage.json`. Live Mission quota for this SHA **NOT_VERIFIED**. |
| Failover | **FAIL** | Mission infers quota via substring; publishes `QuotaFailoverCompleted` before destination success. No live failover campaign. |
| Maestro OFF | **FAIL** | Default project mode is `ASSIST`. Composer requires `READY`; `PrepareContext` **fails** if Maestro is down and mode ≠ OFF. PATH-only hide still found Maestro (`available: true`). Overnight A not started. |
| Maestro ON | **NOT_VERIFIED** | Host Maestro 0.3.5 answered. Enrichment in a real Mission and crash-during-run ownership **not executed**. Overnight B not started. |
| Attention | **PARTIAL** | Attention Center exists. Flow Run page does not present intervention options (UX-03). No live `NEEDS_YOU` product case in this audit. |
| Resume | **PARTIAL** | Durable runner tests exist. Live resume after Nexus restart **NOT_VERIFIED**. |
| Updater | **FAIL** | Fail-closed without production trust root. No signed `v0.5.0-beta.23` release. Not production auto-update. |

Allowed statuses used above: CERTIFIED was never earned.

## Security

Independent [Security Review](3f67e3e3-b4fe-47ed-99ea-5ce3ef3ca97c) dimension: **FAIL** for production, **PARTIAL** for loopback. The first security-reviewer launch failed the required prompt contract; the retry is persisted in `reviewers/security.md`.

Local loopback hardening is real (401 without session, CSP, Origin not treated as auth in current tests). That does not certify production.

Blockers/high:

- Production Ed25519 trust root not configured → updater cannot verify official manifests.
- Same-SHA hosted security/E2E matrix missing; last pushed CI failed the desktop bootstrap test across OSes.
- Desktop/NSIS are **outside** the Ed25519 update manifest; desktop install trusts unsigned SHA256 sidecars (`SEC-03`).
- Detached SessionHost may survive handshake failure.
- WS query token residual on loopback.
- macOS desktop CI can embed an empty trust root via `${NEXUS_UPDATE_PUBLIC_KEY:-}`.
- Page title still “Powered by Orquestrador Maestro” (branding coupling, not an exploit).

`make security` PASS is govulncheck only. It is not a substitute for native installer/auth/E2E.

## Release

| Gate | Status |
| --- | --- |
| Frozen immutable SHA | **FAIL** — HEAD moved; dirty tree |
| same-SHA CI | **FAIL** — commit not on GitHub; `gh` empty |
| Last hosted CI on branch | **FAIL** on `b142171` (Linux/Windows/macOS E2E + Browser). Snapshot/NSIS **skipped** |
| Artifacts / checksums / signatures | **NOT_VERIFIED** — no GitHub release `v0.5.0-beta.23` |
| Installers | Windows NSIS/macOS not executed for this SHA |
| Updater | **FAIL** for production (fail-closed empty keyring) |
| Reproducibility | Local Linux build works; hosted matrix does not exist for the SHA |

Contradiction: `DEV/validation/production-finalization/FINAL_REPORT.md` wrote “1 commit ahead / 88 commits behind `origin/main`”. Fresh `git rev-list --left-right --count origin/main...HEAD` was `1 88` then `1 91` — **88/91 ahead, 1 behind**. The release document inverted ahead/behind.

Contradiction: finalization claims local browser PASS; hosted Browser job on the nearest completed SHA **FAILED**.

Contradiction: docs say Maestro optional; schema/default `maestro_mode DEFAULT 'ASSIST'` and Composer fail-closed without Maestro.

## Overnight evidence

Read raw abort report, not the summary.

- Verdict: `ABORTED_PRECONDITION_FAILED`
- Soak hours: **0**
- Missions executed: **none**
- Maestro OFF/ON, recovery, failover, multi-agent: **NOT_STARTED**
- Finalization later published `NOT_READY_FOR_OVERNIGHT_CERTIFICATION` and listed overnight as **out of scope**

This is absence of overnight certification, not a failed 8h run. It still **blocks** `PRODUCTION_GO`.

`make overnight-smoke` (0.243s, `TestOvernightAcceptanceSandbox` family) is a sandbox. Treating it as overnight would be false success. Rejected.

## Human interventions

None requested of a human during this red-team session for operational product decisions.

| Event | Why | Could Nexus reasonably have solved it? |
| --- | --- | --- |
| Overnight campaign stopped | Precondition GO artifact missing at start; later explicitly NOT_READY | Yes as a gate. Completing overnight is a prior campaign, not optional polish. |
| No native Windows/macOS | This host is Linux | No — requires native runners. |
| Updater public key | Human-controlled protected config | No — red team must not invent a production key. |
| Isolated `nexus run` failed | No eligible registered profiles; error blamed quota/rate limit | Nexus should fail **explicitly** for unregistered/unauthenticated, not as quota failover language. |

Authenticated Codex exists on the **host** profile store. This audit did not spend live provider tokens to force a Mission after the isolated path failed in 0.2s. That is classified as **NOT_VERIFIED** Mission evidence, not as a pass.

## Remaining findings

### P0 / BLOCKER

| ID | Finding |
| --- | --- |
| OVN-001 | Overnight not certified (0h, campaigns not started). |
| CI-001 | No same-SHA hosted CI; SHA not on origin. |
| CI-002 | Last completed branch CI failed Linux/Windows/macOS E2E and Browser. |
| WIN-001 | Windows not certified (no native same-SHA evidence). |
| MAC-001 | macOS not certified. |
| UPD-001 | Production updater trust root missing; auto-update not releasable. |
| MSN-001 | No authenticated Mission `COMPLETED_VERIFIED` on the public surface for this SHA. |
| MSN-002 | Public Mission status can be PATCHed to COMPLETED without runner/DoD/evidence. |
| MSN-003 | Global DoD executes on canonical checkout while agents write worktrees — false `COMPLETED_VERIFIED` risk. |
| AP-001 | Public `nexus run "<goal>"` is non-autonomous and aborts on first step error. |
| MAE-001 | Default Maestro mode ASSIST + Composer READY gate = Nexus not proven usable Maestro-OFF. |
| QUO-001 | `UNKNOWN` quota remains selectable/successful for routing. |
| QUO-002 | Isolated run labeled missing profiles as “quota/rate limit failure”. |
| REC-001 | Detached runtime can remain alive after launcher handshake failure; Stop timeout does not force-kill. |
| SHA-001 | Candidate SHA not frozen; certification invalid for current HEAD. |

### P1 / HIGH

| ID | Finding |
| --- | --- |
| AUTH-WS | Loopback WS query token residual. |
| SEC-03 | Desktop/NSIS not covered by Ed25519 update manifest. |
| SEC-09 | macOS desktop CI defaults missing updater public key to empty. |
| WD-001 | Progress watchdog not sampled during a blocked provider call. |
| UX-01/02/03 | Provider setup dead end; onboarding not first-run; Flow Run lacks intervention options. |
| FAILOVER-001 | Mission failover uses substring matching; completion event fires before destination success. |
| REL-DOC | Finalization inverted ahead/behind vs `origin/main`. |
| BRAND-MAE | Web title “Powered by Orquestrador Maestro”. |

### P2 / MEDIUM

i18n/inline-style debt in large Flow surfaces; npm moderate Vitest advisories; incomplete `NEXUS_DATA_DIR` isolation (doctor still used host `~/.config/ai-cli`); Windows ARM64 cross-only; no Authenticode/notarization; Docker host-parity opt-in mounts `$HOME`; Docker image pipes `opencode.ai/install` unpinned.

### P3 / LOW

Large files / god objects; visual pixel-diff not implemented; Darwin cloudflared pin missing until a digest exists.

## Accepted snowball

These may remain backlog **after** blockers die. They are **not** reasons to GO today:

- Large `runner.go` / PlanBuilder files.
- Broader i18n cleanup beyond the primary blocked Flow recovery path.
- Visual pixel-diff comparator.
- Additional providers.
- Future UX density reductions.
- Moderate Vitest dependency upgrade (compatibility-reviewed).
- Historical Markdown `FINAL_*` clutter (if a current SHA ledger exists).

## Rejected snowball

Items that look like “later” but **block production**:

- “Overnight was out of scope for finalization” — still required for GO.
- “Windows jobs exist in YAML” — not native evidence.
- “Updater is fail-closed so it is secure enough to ship auto-update” — fail-closed means **updates do not work**, not that the release path is certified.
- “`make overnight-smoke` passed” — not 8h.
- “Maestro is optional in docs” — default ASSIST + Composer coupling.
- “UNKNOWN is just a label” — it currently authorizes allocation.
- “Mission COMPLETED in the API” — can be a CRUD write, not verification.
- “Desktop Wails compiled” — not a running Windows/macOS app.
- “CLI manifest is Ed25519 so the whole release is signed” — desktop/NSIS still sit on unsigned checksum sidecars.

## Independent reviewers

Reject-oriented lanes (one blocker wins; no vote):

| Lane | Agent / artifact | Lane verdict |
| --- | --- | --- |
| Security | [Security Review](3f67e3e3-b4fe-47ed-99ea-5ce3ef3ca97c) `reviewers/security.md` | FAIL for production |
| Windows | [Windows](3cd12d82-f37d-41c4-9dc7-7470499cf4db) `reviewers/windows.md` | NOT_VERIFIED / FAIL |
| Runtime/recovery | [Runtime](2d336b7e-d9da-4385-a404-381afafed600) `reviewers/runtime-recovery.md` | FAIL |
| Autopilot/Missions | [Autopilot](d1db47c7-7b34-4e61-b261-f8acbeeb604a) `reviewers/autopilot-missions.md` | FAIL |
| Providers/quota | [Providers](61b0431e-3420-4da8-b876-a905a38f5149) `reviewers/providers-quota.md` | FAIL |
| Frontend/UX | [Frontend](c01d94de-9235-4694-a313-f3a334987ba9) `reviewers/frontend-ux.md` | PARTIAL |
| Maestro | [Maestro](f63a7dcb-8e7c-490a-835b-2d03ea07b8e7) `reviewers/maestro.md` | OFF FAIL / ON NOT_CERTIFIED |
| Release | [Release](992c59de-d563-40e0-9fa4-3502af7bdbb9) `reviewers/release.md` | NO_GO |

Lead synthesis does not average these. OVN-001, CI-001, WIN-001, UPD-001, MSN-001/002/003, MAE-001 and QUO-001 each independently forbid GO.

## Re-executed gates (this session)

| Command | Result | Notes |
| --- | --- | --- |
| `make quality` | PASS (exit 0) | Dirty tree; `TestCoreLifecycle` attempted `sudo` and failed auth, suite still green |
| `make docs-verify` | PASS | |
| `go vet ./...` | PASS | |
| `make security` | PASS | govulncheck only |
| `make build` | PASS | Linked as `37f970b`, not `e53f835` |
| `go test -race ./...` | PASS | 1079 / 65 packages |
| `make overnight-smoke` | PASS | 0.243s — **not overnight** |
| Hosted `ci.yml` for `e53f835` | **NOT RUN** | |
| Native Windows/macOS | **NOT RUN** | Linux host |

Logs: `DEV/validation/production-go-no-go/gates/`.

## Real product smoke

Isolated `NEXUS_DATA_DIR`:

1. `nexus version --json` → `0.5.0-beta.23` commit `37f970b`.
2. `nexus run --help` → public goal form exists.
3. `nexus run` without project → `nenhum projeto ativo; registre um projeto primeiro`.
4. `nexus projects add` → `prj_06G9J3F643Y6J2W28GMTM76EXW`.
5. `nexus run "corrija um typo no README"` → **exit 1 in 0.2s**: `no eligible alternative provider profiles available … after quota/rate limit failure`.
6. `nexus control web --port 13137` → listen `http://127.0.0.1:13137`; `/` 200; `/api/v1/projects` 401.
7. Certification browser: document complete, `#root` length 0. Interactive flow **NOT_VERIFIED**.

Host read-only: doctor checks PASS/SKIPPED desktop; Codex authenticated; other CLIs installed but not registered.

No merge, deploy, force-push, or invented credentials.

## Final recommendation

Do **not** declare PRODUCTION READY. Do **not** merge to `main`. Do **not** publish `v0.5.0-beta.23` as a production release.

### Smallest sequence to become eligible for a later GO attempt

Freeze **one** SHA. Recertify that SHA only. Do not reuse this folder as PASS.

1. **Stop the moving HEAD.** One executable commit; no silent extra fixes during certification. Push it. Run hosted `ci.yml` to completion on **that exact SHA**. Linux, Windows, macOS E2E, Browser, Desktop, NSIS must be green or explicitly waived with a written production risk acceptance — waiving Windows/macOS is still **NO_GO** under this campaign’s own criteria.
2. **Provision the production updater keypair** in the protected GitHub environment. Embed the public trust root. Produce a signed manifest that also covers **desktop and NSIS**, not only `nexus_*` CLI archives. Prove fail-closed still rejects unsigned/wrong-key artifacts.
3. **Make Maestro actually optional in the default product path.** Default new projects to `OFF`, or make Composer/Mission work when Maestro is absent without a hidden ASSIST failure. Prove Maestro OFF with Maestro binaries unreachable (not unit tests). Prove Maestro ON enriches without owning runtime. Prove a mid-run Maestro crash does not kill Nexus runtimes.
4. **Close false-success holes before claiming autonomy:** Mission PATCH cannot mark COMPLETED without runner+evidence; global DoD must verify the work that agents produced; `UNKNOWN` must not mean allocatable success; public `nexus run` must use the autonomous worker/watchdog path or stop claiming autopilot; launcher must kill orphans; watchdog must fire during blocked provider calls.
5. **Run a real public-surface Mission** with an authenticated provider: intent → plan → execution → verification → DoD, durable evidence stream, reconstructable IDs. Then recovery (controlled Nexus restart) and a true no-progress/NEEDS_YOU case.
6. **Run the 8h soak** on that same SHA. Multiple real Missions/idle/resume — not `sleep 8h`, not `make overnight-smoke`.
7. **Independent red team again** on the new SHA. If SHA changes, invalidate previous evidence.

Until that sequence exists, the honest user-visible outcomes of `nexus run --autopilot "<objetivo real>"` are **not certified**. The only overnight-class evidence on file is that the overnight campaign **did not run**.
