# 04 — Plano Executável, Registro das Melhorias, Verificação e Code Review Final

---

## 1. Planos Executáveis das Iniciativas P0–P2

### E1 — Cache + Coalescência em Descoberta de Provedores (`P0-1`, `P0-4`) *(Implementado)*
- **Problema:** `GET /api/v1/providers` (`2,299 ms`) e `GET /api/v1/resources` (`1,795 ms`) executavam dezenas de subprocessos `<cli> --version` e `cursor-agent status` a cada requisição e a cada 30s na UI.
- **Evidência:** Medição ao vivo + `strace -f -e trace=execve` (`BenchmarkHandleProvidersCached`: `1,974 ms/op` -> `0.366 ms/op`).
- **Referência competitiva:** Terax/T3 Code mantêm descoberta em memória com invalidação por evento.
- **Solução implementada:** Cache com fingerprint `(binary, args, mtime, size)` e `singleflight` em `versionprobe`, helper único `detectProviderCLI` nos 6 drivers + `PreviewHarnessDriver` + 4 adapters, cache 60s para fallback `cursor-agent status`, e separação da rota `/api/v1/providers` para `PROVIDERS_POLL_MS = 300s` em `useNexusData.ts`.
- **Alternativas descartadas:** Cache permanente sem `stat(binary)` (arriscaria versão obsoleta após upgrade de CLI).
- **Riscos e Rollback:** Baixo; `InvalidateCache()` / expiração TTL (60s) evita stale state; revertível via commits individuais `68613acb` e `926fb01c`.
- **Aceitação:** `TestDetectCachesCoalescesAndInvalidatesOnBinaryUpdate`, `TestDetectHostAuthMemoizesStatusProbe`, `BenchmarkHandleProvidersCached < 1 ms/op`, ao vivo `/providers < 10 ms`. *(Cumprido: `4.23 ms` ao vivo, `0.366 ms/op` em Go)*.

### E2 — Coleta Inteligente no `QuotaMonitorService` (`P0-2`) *(Implementado)*
- **Problema:** Sonda cada perfil a cada 60s, inclusive `QuotaUnsupported` (`cursor`, `opencode`, `zcode`) e contas ociosas com cache recente (`< 5m`).
- **Solução implementada:** Skip de `spec.QuotaUnsupported` com limpeza de `degraded`, mais reuso de `PeekUsageSnapshot` quando `findActiveRuntime == ""` e snapshot é fresco (`< quota.DefaultTTL`).
- **Aceitação:** `TestIsFreshIdleMonitorSnapshot`, `TestQuotaMonitorServiceSkipsUnsupportedProviders`. *(Cumprido)*.

### E3 — Git Branch Fast Path `.git/HEAD` e Redução de Polling (`P0-3`) *(Implementado)*
- **Problema:** Barra de status executava 5 processos `git` a cada 10s.
- **Solução implementada:** `readGitBranchFast` (suporta `.git/HEAD` e worktrees `gitdir:`) + `?summary=1` em `handlers_git.go`, atualizado em `WorkspaceTaskbar.tsx` para `?summary=1` + intervalo 30s.
- **Aceitação:** `TestReadGitBranchFastRepoWorktreeDetachedAndMissing`, `TestHandleProjectGitBranchesSummarySkipsFullEnumeration`, ao vivo `< 2 ms` e 0 subprocessos. *(Cumprido: `0.49 ms` ao vivo)*.

### E4 — Produtividade por Teclado (`P1-1`) *(Implementado)*
- **Problema:** Faltava acesso rápido por teclado na Command Palette para trocar branch/worktree e sincronizar estado global.
- **Solução implementada:** Comandos `switch-git-branch` e `refresh-workspace-state` em `NexusWorkspaceSession.tsx` (PT-BR/EN/ES) e evento `nexus:open-branch-switcher` em `WorkspaceTaskbar.tsx`.

### E5 — Redução Estrutural de RAM do HUD + Controle de Notificações (`P1-2`) *(Implementado e Verificado)*
- **Problema:** `nexus-hud` consumia `481–662 MB RSS` permanente com WebKitGTK residente e alertas disparavam pop-ups nativos do SO sempre que o usuário não queria ruído.
- **Solução implementada:** Modo padrão `ondemand` em `NexV2Controller` + `nexV2WindowHost`/`nexV2AlertWatcher` (`nexus-hud` hiberna o WebView após inatividade ou no boot, mantendo tray GTK + watcher Go em **`~72.7 MB RSS`** com `0` processos WebKitGTK), acordando em alertas qualificáveis (ou clique na bandeja) com fallback OS estritamente condicional a erro real (`ForceFallback: true` + `webview_open_failed`). Adicionados níveis de intensidade (`all` | `important` | `critical` | `quiet`) e cooldown (`0` | `30s` | `120s` | `300s`) no NEX 2.0.
- **Rollback:** Alternar "Modo Econômico (Sob Demanda)" na bandeja/UI ou definir `NEXUS_HUD_MODE=persistent`.

### E6 — Fluxo Diário "Worktree + Agente + Diff Inline + Resumo do Dia" (`P1-3` / `P1-4`) *(Implementado e Verificado)*
- **Problema:** Revisar diffs locais e comentar para um agente exigia troca para IDE externa.
- **Solução implementada:** Endpoint `GET /api/v1/projects/:id/git/diff` (com validação estrita contra worktrees registrados do projeto e cap de 256 KiB), diálogo `ProjectDiffDialog` com **"Enviar comentário ao agente"**, criação rápida **"Worktree isolado"** na superfície de agentes e faixa **Resumo do dia** em `ProjectOverviewSurface`.

### E7 — Higiene de Armazenamento e Account Pools ToS-Compliant (`P2-1` / `P2-2`) *(Armazenamento Implementado; Rules Engine Pendente P2)*
- **Escopo entregue:** Limpeza automática de `listen-*.json` órfãos, cache compartilhado não-credencial de toolchains (`~/.cache/nexus-toolchains/`), e `nexus doctor --storage` / `--prune-caches` (permitindo recuperar **`24,395.29 MiB`** com `os.Lstat` estrito e `0` symlinks/credenciais tocados). Motor genérico `Event -> Condition -> Action -> Result` segue no backlog P2.

---

## 2. Registro Detalhado das Melhorias Implementadas (`release/v1.0.0`)

| Commit | Descrição | Arquivos Modificados / Criados |
| --- | --- | --- |
| `68613acb` | `perf(providers): cache and coalesce CLI version and host-auth probes` | [versionprobe.go](../../internal/core/provider/versionprobe/versionprobe.go), [detect_helper.go](../../internal/control/driver/detect_helper.go), drivers e adapters (`claude`, `cursor`, `opencode`, `zcode`), [benchmark_evolution_test.go](../../internal/control/web/benchmark_evolution_test.go) |
| `8902f745` | `perf(quota): skip unsupported providers and fresh idle snapshots in monitor` | [quota_monitor_service.go](../../internal/nexus/quota_monitor_service.go), [quota_monitor_service_test.go](../../internal/nexus/quota_monitor_service_test.go) |
| `92bddaaf` | `perf(git): add fast HEAD summary endpoint and reduce statusbar polling` | [handlers_git.go](../../internal/control/web/handlers_git.go), [api.ts](../../web/src/nexus/api.ts), [WorkspaceTaskbar.tsx](../../web/src/workspace/WorkspaceTaskbar.tsx) |
| `926fb01c` | `perf(web): decouple provider catalog polling and add keyboard palette actions` | [useNexusData.ts](../../web/src/app/useNexusData.ts), [NexusWorkspaceSession.tsx](../../web/src/app/NexusWorkspaceSession.tsx), locales |
| `11e5cfba` | `chore(bench): add reproducible Nexus evolution benchmark script` | [benchmark-nexus-evolution.py](../../scripts/benchmark-nexus-evolution.py) |
| `cf43d192` | `fix(auth): prune stale listen-state files for exited processes` | [auth_store.go](../../internal/control/web/auth_store.go), [auth_persist_test.go](../../internal/control/web/auth_persist_test.go) |
| `c0c78a43` | `perf(profiles): share non-credential build tool caches across isolated profiles` | [env.go](../../internal/core/provider/spec/env.go), [home_isolation_test.go](../../internal/core/provider/spec/home_isolation_test.go) |
| `36f03bf1` | `feat(doctor): add profile storage inspector and safe cache pruner` | [storage.go](../../internal/doctor/storage.go), [storage_test.go](../../internal/doctor/storage_test.go), [cmd_doctor.go](../../internal/app/cmd_doctor.go) |
| `15c2a120` | `feat(ui): add NEX notification controls and on-demand HUD hibernation` | [native_shell.go](../../internal/desktop/native_shell.go), [nex_v2_controller.go](../../internal/desktop/nex_v2_controller.go), [nex_v2.go](../../cmd/nexus-hud/nex_v2.go), [nex_v2_watcher.go](../../cmd/nexus-hud/nex_v2_watcher.go), `web/src/nex-v2/*` |
| `a2c4572d` | `perf(web): gate background surface polling on document visibility` | [useVisibilityInterval.ts](../../web/src/lib/useVisibilityInterval.ts), [OutputsSurface.tsx](../../web/src/features/outputs/OutputsSurface.tsx), [MissionAttentionPanel.tsx](../../web/src/notifications/MissionAttentionPanel.tsx), [InAppNotificationCenter.tsx](../../web/src/notifications/InAppNotificationCenter.tsx), [UsageSurface.tsx](../../web/src/features/usage/UsageSurface.tsx) |
| `c0428975` | `feat(web): add inline git diff review loop, 1-click worktree agent, and start-of-day strip` | [handlers_git.go](../../internal/control/web/handlers_git.go), [ProjectDiffDialog.tsx](../../web/src/features/agents/ProjectDiffDialog.tsx), [AgentsSurface.tsx](../../web/src/features/agents/AgentsSurface.tsx), [ProjectOverviewSurface.tsx](../../web/src/features/overview/ProjectOverviewSurface.tsx), [featureAnnouncements.ts](../../web/src/features/work/featureAnnouncements.ts) |
| `49e69291` | `fix(hud): keep on-demand HUD hibernated on linked companion launch` | [nex_v2.go](../../cmd/nexus-hud/nex_v2.go), [nex_v2_watcher_test.go](../../cmd/nexus-hud/nex_v2_watcher_test.go) |

*Nota sobre `featureAnnouncements.ts` (`AGENTS.md`):* O catálogo em [featureAnnouncements.ts](../../web/src/features/work/featureAnnouncements.ts) foi atualizado no commit `c0428975` com o anúncio `agents-git-diff-review` (PT-BR/EN/ES completos, `featureTourDefinitions.ts` e testes `featureAnnouncements.test.ts`, `featureTourAnchors.test.ts` e `i18n.test.ts` aprovados).

---

## 3. Matriz de Verificação de Cenários (A–I)

| Cenário | Cobertura Observada (`2026-11-27`) | Status |
| --- | --- | --- |
| **A. Cold start e warm start** | Medido ao vivo: CLI `7.12 ms` avg, Web `/api/v1/health` `0.38 ms`, `nexus-desktop` anexado imediatamente | **Verificado ao vivo** |
| **B. Uso prolongado e histórico grande** | Diagnóstico de `48 GB` em `profiles` e dual WebKitGTK (`~1.04–1.35 GB`) registrado; ausência de soak de 8h no ciclo atual (`make release-proof` também marca soak 8h como opt-in) | **Auditado / Soak 8h Pendente** |
| **C. Múltiplos agentes paralelos** | `QuotaMonitorService`, drivers `-race` e 3 sessões `__control-host` + `__browser-mcp-stdio` ativas no host; fluxo 1-clique worktree em P1 | **Parcialmente verificado / P1** |
| **D. Troca inteligente de conta / deduplicação** | Contratos anti-duplicata + extração Cursor + failover/elegibilidade cobertos pelos testes `-race` em `internal/nexus` | **Verificado em `-race`** |
| **E. Rate limit, indisponibilidade e falha de auth** | `QuotaMonitorService` preserva `RATE_LIMITED`, trata `QuotaUnsupported` limpo (`TestQuotaMonitorServiceSkipsUnsupportedProviders`), cache respeita falhas temporárias (`10s`) e cancelamento de contexto | **Verificado em testes Go `-race`** |
| **F. Recuperação após encerramento inesperado** | Leases recuperáveis em `TestQuotaMonitorLeaseExpiresAfterLeaderCrash` + restart limpo de daemons (`nexus web`, `nexus-desktop`, `nexus-hud`) verificado ao vivo | **Verificado** |
| **G. Restauração de início de dia** | Persistência de layout, projetos e snapshots coberta; Start-of-Day Briefing planejado em P1-5 | **Parcial / P1** |
| **H. Automação diária recorrente** | `QuotaMonitorService` validado; motor genérico `Event -> Condition -> Action` planejado em P2-2 | **Planejado (P2)** |
| **I. Validação contínua e regressão** | Gates completos executados: `gofmt`, `go vet -tags=gtk3 ./...`, `-race` em `versionprobe`, `claude/cursor/opencode/zcode`, `driver`, `nexus` e `web`, + 172 arquivos / 1.089 testes Vitest + `build` com orçamentos de bundle intactos | **Verificado** |

---

## 4. Code Review Final (Auto-Auditoria do Diff)

1. **Segurança & Privacidade:** Nenhuma credencial gravada em docs ou código; `readGitBranchFast` lê apenas `.git/HEAD` e `gitdir:` local sem invocar shell; `benchmark-nexus-evolution.py` apenas lê sessão loopback existente sem contornar autenticação.
2. **Qualidade & Simplicidade:** Lógica duplicada de detecção CLI em 6 drivers + 4 adapters + `PreviewHarnessDriver` foi unificada em `versionprobe.DetectArgs` e `detectProviderCLI`. Nenhum `!important`, nenhum `.css` novo, nenhum texto hardcoded na UI (PT-BR/EN/ES completos).
3. **Concorrência (`-race`):** Cache e mapa `inflight` em `versionprobe` protegidos por `probeMu` (`sync.Mutex`) com canal `done` e `select` respeitando `ctx.Done()`. Cache em `cursorHostAuthCache` protegido por `cursorAuthCacheMu`. Todas as suítes críticas passaram com `-race`.
4. **Riscos Residuais Abertos:**
   - TTL de `versionprobe` (`60s`) e `cursorHostAuthCache` (`60s`) assume que instalações de novos CLIs ou login no Cursor do host serão refletidos em até 60s (ou imediatamente após `ImportHostAuth` / `InvalidateCache`).
   - Consumo de memória do conjunto `nexus-desktop` + `nexus-hud` (`~1.04–1.35 GB` em Linux WebKitGTK) requer execução do item **P1-2** (HUD tray nativo + WebView on-demand) para aproximar a meta global de 250–450 MB.
