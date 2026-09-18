# BASELINE — Nexus Maximum Delivery (real, código como fonte de verdade)

Data (UTC): 2026-09-17
Autor: Principal Engineer (missão v1.0.0-beta.1)

> Princípio: relatórios antigos em `DEV/validation/*`, READMEs e vereditos GO históricos foram ignorados.
> Fonte de verdade: código atual + comportamento executável + CI do SHA atual.

## 1. Branch / SHA / working tree

- Branch atual: `feat/nexus-maximum-delivery`
- SHA (HEAD): `b3db75c7756fe25a70f4dc49760c90c38af735a9`
- `main`: `f71eb515278168e33d626fd631cd89dfb5e58faf`
- Delta: **146 commits à frente de `main`** (`git log main..HEAD --oneline | wc -l = 146`); diff `main...HEAD`: 1049 arquivos, +114448/-15128.
- `VERSION`: `1.0.0-beta.1` | `web/package.json`: `iapro-nexus-web 1.0.0-beta.1`, `packageManager bun@1.3.9`.
- Remote: `origin https://github.com/IAPro-Community/nexus.git`.
- PR alvo mencionado: `#2 -> main` (não verificado via `gh`; binário `gh` ausente no ambiente — marcado UNVERIFIED).

### Working tree (preservar, não destruir)

Modificados (não commitados):
- `DEV/HANDOFF.md`, `DEV/VERIFY.md`, `DEV/WORKLOG.md` (apenas docs de trabalho 2026-09-16 sobre guard de teclado Codex)
- `internal/app/control_cmd.go` (+43/-3 aprox): `containsTerminalInterrupt` + `forwardAttachedInput` byte-transparente + reassert de modos de teclado após Ctrl+C no attach direto e Lead.
- `internal/runtime/runtime.go` (+20/-1): reset agora inclui `\x1b[?2004l` (desliga bracketed paste) + `ReassertInteractiveTerminalKeyboardModes` (imediato + 75ms + 300ms) + reassert em SIGINT.
- `internal/runtime/runtime_test.go` (+7): novo `TestTerminalKeyboardModeResetDisablesBracketedPaste`.

Não commitado não rastreado:
- `?? internal/app/terminal_keyboard_guard_test.go` (conteúdo ainda não inspecionado nesta fase; preservar e cobrir em Fase 1/2).

Stash relevante (7 entradas, preservar):
- `stash@{0}` wip: conversation spike parked outside v1.0.0-beta.1
- `stash@{1}` frente-b-wip-before-oauth-fixture-fix
- `stash@{2}` pre-merge maximum-delivery working tree
- `stash@{3}` WIP e28e38c agent matching/classification
- `stash@{4}` preserve local work before documentation integration
- `stash@{5}` WIP 4b2b267 splash screen
- `stash@{6}` WIP 81ff0c2 engineering baseline

Estratégia: nenhum `checkout -- .`, nenhum `stash drop`, nenhum `reset --hard`. As mudanças de teclado serão validadas por testes e commitadas como fix pequeno e separado.

## 2. CI atual (lido do código, não de relatórios)

`.github/workflows/ci.yml` (746 linhas, lidas 1–120 + amostragem):
- `frontend` (ubuntu): Bun 1.3.9, `bun install --frozen-lockfile`, format:check, typecheck, lint, lint:styles, check:styles, unit tests, build, verify embedded bundle.
- `security`: Go 1.25.14, `govulncheck ./...`.
- `browser` (needs frontend): build binário Go + Playwright chromium + E2E/axe/visual.
- Jobs adicionais presentes no arquivo (a confirmar execução no SHA): Go tests, race, desktop/package por OS (leitura completa pendente na Fase 27).
- `.github/workflows/release.yml`: `sign-update-manifest`, `channel-pointer`, manifest assinado; `.goreleaser.yaml` v2: builds linux/darwin/windows amd64+arm64, archives tar.gz/zip, checksums, nfpms deb/rpm, `ProductionTrustRootHex` via `NEXUS_UPDATE_PUBLIC_KEY`.
- Status do CI no SHA `b3db75c`: **UNVERIFIED** neste ambiente (sem `gh`, sem rede GitHub consultada). Não se afirma verde/vermelho sem evidência. Fase 1/27 vai executar localmente o equivalente e registrar.

## 3. Mapa de módulos (verificado por leitura direta + investigators read-only)

| Domínio | Local | Estado real |
|---|---|---|
| Frontend | `web/package.json`, `web/bun.lock` (142KB, 2026-09-11) vs package 2026-09-15 | Risco de drift: package modificado após lock; `bun` ausente no ambiente (precisa instalar 1.3.9 e rodar `--frozen-lockfile` como teste). Scripts: prettier/eslint/stylelint/allowlist/typecheck/vitest/build/verify/e2e-hardening/axe/visual |
| Providers | `internal/core/provider/*`, `spec/*`, `adapters/{codex,claude,gemini,cursor,opencode,agy}` | Detect via `LookPath`+candidatos; `InspectAuth` por provider. Gaps: `spec` fixa só `HOME`, sem `USERPROFILE`; `internal/runtime/executable.go:51 splitPathList(getenv("PATH"), ";")` hardcoded — quebra Unix. `runtime.go:175,192` e `launchenv/env.go:65` usam `os.PathListSeparator` corretamente |
| PATH/HOME | `internal/runtime/runtime.go:89 LookPath`, `:135 EnhancedPATH`, `executable.go:32 ResolveCommand`, `testutil/fakebinary.go` existe | Helper existe; cobertura multiplataforma a verificar na Fase 2 |
| Attachments | `internal/control/web/handlers_conversation.go:91-228`, `store/composer.go:457-530`, `store.go:179-192`, `migrations/0021` | **P0 inseguro** (detalhe §5). Zero testes de handler/segurança |
| Auth/control plane | `internal/control/web/auth*.go`, `server.go:253,287,389,456,499,584`, `bind.go`, `originpolicy/`, `desktop_attach.go`, `exposure.go:32`, `handlers_tunnel.go`, `handlers_fs.go`, `handlers_git.go`, `handlers_api.go`, `handler_terminal.go` | Boas propriedades confirmadas: FS loopback-only + allowed-roots, tunnel => PUBLIC_REMOTE + limpa sessões persistidas, WS query token só loopback sem tunnel, rotate atômico, device token 0600 + constant-time. Falhas/atenções §5 |
| Conductor | `internal/nexus/conductor*.go`, `runner/types.go:38-48`, `conductor_dispatch.go:41-44,50-161` | **P0 semântico**: `Dispatch` nunca retorna ACCEPTED; `waitPromotedMission` retorna `DispatchCompleted` para PAUSED/BLOCKED_NEEDS_USER; `IsTerminalState` exclui paused/blocked mas dispatch trata como completo. Sem rota HTTP (só CLI + promotion fire-and-forget); poll 500ms, sem stream |
| Store/migrations | `internal/nexus/store/store.go:22-326`, `migrations/0001-0021`, `flow_evidence.go`, `composer.go` | WAL+FK ligados, checkpoint TRUNCATE (retorno ignorado). **P0 dados**: drift de `flow_context_capsules` renomeia para `_legacy_*` e copia só `flow_work_receipts` — capsules viram invisíveis; `INSERT OR IGNORE` pode pular linha colidida silenciosamente; só newest legacy migrado; `conversation_attachments` sem validação de schema pré-existente |
| IDs | `internal/control/ids/ids.go`, `store/*` (`evt_`,`att_`,`prj_`,`agt_`,`mis_`,`plan_` + ULID), `conductor_dispatch.go:61` | Store usa ULID com `rand.Read` cripto. **Exceção**: `GoalID = goal_%d UnixNano` no Dispatch — colisão/debug ruim |
| Telemetry | `internal/core/telemetry/telemetry.go:45,132`, `handlers_telemetry.go`, `routes.go:94` | Só local JSONL + agregação; sem export/opt-out/product analytics; sem redaction auditada aqui |
| Entitlements/device | `machine.go:17-19` (`/etc/machine-id`, fallback hostname hash), `launcher.go:156` | **Zero** entitlement/license enforcement (só quota). Machine ID Linux-only |
| Updater/registry | `internal/update/{service,manifest,keyring,registry,installation}.go`, `scripts/sign-update-manifest`, `install.sh:9,192-194`, `install.ps1:20,75` | Ed25519 existe + downgrade rejeitado (`ErrManifestDowngrade`) + install.sh verifica assinatura fail-closed. Gap arquitetural: `registry.go:12,38` hardcoded GitHub (`DefaultUpdateTrustRepo` + URL builder) — sem interface `ArtifactRegistry`; cliente acoplado ao repo source |
| Release/supply chain | `release.yml`, `.goreleaser.yaml`, `packaging/{linux,windows,winget}`, `SBOM` | deb/rpm via nfpms, NSIS `packaging/windows/nexus.nsi`, winget manifest; SBOM/provenance/vuln scan a verificar (govulncheck no CI; frontend audit pendente) |
| Desktop/Wails | `cmd/nexus-desktop/main.go:87`, `web/src/wailsjs/*`, `web/src/platform/*` | Bootstrap repassa SessionToken+CSRF ao Wails; Origin reescrito no backend loopback — precisa teste adversarial Fase 4 |
| i18n/frontend prontidão | `web/src/features/conversation/*`, `ComposerSurface`, locales pt-BR/en/es | Hardening recente de Composer; auditoria de produto (empty/offline/loading/a11y/teclado/responsivo) pendente Fase 23 |
| Licença/legal | `LICENSE` = MIT (AI Manager Contributors), sem `docs/legal/` | **P1 comercial**: ainda open-source; sem LICENSE proprietária, sem BETA_TERMS/PRIVACY/THIRD_PARTY; sem auditoria de contribuidores externos |
| Docs source-dist | `README.md`, `install.sh/ps1`, `docs/` | Caminho atual ainda centrado em source/GitHub; desacoplamento pendente Fases 10–11 |

## 4. Testes quebrados / a executar (nada afirmado sem rodar)

- Go: não executado nesta fase. Alvos Fase 1: `go vet ./...`, `go test ./...`, `-race` nos pacotes críticos, `golangci-lint`.
- Frontend: não executado (sem Bun no PATH). Alvos: `bun install --frozen-lockfile`, format:check, lint, lint:styles, check:styles, typecheck, vitest, build, verify, Playwright/axe/visual.
- E2E desktop/instaladores (Linux/Windows/macOS), update E2E, attachment adversarial, migration upgrades, provider contracts: inexistentes ou parciais — a construir nas Fases 2–6, 26–27.
- Histórico `DEV/validation/*` contém dezenas de relatórios GO/RC1 (ex.: `FINAL_RC1_REPORT.md`, `RELEASE_CANDIDATE_v1.0.0-beta.1.md`); **explicitamente desconsiderados** como evidência.

## 5. Riscos

### P0 (bloqueiam RC)
1. **Attachments inseguros** (`handlers_conversation.go:115-228`): confia em `Content-Type` do cliente (`:127`), sem `MaxBytesReader`, sem sniff/magic/decode/reencode, `sessionID` da URL em `Join+MkdirAll` (`:149-150`, traversal), linha DB criada antes dos bytes (`:160`, órfão), `StorageRef` absoluto exposto/servido via `ServeFile` (`:217`, polyglot inline), sem check de ownership (qualquer ID autenticado lê/apaga), sem dimensão/bomb-check. Módulos: `control/web`, `nexus/store`, `core/config`.
2. **Conductor mente estado terminal** (`conductor_dispatch.go:138,152`): PAUSED/BLOCKED_NEEDS_USER retornam COMPLETED; ACCEPTED definido e nunca retornado; `GoalID` UnixNano (`:61`); HTTP sem 202+run_id/stream. Módulos: `nexus`, `runner`, `app`, `control/web/promotion.go`.
3. **Perda silenciosa em repair** (`store.go:222-259`): capsules com drift vão para `_legacy_*` sem migração de conteúdo; `INSERT OR IGNORE` engole colisão; só newest legacy migrado; `wal_checkpoint` sem checagem. Módulos: `nexus/store`.
4. **PATH Windows-only hardcoded** (`runtime/executable.go:51`): `splitPathList(..., ";")` quebra Unix; specs sem `USERPROFILE`. Módulos: `runtime`, `core/provider/spec`.
5. **Superfície de controle a testar adversarialmente**: `CheckOrigin` permite Origin vazio em não-WS (`auth.go:119`); bootstrap via POST sem origin obrigatório (`server.go:456`); `handleProjectShell` (`handlers_projects.go:186`) e `handleRuntimes` (`handlers_api.go:245`) lançam shell/args — precisam de prova de auth+CSRF+loopback; `handleProjectGitCheckout` sem `--` (`handlers_git.go:113`). Módulos: `control/web`.

### P1 (bloqueiam beta comercial)
6. Updater acoplado ao GitHub source (`update/registry.go`, `install.sh/ps1`, `release.yml`); sem abstração `ArtifactRegistry`/channel manifest assinado por domínio próprio.
7. Sem `docs/legal/` (LICENSE ainda MIT), sem inventário de licenças/SBOM publicado, sem auditoria de contribuidores externos para relicenciamento.
8. Sem `EntitlementService`/offline entitlements assinados/device `installation_id` privacy-safe; sem `ProductAnalytics` opt-in/offline-resiliente; sem crash-report boundary com redaction.
9. Instaladores: NSIS/deb/rpm/winget existem mas sem prova atual fresh-install/upgrade/uninstall; sem Authenticode (Windows) nem notarization (macOS) — gates externos prováveis.
10. Frontend lock drift (package 2026-09-15 vs lock 2026-09-11) pode quebrar `--frozen-lockfile`.

### P2 (endurecer antes de escalar)
11. `handleAttachmentByID` DELETE remove arquivo antes do DB sem transação; GET lista expõe `StorageRef` absoluto.
12. Polling 500ms sem backoff/stream; `On DELETE CASCADE` apaga evidência/attachments junto ao pai.
13. `machine.go` usa `/etc/machine-id` + fallback hostname — revisar para `installation_id` aleatório.
14. `install.sh NEXUS_PUBKEY` vs trust root canônico (`ProductionTrustRootHex`) — unificar fonte de verdade.
15. `DEV/validation/*` e artefatos transitórios (`.db-wal/.db-shm`, logs, coverage, screenshots) poluindo repo — cleanup Fase 22.

## 6. Estratégia de resolução (ordem)

1. **Fase 1**: instalar Bun 1.3.9, `bun install --frozen-lockfile` (regenerar lock via Bun se drift real), rodar toda a qualidade frontend + Go vet/tests/race; validar working-tree de teclado e commitá-lo separado.
2. **Fase 2**: corrigir `splitPathList` multiplataforma + `USERPROFILE` nos specs; hermetizar provider tests (PATH vazio/fake, HOME/USERPROFILE isolados, instalado×autenticado×healthy×quota).
3. **Fase 3**: reescrever upload/serve de attachments conforme checklist da missão (MaxBytesReader, limites, ID cripto, sniff+decode+reencode PNG/JPEG/WebP, MIME servidor, nosniff, Content-Disposition, storage fora de public, cleanup transacional) + testes negativos + via tunnel.
4. **Fase 4**: testes adversariais do control plane sem quebrar boas propriedades existentes.
5. **Fase 5**: estados ACCEPTED/RUNNING/BLOCKED/PAUSED/COMPLETED/FAILED/CANCELED/REJECTED; Dispatch nunca COMPLETED sem conclusão real; 202+run_id para Web.
6. **Fase 6**: repairs sem perda (migrar capsules ou abortar com erro + ledger; checar checkpoint; fixtures `store/testdata/upgrades/`).
7. **Fases 7–8, 12–14, 18–21**: UUIDv7/ULID em Goal/Run; ProductAnalytics opt-in; EntitlementService + offline doc assinado + installation_id; keyring com rotação + downgrade + SBOM/provenance.
8. **Fases 9–11, 15–17, 22–38**: licença proprietária + blockers jurídicos, distribuição sem source via registry abstraído, instaladores com gates externos explícitos, cleanup, performance/resiliência/update-E2E, CI tri-OS, proteção de branch, workflow de release, docs de cliente + beta playbook + flags, coverage e FINAL_REPORT com veredito honesto.

## 7. Evidência desta fase (reproduzível)

- `git branch --show-current` → `feat/nexus-maximum-delivery`
- `git rev-parse HEAD` → `b3db75c7756fe25a70f4dc49760c90c38af735a9`
- `git rev-parse main` → `f71eb515278168e33d626fd631cd89dfb5e58faf`
- `git log --oneline main..HEAD | wc -l` → `146`
- `git status --short --branch` → 6 modificados + 1 untracked (listados §1)
- `git stash list` → 7 entradas (listadas §1)
- `cat VERSION` → `1.0.0-beta.1`
- Leituras: `.github/workflows/ci.yml:1-120`, `web/package.json`, `handlers_conversation.go:91-228`, `conductor_dispatch.go:1-161`, `runtime/executable.go:1-80`, `.goreleaser.yaml:1-60`, `LICENSE:1-30`.
