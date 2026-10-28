# Nexus 1.0 — Auditoria de implementação da release (28/10/2026)

> **Auditoria de código estática; NÃO é homologação.** Base imutável examinada: [`1c67e3b6509cdfdd9d84cc79e3dfcef547b63828`](https://github.com/IAPro-Community/nexus/tree/1c67e3b6509cdfdd9d84cc79e3dfcef547b63828) da `release/v1.0.0`. O HEAD de trabalho foi confirmado após compará-lo ao baseline anterior `1175bef9`: os cinco commits adicionais modificavam workflows e exportação de assets NEX, não os componentes funcionais abaixo. Se a branch avançar, reexecutar a auditoria nas áreas modificadas.
>
> Fonte do contrato: [`docs/reference/agent-workspace-v1.json`](../reference/agent-workspace-v1.json); fontes adicionais: arquivos reais da API, Core, Web, testes e [matriz funcional](../reference/feature-matrix.md). Esta classificação não diz que testes passaram neste SHA. A ausência significa **não localizado como recurso final integrado**, não inexistência de bibliotecas primitivas.

## Sumário executivo

- **48 obrigações do contrato:** 3 marcadas `present`, 31 `partial` e 14 `missing` no inventário versionado. Entre as 14 faltantes estão **2 exclusivamente de homologação/E2E** (`two-client-real-e2e`, `cross-platform-exact-sha`); **12 são experiências de produto ainda sem entrega completa**. Os estados são preliminares e não equivalem a 3 recursos prontos para publicação.
- Há **implementação real não refletida pela leitura literal das issues**: base ACP JSON-RPC + RPC JSONL (`internal/control/agentwire`), backend supervisionado integrado ao `SessionHost` (`internal/control/host/structured_backend.go`), drivers preview opt-in, splits/mosaic/layout persistido, worktrees por Agent e enumeração Git, Attention Radar, Browser MCP/handoff e Outputs/Project Files completo como MVP.
- Principal lacuna: **composição das bases existentes** em uma jornada agente→worktree→browser→review→Mission→evidências→mobile. As 48 obrigações devem ser agrupadas por responsabilidade sem duplicar Core, Scheduler, MissionRunner, memória ou autorização.
- A presença de endpoint, componente e teste fixture **não garante E2E com providers reais, RBAC multiusuário, licença permissiva ou plataforma certificada**. Sem inventar taxas de cobertura/percentuais de conclusão.

## Matriz por requisito (48/48)

Legenda: `BASE_PRESENTE` = inventário declara `present`; `PARCIAL` = contrato já declara `partial` e foram encontrados componentes; `NAO_LOCALIZADO_NA_FORMA_FINAL` = contrato marca `missing`, ainda que existam fundamentos reutilizáveis. **Status de implementação, nunca de aprovação.**

| ID | Estado da entrega | Código/recurso encontrado no SHA | Delta necessário para finalizar | Issue |
| --- | --- | --- | --- | --- |
| `terminal-studio` | BASE_PRESENTE | [`web/src/workspace/WorkspaceRenderer.tsx`](../../web/src/workspace/WorkspaceRenderer.tsx) — Tabs/Mosaic/Split, host supervisionado | Ciclo completo recovery/reconnect/resize por plataforma | #78 |
| `agent-command-center` | PARCIAL | [`web/src/features/agents/AgentsSurface.tsx`](../../web/src/features/agents/AgentsSurface.tsx) — Lista de agentes, estados e ações start/stop/recover | Inspector unificado, lineage, runtimeGeneration e filtro transversal | #104 |
| `review-studio` | PARCIAL | [`internal/nexus/runner/review.go`](../../internal/nexus/runner/review.go) — Ledger de findings e provas de Mission | UI diff+hunk+test+feedback vinculada ao Agent e confirmação de escrita | #109 |
| `browser-studio` | PARCIAL | [`web/src/features/browser/BrowserSurface.tsx`](../../web/src/features/browser/BrowserSurface.tsx) — BrowserSurface e sessão CDP/MCP | Integração inspect/review/diff/preview em pane interoperável | #107 |
| `comparison-lab` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`web/src/features/agents/AgentsSurface.tsx`](../../web/src/features/agents/AgentsSurface.tsx) — Bases para executar agentes separados | Ausente fluxo dedicado de alternativas paralelas, comparação e escolha | #108 |
| `remote-companion` | PARCIAL | [`web/src/features/settings/RemoteAccessTab.tsx`](../../web/src/features/settings/RemoteAccessTab.tsx) — Acesso remoto e Outputs no projeto | Inbox de intervenções, handoff entre devices e controle exclusivo | #111 |
| `mission-observatory` | PARCIAL | [`web/src/features/work/FlowRunSurface.tsx`](../../web/src/features/work/FlowRunSurface.tsx) — Tela Mission e Evidence Bundle no backend | Navegação integrada por attempts, bloqueios e proof links | #85 |
| `smart-layouts` | PARCIAL | [`web/src/workspace/arrangePresets.ts`](../../web/src/workspace/arrangePresets.ts) — Presets e motor de layout | Aprimorar orquestração contextual e persistência ao alternar layouts | #105 |
| `workspace-snapshots` | PARCIAL | [`web/src/workspace/presentation.ts`](../../web/src/workspace/presentation.ts) — Snapshots internos de modo/layout e persistência | Salvar, nomear, listar e restaurar snapshots de usuário com migração | #105 |
| `agent-groups` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`web/src/features/agents/AgentsSurface.tsx`](../../web/src/features/agents/AgentsSurface.tsx) — Agents persistentes e Mission teams | Agrupamento visual por tarefa/worktree com persistência e ownership | #105 |
| `quick-switcher` | BASE_PRESENTE | [`web/src/app/commands/registry.ts`](../../web/src/app/commands/registry.ts) — Busca e atalhos de comando | Refinar cobertura e navegação de teclado | #84 |
| `follow-agent` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`web/src/workspace/WorkspaceRenderer.tsx`](../../web/src/workspace/WorkspaceRenderer.tsx) — Motor de seleção/foco existente | Modo Follow Agent com pausa e escolha do runtime observado | #105 |
| `floating-inspector` | PARCIAL | [`web/src/features/work/ComposerInspector.tsx`](../../web/src/features/work/ComposerInspector.tsx) — Inspector Composer e superfícies flutuantes | Inspector global por agente, fixação, foco e lifecycle | #104 |
| `agent-parking` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`web/src/features/agents/AgentsSurface.tsx`](../../web/src/features/agents/AgentsSurface.tsx) — Stop/recover e estados de runtime | Park/ocultar UI sem alegar pause de processo sem suporte | #114 |
| `attention-radar` | PARCIAL | [`web/src/components/GlobalAttentionRadar.tsx`](../../web/src/components/GlobalAttentionRadar.tsx) — Radar global e detector OSC/PTY | Todos providers, handoff de intervenção, estado autoritativo e dedup | #100 |
| `focus-queue` | PARCIAL | [`web/src/app/attentionRadarModel.ts`](../../web/src/app/attentionRadarModel.ts) — Leitura e agrupamento de atenção | Fila única priorizada de interações, idempotência e persistência | #100 |
| `agent-timeline` | PARCIAL | [`web/src/app/activityModel.ts`](../../web/src/app/activityModel.ts) — Eventos e atividade por runtime | Timeline histórica por runtimeGeneration, status/proveniência | #114 |
| `resource-recommendations` | PARCIAL | [`internal/nexus/resource_recommendation.go`](../../internal/nexus/resource_recommendation.go) — Recomendações por elegibilidade/capability | Racional exibido LIVE/CACHED/UNKNOWN por conta+modelo, PIN e fallback | #87 |
| `runtime-health` | PARCIAL | [`internal/control/registry/lifecycle.go`](../../internal/control/registry/lifecycle.go) — Lifecycle/processos supervisionados | Inspector unificado com heartbeat/lease/PTY/processo diagnóstico | #114 |
| `agent-compare` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`internal/nexus/runner/review.go`](../../internal/nexus/runner/review.go) — Worktree + review primitivas | Comparação simultânea de commits/diffs/testes/custo com proveniência | #108 |
| `review-handoff` | PARCIAL | [`internal/nexus/mission_team.go`](../../internal/nexus/mission_team.go) — Team/review no engine | Pedido formal de revisão e saída ligada a WorkReceipt | #110 |
| `consult-another-agent` | PARCIAL | [`internal/nexus/mission_team.go`](../../internal/nexus/mission_team.go) — Delegação e coordenação no Core | Interação explícita, consentimento, timeout e não duplicar Maestro | #110 |
| `change-radar` | PARCIAL | [`internal/control/web/handlers_git_worktrees.go`](../../internal/control/web/handlers_git_worktrees.go) — Leitura de worktrees e findings | Agregação de arquivos/hunks por agente e atualização em tempo real | #109 |
| `test-matrix` | PARCIAL | [`internal/nexus/evidence_bundle.go`](../../internal/nexus/evidence_bundle.go) — Receipt/evidência de execução | Matriz de testes command/exit-code/sha/ambiente na UI | #109 |
| `release-desk` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`internal/nexus/evidence_bundle.go`](../../internal/nexus/evidence_bundle.go) — Projeção de provas disponível | Tela de coordenação issue→PR→check e bloqueio de entrega | #109 |
| `browser-pane` | BASE_PRESENTE | [`web/src/features/browser/BrowserSurface.tsx`](../../web/src/features/browser/BrowserSurface.tsx) — Browser como surface no Workspace | Testes cross-client e ergonomia do painel | #107 |
| `design-inspect` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`internal/browsercontrol/browser_semantic.go`](../../internal/browsercontrol/browser_semantic.go) — Snapshot/DOM semântico e sessões browser | Captura seletor/estilos/crop redigida enviada ao Agent com autoria | #107 |
| `live-preview` | PARCIAL | [`internal/browsercontrol/browser_stream.go`](../../internal/browsercontrol/browser_stream.go) — Visualização live da sessão no host | Acoplamento automático ao output e revisão no mesmo workspace | #107 |
| `browser-automation` | PARCIAL | [`internal/control/web/handlers_browser_mcp.go`](../../internal/control/web/handlers_browser_mcp.go) — Ferramentas MCP + grants/handoff humanos | Negociação real e2e por CLI, revogação e limites de permissão | #83 |
| `visual-diff` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`web/src/features/browser/BrowserSurface.tsx`](../../web/src/features/browser/BrowserSurface.tsx) — Captura/snapshot existentes | Ausente diff visual antes/depois com baseline e threshold | #107 |
| `device-preview` | PARCIAL | [`web/src/features/browser/BrowserSurface.tsx`](../../web/src/features/browser/BrowserSurface.tsx) — Preview da página remota | Controles reais e cenários telefone/tablet/desktop | #107 |
| `mobile-inbox` | PARCIAL | [`web/src/components/GlobalAttentionRadar.tsx`](../../web/src/components/GlobalAttentionRadar.tsx) — Notificações/atenção e sessão Web remota | Fila móvel autenticada para perguntas e aprovações com receipts | #111 |
| `remote-terminal` | PARCIAL | [`web/src/nexus/AgentTerminal.tsx`](../../web/src/nexus/AgentTerminal.tsx) — Terminal Web existente e tunnel | Handoff de input, leases e reconnect multi-device seguros | #111 |
| `workspace-sharing` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`internal/browsercontrol/browser_collaboration.go`](../../internal/browsercontrol/browser_collaboration.go) — Grants para Browser, não para Workspace | Não há RBAC multiusuário de workspace + observer/revogação | #112 |
| `voice-actions` | PARCIAL | [`web/src/features/voice/VoiceCaptureButton.tsx`](../../web/src/features/voice/VoiceCaptureButton.tsx) — Capture/control por voz | Comandos seguros sem Enter automático, permissões e origem | #114 |
| `contextual-hud` | PARCIAL | [`web/src/features/hud/NexusHud.tsx`](../../web/src/features/hud/NexusHud.tsx) — HUD e projeção de estado | Consistência real com estado, baixa CPU/reduced motion | #114 |
| `nex-companion` | PARCIAL | [`web/src/features/hud/NexPixelSprite.tsx`](../../web/src/features/hud/NexPixelSprite.tsx) — NEX sprite e mascote contextual | Completar animações/estados baseados em eventos, sem 2º mascote | #114 |
| `workspace-recipes` | PARCIAL | [`web/src/workspace/arrangePresets.ts`](../../web/src/workspace/arrangePresets.ts) — Cinco receitas existentes (Frontend, Backend, Review, Debug, Release) | Restaurar com snapshots/grupos e sem lançar processos | #105 |
| `parallel-challenge` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`internal/nexus/worktree.go`](../../internal/nexus/worktree.go) — Trabalho isolado e recomendação como bases | Ausente modo de comparação de soluções/seleção/veredito | #108 |
| `change-collision-detector` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`internal/control/web/handlers_git_worktrees.go`](../../internal/control/web/handlers_git_worktrees.go) — Enumeração Git de worktrees | Detecção de colisão de hunks/paths entre agents | #109 |
| `device-handoff` | PARCIAL | [`web/src/features/settings/RemoteAccessTab.tsx`](../../web/src/features/settings/RemoteAccessTab.tsx) — Web remoto e terminal compartilhável | Retomada desktop↔celular sem segundo writer nem replay | #111 |
| `maestro-managed-dependency` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`internal/nexus/maestro.go`](../../internal/nexus/maestro.go) — Integração opcional, modos OFF/ASSIST/ORCHESTRATE | Dependência gerenciada com install/update/uninstall reversível | #113 |
| `maestro-canonical-memory` | PARCIAL | [`internal/nexus/intelligence_memory.go`](../../internal/nexus/intelligence_memory.go) — IntelligenceMemory legado + context.brief via Bridge | Backfill, rollback e semantic write autoritativo no Maestro | #113 |
| `single-execution-owner` | PARCIAL | [`internal/nexus/mission_executor.go`](../../internal/nexus/mission_executor.go) — Autoridades de execução e handoff parciais | Provar ownership único, fencing/idempotência em falha | #113 |
| `composer-flow-onboarding` | PARCIAL | [`web/src/features/work/ComposerSurface.tsx`](../../web/src/features/work/ComposerSurface.tsx) — Composer/Flow, ProductTour, docs de uso | UX por 6 estados, i18n, fallback e fluxo end-to-end | #102 |
| `remote-generated-outputs` | PARCIAL | [`internal/nexus/artifacts/catalog.go`](../../internal/nexus/artifacts/catalog.go) — Scanner seguro, thumbs, original, revision history, API e galeria | Proveniência automática para todos agentes, RBAC por usuário, mídia grande, mobile real | #94 |
| `two-client-real-e2e` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`internal/control/web/multi_runtime_attention_e2e_test.go`](../../internal/control/web/multi_runtime_attention_e2e_test.go) — Testes de dois clientes com runtime fixture | Prova com 2 providers reais, dois clientes e worktrees seguros | #81 |
| `cross-platform-exact-sha` | NAO_LOCALIZADO_NA_FORMA_FINAL | [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) — Workflows de validação e contratos | Evidências same-SHA por SO, assinatura e instalação | #27 |

## Auditorias transversais: correções de classificação anteriores

### 1. Harnesses/ACP/Providers — issues #97–#101

- **Implementado:** `internal/control/agentwire/{acp,jsonline,pi,bridge,state}.go` oferece JSON-RPC ACP bidirecional, limite 4 MiB/frame, request/response correlacionados, Pi/OMP RPC e reconciliador; testes `agentwire_test.go` e `bridge_test.go`. `internal/control/host/structured_backend.go` foi integrado ao `SessionHost` com handshake e comandos `/cancel`, `/permit`, `/answer`; launcher/driver suportam seleção explícita `agent_protocol`. Logo, **não escrever que transporte ACP/RPC está ausente**.
- **Parcial/experimental:** `internal/control/driver/registry.go` registra `pi`, `omp`, `mimo`, `dsh` apenas com `NEXUS_EXPERIMENTAL_HARNESSES=1`. `harness_preview.go` informa capabilities `UNSUPPORTED/PARTIAL`, `Tested:false`; `dsh` recusa PTY. `NewAgentModal.tsx` filtra modos pelo report do Core. Faltam ciclo real por provider e version negotiation, autenticação/permission HTTP vinculada à sessão, handling de erros e teste de contas isoladas.
- **Atenção/AGY:** `internal/control/host/attention.go` implementa PTY heurístico e desativa scraping quando há eventos estruturados; `osc7501.go` e `web/src/app/attentionRadarModel.ts` dão base. Falta conciliar todos os signals no Core e distinguir observação vs inferência antes de liberar resposta remota. A nota [AgentWire](../../DEV/validation/AGENTWIRE_IMPLEMENTATION_STATUS_2026-10-27.md) tem um addendum mais recente que invalida alguns blockers históricos no fim do mesmo documento.

### 2. Primeiro uso — #77/#79

`web/src/app/modals/WelcomeModal.tsx`, `web/src/features/projects/firstRunReadiness.ts` e `ProviderReadinessBanner.tsx` já detectam providers e diferenciam `REGISTERED_AUTHENTICATED`, `PENDING_AUTH`, `REGISTRATION_UNKNOWN`. `DirectSessionLauncher` já existe. Trabalho real: unir projeto→provider/profile→sessão em percurso com diagnóstico e retry, tratar falha sem recurso, medir fricção em instalações limpas; **não criar segunda Welcome UI**.

### 3. Workspaces/Agents — #80/#81/#104/#105

`presentation.ts` já tem `TABS|DESKTOP|MOSAIC|SPLIT`, orientações/ratios, snapshots internos de modo e receitas; `WorkspaceProvider.tsx` persiste layout. `internal/nexus/worktree.go` cria/reusa worktree de Agent e `handlers_git_worktrees.go` fornece leitura Git. Trabalho restante: juntar visualização de worktree com runtime ownership; inspector único; agrupamento/follow; detectar alterações concorrentes; E2E 2 agentes reais/2 clientes. `workspace-snapshots` internos **não** são ainda recurso de snapshots nomeados pelo usuário. `Mission teams` não equivalem a `Agent Groups` visuais.

### 4. Browser — #83/#107

`BrowserSurface.tsx`, `internal/browsercontrol/browser_collaboration.go` e `handlers_browser_mcp.go` já oferecem browser, grants temporários, takeover/handoff, snapshots e controle. Falta Design Inspect seletivo com autoria/proveniência, Visual Diff reproduzível, Device Preview completo, integração com revisão e evidência E2E com CLIs reais. Não adicionar segundo Chromium/control plane.

### 5. Outputs e mobile — #94/#88/#111/#112

`internal/nexus/artifacts/{catalog,history,thumbnail}.go` contém `os.Root`, whitelist/MIME sniff, arquivo original, cache thumbnail, limites e snapshots; `handlers_project_outputs.go` expõe API sob sessão do projeto; `handlers_output_publish_mcp.go` atribui publicação a runtime autenticado; `OutputsSurface.tsx` tem galeria/preview/download/histórico e modo arquivos. **É falso tratá-lo como galeria inexistente.** A própria [documentação de implementação](../product/nexus-outputs-project-files.md) registra pendência de proveniência automática, RBAC por membro, mídias grandes e mobile real. `RemoteAccessTab.tsx` expõe túnel/QR; isso **não** é Workspace Sharing multiusuário com RBAC.

### 6. Maestro — #86/#113

`internal/nexus/maestro.go` tem modos `OFF/ASSIST/ORCHESTRATE`; `MaestroSurface.tsx` consome status/skills; `internal/control/handoff/maestro_context.go` usa `context.brief`; `internal/nexus/intelligence_memory.go` mantém memória legada. **Falta** instalação compatível gerenciada, version handshake final, migração idempotente com backup/rollback e escrita semântica canônica apenas no Maestro. Deve operar independente de Nexus.

### 7. Missions/Review/custos — #82/#85/#87/#89/#109/#110

`internal/nexus/evidence_bundle.go`, `internal/nexus/runner/review.go`, `mission_*.go`, `FlowRunSurface.tsx` e `resource_recommendation.go` já oferecem execução, evidência, findings e seleção explicável. **Não criar outro runner, review store ou scheduler**. Priorizar a projeção UI de diff/tests/vereditos reais, upload/links de evidências, aprovação antes de Git e fluxo issue→PR e consulta entre agentes.

### 8. Publicação/comunidade fora da homologação técnica — #65/#74/#90/#96

A licença ainda exige clearance jurídico de contribuições/assets antes de Apache-2.0. GitHub branch rulesets e canal privado de vulnerabilidades exigem ação de administrador. São impedimentos à proposta Community/open source, não prova de que o runtime Core esteja ausente. Não anunciar licença não autorizada.

## Qualidade da evidência e limitações

Esta revisão leu arquivos de código e testes na árvore Git e confrontou o contrato; **não executou build/Go/Vitest/Playwright em host local, não iniciou CLIs reais e não certificou uma distribuição instalável**. A CI pode conter sucesso/falha em SHAs diferentes e não substitui essa ressalva. Itens `missing` necessitam busca de implementação aditiva antes de novo código; nunca deletar componentes históricos por classificação documental.

**Plano em ordem de implementação:** [ACTION_PLAN_2026-10-28.md](ACTION_PLAN_2026-10-28.md).
