# 01 — Diagnóstico Atual do Nexus e Análise Competitiva

> **Data de consulta e verificação:** `2026-11-27`
>
> **Escopo:** Repositório **IAPro Nexus** (branch `release/v1.0.0`, commit base `4c381001`) comparado com ferramentas de mercado pesquisadas em fontes oficiais e repositórios abertos.

---

## 1. Realidade Arquitetural do Nexus (Validada no Código)

Ao contrário da hipótese inicial de um monorepo Nx/Node, o **IAPro Nexus** é um **sistema monolítico modular em Go + frontend React embutido via `go:embed`**:

| Camada | Tecnologia / Versão Real Verificada | Evidência no Repositório |
| --- | --- | --- |
| Backend / CLI / Daemon | Go `1.26.11` (`github.com/IAPro-Community/nexus`, `580` arquivos `.go` não-teste em `internal/`) | [go.mod](../../go.mod) |
| Frontend Web | React `19`, TypeScript `5.9`, React Router `7`, Zustand `5`, Tailwind CSS `v4` (`@tailwindcss/vite`), Vite `7`, esbuild | [web/package.json](../../web/package.json) |
| Estilização | SCSS Modules (`*.module.scss`) + CSS Custom Properties (`--nx-*`), sem `.css` local | [AGENTS.md](../../AGENTS.md) |
| Desktop & HUD Nativos | **Wails `v3.0.0-beta.25`** sobre WebKitGTK (`4.1` no Linux), tag `gtk3` | [go.mod](../../go.mod), [Makefile](../../Makefile) |
| Banco Local | SQLite (`modernc.org/sqlite`) em WAL mode (`busy_timeout=5000`, `synchronous=NORMAL`, `MaxOpenConns=1`) | `~/.local/share/ai-manager/nexus.db` |
| Contratos Compartilhados | `github.com/IAPro-Community/IAPro-SDK v0.0.0` (com `replace => ../IAPro-SDK`) | [go.mod](../../go.mod), [ADR-iapro-sdk-consumer.md](../architecture/ADR-iapro-sdk-consumer.md) |
| Runtime Autônomo Externo | **Orquestrador Maestro** como autoridade externa (governança/memória/ledger) | [MAESTRO_EXTERNAL_AUTHORITY_ADR.md](../architecture/MAESTRO_EXTERNAL_AUTHORITY_ADR.md) |

---

## 2. Diagnóstico Integral dos Módulos do Nexus

Classificação rigorosa com base em leitura direta do código e testes em execução (`2026-11-27`):

| Capacidade / Módulo | Classificação | Evidência Verificada no Código | Problema / Gargalo / Observação |
| --- | --- | --- | --- |
| **CLI `nexus` (roteamento, launch, perfil)** | Implementado e funcional | Binário Go único; cold start `--version` medido em **`~8–15 ms`** | Sobra clareza e velocidade; não há gargalo de cold start na CLI |
| **Controle PTY, TUI e `__control-host`** | Implementado e funcional | `internal/control/host`, `broker`, reconexão de runtime, `~28.5 MB RSS` por host ativo | Estável; `xterm.js` no frontend pode crescer se muitos terminais ficarem montados |
| **Gerenciamento de Perfis e Contas (`internal/profile`)** | Implementado e funcional | Diretórios isolados (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, etc.), dedup automática (`53956e84`, `ed891a0d`), provenance e bloqueio de duplicatas | Armazenamento local em `~/.local/share/ai-manager/profiles` atingiu **48 GB** no host sem política automática de retenção/limpeza de caches antigos |
| **Monitoramento de Quota & Codex App-Server Isolation** | Implementado, otimizado em P0 | `internal/nexus/quota_monitor_service.go`, `internal/core/quota`, lock TUI Codex (`IsTUILocked`) | **Problema encontrado:** `QuotaMonitorService.check()` varria inclusive providers `QuotaUnsupported` e perfis ociosos com cache fresco a cada 60s (corrigido em P0-2) |
| **Descoberta de Provedores (`GET /api/v1/providers`, `/resources`)** | Implementado, porém problemático | `internal/control/web/handlers_api.go`, `versionprobe.go`, `resource_discovery.go` | **Gargalo crítico medido:** `2,299 ms` em `/providers` e `1,795 ms` em `/resources` por subprocessos `<cli> --version` sem cache nem singleflight (corrigido em P0-1/P0-4) |
| **Notificações & Hub NEX** | Implementado e funcional | Hub primário em NEX com fallback automático (`94a044af`, `4c381001`) + dedup/episódios no `QuotaDropMonitor` | Evita flood de notificações nativas; ainda depende de `nexus-hud` separado em WebKitGTK |
| **NEX Mascote / HUD Desktop (`cmd/nexus-hud`)** | Implementado, Gate de Release `BLOCKED` | `docs/design/nex/RELEASE_READINESS.json` + WebKitGTK separado | **`662 MB RSS`** adicionais em idle; proibido publicar Hero/site/atlas 360° até completar os 11 gates humanos (`AGENTS.md` §6–8) |
| **Desktop Wrapper (`cmd/nexus-desktop`)** | Implementado, pesado no Linux | Wails v3 + WebKitGTK 4.1 | **`653.5 MB RSS`** em idle (fallback de renderização software `VMware: No 3D enabled` / WebKitProcess `~409 MB`) |
| **Projects, Snapshots e Git Branches / Worktrees** | Implementado, polling pesado em taskbar | `handlers_git.go`, `handlers_git_worktrees.go`, `WorkspaceTaskbar.tsx`, `BranchSwitcherModal.tsx` | Taskbar disparava 5 subprocessos `git` a cada 10s só para ler `current_branch` (corrigido em P0-3); falta fluxo "1-clique card -> worktree -> diff inline -> comentário de volta ao agente" |
| **Composer & Flow (`internal/nexus/runner`, `internal/control/web`)** | Implementado parcialmente (feature flag) | Feature flag `COMPOSER_FLOW_ENABLED`, `schedules` SQLite (`AT`, `AFTER_RUN`, `WHEN_RESOURCES`), checkpoints | Boa fundação de ledger/checkpoints, mas ainda pouco descobrível e sem motor genérico `Event -> Condition -> Action` fora de missões |
| **Browser MCP & Visual Diff** | Implementado e pouco descoberto | `__browser-mcp-stdio` (`~20.2 MB` por sessão), `handlers_browser_visual_diff*.go` | Diferencial forte invisível ao usuário iniciante no dia a dia |
| **Outputs / Artifacts Gallery** | Implementado e pouco descoberto | `outputs` surface (`NexusWorkspaceSession.tsx`), captura de artefatos locais | Pouco promovido após fim de run |
| **Detecção de Providers & Contratos** | Duplicado entre Nexus e `IAPro-SDK` | `internal/core/provider/*` vs `../IAPro-SDK/providers/*`, `capacity`, `quota` | Consolidação parcial via `ADR-iapro-sdk-consumer.md`; adapters ainda mantêm lógica duplicada local |
| **Editor de Código / Diff Review** | Apenas inspecionador parcial / delegado a IDE externa | `open-os` (`editor`/`filemanager`), collision alerts, release desk | Sem visualizador de diff inline com comentários diretos enviados ao agente (gap claro vs Orca/Superset/Vibe Kanban) |
| **Motor de Execução Próprio vs Maestro** | Não duplicado (intencional) | `MAESTRO_EXTERNAL_AUTHORITY_ADR.md` | Nexus controla ambiente, perfis, quotas, terminais e workspace; execução autônoma profunda pertence ao Maestro |
| **Funcionalidades apenas planejadas / simuladas** | Planejado (com guardas honestos) | NEX Retro HD Hero (`RELEASE_READINESS.json`), Composer/Flow certificação visual completa, E2E multi-plataforma Windows/macOS | Mantidos explicitamente marcados como não homologados (`make release-proof`) |

### Funcionalidades Existentes Subdescobertas pelos Usuários
1. **Browser MCP embutido e Visual Diff:** permite que agentes e usuário inspecionem páginas e diferenças visuais sem configuração externa pesada.
2. **Release Desk & Change Collisions (`/release-desk`, `/change-collisions`):** detecta colisão entre agentes editando o mesmo projeto antes do merge.
3. **Workspace Snapshots & Execution Roles (`nexus:execution-role-switch`):** restauração de estado do workspace e troca rápida de papel do terminal IA via Command Palette.
4. **Git Worktrees no Branch Switcher & Launcher:** já existe suporte técnico em backend/API, porém sem fluxo instantâneo "Novo agente em worktree isolado" em 1 passo na tela principal.
5. **Schedules Orientados a Recursos (`WHEN_RESOURCES`):** permite adiar runs quando capacidade/quota não atende aos requisitos, escondido dentro de Composer/Flow.

---

## 3. Pesquisa Dinâmica de Mercado (`2026-11-27`)

Todas as referências abaixo foram consultadas na web em **2026-11-27**, diferenciando **recursos verificados em código/documentação** de **posicionamento comercial** e verificando **compatibilidade de licença** com o modelo de distribuição do Nexus.

| Ferramenta | URL / Repositório | Licença Verificada | Pontos Fortes Verificados (Real vs Promessa) | Limitações / Problemas Conhecidos |
| --- | --- | --- | --- | --- |
| **Orca** | `https://orca.dev/` | Proprietária / source-available | ADE em Electron/TS; BYO-Agent; worktrees Git isolados; **Chromium Design Mode** (clique no DOM envia HTML+CSS+screenshot recortado ao agente); **comentários inline em diffs enviados direto ao agente**; triggers GitHub/Linear/Jira; mirror móvel E2E criptografado | Consumo Electron alto (~600 MB–1 GB+ típico); stack JS acoplada; código não reutilizável diretamente por licença |
| **Herdr** | `https://github.com/herdrdev/herdr` | Open Source (verificada no GitHub) | Multiplexador Rust cliente-servidor (`herdr-core`/`hc`), binário único leve; PTYs sobrevivem ao fechamento de janela; máquina de estados nativa de agente (`working`/`blocked`/`idle`/`done`) via processo foreground + tela + `OSC 7501`; **skip-render** em painéis ocultos; API socket para agente líder comandar workers | Foco exclusivamente em terminal TUI/CLI; sem camada visual rica para contas/quotas ou Browser Design Mode |
| **T3 Code** | `https://github.com/pingdotgg/t3code` | **MIT** | Arquitetura cliente-servidor BYOS (Desktop + Web + mobile via WS); adaptadores canônicos de provedores; worktrees Git isolados + checkpoints via refs ocultas do Git; handoff entre agentes no meio da thread; políticas enterprise JSON/GPO | Ecossistema TS/Electron; gestão de múltiplas contas/quotas por assinatura inferior à do Nexus |
| **Superset** | `https://superset.sh/` | Open-source / Desktop IDE-companion | Padrão `.superset/setup.sh` por worktree; revisão unificada de diff; handoff 1-clique para IDE externa | Sem monitor profundo de quotas multi-conta ou coordenação nativa de runtime como Nexus+Maestro |
| **Conductor** | `https://www.conductor.build/` | Proprietária | UX desktop polida (macOS) para Claude Code/Codex em worktrees paralelos; pipelines GitHub/Vercel | Apenas macOS; foco em BYO API key; menos controle sobre múltiplas contas de assinatura |
| **cmux** | `https://www.cmux.com/` | Proprietária (macOS) | Abas verticais, anéis de notificação visual por terminal, app companion iOS, CLI scriptável | Exclusivo macOS; acoplado a terminal |
| **Vibe Kanban** | Open Source local (Kanban ADE) | Open Source | Fluxo simples: Card Kanban -> worktree automática -> diff + comentários de revisão -> preview local | Menos adequado para sessões livres ad-hoc e sem engine de capacidade multi-conta |
| **Paseo** | `https://github.com/getpaseo/paseo` | Open Source | Orquestrador desktop + mobile (iOS/Android) + web + CLI sobre worktrees + voz | Maturidade operacional menor; sem telemetria profunda de quotas por janela 5h/7d |
| **Terax** | `https://github.com/crynta/terax-ai` | Open Source | Rust + Tauri 2 ultraleve (binário ~7–8 MB, cold start < 1s); WebGL xterm + CodeMirror 6 + Vim + BYOK/Ollama | Escopo enxuto de IDE local; sem governança multi-agente/capacidade |
| **Paperclip AI** | `https://github.com/paperclipai/paperclip` | Open Source | Control plane em Node.js+React modelando org charts de agentes, papéis, aprovações e orçamentos | Abstração pesada voltada a times/empresas virtuais; overhead alto para uso solo diário |
| **AionUi** | `https://github.com/iOfficeAI/AionUi` | Open Source | Desktop Electron + WebUI + hooks Telegram + automações cron 24/7 + Team Mode | Electron pesado; menor foco em Git worktrees e fluxo de código |
| **Happy Coder** | `https://happy.engineering/` / `github.com/slopus/happy` | Open Source | Espelhamento E2E móvel de agentes de terminal com diffs coloridos, prompts de permissão e voz | Dependência de relay móvel; não gerencia ciclo completo de contas/projetos |
| **Claude Squad (`cs`)** | Open Source (Go TUI) | Open Source | TUI Go leve para gerenciar múltiplos agentes de terminal em worktrees Git (`tmux`/PTY) | Sem dashboard web, sem monitor de quotas/contas, sem automações |
| **Composio Agent Orchestrator** | `https://github.com/mnemom/composio-ao` | Open Source | Worktrees + loops autônomos de CI-fix e PR review com Linear/GitHub | Orientado a CI/PR pipeline; fraco como cockpit interativo diário |
| **Cursor / Claude Code / Codex App / OpenCode / Antigravity** | Oficiais respectivos | Variadas (OpenCode: **MIT**) | **Codex App**: `app-server` JSON-RPC desacoplado + sandbox nativo + skills; **Claude Code**: Agent Teams + `CLAUDE.md`/`AGENTS.md` + MCP; **OpenCode**: Go Bubble Tea + SQLite local + contexto LSP; **Antigravity**: Agent Manager + Artifacts; **Cursor**: Background Agents + BugBot | Cada um otimiza seu próprio ecossistema; **nenhum resolve gerenciamento unificado multi-provedor + multi-conta + capacidade real** |
| **CCSwap** | `https://github.com/errhythm/ccswap` | **MIT** | OS keyring + janelas 5h/7d + estratégia `consume-first` + `ccswap run` paralelo isolado + Raycast/menu-bar | Exclusivo para ecossistema Claude; sem workspace/orquestração geral |
| **AgentSwap** | `https://github.com/bojieli/agentswap` | **MIT** (Go zero-dep) | Pool de credenciais + `teleport`/`handoff` de sessão entre harnesses diferentes | Sem interface desktop, sem gerenciamento de projeto/worktree |
| **Agent LB** | `https://github.com/tomaasz/agent-lb` | Open Source | Balanceador multi-conta Claude + dashboard | **Zona de risco de ToS** se operar como proxy/relay de contas de assinatura; Nexus já opta corretamente por **nunca** atuar como relay proxy |
| **ControlCode** | `https://github.com/luis3132/ControlCode` | Open Source | Extensões modulares que expõem servidores MCP isolados por extensão | Foco estreito em extensibilidade MCP |

### Conformidade com Termos de Serviço (ToS) — Verificação `2026-11-27`
- Tanto a **Anthropic** (Claude Code) quanto a **OpenAI** (Codex) proíbem explicitamente extrair/repassar tokens OAuth de assinaturas consumidoras (Pro/Max/Plus) através de relays/proxies de terceiros ou compartilhamento multiusuário.
- Formas legítimas e conformes verificadas em 2026: execução local dos **binários oficiais** em **diretórios isolados por perfil** (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `HOME` dedicado por conta), setup tokens oficiais, chaves API ou contas Service/Enterprise autorizadas — exatamente o modelo nativo do Nexus, que deve ser preservado como regra inviolável.

---

## 4. Matriz Competitiva (26 Dimensões)

Classificação: **Nexus Superior** · **Nexus Equivalente** · **Nexus Inferior** · **Nexus Não Possui** · **Não Relevante** · **Não Verificável**.

| # | Critério | Classificação do Nexus | Referência Principal | Problema Real do Usuário que o Gap / Vantagem Resolve |
| --- | --- | --- | --- | --- |
| 1 | Tempo de inicialização (CLI/Core vs Desktop) | **Equivalente / Misto** (`8–15 ms` CLI vs Wails WebKitGTK) | Terax/Herdr (Rust ~8ms/sub-1s) vs Orca (Electron) | CLI e Core Go sobem instantaneamente, mas Desktop+HUD consomem ~1.35 GB no Linux |
| 2 | Consumo de memória e CPU em idle | **Inferior** (Desktop+HUD `1,358 MB`; Core `41.8 MB`) | Herdr / Terax / Claude Squad (`< 80 MB`) | 2 WebKitGTK completos abertos em idle desperdiçam >1.2 GB RAM e concorrem com builds/agentes |
| 3 | Responsividade da interface | **Inferior -> Melhorado em P0** (`/providers` caiu de `2,299 ms`) | T3 Code / Conductor | Abrir telas ou polling travava ou gastava 2% CPU por subprocessos sem cache |
| 4 | Facilidade de instalação | **Superior** | Go single binary + `make install-local` | Instalação direta em `~/.local/bin` sem cadeia pesada de Node/Python em runtime |
| 5 | Onboarding | **Inferior** | Conductor / Claude Code | Muitas opções/superfícies expostas cedo; usuário iniciante precisa entender conceitos internos |
| 6 | Facilidade de iniciar uma tarefa | **Inferior** | Vibe Kanban / Conductor / Superset | Falta atalho único "Descreva tarefa -> escolhe conta saudável -> abre worktree -> inicia" |
| 7 | Navegação por teclado | **Equivalente** (Command Palette + shortcuts P1-1) | Herdr / cmux / T3 Code | Atalhos de layout e paleta já existem, mas faltavam ações Git/Sync diretas (mitigado em P1-1) |
| 8 | Gerenciamento de projetos | **Superior** | T3 Code / Antigravity | Cadastro canônico, `release-desk`, `change-collisions`, detecção de tech stack |
| 9 | Gerenciamento de workspaces | **Equivalente** | Orca / Superset | Mosaic/Split + snapshots de layout funcionam, falta perfil de início de dia automático |
| 10 | Gerenciamento de agentes | **Superior** | Claude Squad / Herdr | Roles de execução, PTY persistente (`__control-host`), observabilidade e restart |
| 11 | Execução paralela | **Equivalente** | Herdr / Conductor / Claude Squad | Suporta múltiplos runtimes e worktrees, porém criação paralela ainda exige passos manuais |
| 12 | **Gerenciamento de contas** | **Nexus Superior** | CCSwap / AgentSwap / Agent LB | Único com **perfis isolados multi-provedor**, **bloqueio/limpeza de duplicatas**, provenance e conformidade ToS |
| 13 | **Monitoramento de quotas** | **Nexus Superior** | CCSwap | Coleta multi-janela (5h/7d/primary/auxiliary), lock TUI Codex seguro e projeção de elegibilidade |
| 14 | Balanceamento autorizado de workloads | **Superior (com gap de UI)** | AgentSwap / CCSwap | `RecommendResources` e failover já existem em Go; falta expor pools e score com transparência |
| 15 | Continuidade de sessões | **Equivalente** | Herdr / T3 Code / AgentSwap | `__control-host` sobrevive e retoma threads, mas cross-harness handoff ainda é menos fluido que AgentSwap |
| 16 | Integração Git | **Equivalente -> Melhorado em P0** | Orca / Superset / T3 Code | Branch switcher + worktrees + fast HEAD summary P0-3 existem; falta `setup.sh` automático por worktree |
| 17 | Revisão de código | **Inferior** | Orca / Superset / Vibe Kanban | Não há diff visual inline com seleção de linhas enviando comentário direto ao agente |
| 18 | Editor | **Não Possui (intencional / fronteira)** | Terax vs Conductor | Delegar edição pesada à IDE (`open-os`) evita manter outro VS Code; falta apenas preview/diff leve |
| 19 | Comunicação entre agentes | **Inferior (delegado ao Maestro)** | Claude Code Teams / Herdr socket | Nexus não coordena diálogo direto entre agentes sem Maestro; falta API local de handoff/inbox |
| 20 | Automação | **Inferior** | AionUi / Composio AO / Orca | Existem schedules de Flow (`AT`/`AFTER_RUN`/`WHEN_RESOURCES`), falta motor simples `Evento -> Condição -> Ação` |
| 21 | **Notificações** | **Nexus Superior** | cmux / Orca | Hub centralizado no NEX com deduplicação por episódio e fallback automático para OS |
| 22 | Controle de custos | **Equivalente** | Paperclip / IAPro-SDK `cost` | Estrutura de eventos e `IAPro-SDK/cost` pronta; UI diária ainda foca mais quota% do que burn rate$/token |
| 23 | Recuperação após falhas | **Superior** | Herdr / T3 Code | WAL SQLite, `QuotaMonitorService` com lease recuperável e reidratação de runtimes |
| 24 | Suporte multiplataforma | **Inferior (não verificado fora Linux)** | T3 Code / Electron tools | Linux verificado; Windows/macOS explicitamente `UNVERIFIED` no `Makefile` |
| 25 | Integrações externas (GitHub/Linear/Telegram) | **Inferior** | Orca / Composio AO / AionUi | MCPs e hooks existem de forma isolada, sem triggers de PR/Issue/CI out-of-the-box |
| 26 | Manutenibilidade | **Equivalente** | Go + TS strict vs Rust/TS | Código bem estruturado por pacotes, porém duplicações entre `internal/` e `IAPro-SDK` aumentam custo de mudança |

> **Observação sobre limitações de comparação:** métricas de memória/latência de concorrentes citadas apenas como faixas arquiteturais públicas observadas (Electron vs Rust/Tauri) quando não executados localmente no mesmo hardware; nenhum benchmark artificial foi fabricado para concorrentes.
