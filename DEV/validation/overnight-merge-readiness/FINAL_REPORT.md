# Overnight Merge Readiness — Final Report

Generated: 2026-09-12  
Maestro SHA final: `37b9ddec49ce72d869bbeb2ad402a1d4dd2bc674`  
Base vs `origin/main`: `1899ca6334576d859056d48a394e51d03758f313`

## VERDICT

**NOT_READY_FOR_MAIN**

Local implementation gates (`make quality`, `make build`) are green after overnight fixes.
Independent red team + AGENTS.md contract review still reject merge due to open HIGH/BLOCKER
items (i18n contract, release evidence NO-GO, updater trust root placeholder, dirty doc tree).

## Git

| Field | Value |
|-------|-------|
| Target | `main` |
| Base SHA | `1899ca6334576d859056d48a394e51d03758f313` |
| Initial HEAD | `3760df2d5b3c6656c9c32d9cfc5d922d352c2644` |
| Final HEAD | `37b9ddec49ce72d869bbeb2ad402a1d4dd2bc674` |
| Commits created | 3 (`cd5cb87`, `b2c4645`, `37b9dde`) |
| Ahead of origin/feat | 3 |
| Ahead/behind main | ~76 ahead / 1 behind |
| Working tree | Dirty: DEV validation doc banners + untracked inventory (not committed) |

## Branch intent

Deliver Nexus maximum product surface vs `main`: WorkPlan/Mission runner, quota routing,
desktop/web control center, provider adapters, evidence streams, Attention Center, and
final-closure hardening. Campaign docs remain **NO-GO** for production certification.

## Correções realizadas (esta sessão)

| Problema | Causa raiz | Correção | Validação |
|----------|------------|----------|-----------|
| Desktop auth via Origin spoof | `AuthenticateRequest` granted `desktopSession` on Origin/Referer alone | Require cookie/Bearer/X-Nexus-Session | `desktop_auth_test.go` |
| WS session token on public tunnel | Query token accepted even with tunnel | Reject query token when `tunnelActive`; omit token on `.trycloudflare.com` URLs | Go + Vitest |
| Claude credentials fail-open | Non-empty file ⇒ authenticated | Require parseable OAuth/email/token fields | compile + suite |
| AGY weak heuristics | google_accounts / jetski / keyring alone | OAuth token required | `agy_test.go`, `account_test.go` |
| Codex ActiveUntil ignored | Parsed but unused | Fail-closed when subscription expired | `codex` tests |
| Intent silent skip | Errors discarded | Persist `intent_decision_status/error` facts | nexus tests |
| Mission stall → NEEDS_YOU risk | No durable progress watchdog | `ApplyProgressWatchdog` → `FAILED_NO_PROGRESS` | `watchdog_test.go` |
| AUTO model leftover | `cfg.Model` singleton mint | Empty inventory clears model | `runtime_routing_test.go` |
| lint-go unparam | always-nil error return | void `mergeSessionIndexEntry` | `make lint-go` |

## Findings

| ID | Severity | Status | Descrição | Resolução |
|----|----------|--------|-----------|-----------|
| AUTH-001 | HIGH | FIXED | Desktop Origin-only auth | cd5cb87 |
| AUTH-002 | HIGH | PARTIAL | WS query token | Rejected under tunnel; loopback residual |
| PROV-002 | HIGH | FIXED | Claude credentials.json | cd5cb87 |
| PROV-003 | HIGH | FIXED | AGY weak auth heuristics | cd5cb87 |
| PROV-001 | HIGH | FIXED | Codex ActiveUntil | cd5cb87 |
| INTENT-001 | HIGH | FIXED | Silent intent errors | b2c4645 |
| WD-001 | HIGH | FIXED | Stall watchdog | b2c4645 |
| UPD-001 | BLOCKER (release) | OPEN | Trust root placeholder | Needs human keygen |
| I18N-001 | HIGH | OPEN | PlanBuilder/FlowRun hardcoded strings | AGENTS §2.5 |
| REL-001 | BLOCKER (cert) | OPEN | No authenticated Mission / Win / macOS / 8h soak | External |
| AUTH-003 | HIGH | OPEN | Bootstrap CSRF/reuse on loopback | Documented residual |
| TREE-001 | HIGH | OPEN | Uncommitted DEV doc churn | Stabilize before merge |

## Verification (fresh)

| Comando | Exit | Resultado |
|---------|------|-----------|
| `make quality` | 0 | format, lint, typecheck, go tests, 67 frontend files / **346 tests** PASS, version 0.5.0-beta.23 |
| `make build` | 0 | web dist + `./nexus` built at 37b9dde |
| `go test ./internal/control/web ./internal/core/provider/adapters/... ./internal/nexus/... ./internal/profile` | 0 | targeted packages PASS during fixes |
| `make docs-verify` | FAIL (known) | visual manifest stale — residual |

## Dívida preexistente / residual aceitável para *continuar na branch*

- Scheduler fallback to unauthenticated profiles for interactive login-on-launch
- Maestro lifecycle not fully wired into runner (methodology optional)
- WS query token still used on loopback (browser limitation)
- Updater fail-closed without production key (safe, non-functional)
- Large god files (`PlanBuilderSurface`, `runner.go`)

## Riscos residuais reais

1. Merging to `main` would publish AGENTS.md i18n/CSS/`any` violations in primary Flow UI.
2. Production auto-update cannot verify manifests until a real Ed25519 trust root is embedded.
3. Release certification remains NO-GO without live provider Mission evidence and native CI.
4. Dirty uncommitted validation docs can confuse release readers if pushed partially.

## Próxima ação

**NOT_READY_FOR_MAIN** — do not merge.

Human / follow-up sequence:

1. i18n + remove static inline styles/`any` in `PlanBuilderSurface` / `FlowRunSurface` (or formal waiver).
2. Generate and embed production updater public key; sign release manifests.
3. Clear or commit remaining DEV validation doc inventory consistently.
4. Obtain authenticated Mission + Win/macOS evidence if claiming production GO.
5. Re-run `make quality` + independent red team → only then consider READY_FOR_MAIN.

Merge/push to `main` **not** executed. Branch is **3 commits ahead** of `origin/feat/nexus-maximum-delivery` (local only).

## Follow-up (post-audit notifications)

- Restored Codex `info.ExpiresAt` from `ActiveUntil` (regression noted by [Red team merge reject](4e787dc4-4a9a-4cf6-9afa-b1a51f17441f)).
- Evidence FAIL now wins over missing HEAD (MR-003 from [Backend behavior audit](383e8029-c4a4-4725-9e33-f05cbdac822e)).
- MR-002 intent silent swallow already mitigated earlier via `intent_decision_status` facts.
- Frontend i18n BLOCKERs from [Frontend QA audit](f86ed862-81c0-4574-8122-6bc72c3d0591) remain open → verdict still **NOT_READY_FOR_MAIN**.
