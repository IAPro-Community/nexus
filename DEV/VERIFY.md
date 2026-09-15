# Verification: Nexus V1 (post-pending-issues)

## 2026-09-15 — Compilação e Instalação Local do Binário Nexus (v1.0.0-beta.1 @ 2b9c956)

- PASS — `make build`: Bundles frontend gerados com sucesso e binário `./nexus` compilado com commit `2b9c956`.
- PASS — `make install LOCAL_BIN=/home/desenvolvedor/.local/bin`: Binário instalado atomicamente em `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `which nexus`: Resolvido para `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `nexus version`: `IAPro Nexus 1.0.0-beta.1 (linux/amd64, go1.25.0) commit 2b9c956 built 2026-09-15T03:22:18Z`.
- PASS — `nexus usage --json`: Retorno estruturado de cotas e provedores funcionando sem anomalias.
- PASS — `curl -sI http://127.0.0.1:13000/`: HTTP 200 OK com novos assets do Web OS servidos.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador).



- PASS — `go test -v -run TestUnifiedUsage_AddProfileModal ./internal/tui/...`: 4/4 novos testes unitários aprovados (abertura por tecla `a`, abertura por `enter` em provedor sem perfil, submissão de `ActionAdd` com nome informado e cancelamento por `esc`).
- PASS — `go test ./internal/tui/...`: 100% da suite do TUI aprovada.
- PASS — `go test ./internal/app/...`: 100% da suite do App CLI aprovada, validando despacho de `tui.ActionAdd` para `addCmd`.
- PASS — `npm --prefix web run check:styles`: Conformidade estrita com SCSS Modules e allowlist global.
- PASS — `npm --prefix web run typecheck`: `tsc --noEmit` compilado limpo (0 erros).
- PASS — `npm --prefix web run lint`: ESLint com `--max-warnings=0` aprovado sem erros ou avisos.
- PASS — `npm --prefix web test`: Vitest executou 73 arquivos e 388 testes com 100% de aprovação.
- PASS — `make web-verify`: Todos os 10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers). Relatório em `DEV/validation/FRONTEND_LATEST.md`.
- PASS — `make build`: Binário `./nexus` v1.0.0-beta.1 gerado com sucesso.
- PASS — `install -m 755 ./nexus ~/.local/bin/nexus`: Binário instalado no ambiente de execução do usuário.
- NOT EXECUTED — `git commit` e `git push` (em conformidade com as diretrizes do Orquestrador).

## 2026-09-15 — Redesign da Superfície de Uso (UX/UI, Responsividade & Paridade CLI)

- PASS — `npm --prefix web run check:styles`: validação de CSS Modules e conformidade com allowlist de estilos globais.
- PASS — `npm --prefix web run format:check`: 100% dos arquivos formatados segundo regras do Prettier.
- PASS — `npm --prefix web run lint`: ESLint aprovado com 0 erros e 0 warnings (`--max-warnings=0`).
- PASS — `npm --prefix web run lint:styles`: Stylelint aprovado sem nenhuma violação.
- PASS — `npm --prefix web run typecheck`: TypeScript `tsc --noEmit` compilado com 0 erros.
- PASS — `npm --prefix web test`: Vitest executou 73 arquivos de teste e 388 testes com 100% de aprovação (incluindo testes de modelos de uso e paridade i18n).
- PASS — `npm --prefix web run build`: Geração completa de bundles web com Tailwind CSS v4 e esbuild em ~1.3s.
- PASS — `make web-verify`: Todos os 10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers). Relatório gerado em `DEV/validation/FRONTEND_LATEST.md`.
- PASS — `go test -v ./internal/control/web/...`: Todos os testes da API de controle Web e endpoints de perfis/recursos aprovados (0 falhas).
- PASS — `make build`: Binário `./nexus` (v1.0.0-beta.1) compilado com sucesso com os novos assets embutidos.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador: commits e pushes somente sob ordem explícita do Maestro).

## 2026-09-15 — Limpeza de galhos + pick terminal (HEAD `356f1a3`)

- PASS — `go test ./internal/control/host/ -count=1` (pós-pick e pós-recuperação).
- PASS — `go vet ./internal/control/host/` + `gofmt -l` limpo nos 3 arquivos do pick.
- PASS — `make quality` (frontend 73 arqs/385 testes, version contract `1.0.0-beta.1`).
- PASS — `make build` (binário `./nexus` 20,5M) + `git diff --check`.
- PASS — `make web-verify` 10/10 gates (format, typecheck, lint, stylelint,
  null-arrays, test, i18n, build, embed-sync, ui-markers).
- PASS — loss audit: 12/12 galhos apagados recuperáveis via
  `archive/2026-09-15/*`; `git bundle verify` OK.
- NOT EXECUTED — commit/push dos 5 arquivos de release staged, CI hosted
  same-SHA, `release-proof`/`overnight-soak`/`docs-verify` (exigem SHA
  congelado + Go 1.25.1/covdata).

## 2026-09-14 — Remediação do review pré-main

- PASS — `GOTOOLCHAIN=go1.25.1 make release-proof`: gates locais completos,
  cobertura Go agregada 52,4%, segurança e version contract aprovados;
  `OVERNIGHT_SOAK_EVIDENCE=MISSING` foi mantido como pendência explícita.
- PASS — `go test ./...` e `go test -race ./...` sem falhas ou data races.
- PASS — `make quality`, `make test-e2e` e `make security`.
- PASS — lint frontend com `--max-warnings=0`.
- PASS — `GOOS=windows GOARCH=amd64 go test -c` para `scripts`, `internal/control/web`
  e `internal/control/host`.
- PASS — regressão de persistência do binding `--lead` em Registry reloadado.
- BLOCKED — `make docs-verify`: manifesto visual stale para o worktree não
  commitado; exige novo SHA e evidência visual após congelamento.
- NOT EXECUTED — commit, push, CI hosted same-SHA, rulesets e branch protection.

## 2026-09-14 — CLI supervised sem Lead no teclado

- PASS — `wantInteractiveLead` default false; `--lead` true (case-insensitive); `StripControlFlags` remove `--lead` junto de `--direct`/`--supervised`.
- PASS — `go test ./internal/app/` (flag/strip + suite existente).
- PASS — smoke `nexus codex --yolo` (PTY): runtime `nexus.interactive_lead=false`; **0** agentes novos; `/status` chega ao Codex (autocomplete de slash) sem hang de promoter.
- PASS — smoke `nexus codex --yolo --lead`: label `nexus.interactive_lead=true` + bind Lead; caminho LineShadow preservado.
- PASS — `nexus control list` lista o runtime do atalho; help/completions expõem `--lead`.
- PASS — `make build` + install `~/.local/bin/nexus` (`0.5.0-beta.23`).

## 2026-09-14 — Enter preso em /status (qualquer CLI supervised)

- PASS — `IsImmediateSubmitLine` cobre `/status`, `/model`, `/nexus`, `:help`, vazio; prompts normais não bypassam.
- PASS — `decideInteractiveSubmit` não chama DecideIntent para `/status`; Decide lento estoura 250ms → KeepWithLead; Promote ainda funciona em goal multi-stream.
- PASS — `go test` promotion+app focados (11); binário reinstalado em `~/.local/bin/nexus`.
- UNVERIFIED — smoke manual `nexus codex` → `/status` responde na TUI (requer sessão interativa).

## 2026-09-14 — CLI shortcuts: busy failover + orphan host reap

- PASS — `go test ./internal/app ./internal/control/launcher ./internal/nexus ./internal/control/tui` (350): failover auto cai no próximo perfil; explícito não faz fallback; `ReapOrphanHostsFrom` mata host sem registry e preserva RUNNING.
- PASS — lock exclusivo em `kivergmail` + `nexus codex --yolo`: stderr `Profile codex:kivergmail is busy. Automatically falling back...` e attach em `codex:kiver.omegasistemas` (~8s, TUI YOLO). `nexus codex:kivergmail --yolo` falhou em 159ms sem fallback.
- PASS — `nexus control cleanup` reaped 1 orphan host de teste; 3 RUNNING reais intactos (`agy-06G9ZZ…`, `runtime_06GA0Z…`, `codex-06G9YT…`).
- UNVERIFIED — Windows named-pipe scan of `__control-host` (Linux `/proc` only).

## 2026-09-14 — Rail flicker + terminal recover

- PASS — `useNexusData` só atualiza estado quando o snapshot muda; `resolveProjectSelection` não bounceia para `projects[0]` em poll vazio; outline não re-entra em loading com cache.
- PASS — overlay de Recover só em ERROR/DISCONNECTED/recovering; `CONNECTING` usa indicador leve; WS effect sem dep de `t`.
- PASS — attach prefere `surface.data.runtimeId`; `runtime_changed` faz rebind em `new_runtime_id`; sticky `nextBoundRuntimeId` ignora poll stale.
- PASS — `make web-verify` (FRONTEND_LATEST PASS), testes focados rail/terminal, `make build`, smoke browser: expand estável + Recover Agy conectou Antigravity CLI.

## 2026-09-14 — Codex terminal input

- PASS — `web/src/nexus/terminalInput.ts` shared filter drops focus/CPR/`[i`; `AgentTerminal` and `TerminalPane` use it; mouse/CSI-u/UTF-8/`0x03`/literals forwarded.
- PASS — `internal/control/host` forwards `CmdInput` byte-for-byte; `/nexus stop` never intercepted; `LineShadow` keeps control/paste/OSC/CPR out of prompt.
- PASS — `CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT=1` in direct adapter and supervised spec/driver.
- PASS — rebuilt `/home/desenvolvedor/.local/bin/nexus` from `1b840ca`; `nexus version` `1b840ca`, `go version -m` module `github.com/IAPro-Community/nexus`.
- PASS — focused web `terminalInput.test.ts` (10), Go race (`host`, `promotion`, `driver`, `spec`), `go test ./...`, `make quality`, `make build`, `git diff --check`.
- UNVERIFIED — manual Web/Desktop + `nexus codex` acceptance (no mouse `[i`/`[I`/CSI residue; common keys no interrupt; explicit `0x03` interrupts; literals intact).

## 2026-09-14 — Codex autonomous launch compatibility

- PASS — Codex autonomous argument regression: approval/sandbox flags are
  placed before `exec` so Codex 0.154.0 accepts the command.
- PASS — Codex reviewer argument contract: read-only sandbox and no-approval
  flags are placed before `exec`; provider contract golden updated.
- PASS — targeted race tests and `make quality`; alternate Go 1.25.1
  `make release-proof` completed all local gates.
- PASS — authenticated Mission `run_06G9VJV1K6J649PN2EMEHR1EMG` routed to
  `codex:kivergmail`, executed the implementer in a dedicated worktree,
  received an approved verdict from a distinct read-only Codex reviewer, and
  completed package plus global verification as `COMPLETED_VERIFIED`.

## 2026-09-13 — P0/concurrency and release closure refresh

- PASS — `make attention-e2e`, `make overnight-smoke` and
  `make overnight-soak DURATION=45m` on the current worktree.
- PASS — `GOTOOLCHAIN=go1.25.1 make release-proof` completes all local gates
  with aggregate coverage at 52.4%; the default Go 1.25.0 installation still
  lacks `covdata` and is recorded as an environment-only block.
- PASS — authenticated Codex Mission `run_06G9VJV1K6J649PN2EMEHR1EMG`
  completed with a distinct read-only reviewer and durable package/global
  verification evidence (artifacts in
  [`validation/current/real-provider-mission/`](validation/current/real-provider-mission/)).
- BLOCKED — `make docs-verify` still sees a stale visual manifest for
  uncommitted frontend changes.
- NOT EXECUTED — push, hosted same-SHA CI, rulesets and branch protection;
  these require explicit maestro authorization.

## 2026-09-13 — Fase A promoção adaptativa

- PASS — `internal/nexus/promotion` cobre Enter retido, UTF-8/backspace,
  ESC/CSI/OSC, CPR, bracketed paste, controles e OFF passthrough.
- PASS — CLI attach e Web terminal integram `KEEP_WITH_LEAD`, `ASK` e
  `PROMOTE`; Web expõe confirmação `promotion_answer`.
- PASS — `go test ./... -count=1`, `go test -race` nos pacotes tocados,
  `go vet ./...`, `make quality` e `git diff --check`.
- UNVERIFIED — Mission com providers autenticados, CI hosted same-SHA e
  soak de oito horas; nenhum push ou branch protection foi executado.

## 2026-09-13 — Fase 0 pós PRODUCTION_NO_GO

- Scope: código apenas (falso sucesso / Maestro / Mission PATCH / quota /
  launcher / installers / CI key gate). **Não** overnight, **não** CI hosted,
  **não** Mission autenticada.
- PASS — `make quality`; `go test -race` em store/nexus/runner/quota/launcher/
  release/sign-update-manifest; smoke isolado (`maestro_mode=OFF`,
  `nexus run` → strict admission, sem wording “quota/rate limit”).
- PASS (contrato) — CI desktop Linux/Windows/macOS falham se
  `NEXUS_UPDATE_PUBLIC_KEY` vazia; installers desktop recusam
  `desktop-checksums.txt` unsigned.
- BLOCKED / NEXT — congelar SHA (commit); humano provisiona Ed25519 em
  `beta-signing` / `vars.NEXUS_UPDATE_PUBLIC_KEY`; só então push e CI same-SHA.
- UNVERIFIED — overnight 8h, Windows/macOS nativos same-SHA, Mission
  `COMPLETED_VERIFIED` autenticada, red team no SHA novo.

## 2026-09-13 — Current local closure audit

- Candidate observed: `eed65e5a7e46adc896165c94accb7fc73fc6750f`; worktree was
  not frozen, so no release certification is asserted.
- PASS — `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet
  ./...`, `make lint-go`, `make quality`, `make security`, `make build`,
  `make build-desktop-wails`, `make web-verify`, and
  `bash scripts/verify-update-gate.sh`.
- PASS — Go/browser regressions preserve PTY bytes including ESC/OSC/kitty,
  UTF-8 and literal `/e[...]` or `\\e[...]` provider text.
- BLOCKED — `make docs-verify` because the visual manifest is stale for
  uncommitted frontend changes.
- UNVERIFIED — authenticated Mission evidence, production updater key
  provisioning, native Windows/macOS/NSIS, hosted same-SHA browser/release
  matrix, and eight-hour overnight certification.

## 2026-09-13 — Fase B revisão/remediação

- PASS — ciclo canônico Planner → Builder → Reviewer (`CHANGES_REQUESTED`)
  → remediação com finding novo → `APPROVED` → Verifier sem write turn.
- PASS — ledger distingue quatro identidades e termina sem findings abertos;
  `make overnight-smoke` executa o teste.
- BLOCKED — `make docs-verify` exige manifesto visual renovado para as
  alterações frontend não commitadas; nenhuma regeneração foi feita.
- BLOCKED — `make docs-verify` still requires a refreshed visual manifest for
  uncommitted frontend changes; no manifest regeneration was performed.

## 2026-09-13 — Mission Teams validation

- PASS — full Go tests, full race suite, `go vet`, security, build, desktop
  build, E2E, Attention E2E and bounded overnight smoke.
- PASS — canonical deterministic Planner → Builder → Reviewer → remediation →
  Verifier test and direct-task anti-overprocessing coverage.
- PASS — structured finding identity/deduplication/resolution, durable phase
  state, Flow/WorkPlan round-trip and desired-vs-actual routing preference.
- FAIL / ENVIRONMENT — `make coverage` and `make release-proof` stop because
  the active Go 1.25 toolchain lacks `covdata`.
- FAIL / CONCURRENT WORKTREE — latest `make web-verify` reports formatting in
  unrelated `web/src/app/NexusSplashScreen.tsx`; the Mission Teams surface
  itself passed typecheck, lint, styles, tests and build.
- UNVERIFIED — authenticated real-provider Mission, provider failover proof,
  native Windows/macOS and eight-hour overnight certification.
- Report: [`validation/current/MISSION_TEAMS_REPORT.md`](validation/current/MISSION_TEAMS_REPORT.md).

## 2026-09-13 — Production GO/NO-GO red team

- Verdict: `PRODUCTION_NO_GO` / `CERTIFICATION_INVALID_FOR_CURRENT_HEAD`.
- Candidate executable SHA: `e53f8352e4c3647632ebac7877cf0a6a3bd62647`.
- PASS (local, this session): `make quality`, `go vet`, `make security`,
  `make docs-verify`, `make build`, `go test -race ./...` (1079/65).
- PASS-but-not-soak: `make overnight-smoke` 0.243s.
- FAIL: overnight 0h abort; no same-SHA CI; ancestor CI E2E failures;
  Windows/macOS native; updater trust root; authenticated Mission;
  Maestro default ASSIST; UNKNOWN quota allocation; Mission PATCH complete.
- Evidence: `DEV/validation/production-go-no-go/FINAL_REPORT.md`.

## 2026-09-13 — Overnight production certification

- FAIL / ABORT — precondition `production-finalization/FINAL_REPORT.md` missing.
- FAIL / ABORT — verdict `READY_FOR_OVERNIGHT_CERTIFICATION` not present in repo.
- FAIL / ABORT — HEAD `e53f8352e4c3647632ebac7877cf0a6a3bd62647` is not a certified overnight candidate.
- NOT RUN — campaigns A–F, 8h soak, recovery, provider failover, red team.
- Evidence: `DEV/validation/overnight-production-certification/FINAL_REPORT.md`.

## 2026-09-12 — Capability review intensive

- PASS — CLI JSON: doctor, security, control doctor, providers, profiles, version.
- PASS — `make security` (govulncheck clean).
- PASS — focused tests after fixes: web, codex, nexus, flags, update; `make lint-go` clean.
- PASS — TunnelHost ⇒ `IsTunnelActive` regression test.
- PASS — providers advertise CrossAccountResume only for Codex after honesty fix.
- PASS — follow-up: official cloudflared 2026.8.2 checksums, Darwin `.tgz` extract,
  tunnel `OnExit` disarms auth, `--effort` preserves native `-c`, AGY `-p`→DIRECT,
  supervised Codex bootstrap calls `Prepare` (CrossAccountResume seed).
  Retest: `go test ./internal/control/web ./internal/control/flags ./internal/app ./internal/control/driver ./internal/core/provider/adapters/codex -count=1`.
- SKIPPED — Claude CLI OAuth expired; Gemini CLI client unsupported.
- UNVERIFIED — interactive `/resume` cross-account and live mid-turn Codex TUI.
- DEFERRED — Claude/OpenCode Usage file-stub honesty; developer-preset credential exposures;
  TLS on private `--remote`.

## 2026-09-12 — Auth follow-up (desktop + WS query tokens)

- PASS — `/api/v1/desktop/bootstrap` requires pre-provisioned desktop bearer (Origin alone → 401).
- PASS — WS `?token`/`?session` accepted only on loopback listen; rejected on private `--remote`
  and while tunnel active (`TestWebSocketQueryTokenRejectedOnPrivateRemote`).
- PASS — frontend omits WS query token outside loopback (`shouldAttachWebSocketQueryToken`).
- PASS — `go test ./internal/control/web -count=1`; focused web vitest for agent terminal + networkHost.
- PASS — `TestVerifyCloudflaredChecksumUsesPlatformPin` (Darwin sidecar path covered).

## 2026-09-12 — Codex CrossAccountResume

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/conversation ./internal/control/flags -count=1`.
- PASS — `TestAdoptSessionFromSiblingProfile` copies sibling rollout + index
  into destination canonical sessions and marks cross-account.
- PASS — `TestPrepareSeedsSiblingSessionsForPicker` seeds sibling IDs into
  destination `session_index.jsonl` during Prepare.
- PASS — `TestForeignAccountRolloutIgnoredForQuota` and
  `TestRolloutBelongsRejectsMismatchedAccountID` keep GetUsage from inheriting
  foreign resume artifacts.
- UNVERIFIED — interactive `nexus codex:kivergmail` `/resume` listing of
  omegasistemas threads and live `nexus usage` attribution on account
  `9cba8371-...` (requires local authenticated Codex TUI).

## 2026-09-12 — Mission evidence isolation

- PASS — two persisted Missions in the same project receive only their own
  validation entries from the shared canonical stream.
- PASS — the complete hash chain is verified before the per-Mission filter;
  invalid or missing `run_id` metadata fails closed (`TestRunApplicationRejectsMissionEvidenceWithoutRunID`).
- PASS — `TestRunApplicationProjectsCompleteMissionEvidenceBeyondDefaultPage`
  projects 101 entries without inheriting the bounded 100-entry list page.

## 2026-09-12 — Nexus Codex TUI lock / flag compatibility

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/control/flags ./internal/control/driver ./internal/control/host ./internal/profile -count=1`.
- PASS — `go test ./internal/core/provider/adapters/codex -race -count=1`.
- PASS — `go vet` on the touched packages; focused `Quota|Monitor` Nexus tests.
- PASS — TUI lock blocks `runAppServerRateLimits` / `GetUsage` app-server spawn;
  rollout windows still returned; `thread_history` symlink untouched on usage path.
- PASS — Codex `-c`/`-p` pass through; `--continue` → `resume --last`;
  `--print` → `exec`; `--effort` → `-c model_reasoning_effort=...`.
- PASS — Prepare creates canonical `sessions` + `config.toml` only (no second
  `.codex/sessions` write tree).
- PASS — legacy `.codex/config.toml` is migrated before canonical defaults are
  appended; the regression test preserves the legacy model setting.
- PASS — profile preparation waits for an active per-home TUI lock before
  touching configuration/session artifacts.
- PASS — E2E profile import resolves the real host home when `HOME` points at an
  isolated provider home; explicit source override precedence is covered.
- UNVERIFIED — live authenticated `nexus codex` mid-turn interrupt / `model:
  loading` reproduction after the fix (requires interactive provider session).

## 2026-09-12 — Canonical validation evidence projection

- PASS — `go test ./internal/nexus ./internal/control/web -count=1`.
- PASS — `RunApplicationService.ValidationEvidence` reads the existing
  append-only Mission stream, verifies its hash chain before projection and
  returns explicit empty/no-chain state when no stream exists.
- PASS — the projection test closes and reopens the same SQLite store, then
  reloads the run, stream, entry and verified chain successfully.
- PASS — Codex TUI lock tests cover non-blocking ownership, app-server rate
  limit suppression, session migration safety and canonical home preparation.
- PASS — Web route `GET /api/v1/runs/{id}/validation-evidence` exposes that same
  projection; no second ledger or report store was introduced.
- PASS — Web client typed method `getRunValidationEvidence` and its transport
  test are green; frontend typecheck and build pass.
- UNVERIFIED — no authenticated Mission produced a durable production stream
  in this campaign, so this verifies the projection contract only.

## 2026-09-12 — Final closure continuation after routing/guidance integration

- PASS — `go test ./... -count=1`.
- PASS — `go test -race ./... -count=1`.
- PASS — `go vet ./...` and `git diff --check`.
- PASS — `make security`, `make build`, `make build-desktop`.
- PASS — `make quality` (aggregated formatting, lint, staticcheck, Go tests,
  frontend tests and version consistency). One existing ESLint warning at
  `web/src/nexus/AgentTerminal.tsx:218` is non-blocking.
- PASS — `make web-verify` (format, typecheck, lint, stylelint, null-array,
  tests, i18n, build, embed-sync and UI markers).
- PASS — `./nexus doctor --json` at `2026-09-12T03:02:22Z`; desktop shell is
  correctly reported as `SKIPPED`, not PASS.
- PASS — focused AGY/runtime hardening tests: expired-token quota probes fail
  closed and no-browser helper paths do not open external OAuth routes.
- PASS — task requirements now carry bounded generic `ExecutionGuidance` into
  prompt compilation; persisted routing decisions project canonical Skill refs
  and Maestro guidance refs only when an actual source/reference is present,
  including selected Skill source/version/hash/reason provenance.
- PASS — the local autopilot contract now exercises package/global recorder
  integration into `ValidationEvidenceStream`, chain verification and
  Git-SHA-bound `VERIFIED` entries. Latest ephemeral test stream was sequence 2;
  it is not a production Mission stream.
- PASS — Mission evidence captures bounded Go/OS/arch metadata and Node plus
  package-manager versions when applicable, using fixed no-shell probes with
  a two-second timeout.
- PASS — E2E harness now supports an explicit empty stable root for providers
  that reject temporary credential homes; focused root-safety tests pass.
- UNVERIFIED — Codex remained at `model: loading` even with stable root;
  AGY reached the runtime but reported `not signed in`. No real provider
  marker or authenticated Mission evidence was promoted.
- UNVERIFIED — authenticated Nexus provider Mission, durable production
  evidence stream, live failover/escalation/handoff, Windows/macOS and
  overnight acceptance remain unavailable.

## 2026-09-12 — Final closure local autopilot continuation

- PASS — `go test ./... -count=1` after the local autopilot acceptance test.
- PASS — `go test -race ./... -count=1`.
- PASS — `go vet ./...` and `git diff --check`.
- PASS — `TestLocalAutopilotContractTraversesDiscoveryRoutingSkillsAndVerification`:
  bounded discovery → `DIRECT` → Builtin Skill → task-aware model → Mission
  Runner → `COMPLETED_VERIFIED`.
- PASS — SQLite restart reload of `RoutingDecisionJSON` and durable routing
  report projection.
- PASS — persisted routing decisions now include Agent score/confidence/reason
  and selected account scope in addition to provider/profile/model.
- Release remains `NO-GO`: authenticated provider Mission, live failover/
  handoff, native Windows/macOS and overnight evidence are unavailable.

## 2026-09-11 — Codex frescor + TUI Contas & Quotas

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/core/quota ./internal/profile ./internal/tui ./internal/control/host -count=1`.
- PASS — host same-account ingest + adopt; parser aceita qualquer evento com `rate_limits.primary`; evento velho → `ESTIMATED`.
- PASS — `AvailabilityLabel` UNKNOWN → `SEM DADOS` (nunca DISPONIVEL); 5h=0 → `QUOTA ESGOTADA`.
- PASS — last-known Codex aceita bump de `IdentityVersion` (email → `chatgpt_account_id`) com mesmo `AccountID` durável.
- PASS — TUI: chrome compacto (`usageChromeLines=6`), colunas sem MODELO, STATUS `OK`/`ESGOTADA`/`SEM DADOS`/`OFF` cabem em width 80/132/160.
- PASS — `SessionHost.Stop`/`Terminate` disparam `RefreshUsageSnapshot` async para codex/agy.
- PASS — `nexus usage --json` pós-rebuild: `codex:kiver.omegasistemas` LIVE; `codex:kivergmail` ESTIMATED com last-known (rollout isolado tinha `primary:null` do Codex — sem inventar LIVE).
- NOTE — cota LIVE da gmail exige sessão isolada que grave `rate_limits.primary` (não só splash com primary null).

## 2026-09-11 — Native Skill contract transport continuation

- PASS — `go test ./internal/nexus/... -count=1` after adding canonical
  `SkillIDs` to WorkPlan, Flow, runner packages and ContextCapsule.
- PASS — builtin `coding`/`testing` resolve with Maestro unavailable; unknown
  IDs reject deterministically and legacy/generic packages can coexist.
- PASS — the actual execution freeze path accepts a real project plan with
  builtin `SkillIDs` while Maestro is offline and preserves compatibility aliases.
- PASS — `go test ./... -count=1` and `go test -race ./... -count=1` after
  correcting handoff receipts and reconciling latest Codex/TUI contracts.
- PASS — `go vet ./...`, `gofmt` cleanliness and `git diff --check`.
- PASS — `make web-verify`, `make build`, `make build-desktop`,
  `git diff --check` and repository-wide `gofmt` cleanliness.
- Release remains NO-GO for missing authenticated Mission/provider/platform
  evidence.

## 2026-09-11 — Codex quota attribution by chatgpt_account_id + isolated sessions

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/core/provider/adapters/agy ./internal/core/scheduler ./internal/profile ./internal/core/quota ./internal/tui ./internal/app`.
- PASS — e-mails no texto do rollout **não** roubam atribuição; no máximo um perfil reivindica cada rollout host.
- PASS — `FetchedAt` vem do evento `token_count`; `resets_at` no passado vira `window rolled over` e não conta como esgotado.
- PASS — AGY com `refresh_token` permanece autenticado (`Token refresh pending`) e o probe de quota pode rodar com `BROWSER=false`.
- PASS — `nexus usage --json` após rebuild: `codex:kiver.omegasistemas` LIVE com 5h/weekly alinhados ao `/status` Codex (reset weekly `05:35 on 15 Sep`); `codex:kivergmail` UNKNOWN com erro `aguardando primeira sessão isolada desta conta` (esperado até a primeira sessão isolada).
- PASS — symlinks `sessions` → `~/.codex/sessions` migrados para diretórios reais por perfil no `GetUsage`/`Prepare`.
- PASS — scheduler prefere conta Codex com capacidade sobre 5h esgotado (`TestCodexExhaustedFiveHourLosesToHealthyAccount`).

## 2026-09-11 — Codex/CSI-u no terminal Nexus

- PASS — `go test ./internal/control/driver ./internal/core/provider/adapters/codex ./internal/runtime`.
- PASS — `gofmt` nos arquivos Go alterados e `git diff --check`.
- PASS — o driver supervisionado injeta
  `CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT=1`; o adapter direto usa a mesma
  proteção.

## 2026-09-11 — Evolution corrective closure (current working tree)

- PASS — focused Go regression: `internal/nexus`, `internal/core/scheduler`,
  `internal/control/web`, `internal/control/handoff`.
- PASS — natural-language classification covers React, Go REST, Kubernetes,
  security, review and complex ecommerce; classified requirements persist on
  generated Flow steps.
- PASS — AgentMatcher no longer treats provider resource gates as Agent
  personality; generic `implementer` labels no longer erase specialized task
  roles.
- PASS — same-provider handoff preserves Agent/Project identity and reports
  `NATIVE_RESUME_UNVERIFIED` without provider-level confirmation; cross-provider
  context handoff is labeled `CONTEXT_HANDOFF`.
- PASS — `nexus run "<goal>"` dispatches through Flow, AgentMatcher,
  ResourceScheduler and MissionRunner while provider-native `nexus run <provider>`
  remains compatible.
- PASS — scheduler tie-breaking is deterministic by profile name.
- PASS — `make quality`, `make security`, `make build`, `make web-verify`,
  `go vet ./...` and `make lint-go` on the corrected tree.
- PASS — `make test-e2e` (terminal, protocol, host and web race scenarios).
- UNVERIFIED — `make docs-verify` rejects the stale visual manifest because the
  Web change is intentionally uncommitted; refreshing that manifest requires a
  committed source SHA or an explicit release commit.
- UNVERIFIED — full Go race completion, authenticated PTY/failover E2E, live
  providers, and native Windows/macOS execution; see final validation report.

## 2026-09-10 — Gates de format/lint alinhados

- PASS — `make format-check` usa o Prettier local e verifica TypeScript, CSS,
  SCSS e JSON.
- PASS — `make lint-styles` verifica CSS e SCSS.
- PASS — `make lint-frontend` usa o ESLint local; permanece apenas o warning
  preexistente de diretiva não utilizada em `NexusWorkspaceApp.tsx`.
- PASS — `.lintstagedrc.json` usa binários locais e inclui SCSS.

## 2026-09-10 — Fechamento dos últimos caminhos de quota

- PASS — AGY só persiste usage live com `AccountScope` verificável; arquivos
  legados sem identidade permanecem não atribuídos até refresh autenticado.
- PASS — `BatchFetch` e seleção de modelo AGY não leem/escrevem cache de quota
  sem escopo; runtimes carregam o escopo quando o perfil registrado o fornece.
- PASS — monitor de quota associa runtime afetado por escopo exato quando
  disponível, evitando colisões entre contas do mesmo provider/perfil.
- PASS — perfis pendentes/sem identidade mantêm apenas o identificador local
  reservado; `AccountScope.Verifiable()` só fica verdadeiro após identidade
  autenticada, impedindo cache e routing atribuíveis antes do login.
- PASS — `go test ./...` após o fechamento.

## 2026-09-10 — Review corrective pass

- PASS — `go test ./...` após as correções.
- PASS — `go test -race ./...` (processo concluído sem detector de corrida).
- PASS — `go vet ./...`.
- PASS — `npm --prefix web run typecheck`, testes Web (320/320), build e
  Prettier nos arquivos Web alterados.
- PASS — `git diff HEAD --check`.
- NOT VERIFIED — execução nativa Windows/macOS e Cloudflare real; o tunnel
  recebeu correção de lifecycle e readiness, mas não foi iniciado contra uma
  instalação real de `cloudflared` nesta sessão.

## 2026-09-10 — AccountScope e isolamento de estado

- PASS — `go test ./internal/core/model ./internal/core/quota
  ./internal/core/cooldown ./internal/profile ./internal/control/events
  ./internal/nexus`.
- PASS — regressões A/B: cache de quota e cooldown da conta A não são visíveis
  para a conta B; troca de credencial preserva `account_id` e incrementa
  `identity_version`.
- PASS — `go test -race ./...`.
- PASS — `git diff --check`.
- NOT VERIFIED — registro TUI/CLI de instalação não registrada e execução nativa
  Windows/macOS permanecem fora deste slice.

## 2026-09-10 — Registry progressivo de CLIs

- PASS — `InstallationRegistry` separa descoberta de binário do registro de
  conta e persiste `installations.json` com permissões restritas.
- PASS — `providers status --json` expõe estado de instalação/registro.
- PASS — `providers register` cria perfil isolado e permite adiar autenticação.
- PASS — testes de contrato e `go test -race` nos pacotes alterados.
- PENDENTE — integração equivalente na TUI/Web e modo `UNMANAGED_EPHEMERAL`.

## 2026-09-10 — Integração Web/TUI

- PASS — API de providers retorna estado de instalação/registro e aceita
  registro de perfil via POST.
- PASS — TUI exibe Providers separadamente e registra o item selecionado como
  `PENDING_AUTH` sem copiar credenciais.
- PASS — cliente Web tipado expõe listagem e registro.
- PASS — `go test -race ./...` e `npm --prefix web run typecheck`.
- PASS — `SaveUsageForExecution(..., managed=false)` garante que execuções
  efêmeras não persistam usage; o wiring de lançamento com HOME temporário
  permanece pendente.

## 2026-09-10 — Correção de uso compartilhado entre contas Codex

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/profile
  ./internal/core/quota ./internal/nexus -count=1`.
- PASS — `go vet` nos mesmos pacotes.
- PASS — regressão cobre perfis distintos apontando por symlink para o mesmo
  `~/.codex/sessions`; apenas a conta do host recebe o rollout LIVE.
- PASS — `nexus usage --json` foi executado localmente; a resposta manteve
  `profile_id`, `account`, `status` e `source` por perfil, com fallback
  `ESTIMATED` quando não há rollout atribuível.
- PASS — `git diff --check`.
- NOT VERIFIED — execução nativa Windows/macOS; esta correção foi verificada
  no Linux.
- CONDITIONAL — `go test ./...` foi tentado, mas o sandbox bloqueou sockets
  Unix/TCP e escrita em `/home/.../checkpoints`; falhas ocorreram em testes
  de host/protocol/web/update/core lifecycle fora da área alterada. Os testes
  focados do domínio passaram.

## 2026-09-10 — Composer application boundary and context propagation

- PASS — TDD RED confirmou que Composer ignorava contexto cancelado; após a
  correção, `TestComposerOperationsRejectCanceledContext` passou cobrindo
  criação, listagem, leitura, turno, skill, finalização e resolução.
- PASS — `go test ./internal/nexus ./internal/control/web`.
- PASS — `make quality-full`: Web 62 arquivos / 315 testes; Go completo com
  race/vet; lint; build; security (`No vulnerabilities found`).
- WARNING PRE-EXISTING — ESLint mantém um warning de dependência ausente em
  `web/src/app/NexusWorkspaceApp.tsx`; nenhum erro novo.
- NOT VERIFIED — execução nativa em Windows/macOS; esta fatia foi verificada
  no Linux.
- PASS — smoke não destrutivo: `go run ./cmd/nexus --help` e `go run
  ./cmd/nexus doctor` retornaram exit 0; o diagnóstico listou os providers
  instalados e manteve estados não verificáveis como WARN/SKIPPED.

## 2026-09-10 — WorkPlan application boundary

- PASS — `go test ./internal/nexus ./internal/control/web` após extrair CRUD
  e revisões para `internal/nexus/plan_application.go`.
- PASS — `make quality-full`: suíte Web com 62 arquivos / 315 testes; testes
  Go completos incluindo race; `go vet`; lint; build; security (`No
  vulnerabilities found`).
- WARNING PRE-EXISTING — ESLint mantém um warning de dependência ausente em
  `web/src/app/NexusWorkspaceApp.tsx`; sem erros novos.
- NOT VERIFIED — execução nativa em Windows/macOS; esta etapa foi verificada
  no Linux.

## 2026-09-08 — Project Shell IPC timeout

- PASS — teste de fallback do ShellDriver com `SHELL` inválido.
- PASS — `go test ./internal/control/driver ./internal/control/web ./internal/nexus`.
- Causa confirmada nos logs: `/usr/bin/zsh` não pôde ser executado pelo
  SessionHost e o handler anterior convertia o erro em `409`.
- PENDENTE — smoke real após reiniciar o processo `nexus web`; o processo atual
  foi iniciado antes da recompilação.

## 2026-09-08 — Rail inteligente e gestão de projetos

- PASS — `cd web && bun run typecheck`.
- PASS — `cd web && bun run lint:styles` e `bun run check:styles`.
- PASS — `cd web && bun run test`: 62 arquivos / 313 testes.
- PASS — `cd web && bun run build`.
- PASS — `make quality` (inclui lint Go, testes Go e gates frontend); ESLint
  manteve somente o warning preexistente de hook em `NexusWorkspaceApp.tsx`.
- PENDENTE — screenshot/browser visual específico do rail nos cinco breakpoints;
  a implementação foi revisada estaticamente, mas nenhum harness autenticado foi
  executado nesta sessão.

## 2026-09-07 — Deep review fixes (commit 059bb5c)

- `go test -count=1 ./...` — PASS (all packages)
- `go vet ./...` — PASS
- `cd web && bun run typecheck` — PASS
- `cd web && bun run lint` — PASS (1 pre-existing warning)
- `cd web && bun run lint:styles` — PASS
- `cd web && bun run check:styles` — PASS
- `cd web && bun run test -- --run` — PASS (62 files, 313 tests)
- `make quality` — PASS
- `make security` — PASS (no vulnerabilities)
- `make build` — PASS (nexus v0.5.0-beta.23)
- `git diff --check` — PASS
- Commit: `059bb5c` (29 files, +1062/-271)

## 2026-09-07 — Preparar tarefa

- PASS — `npm run verify` (10/10 gates), `npm run build` e 311 testes Vitest.
- PASS — `go test ./...`, `go vet` afetado, typecheck, lint, format, stylelint,
  allowlist e `git diff --check`.
- PASS — validação visual/Axe específica do novo diálogo nos cinco breakpoints
  registrada na seção seguinte.

## 2026-09-07 — Visual/Axe do diálogo de preparação

- PASS — `node web/scripts/task-preparation-visual-verify.mjs`.
- PASS — screenshots em 320×568, 390×844, 768×1024, 1024×768 e 1440×900.
- PASS — ausência de overflow horizontal, navegação por teclado e Axe sem
  violações `serious`/`critical`.

## 2026-09-07 — Browser hardening e bootstrap persistido

- PASS — `timeout 180 node web/scripts/e2e-hardening-verify.mjs`; seis
  breakpoints, deep-links, Axe, Settings/ARIA e density.
- PASS — harness atualizado para resolver o bootstrap por `nexus web url`, sem
  exigir token impresso pelo servidor.

`make docs-verify` permanece **CONDITIONAL** neste worktree: o verificador
rejeita corretamente o manifesto visual ancorado em `2358618` enquanto há
alterações visuais não commitadas. A âncora só deve ser atualizada junto com um
novo SHA e screenshots correspondentes.

## 2026-09-07 — Verificação same-SHA de evidência visual

- `scripts/docs-verify.mjs` também inspeciona alterações staged e unstaged nos
  caminhos visuais; não é permitido certificar screenshots de um candidato
  imutável enquanto o worktree contém mudanças locais.
- `make docs-verify` falha de forma esperada no checkout atual, identificando
  o manifesto em `23586183...` e os paths dirty. Isso é um bloqueio honesto,
  não uma falha de teste a ser mascarada.

## 2026-09-07 — Resume-first local slice

- PASS — `overviewResumeModel.test.ts`: prioridade Needs You → ativo →
  recuperável → recente, filtragem por projeto e associação ao agente.
- PASS — `bun run test` (62 arquivos / 313 testes), `bun run typecheck`,
  `bun run lint:styles`, Prettier e build Web.
- PASS — `node web/scripts/e2e-hardening-verify.mjs` após a mudança; os seis
  breakpoints e Axe continuam verdes.
- CONDITIONAL — o painel usa runtimes/agentes já carregados; runs históricos de
  missão ainda não são propagados ao Overview e precisam de E2E de retorno.
- PASS — o hardening E2E passou a verificar explicitamente
  `[aria-labelledby="overview-resume-title"]` no Overview.

## 2026-09-07 — Composer permissões por destino

- `cd web && bun run typecheck` — PASS.
- `cd web && bun run test -- --run src/features/work` — PASS (10 arquivos,
  40 testes), incluindo Composer `MISSING` editável e destinos bloqueados.
- `cd web && bun run lint && bun run lint:styles && bun run check:styles` — PASS
  (apenas warning preexistente de dependência de hook em `NexusWorkspaceApp`).
- `cd web && bun run build` — PASS.
- `node web/scripts/build.mjs` sincronizou `web/dist` com o bundle embutido;
  `go test ./...` — PASS.
- `go test ./internal/nexus/runner -run 'Test(OvernightAcceptanceSandbox|RemediationPersistsStrategyChangeBeforeRetry|NeedsHumanContractIsStructuredAndDurable)'` — PASS; sandbox determinístico cobre plano paralelo, receipt de dependência, falha injetada, remediação e DoD global.
- `make quality` — PASS; frontend 61 arquivos/311 testes e `golangci-lint` sem issues após corrigir a mensagem de erro capitalizada em `internal/nexus/maestro.go`.

## 2026-09-07 — Correção do bootstrap de autenticação Web

- `nexus web` + `curl` em processo real — PASS: fragmento de bootstrap trocado
  por cookie; `/api/v1/session` retornou `authenticated=true` e CSRF válido.
- `go test ./internal/control/web -run 'TestServerBootstrapURLKeepsTokenInFragment|TestProvidersDocsCaptureUsesSyntheticInventory|TestServer_BootstrapAndAuth'` — PASS.
- `npm --prefix web run test -- --run src/api.test.ts` — PASS, incluindo
  consumo do fragmento e limpeza via `history.replaceState`.
- `npm --prefix web run typecheck` — PASS; `make build` — PASS.

## 2026-09-07 — Higiene de mídia documental

- `npm --prefix web run docs:capture` — PASS em fixture sintética isolada; shell
  real validado por round-trip do marcador `__NEXUS_DOCS_TERMINAL_OK__`.
- `go test ./internal/control/web -run 'TestProvidersDocsCaptureUsesSyntheticInventory|TestServer_BootstrapAndAuth'` — PASS.
- `node scripts/docs-verify.mjs` — PASS; manifesto com VIS-001…VIS-011,
  classificação `SYNTHETIC` e rejeição do cenário `real-local-bootstrap`.
- Inspeção visual — PASS para terminal, providers, workspace e Direct; sem
  projetos, paths de usuário, credenciais ou versões reais de providers.
- `git diff --check` continua apontando whitespace em bindings Wails já
  modificados por outra alteração do worktree; não foi alterado nesta tarefa.

## 2026-09-07 — Review closure local

- `npm run verify` — PASS, 10/10 gates; 61 arquivos Vitest / 310 testes.
- Browser E2E — PASS: bootstrap/API readiness, Overview → Terminais, Axe,
  deep-links, 320/390/768/1024/1280/1440px, accordion e density delta.
- `go test ./...` — PASS; `go vet ./...` — PASS.
- `make docs-verify` — PASS com manifest VIS-001…VIS-011.
- `make build-desktop-wails` — PASS com Wails Linux production + `webkit2_41`.
- `VIS-011` — PASS visual em janela Wails real, com Core Web isolado e captura
  da tela de boas-vindas funcional; captura não é mock.
- `git diff --check` — PASS.
- `go run github.com/goreleaser/goreleaser/v2@v2.18.0 release --snapshot --clean`
  — PASS; archives, DEB/RPM packages and checksums generated locally.
- Pendentes obrigatórios para GO: execução nativa Windows/macOS, snapshot
  GoReleaser, CI em SHA único após integração com `origin/main` e regeneração
  do publication report nesse SHA. Nenhum commit/push foi feito.

## 2026-09-07 — Correções iniciais de desempenho Web/Desktop

- `npm --prefix web run verify` — PASS, 10/10 gates.
- `go test ./...` — PASS.
- `go test -race` dos pacotes Desktop/Web/Terminal/Host — PASS.
- `go vet ./...` — PASS.
- `make build-desktop-wails` — PASS em produção com `webkit2_41`.
- `git diff --check` — PASS após restaurar bindings gerados automaticamente.
- Benchmark comparativo e smoke nativo de performance — PENDENTES.

## 2026-09-07 — Plano de paridade de desempenho Web/Desktop

- Artefato `.omx/plans/desktop-web-performance-parity.md` criado e inspecionado.
- Referências principais conferidas em `cmd/nexus-desktop/main.go`,
  `web/src/platform/desktopBridge.ts`, `web/src/api.ts`,
  `web/src/app/NexusWorkspaceApp.tsx`, `web/src/nexus/AgentTerminal.tsx` e
  `web/src/features/work/FlowCanvas.tsx`.
- `git diff --check -- .omx/plans/desktop-web-performance-parity.md` — PASS.
- Nenhum benchmark foi executado e nenhuma causa foi declarada confirmada.

## 2026-09-07 — Execução Luna P0/P1 e fundação de continuidade

- `go test -count=1 ./...` — PASS.
- `go vet ./...` — PASS.
- `cd web && bun run verify` — PASS, 10/10 gates; a execução anterior foi
  corrigida formatando `web/src/nexus/api.ts`.
- `go test ./internal/app ./internal/control/handoff ./internal/nexus -count=1`
  — PASS após adicionar launch supervisionado, checkpoint Maestro schema 4 e
  lease do quota monitor.
- `git diff --check` — PASS antes das alterações atuais; repetir no gate final.
- Cross-platform native runtime — PENDENTE; cross-build não substitui Windows
  ConPTY/Named Pipe nem macOS PTY/race.

## Evidência de cobertura de quota — 2026-09-07

- `go run ./cmd/nexus providers --json` — PASS; binários e capabilities foram
  enumerados sem tratar capability `Usage` como dado atual.
- `go run ./cmd/nexus profiles --json` — PASS; AGY/Codex autenticados, porém
  todos `UNKNOWN`/`NONE` sem janelas/freshness. OpenCode não autenticado e Cursor
  sem Usage. O monitor corretamente não emite alerta de percentual nesses casos.
- Teste de prioridade cross-provider — PASS; `TestResourceRecommendation_ProviderPriorityControlsFallbackOrder`.

## Terminal/continuidade — 2026-09-07

- `go test -race -count=1 ./internal/control/host ./internal/control/handoff
  ./internal/nexus ./internal/app ./internal/core/config` — PASS.
- `go vet` dos pacotes afetados e `git diff --check` — PASS.
- Cross-build de testes dos pacotes host, handoff e nexus para Windows amd64 e
  macOS arm64 — PASS como compilação; execução nativa permanece PENDENTE.
- `--supervised` e `/nexus<TAB>` ainda não têm PASS nativo Windows/macOS.

## 2026-09-06 — Plano Luna (somente documentação)

- Código de lançamento, attach, quota monitor, checkpoint/handoff e workflow
  inspecionado para fundamentar DEV/SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md.
- Nenhuma implementação ou suíte runtime executada nesta etapa; não há novo
  resultado de CI remoto. Verificações de execução estão definidas em P0–P12.

## 2026-09-06 — CI closure pass after remote failure reproduction

- Remote run `34060911997` was inspected by job/step metadata. Frontend failed
  on format; Windows and macOS failed on their full Go test/race step. Logs are
  unavailable through the repository API without admin log permission.
- Frontend: **PASS**, `cd web && bun run verify` (10/10 gates), 306 tests,
  including build and embed synchronization.
- Go: **PASS**, `go test -count=1 ./...`, `go test -race -count=1 ./...`, and
  `go vet ./...`.
- Lint: **PASS**, `golangci-lint v2.12.2 run --timeout=5m ./...` (`0 issues`).
- Platforms: **PASS (compile evidence)**, Go test binaries and `cmd/nexus`
  compile for Windows amd64 and Darwin arm64; native execution still belongs
  to the GitHub Windows/macOS runners.
- Desktop/release: **PASS**, Wails Linux package and GoReleaser v2.18.0
  snapshot artifacts/checksums.
- Repository hygiene: **PASS**, `gofmt` and `git diff --check`.
- Generated Wails bindings: **PASS**, Prettier normalized the generated model
  file and the aggregate `make quality` gate passed afterward.
- Remote CI must be rerun on the resulting commit; no commit or push was made.

## 2026-09-06 — Monitor residente de quotas

- Go: `go test ./...` PASS; `go vet ./...` PASS; testes focados de Nexus,
  eventos, Web e App PASS.
- Web: `npm run typecheck` PASS; Vitest PASS (61 arquivos / 306 testes);
  ESLint PASS com warnings preexistentes, Stylelint PASS, `check:styles` PASS,
  build PASS.
- `git diff --check` PASS.
- Cobertura adicionada para Gemini 28% independente de Claude/GPT 100% e para
  histórico global sem filtro de runtime.

## 2026-09-06 — Workspace tabs and terminal viewport regression

- Route synchronization now keys off the actual pathname/search and does not
  reopen `/overview` on every workspace state update.
- Product tab clicks route through the application opener, preserving the URL
  and active surface together.
- Hidden xterm panels defer output, focus, and resize until `data-active=true`;
  activation flushes output and performs a guarded fit.
- Frontend verification: Prettier, TypeScript, and 292 Vitest tests passed.
- `bun run build`, `make build-desktop-wails`, installation of the Linux Wails
  artifact, and `git diff --check` passed.

## 2026-09-06 — Desktop input hardening

- Deep links now enforce bounded length, one resource ID, no fragments/users,
  and reject path/control/whitespace ambiguity.
- Linux autostart quotes the executable path using Desktop Entry `Exec` rules.
- `go test ./internal/desktop -run 'Test(AutoStart|DeepLink)'` passed.

## 2026-09-06 — Desktop capability evidence

- `DesktopBridge` now consumes the backend `GetCapabilities()` result instead
  of equating generated Wails methods with available native functionality.
- Pre-bootstrap state is conservative; capability evidence is cached after the
  authenticated desktop bootstrap.
- `cd web && bun run verify` passed all checks, including the regression test.

## 2026-09-06 — Native notification input safety

- AppleScript and PowerShell notification paths now pass title/body as
  arguments to static scripts instead of interpolating user-controlled text.
- `go test -race ./internal/control/notify -count=20` passed; Windows package
  cross-compilation also passed.

## 2026-09-06 — External URL fallback safety

- Added shared structural URL validation to Web/Desktop bridges and all
  browser-opening fallbacks.
- Unsafe schemes (`javascript`, `data`, `file`, malformed URLs) are rejected;
  the frontend regression and full `bun run verify` passed.

## 2026-09-06 — Handoff secret hygiene

- Redacted two persisted loopback bootstrap token URLs from `DEV/HANDOFF.md`
  and removed machine-local installation paths.
- Current-tree scan found no long hexadecimal bootstrap token URLs outside Git
  metadata; history was not rewritten because local user work must be preserved.
- `internal/core/security` and script tests passed.

## 2026-09-06 — Same-SHA security gate

- Added a dedicated CI `Security` job running pinned `govulncheck` with Go
  1.25.14.
- Added `Security` to the release workflow's required same-SHA job list.
- `actionlint` and release package tests passed; remote CI execution remains
  pending because no push was performed.

## 2026-09-06 — Detached update signature consumption

- Fixed the shared Update Service to consume `update-manifest.sig` beside a
  static `update-manifest.json` when no signature header is present.
- Missing, malformed, unavailable, or invalid signatures remain fail-closed.
- RED/GREEN evidence: detached-signature unit test passes normally and under
  `go test -race ./internal/update -count=5`.

## 2026-09-06 — Explicit Maestro update boundary

- Added the explicit CLI namespace `nexus maestro status|doctor|update`.
- `nexus update` remains owned by the shared Nexus Update Service and no longer
  claims to update Maestro in help/completion text.
- Added a degraded-status test for an unavailable Maestro installation and
  aligned Web settings copy with the explicit Maestro operation.

## Current closure pass — 2026-09-06

- Frontend: **PASS**, `make web-verify` 10/10 at `2026-09-06T05:20:55Z`.
- Go: **PASS**, `gofmt`, `go vet ./...`, `go test ./...`, and
  `go test -race ./...`.
- Go lint: **PASS**, `golangci-lint v2.12.2 run ./...` (`0 issues`).
- Aggregate quality: **PASS**, `PATH=/tmp/nexus-tools:$PATH make quality`.
- Stress: **PASS**, Desktop/Update/SessionHost/Workspace at `-count=20`.
- macOS fallback input handling: **PASS** by static review and Go lint; native
  execution remains pending on a macOS runner.
- Installers: **PARTIAL**, pinned release + SHA256 checks pass; signed
  manifest/keyring publication is still unavailable.
- Release workflow: **STATIC PASS**, YAML parses and publication is gated on
  named same-SHA CI jobs; runtime execution remains unverified until a new
  branch/tag CI run exists.
- Update signer byte binding: **PASS**, published manifest bytes are covered by
  the detached signature and regression-tested.
- Wails Linux packaging: **PASS**, `make build-desktop-wails` generated and
  packaged the native artifact; Windows/macOS execution remains pending.
- Native release artifact promotion: **STATIC PASS**, CI workflow uploads and
  release workflow promotes artifacts from the selected same-SHA run; runtime
  evidence remains pending.
- GoReleaser snapshot: **PASS locally** with v2.18.0; all six CLI archives,
  checksums, and Linux DEB/RPM artifacts were generated. Deprecation warnings
  remain non-blocking follow-up items.
- Desktop Linux build: **PASS**, `make build-desktop`.
- Security: **PASS**, `make security` reported `No vulnerabilities found`.
- Native Windows/macOS: **NO-GO**, CI run `34012236345` failed those jobs;
  browser/Desktop downstream jobs were skipped. See
  [`FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md`](validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md).

## 2026-09-05 — Implementação Completa Web + Desktop Multiplataforma

- `go test ./...` — PASS (100% de sucesso nos pacotes `internal/app`, `internal/desktop`, `internal/update`, `internal/control/web`, `internal/doctor`, `internal/release`, etc.).
- `go vet ./...` — PASS.
- Frontend Quality (`bun run quality`):
  - `check:styles` — PASS (allowlist SCSS Modules estrita).
  - `format:check` — PASS.
  - `lint` — PASS (0 erros).
  - `lint:styles` — PASS.
  - `typecheck` — PASS (`tsc --noEmit` 0 erros).
  - `vitest run` — PASS (58 arquivos de teste, 280 testes unitários).
- Frontend Build (`bun run build`):
  - Geração de chunks ESM sob demanda em `web/dist` — PASS.
  - Sincronização e verificação de integridade do bundle embutido no Core e no Desktop (`web.EmbeddedDistFS()`) — PASS.
- Desktop Shell & Capabilities:
  - Wails v2 estável (`v2.15.0`) — build e testes unitários do bridge nativo (`internal/desktop`) — PASS.
  - Deep links (`nexus://`), window management e PlatformBridge — PASS.
- Maestro Desacoplado:
  - Scripts de instalação (`install.sh` e `install.ps1`) com opt-in explícito (`--with-maestro` / `-WithMaestro`) verificados — PASS.
  - Modo degradado `MAESTRO_DEGRADED` sem falhar o produto — PASS.

- `bun install --frozen-lockfile` — PASS com Bun 1.3.9 após regeneração do lockfile.
- Frontend format/typecheck/lint/stylelint — PASS; lint com 0 erros e 36 warnings preexistentes.
- Frontend Vitest — PASS: 51 arquivos, 253 testes.
- golangci-lint `v2.12.2 config verify` — PASS; execução completa reproduz 59 achados existentes, sem supressão.
- `go test ./...` — PASS após ExecutableResolver.
- `go test -race ./internal/control/terminal ./internal/control/protocol ./internal/control/host` — PASS.
- Windows cross-compile `go test -c` para terminal, launcher, host e protocol — PASS; execução nativa ainda pendente.

Estado: runtime/release continuam NO-GO até lint debt, build/embed gate e evidência nativa Windows/macOS serem fechados.

## 2026-09-05 — Path identity and credential capability increment

- `go test ./internal/core/config ./internal/nexus/store` — PASS após migração `0012_path_identity.sql`.
- Cross-compile de config/runtime para Windows e Darwin — PASS.
- `go test ./...` — PASS.
- `go vet ./...` — PASS.
- `git diff --check` — PASS.

Limitações mantidas: identidade nativa por handle no Windows, execução nativa Windows/macOS e bundle do `nexus doctor` ainda não foram validados.

## 2026-09-05 — Doctor CLI and diagnostic bundle

- `nexus doctor --json` com diretórios temporários isolados — PASS.
- `nexus doctor --bundle` — PASS; ZIP contém somente `report.json` e `MANIFEST.txt`.
- `go test ./internal/doctor ./internal/app` — PASS.
- `go test ./...` — PASS.

O bundle foi validado por allowlist; execução nativa Windows/macOS e reuso na Web permanecem pendentes.

## 2026-09-04 — Topbar Overlap Refactor, Responsive Hardening & Accordion Theme Quality Gates

- `make web-verify` — PASS (8/8 gates verdes: typecheck, lint, null-arrays, vitest [51 files, 249 tests], i18n, build, embed-sync, ui-markers).
- `make build` — PASS (binário `nexus v0.5.0-beta.23` gerado e instalado com bundle web embutido).
- `npm --prefix web run test:e2e-hardening` — PASS (100% de sucesso nos 6 viewports):
  - `320x568` (mobile-se): botão Terminal desobstruído (`elementFromPoint` verificado sem overlap).
  - `390x844` (mobile-iphone): botão Terminal desobstruído.
  - `768x1024` (tablet): botão Terminal desobstruído.
  - `1024x768` (laptop-compact): layout estável, ações secundárias colapsadas.
  - `1280x800` (laptop-critical): terminal acessível em 1º lugar, sem clipping.
  - `1440x900` (desktop): renderização completa sem colisão.
  - Validação WAI-ARIA do Accordion de Temas com paletas visuais e medição real do delta de densidade (Comfortable 497px vs Compact 481px).


- `make web-verify` — PASS (typecheck, lint, null-arrays, vitest (244 tests), i18n, build, embed-sync e ui-markers).
- `make build` — PASS (binário `nexus v0.5.0-beta.23` compilado com assets embutidos atualizados).
- Auditoria Playwright multi-viewport (1440x900, 1024x768, 768x800, 480x800) em `/usr/bin/google-chrome`:
  - Visualização de screenshots reais renderizados:
    - `verify_1440x900_desktop.png`: layout limpo com rail lateral, topbar sem colisão, janelas em cascata e radar visível.
    - `verify_1024x768_compact.png`: botões redundantes de criação escondidos, hierarquia de janelas com z-index correto.
    - `verify_768x800_tablet.png`: rail colapsado em drawer/overlay, topbar compacto com badges e menus sem quebra.
    - `verify_480x800_mobile.png`: zero sobreposição/encavalamento de controles; comandos e labels longos suprimidos graciosamente, radar e switcher de idiomas alinhados no topo sem overflow horizontal.
- Z-Index verificado: radar e popovers operando em camada superior (`--nx-z-popover: 4000`) a janelas focadas (`500`).
- Primitivas `<Select>` e `<Input>` do design system adotadas em 100% dos modais e superfícies auditadas, eliminando selects HTML crus não estilizados.

## 2026-09-04 — Terminal recover após reboot/serviço

- `go test -count=1 ./internal/control/registry ./internal/control/web` — PASS.
- `make web-verify` — PASS (typecheck, lint, null-arrays, Vitest, i18n,
  build, embed-sync e ui-markers). Relatório:
  `DEV/validation/FRONTEND_LATEST.md`.
- Reprodução manual restante: reiniciar `nexus web` (ou a máquina) com uma
  janela de terminal de agente aberta e confirmar overlay **Iniciar /
  Recuperar Agente** em vez de xterm preto.

## 2026-09-04 — Baseline atual e Composer skills

- `make web-verify` — PASS (typecheck, lint, null-arrays, Vitest, i18n,
  build, embed-sync e UI markers).
- `go test -count=1 ./...` — PASS.
- `go vet ./...` — PASS.
- `git diff --check` — PASS.
- CI frontend passou a fixar Bun `1.3.9` e o `packageManager` do workspace foi
  declarado como `bun@1.3.9`.
- Composer agora expõe e persiste aceitar/dispensar de Maestro skills e envia
  as skills aceitas na finalização; Readiness, Unknowns e Assumptions são
  visíveis.

## Identidade visual e assets de marca — 2026-09-04

- `make web-verify` — PASS (typecheck, lint, null-arrays, vitest, i18n, build, embed-sync e ui-markers).
- `go test ./...` — PASS (todos os pacotes Go testados e passando).
- `make build` — PASS; binário `nexus v0.5.0-beta.23` gerado e instalado com novos assets embutidos em `internal/control/web/dist`.
- Validação visual: extração do master em alta resolução com desmatting transparente, verificação de contraste em fundo escuro (`#080a0f`) e fundo claro, geração do pacote favicon e renderização no rail e hero.

## Controles de terminais e resumo de Agentes — 2026-09-03

- `make web-verify` — PASS (typecheck, lint, null-arrays, testes, i18n, build,
  embed-sync e ui-markers).
- `make build` — PASS; instalou `nexus v0.5.0-beta.23`.
- Cobertura funcional implementada: troca de modo com erro visível e rebinding,
  feedback de lease, fechamento com escolha aba/runtime, confirmação de Project
  Shell e badge operacional clicável com foco/recuperação.

## Bootstrap de contexto durável — 2026-09-03

- `go test ./internal/nexus ./internal/control/web` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web test -- --run` — PASS (42 arquivos, 180 testes).
- `make build` — PASS; instalou `nexus v0.5.0-beta.23`.
- A criação é explícita, gera `AGENTS.md` base e nunca sobrescreve contexto
  existente.

## Recuperação de terminal desconectado — 2026-09-03

- Web: `npm --prefix web run typecheck` — PASS.
- Web: `npm --prefix web test -- --run` — PASS (42 arquivos, 175 testes).
- Build: `make build` — PASS; instalou `nexus v0.5.0-beta.23`.
- Recover/Start agora vincula diretamente a geração retornada; o terminal também
  oferece fechamento explícito da aba.

## Nova sessão de IA — 2026-09-03

- Web: `npm --prefix web test -- --run` — PASS (40 arquivos, 157 testes).
- Web: `npm --prefix web run typecheck` — PASS.
- Build: `make build` — PASS; instalou `nexus v0.5.0-beta.23`.
- O launcher agora é disparado depois que a superfície Composer está montada.

## TERM=dumb em CLIs interativos — 2026-09-03

- `go test ./internal/runtime ./internal/control/terminal` — valida substituição
  de `TERM=dumb` sem mutar o ambiente original e preservação de terminais normais.
- A correção cobre execução direta e runtimes supervisionados com PTY/ConPTY.

## AGY quota por grupo — 2026-09-03

- `go test ./internal/core/quota ./internal/app ./internal/core/scheduler ./internal/nexus` — PASS
- Conta AGY com Gemini disponível e Claude/GPT esgotado agora fica `INDISPONIVEL`/
  `QUOTA ESGOTADA` no nível da conta.
- Conta AGY só fica `DISPONIVEL` quando os dois grupos têm quota.

## AGY keyring e modelo — 2026-09-03

O processo AGY recebe `GNOME_KEYRING_CONTROL` do Secret Service privado, e o
arquivo `antigravity-cli/settings.json` não é mais sobrescrito pelo host a cada
execução. Testes Go focados passaram e `make build` instalou `nexus v0.5.0-beta.15`.

## Radar falso + terminal “connecting” — 2026-09-03

- Go: `go test ./internal/control/host` — PASS
  (lista numerada sozinha não vira pergunta; lixo `\uFFFD`/box-drawing rejeitado;
  `Select an option` + `1. …` continua `choice`)
- Web: `make web-verify` → [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md) — PASS
  (radar colapsa mensagens iguais; honesty usa `sanitizeAttentionText`;
  WS `?runtime_id=`; erro fatal para reconnect; taskbar “agentes degradados”)
- Runtime: `make install` + reinício de `nexus web` + `pkill` dos `__control-host` velhos
- Browser smoke: Radar **ok** / “Nenhum terminal ativo” sem hosts;
  AgentTerminal mostra `error` + “Use Recover/Start…” em vez de `connecting` eterno

## Organizador de terminais + radar — 2026-09-03

Cobertura automatizada do P0:

- Go: `go test ./internal/control/host ./internal/control/notify` — PASS
  (shortcuts/`[y/n]` isolado silenciam; yn real + free_text; clear on `thinking...`;
  OS notify só com evidência + dedupe de fingerprint)
- Web: `make web-verify` → [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md) — PASS
  (título sem emoji/`(N ❓)`; fingerprints de mensagem iguais colapsam; Sim/Não só se `yn`;
  radar/focus model; i18n parity)
- Embed sync: `web/dist` ≡ `internal/control/web/dist` após o gate

Caminho manual ainda necessário no ambiente do operador (depende de `make build` +
reinício de `nexus web`): dois Projetos, pergunta yn real vs chrome de help, e
notificação de SO com a aba fechada.

## `null.forEach` no Flow — 2026-09-03

O bundle servido foi regenerado a partir de `flowModel.ts`, que trata
`plan.phases` e `phase.packages` não-array como coleções vazias. A regressão
para payloads nulos passou em `yarn test src/features/work/flowModel.test.ts`
(9/9), e `yarn build` atualizou `internal/control/web/dist`.

Nesta execução, `yarn typecheck` também passou.

## Normalização de WorkPlan — 2026-09-03

`web/src/nexus/workPlan.ts` passou a ser a fronteira de normalização para
todos os endpoints que retornam WorkPlan, cobrindo planos e fases nulos antes
do acesso da UI. Verificação atual: testes focados (19/19), `yarn lint`,
`yarn typecheck` e `yarn build` — PASS.

## Reanexação de terminal — 2026-09-03

Após Recover/Start, o terminal não reutiliza o `runtime_id` histórico da
superfície. Ele reanexa pelo Agente e acompanha a nova geração. Testes focados
(14/14), typecheck e build passaram; lint passou com 7 warnings preexistentes.

## Tabela TUI de uso — 2026-09-02

`nexus usage` usa a tabela navegável do Charm Bubbles. O filtro cobre
identidade e grupo de modelo; há teste unitário para ambas as buscas.

## Banner de atenção — 2026-09-02

O banner de intermediação não participa mais da linha expansível da grade do
Workspace OS. Ele é exibido como uma faixa compacta sob o cabeçalho, com
truncamento, controles padronizados e quebra responsiva. A heurística ignora
`? for shortcuts`, evitando o aviso falso visto no terminal.

Validação: `go test ./...`, `go vet ./...`, `yarn --cwd web typecheck`,
`yarn --cwd web lint`, `yarn --cwd web test` (31 arquivos/111 testes),
`git diff --check` e `make build` — todos PASS.

## Notificações transitórias — 2026-09-02

`TASK_COMPLETED` e `ERROR` são convertidos em toasts próprios, enquanto
`QUESTION` e `APPROVAL` continuam no componente de resposta. O modelo possui
cobertura para conclusão e exclusão de eventos interativos.

## Uso/quota — 2026-09-02

O fluxo de sessão Web mantém rotação automática antes da expiração absoluta,
propaga novo CSRF e orienta a recuperação via novo Bootstrap quando a sessão
expira. Testes Go e frontend passaram.

CLI e dashboard de recursos foram verificados com tabela/grid por modelo,
layout responsivo e status independente por grupo. Typecheck, lint, 31 arquivos
de testes (111 testes), `make build` e `git diff --check` passaram.

`nexus usage` agora preserva a capacidade por grupo de modelo e por janela,
exibindo percentual restante e reset 5h/semanal. A validação passou em
`go test ./internal/app ./internal/core/quota ./internal/profile`, `go vet ./...`
e `make build` (`v0.5.0-beta.9`). Cache vencido e snapshot de outra identidade
agora são rejeitados antes da exibição/seleção.
Grupos independentes também têm disponibilidade independente; no estado atual
`kiveromegasistemas` aparece como `DISPONIVEL` para Gemini e `INDISPONIVEL` para
Claude/GPT.

## Review update — 2026-09-02

The complete current diff was reviewed against `HEAD` (`c968852`). The current
worktree is **PASS AFTER CORRECTIONS**: `go build ./cmd/nexus`, `go test ./...`,
`go vet ./...`, frontend lint, tests and typecheck pass. The detailed findings
and exact file locations are recorded in
[`DEV/CODE_REVIEW.md`](CODE_REVIEW.md).

The corrections and latest evidence supersede the historical validation
snapshots below.

## AGY quota semantics — 2026-09-02

Legacy AGY quota files expose `percent_left` as consumed percentage. The
adapter and cache reader now normalize it to remaining percentage before
selection or display. Coverage includes boundary and fractional values.

## Status: ALL 7 PENDING ISSUES RESOLVED

### Build & Test
- `go build ./...` — clean
- `go test -race ./...` — ALL PASS
- `go vet ./...` — clean
- `npx tsc --noEmit` — clean
- `npx vitest run` — 44/44 tests PASS
- `make build` — PASS (v0.4.6)

### Pending Issues Resolution
| # | Issue | Status | Key Changes |
|---|-------|--------|-------------|
| 1 | Resources facade | ✅ | `ListResources()` real discovery, `AllocateResource()` persistence, `ResourcePicker` UI |
| 2 | Maestro synthetic state | ✅ | Honest degraded fallback, no hardcoded capabilities/recommendations |
| 3 | Update simulated | ✅ | Returns `501 Not Implemented` |
| 4 | Agent start without provider | ✅ | `ResolveStartParams()` + `REQUIRED_RESOURCE_SELECTION` flow |
| 5 | Config not reaching runtime | ✅ | `LaunchOptions` extended, full `AgentConfig` propagation |
| 6 | Terminal continuity | ✅ | `AgentTerminalBroker` integration, `runtime_changed` frames |
| 7 | Missions scaffold | ✅ | Kept as-is, documented as future work |

### Business Rules Location
All business logic lives in `internal/nexus/` (service layer). Web and TUI consume the same API endpoints. No resource selection, health checking, config propagation, or Maestro degradation logic exists in the frontend.

### Files Modified (this session)
- `internal/nexus/resource_discovery.go` (new)
- `internal/nexus/maestro.go`
- `internal/nexus/nexus.go`
- `internal/nexus/scheduler.go`
- `internal/nexus/store/agents.go`
- `internal/control/launcher/launcher.go`
- `internal/control/web/handlers_nexus.go`
- `internal/control/web/handler_terminal.go`
- `internal/control/web/broker.go`
- `internal/control/web/server.go`
- `web/src/features/agents/AgentsSurface.tsx`
- `web/src/nexus/ResourcePicker.tsx`
- `web/src/nexus/api.ts`

### Documentation Updated
- `DEV/NEXUS_V1_ARCHITECTURE.md` — resource discovery + terminal broker
- `DEV/NEXUS_V1_RESOURCE_SCHEDULER.md` — marked implemented with flow
- `DEV/NEXUS_V1_MAESTRO_INTEGRATION.md` — honest degraded fallback
- `DEV/NEXUS_V1_AGENT_MODEL.md` — updated start flow + config propagation
- `DEV/WORKLOG.md` — session entry with all changes

### Not Yet Started
- Cross-provider scoring with continuity, cooldown, and rate-limit risk
- Project-level and global policy hierarchy
- Account Handoff and Context Handoff recommendations
- Full Maestro CLI contract (`feat/nexus-contracts-v1`)
- Mission execution/orchestration (scaffold only)
- Full E2E verification with live providers
- Performance benchmarks

## Review final — 2026-09-04

- `go test -count=1 ./...`: PASS
- `go vet ./...`: PASS
- `go test -race -count=1 ./internal/control/... ./internal/nexus/... ./internal/core/security/... ./internal/core/session/...`: PASS
- `make web-verify`: PASS (typecheck, lint, null-arrays, 188 testes, build, embed-sync, ui-markers)
- `npm audit --offline --omit=dev --audit-level=high`: 0 vulnerabilidades
- Cross-build `linux/amd64`, `darwin/amd64`, `windows/amd64`: PASS
- `git diff --check`: PASS
- Smoke final `/api/v1/health`: HTTP 200; bootstrap: HTTP 302; servidor ativo em `127.0.0.1:3000`

Parecer e limitações: [`DEV/validation/CURRENT_CODE_REVIEW.md`](validation/CURRENT_CODE_REVIEW.md).

## TUI Usage selection — 2026-09-06

- `go test ./internal/tui -run 'TestUnifiedUsage(RestoresSelectionAfterEmptyFilter|ModeTogglingAndFlags|EscAndQQuit)$' -count=20` — PASS.
- `go test -race ./internal/tui` — PASS.
- `Enter` após filtro seleciona a linha destacada; cursor é restaurado após
  resultado vazio; `Esc` mantém comportamento de apenas fechar o filtro.

## Frontend embedded identity — 2026-09-06

- `cd web && bun run verify` — PASS (10/10 gates, incluindo igualdade entre
  `web/dist` e `internal/control/web/embedded`).
- `.github/workflows/ci.yml` agora usa esse verificador no job Frontend, não um
  teste de mera existência de arquivos.
- `go test ./internal/release -run 'TestInstaller' -count=1` — PASS; a política
  de caminho `IAPro Nexus` com preservação do diretório legado foi verificada.
- Actionlint/YAML dos workflows — PASS; jobs Windows/macOS preservam logs de
  falha em artifacts sem mascarar o resultado.
- O gate Windows inclui parsing explícito do `install.ps1`; execução local não é
  possível porque `pwsh` não está instalado neste runner Linux.
- Cross-build CLI Linux amd64/Windows amd64/macOS arm64 e Desktop Windows
  amd64 — PASS como compilação cruzada apenas; não substitui execução nativa.
- Listener failure: teste Linux 20x PASS; pacote `internal/control/host`
  compilou para Windows amd64 PASS; execução Windows ainda requer runner nativo.
- ConPTY interactive fixture: pacote Linux e compilação Windows amd64 PASS;
  execução nativa ainda pendente.
- FSMkdir: testes FS focados 20x PASS e compilação Windows amd64 do pacote Web
  PASS; execução nativa ainda pendente.
- IPC readiness: Host/Protocol 20x PASS e race PASS sem sleeps de prontidão;
  execução nativa Unix socket/Named Pipe continua dependente dos runners.
- HTTP readiness: bootstrap/session/restart focados 3x PASS e race PASS sem
  sleeps fixos.
- Web E2E/API/túnel completos: PASS normal e com race após remover waits de
  startup artificiais.
- PTY Unix: 20x PASS e race PASS com leitura baseada em output observado, sem
  sleep de readiness.
- Pacotes terminal/host/protocol/web: testes normais, race e golangci-lint PASS;
  `make quality PATH=/tmp/nexus-tools:$PATH` PASS.
- Revalidação final: `go test -race ./...`, `make security`, GoReleaser v2.18.0
  snapshot e `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7`
  passaram.
- Browser E2E real: PASS após corrigir locator case-sensitive de Settings;
  Playwright, Axe, deep-links, breakpoints e screenshots passaram em três
  execuções consecutivas; o script passou pelo Prettier.
- Fixtures Host/QA Windows não usam mais `cat`; usam `cmd.exe /D /Q /C more`.
  Linux normal/race e cross-compilação Windows/macOS dos pacotes afetados
  passaram. Wine/ConPTY permanece explicitamente não nativo.
- Update Service: testes 20x e race PASS; `HOME` de teste isolado e nenhum
  receipt gerado no checkout.
- Doctor: probes read-only e estados `WARN`/`SKIPPED` para capacidades nativas
  sem evidência; pacote passou 20x e race 5x.
- Registry: invalidação cross-process agora usa `ModTime` + tamanho + fingerprint
  SHA-256; teste de concorrência passou 50x, race 10x e `go test ./...` passou.

## Maestro catalog and skill context — 2026-09-07

- `go test ./internal/nexus -run 'TestMaestro|TestFindOrquestrador' -count=1` —
  PASS after global/profile catalog merge and prompt provenance changes.
- Frontend typecheck, lint, stylelint, focused tests and build — PASS after
  replacing command copy with complete skill context.
- Real browser smoke against the rebuilt server — PASS: `53 de 53 disponíveis`,
  53 cards and 53 usage-context panels; `skill-database-migrations` exposed
  its `SKILL.md` prompt.
- This does not replace native macOS/Windows execution; those release gates
  remain pending where recorded above.

## Open Design typography and visual QA — 2026-09-07

- Open Design audit identified undersized semantic typography and one dark-theme
  contrast failure in the shell.
- `node web/scripts/maestro-visual-verify.mjs` — PASS after correction:
  screenshots at five required viewports, no horizontal overflow, search
  interaction, Axe with zero serious/critical violations, and no page errors.
- `bun run format:check`, `bun run typecheck`, `bun run lint:styles`, focused
  theme/Maestro tests (14/14), frontend build, focused Go tests, `go vet` and
  `git diff --check` — PASS. Existing ESLint warnings remain warnings only.
- Typography tokens use `rem`; visual verification was repeated after the
  migration and the rebuilt server health endpoint returned `status: ok`.
- `FontScalePicker` no toolbar foi validado por typecheck/build e usa o mesmo
  `ThemeProvider` da tela Configurações, sem estado paralelo.
- Tipografia completa: nenhuma declaração `font-size`, `font` ou `fontSize` em
  px permanece em `web/src`; o servidor recompilado respondeu `status: ok`.

## Terminal helper do Project Shell — 2026-09-07

- `make build` — PASS; frontend recompilado e binário Nexus gerado.
- Verificação do bundle — PASS; `xterm-helper-textarea` presente em
  `web/dist/bundle.css` e CSS embutido sincronizado.
- Smoke Playwright contra o servidor real — PASS; os helpers do xterm ficaram
  invisíveis, não houve sequência de `W` sobre o terminal e não houve erro de
  página.
- O problema era visual do CSS ausente, não saída indevida do PTY.

## Tokens de espaçamento — 2026-09-07

- `bun run build` — PASS.
- `bun run typecheck` — PASS.
- `bun run lint:styles` — PASS.
- Bundle final contém `--nx-spacing-1`, `--nx-spacing-8`,
  `--nx-spacing-16` e aliases `--nx-space-4` — PASS.
- `make build` — PASS; bundle embutido regenerado.
- Instalação/execução final — PASS; `LOCAL_BIN=/home/desenvolvedor/.local/bin
  make install-local`, CSS HTTP contém `--nx-spacing-4: 16px` e o navegador
  calcula `16px` no `:root`.
- `make -n install` — PASS; fluxo local padrão e fluxo `DESTDIR` ficam explícitos
  no Makefile.

## Cota oficial do Codex via app-server — 2026-09-11

- Fonte primária nova: `codex app-server --stdio` + `account/rateLimits/read`
  (`internal/core/provider/adapters/codex/app_server_usage.go`). Rollouts e
  last-known continuam como fallback honesto, nessa ordem.
- `go test ./... -count=1` — PASS (suíte completa).
- `go test -race -count=1` em `internal/profile`, `internal/core/quota`,
  `internal/core/provider/adapters/codex`, `internal/tui`, `internal/nexus` — PASS.
- `go vet ./...`, `gofmt -l internal/` e `git diff --check` — PASS.
- Validação real com as duas contas, sem abrir sessão e sem enviar prompt:
  - `codex:kivergmail` — `RATE_LIMITED` / `OFFICIAL_API`; 5h em `0%` restante
    (reset 21:55) e semanal em `56%` restante (reset 08:37 de 15 Sep). Confere
    com o splash oficial do Codex CLI.
  - `codex:kiver.omegasistemas` — `LIVE` / `OFFICIAL_API`; 5h `98%` e semanal
    `58%`, com `accountId` próprio. As duas contas não se contaminam.
- `RATE_LIMITED` passou a ser leitura confiável: `quota.Engine.Trustworthy`,
  as barras e o monitor tratam o bloqueio informado pelo provedor como dado
  exato, em vez de descartá-lo para `SEM DADOS`.
- TTL de 60s vale também entre processos para leituras `OFFICIAL_API`; uma
  segunda chamada de `nexus usage` dentro da janela reaproveita o cache e não
  sobe outro app-server. Cache derivado de rollout nunca é reaproveitado.
- Identidade: a sonda exige `chatgpt_account_id` verificável, confere o
  `accountId` da resposta e o `codexHome` ecoado pelo `initialize`. Qualquer
  divergência descarta o payload. Desligável por `NEXUS_CODEX_APP_SERVER=0`.
- Controle estruturado (aprovações/eventos) segue diferido — ver
  [`DEV/AI_CONTROL_DEFERRED.md`](AI_CONTROL_DEFERRED.md), item 6.

## 2026-09-12 — automatic delegation interactive Lead closure

- PASS — focused `go test ./internal/app ./internal/control/launcher ./internal/nexus -count=1`.
- PASS — `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`
  and `git diff --check`.
- PASS — `make build` and `make build-desktop`; both retain explicit Nexus
  build metadata while disabling only redundant Go VCS stamping.
- PASS — canonical workspace lookup/registration, persistent Lead binding and
  runtime generation lifecycle are covered locally; launcher labels preserve
  the interactive policy metadata.
- UNVERIFIED — authenticated `nexus agy`/Codex provider session, real provider
  prompts and a durable authenticated Mission evidence stream remain unavailable.
- UNVERIFIED — the attempted supervised Codex smoke timed out during IPC
  handshake; no runtime remained registered afterward.

<!-- frontend-verify:latest -->
## Frontend gate — 2026-09-15T03:07:25Z

Verdict: **PASS**. Relatório completo: [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md).

