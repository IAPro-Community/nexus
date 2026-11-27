# 02 — Auditoria de Performance, Benchmarks Reproduzíveis e Usabilidade Diária

> **Hardware e ambiente de referência medidos (`2026-11-27`):** Linux `6.17.0-23-generic`, AMD EPYC 7313 (8 vCPUs expostos em `/proc/cpuinfo`), 60 GiB RAM, `release/v1.0.0`, sem agentes externos incluídos no baseline do Nexus.

---

## 1. Principais Gargalos Encontrados na Auditoria

1. **Tempestade de subprocessos `--version` sem cache nem coalescência (`GET /api/v1/providers` = `2,299 ms`; `GET /api/v1/resources` = `1,795 ms`):**
   - `internal/core/provider/versionprobe/versionprobe.go` não tinha cache nem `singleflight`.
   - Os 4 adapters (`claude`, `cursor`, `opencode`, `zcode`) e **todos os 6 `ControlDriver`s** (`internal/control/driver/*_driver.go`) e `PreviewHarnessDriver` duplicavam chamadas diretas `RunCommandCapture(..., "--version")`.
   - Pior: `GET /api/v1/providers` chamava `drv.Detect()` **e** `drv.EffectiveCaps()` (que chamava `drv.Detect()` pela segunda vez!), enquanto `useNexusData.ts` disparava `/api/v1/providers` a cada 30 segundos!
2. **Sonda `cursor-agent status` sem cache dentro de `DetectHostAuth` (`~780 ms/req`):**
   - Detectada via `strace -f -e trace=execve` durante o benchmark de Go: quando não existia `~/.cursor/cli-config.json` no host, cada requisição a `/api/v1/providers` ainda disparava `cursor-agent status` (Node.js CLI de ~750–800 ms).
3. **`QuotaMonitorService.check()` a cada 60s sem filtrar provedores nem snapshots frescos (`internal/nexus/quota_monitor_service.go`):**
   - Iterava todos os perfis a cada minuto, incluindo provedores `QuotaUnsupported` (gerando alertas falsos de degradação e chamadas extras) e perfis ociosos cujo snapshot ainda estava dentro do TTL padrão (`quota.DefaultTTL = 5m`).
4. **Polling de statusbar Git executando 5 comandos `git` a cada 10s (`WorkspaceTaskbar.tsx` + `handlers_git.go`):**
   - Só para exibir o nome do branch na barra inferior, `WorkspaceTaskbar.tsx` chamava `GET /api/v1/projects/:id/git/branches` a cada 10s, disparando `git rev-parse`, `branch --list`, `branch -r`, `status --porcelain` e `worktree list` continuamente.
5. **Arquitetura Dual WebKitGTK no Desktop + HUD Linux (`1,045–1,358 MB RSS` em idle):**
   - `nexus web` (Go Core) consome apenas **`~36–42 MB RSS`**.
   - Entretanto, `nexus-desktop` (`~524–654 MB` entre Go/GTK, `WebKitNetworkProcess` e `WebKitWebProcess`) + `nexus-hud --linked` (`~485–662 MB`) instanciam **duas pilhas WebKitGTK 4.1 independentes**, agravadas por renderização por software no ambiente (`VMware: No 3D enabled` / falha `ZINK`).
6. **Proliferação de timers e disco:**
   - Foram identificados `27` `setInterval` distribuídos em `web/src/`;
   - `~/.local/share/ai-manager/profiles` acumula **`48 GB`** sem política de limpeza de caches antigos;
   - `~/.local/share/ai-manager/web-auth/` acumula dezenas de `listen-<pid>.json` de processos já encerrados (`28` arquivos observados no host).

---

## 2. Benchmarks Reproduzíveis

### Como reproduzir
1. **Microbenchmarks Go (CPU / alocação / latência em handler HTTP e probe):**
   ```bash
   GOCACHE=/tmp/nexus-gocache-clean NEXUS_REAL_HOME="" \
     go test -tags=gtk3 -run='^$' -bench=. -benchmem \
     ./internal/core/provider/versionprobe ./internal/control/web
   ```
2. **Benchmark Ao Vivo no Host (`CLI + HTTP API + Árvore de Processos RSS/CPU`):**
   ```bash
   python3 scripts/benchmark-nexus-evolution.py --runs 5 --cli-runs 10 --json
   ```
   O script [benchmark-nexus-evolution.py](file:///projetos/nexus/scripts/benchmark-nexus-evolution.py) utiliza apenas stdlib Python 3, lê automaticamente o token de sessão loopback em `~/.local/share/ai-manager/web-auth/sessions.json` e separa **estritamente** os processos do Nexus (`nexus_core`, `nexus_desktop`, `nexus_hud`) dos helpers de sessões externas (`__control-host`, `__browser-mcp-stdio`).

---

## 3. Resultados Reais Medidos: ANTES vs DEPOIS

### 3.1. Benchmark Ao Vivo no Host (`4c381001` ANTES vs `11e5cfba` DEPOIS)

| Medição / Endpoint | ANTES (`4c381001`) | DEPOIS (`11e5cfba`) | Melhoria Medida |
| --- | --- | --- | --- |
| **CLI Cold Start (`nexus --version`, 10 runs)** | avg `14.88 ms` (min `7.62`, p95 `24.41 ms`) | avg **`7.12 ms`** (min `6.32`, p95 **`8.26 ms`**) | **2.1x avg / 3.0x p95** |
| **`GET /api/v1/providers`** | avg **`2,299.34 ms`** (p95 `2,393.89 ms`) | avg **`4.23 ms`** (p95 **`4.54 ms`**) | **543.6x mais rápido** |
| **`GET /api/v1/resources`** | avg **`1,795.04 ms`** (p95 `1,812.47 ms`) | avg **`7.67 ms`** (p95 **`9.24 ms`**) | **234.0x mais rápido** |
| **`GET /api/v1/hud`** | avg **`330.30 ms`** (max `988.92 ms`) | avg **`0.51 ms`** (p95 **`0.64 ms`**) | **647.6x mais rápido** |
| **`GET /api/v1/projects/:id/git/branches?summary=1`** | avg `38.93 ms` (`8,456 B`, 5 procs `git`) | avg **`0.49 ms`** (`261 B`, **0 subprocessos**) | **79.4x mais rápido**, **-96.9% payload** |
| **`GET /api/v1/projects/:id/git/branches` (completo)** | avg `49.98 ms` | avg `26.11 ms` | Chamado só ao abrir modal |
| **`GET /api/v1/runtimes`** | avg `1.45 ms` | avg `1.11 ms` | **1.3x** |
| **`GET /api/v1/projects` / `events` / `profiles` / `health`** | `0.36–1.01 ms` | `0.32–1.18 ms` | Sub-2 ms mantido |
| **CPU `nexus web` em idle (janela real 3s `/proc/stat`)** | `2.0%` CPU contínuo observado via `ps` | **`0.00%` CPU** (janela idle 3s em `nexus`, `desktop` e `hud`) | **Eliminação de spikes em idle** |
| **Memória `nexus_core` (`nexus web`)** | `41.79 MB RSS` | **`35.81–38.41 MB RSS`** | **~4–6 MB menor** |
| **Memória `nexus_desktop` (3 procs Wails+WebKitGTK)** | `653.50 MB RSS` (~9h42m uptime) | `524.35 MB RSS` (recém-iniciado, estabiliza ~600–650 MB) | Limite estrutural WebKitGTK |
| **Memória `nexus_hud` (Go+GTK + WebKitGTK)** | `662.42 MB RSS` | `481.91–484.49 MB RSS` (recém-iniciado) | Limite estrutural WebKitGTK |
| **Total Nexus em Idle (sem agentes externos)** | **`1,357.71 MB RSS`** | **`1,044.65 MB RSS`** (recém-iniciado) | **~313 MB menor logo após boot**, mas ainda acima de 250 MB |
| **Helpers de sessão (fora do idle Nexus)** | `146.34 MB` (6 procs `__control-host`/`__browser-mcp`) | `142.86 MB` (6 procs) | Separado conforme regra |

### 3.2. Microbenchmarks Go (`testing.B`, AMD EPYC 7313)

| Benchmark | Resultado | Alocações | Observação |
| --- | --- | --- | --- |
| `BenchmarkDetectCached-8` | **`2,168 ns/op` (`2.17 µs`)** | `320 B/op`, `4 allocs/op` | Cache com fingerprint `(mtime, size)` |
| `BenchmarkHandleProvidersCached-8` (somente `versionprobe` antes do fix Cursor) | `781.8–1,974 ms/op` | `154 KB–370 KB` | Diagnóstico via `strace` isolou `cursor-agent status` |
| **`BenchmarkHandleProvidersCached-8` (com cache `cursorHostAuth` em `11e5cfba`)** | **`365,893 ns/op` (`0.366 ms`)** | `130,319 B/op`, `939 allocs/op` | **~5,395x mais rápido** que `1,974 ms/op` |
| `BenchmarkHandleProjectGitBranchesSummary-8` (`?summary=1`) | **`64,534 ns/op` (`0.065 ms`)** | `11,745 B/op`, `117 allocs/op` | Leitura direta `.git/HEAD` (suporta worktrees) |
| `BenchmarkHandleProjectGitBranchesFull-8` | `6,127,914 ns/op` (`6.13 ms`) | `237,068 B/op`, `576 allocs/op` | **95x mais lento** que `?summary=1` |

---

## 4. Avaliação Honesta das Metas de Engenharia

| Meta Inicial Proposta | Status Atual (`11e5cfba`) | Avaliação / Meta Realista & Custo |
| --- | --- | --- |
| **Interface interativa `<= 2s`** | **Atingida** no browser/Web Core; CLI `7.1 ms`, rotas API `< 8 ms` | Wails WebView fria depende do boot do WebKitGTK (~1–2s); manter rotas rápidas garante interatividade imediata após WebView abrir |
| **Navegação entre telas `<= 100 ms` p95** | **Atingida** para dados locais (`/providers` p95 `4.5 ms`, `/resources` p95 `9.2 ms`) | Rotas React usam chunks `Suspense` já compilados |
| **CPU em idle `< 1%` total da máquina** | **Atingida** em medição limpa de 3s (**`0.00%` CPU** em `nexus`, `desktop`, `hud`) | Quotas ociosas e probes redundantes removidos evitam picos periódicos |
| **Memória em idle `<= 250 MB` (excluindo agentes)** | **Atingida no Go Core (`36–38 MB`)**, **NÃO atingida no combo Desktop+HUD (`1.04–1.35 GB`)** | **Por quê:** 2 processos `WebKitGTK 4.1` completos no Linux (~250–420 MB `WebKitWebProcess` + ~48 MB `WebKitNetworkProcess` + ~195 MB Go/GTK cada). **Meta realista P1:** `<= 450 MB` eliminando o WebView permanente do `nexus-hud` quando recolhido (tray GTK nativo `< 25 MB` + WebView on-demand). **Meta P2 (`<= 250 MB`):** exige rodar a UI principal em browser leve ou shell WebView único compartilhado / avaliação Tauri |
| **Suporte a >= 20 sessões e 5/10/20 agentes** | `nexus web` aguenta dezenas de sessões em `< 50 MB`; cada sessão PTY custa `~28 MB` (`__control-host`) + CLI externo | Limite real está em montar múltiplos `<XTerm>` pesados no DOM ao mesmo tempo -> mitigar com lazy mount / skip-render em P1 |

---

## 5. Auditoria de Usabilidade e Experiência Diária (DX)

### Fluxos com Atrito Atual e Melhorias Recomendadas
1. **Início de Dia ("Abrir e retomar"):**
   - Hoje o desenvolvedor vê projeto + taskbar + abas, mas não tem um resumo imediato consolidado do que ficou bloqueado, quais branches/worktrees estão sujas e qual conta tem folga hoje.
   - *Melhoria (P1-5):* "Start-of-Day Card" não intrusivo no Overview com ações 1-clique (`Retomar sessão`, `Responder bloqueio`, `Limpar worktrees antigas`).
2. **Troca rápida via teclado:**
   - *Já melhorado em P1-1:* Command Palette (`Cmd/Ctrl+K`) agora expõe `Trocar branch Git / worktree` e `Atualizar provedores, contas e sessões` diretamente pelo teclado em PT-BR/EN/ES.
3. **Criação de agente paralelo em worktree:**
   - Hoje requer criar worktree/branch separado ou configurar manualmente no modal.
   - *Melhoria (P1-3):* Ação única na paleta e Overview que cria branch + worktree, escolhe conta saudável e abre o agente em 1 etapa.
4. **Progressive Disclosure na navegação:**
   - Ocultar `legacy-runtimes`, `legacy-providers` e `legacy-events` em grupo "Diagnóstico" para que usuários novos foquem em Overview, Terminals, Resources e Outputs.
