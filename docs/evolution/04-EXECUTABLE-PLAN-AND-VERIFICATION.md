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

### E5 — Redução Estrutural de RAM Desktop+HUD (`P1-2` / `P2-4`) *(Planejado)*
- **Problema:** Dual WebKitGTK consome `~1.04–1.35 GB RSS`.
- **Solução proposta:** Substituir janela WebKitGTK residente do HUD recolhido por indicador/tray nativo GTK (`< 25 MB`), abrindo WebView sob demanda; implementar skip-render/lazy-mount para abas `xterm.js` ocultas (referência Herdr/Terax).
- **Critério de aceitação:** Idle completo `nexus-desktop` + HUD recolhido `<= 450 MB RSS` em P1; `< 250 MB` em P2.
- **Rollback:** Flag `NEXUS_HUD_PERSISTENT_WEBVIEW=1` restaura comportamento anterior.

### E6 — Fluxo 1-Clique "Worktree + Agente + Diff Inline" (`P1-3` / `P1-4`) *(Planejado)*
- **Problema:** Criação paralela de worktree + revisão inline de diffs exige troca manual para IDE externa.
- **Referência competitiva:** Orca, Superset, Vibe Kanban, T3 Code.
- **Solução proposta:** Endpoint único `POST /api/v1/projects/:id/worktree-agents` (cria worktree, executa `.nexus/setup.sh` se autorizado, escolhe conta saudável do pool, inicia runtime) + visualizador de diff unificado leve com botão **"Comentar no Agente"**.
- **Riscos e Rollback:** Worktrees órfãs e hooks maliciosos -> exigir confirmação/hash por `.nexus/setup.sh` alterado e vincular limpeza ao ciclo de vida do agente.

### E7 — Account Pools ToS-Compliant + Automação `Event -> Condition -> Action -> Result` (`P2-1` / `P2-2`) *(Planejado)*
- **Escopo:** Políticas explícitas de pool + `burn_rate` no SDK e regras determinísticas locais com dry-run e rate-limit contra loops.

---

## 2. Registro Detalhado das Melhorias Implementadas (`release/v1.0.0`)

| Commit | Descrição | Arquivos Modificados / Criados |
| --- | --- | --- |
| `68613acb` | `perf(providers): cache and coalesce CLI version and host-auth probes` | [versionprobe.go](file:///projetos/nexus/internal/core/provider/versionprobe/versionprobe.go), [versionprobe_test.go](file:///projetos/nexus/internal/core/provider/versionprobe/versionprobe_test.go), [detect_helper.go](file:///projetos/nexus/internal/control/driver/detect_helper.go), 7 drivers em `internal/control/driver/`, 4 adapters (`claude`, `cursor`, `opencode`, `zcode`), [cursor_test.go](file:///projetos/nexus/internal/core/provider/adapters/cursor/cursor_test.go), [benchmark_evolution_test.go](file:///projetos/nexus/internal/control/web/benchmark_evolution_test.go) |
| `8902f745` | `perf(quota): skip unsupported providers and fresh idle snapshots in monitor` | [quota_monitor_service.go](file:///projetos/nexus/internal/nexus/quota_monitor_service.go), [quota_monitor_service_test.go](file:///projetos/nexus/internal/nexus/quota_monitor_service_test.go) |
| `92bddaaf` | `perf(git): add fast HEAD summary endpoint and reduce statusbar polling` | [handlers_git.go](file:///projetos/nexus/internal/control/web/handlers_git.go), [handlers_git_test.go](file:///projetos/nexus/internal/control/web/handlers_git_test.go), [api.ts](file:///projetos/nexus/web/src/nexus/api.ts), [WorkspaceTaskbar.tsx](file:///projetos/nexus/web/src/workspace/WorkspaceTaskbar.tsx) |
| `926fb01c` | `perf(web): decouple provider catalog polling and add keyboard palette actions` | [useNexusData.ts](file:///projetos/nexus/web/src/app/useNexusData.ts), [useNexusData.test.tsx](file:///projetos/nexus/web/src/app/useNexusData.test.tsx), [NexusWorkspaceSession.tsx](file:///projetos/nexus/web/src/app/NexusWorkspaceSession.tsx), locales `pt-BR.ts` / `en.ts` / `es.ts` |
| `11e5cfba` | `chore(bench): add reproducible Nexus evolution benchmark script` | [benchmark-nexus-evolution.py](file:///projetos/nexus/scripts/benchmark-nexus-evolution.py) |

*Nota sobre `featureAnnouncements.ts` (`AGENTS.md`):* As alterações realizadas no código otimizam cache/polling internos e adicionam atalhos na Command Palette para superfícies já existentes (Branch Switcher e Refresh Global), sem criar uma nova área ou superfície visível anunciável — portanto `featureAnnouncements.ts` permanece inalterado de forma justificada.

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
