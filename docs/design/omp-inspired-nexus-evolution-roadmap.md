# Nexus — Plano sequencial de evolução inspirado em Oh My Pi (sem integração OMP)

**Data:** 2026-10-29  
**Destino canônico:** release/v1.0.0  
**Natureza:** plano de implementação, não declaração de funcionalidade homologada.  
**Baseline de inspeção:** Nexus ec4a527ba1448f24db22fd638a59605227377e5a; Maestro v1 bfeb33fa5f856607f0c8017e1b7baf960f0a4802; OMP main d33a0edb00c8b76b28d71eb976cd82d69617bcba. Revalidar HEAD ao iniciar cada etapa.

## Objetivo e exclusões

Usar OMP como **referência de produto e engenharia**, sem rodar, empacotar, instalar, importar motor ou habilitar OMP como CLI no Nexus nesta iniciativa. O PR experimental Nexus #150 e a issue #98 de provider continuam fora do caminho crítico. Todos os PRs de código devem ter base em release/v1.0.0, jamais main; preservar gates e comportamento legado.

Modelo de ownership:

- **Nexus:** UI/Web/Desktop/Mobile, supervisão dos processos iniciados no Nexus, SessionHost, PTY/ConPTY, Browser/CDP, runtime registry/events, leases, layout, status e evidência operacional. Ao existir Mission Nexus-owned, seu MissionRunner é o owner de execução.
- **Maestro:** memória episódica e semântica, contexto compartilhado, context brief, promoção de conhecimento, evolução de skills, regras/skills e método. Em Runs Maestro-owned, o Runtime do Maestro decide executar/reiniciar/cancelar.
- **Provider:** sessão/transcrição nativa, execução de ferramentas internas e compaction nativa; só oferecer ações de controle quando há API/capability testada.
- **Nexus legado IntelligenceMemory:** somente leitura/compatibilidade/migração reversível durante transição. **Não expandir para novo conhecimento, histórico semântico ou embeddings.**
- **Checkpoint operacional ≠ memória semântica:** armazenar IDs/estado de processo, run, geração, workspace, lease e referência de evidência é permitido no Nexus; decisões e aprendizados recuperáveis pertencem ao Maestro.

## Evidência do código, não da lista de desejos

### OMP examinado

- Direção de workers persistentes: docs/vibe-mode.md, packages/coding-agent/src/vibe/runtime.ts e vibe/lifecycle.ts. O processo mantém workers com identidade, eventos de spawn/turn/tombstone, steering, fila, wait/cancel e transcrição em sessão própria.
- Async jobs: docs/tools/task.md, packages/coding-agent/src/async/job-manager.ts: limites de concorrência, estado running/completed/failed/cancelled, entrega com retry, retenção/cleanup e escopo por owner.
- Agent Hub: docs/agent-hub.md; observação de árvore, atividade, uso e intervenção, com suporte a parked/resume.
- História/fork/restore: docs/session.md e docs/session-operations-export-share-fork-resume.md; entries com parentId, leaf ativa e operações distintas de fork, resume e restore.
- Compaction: docs/compaction.md; entries tipadas e sumários de branch. Não transplantar mecanismo para Nexus.
- Memória/skills: docs/memory.md e docs/tools/learn.md; local, Hindsight, Mnemopi e outras opções. Não copiar.
- Browser: docs/tools/browser.md; supervisor de tabs/autorizações/CDP, referência para Nexus Browser atual.
- Uso: packages/coding-agent/src/stats/activity-worker.ts; sincronização/agregação de sessão em worker, sem bloquear a TUI.

### Maestro examinado

- orquestrador/bin/memory.js contém record, search, searchWithVisibility, promote, consolidate, escopo e validação de evidência; docs/memory-scopes.md documenta repositório/branch/workspace/commit/task.
- runtime/bridge/bridge.js v1 suporta context.brief, skills.list, runs.list/get/inspect/create/cancel, provas, evidência e artefatos. **Não oferece memory.search/record**. runs.subscribe retorna explicitamente unsupported; events.subscribe não implica stream de runs funcional. Interface faltante pertence ao Maestro: issue https://github.com/IAPro-Community/Orquestrador-Maestro/issues/41.
- docs/context-economy.md e MAESTRO_INTEGRATION_API.md documentam manifest/digest e brief limitado. Não confundir brief com compaction nativa do provider.
- runtime/skills/registry.js e MAESTRO_SKILLS.md já são autoridades de catálogo/contrato de skills. Learn-to-skill deve entrar por esse domínio.
- runtime/MAESTRO_RUNTIME.md: RunStore tem persistência operacional local; não presumir assinaturas/controle distribuído que Bridge v1 não oferece.

### Nexus examinado

- Lead/roles/teams: internal/nexus/interactive_lead.go, execution_roles.go, mission_team.go, agent-routing e MissionRunner já existem.
- Background/evidência: mission_execution.go, mission_notifications.go, runner/review.go, evidence_bundle.go; UI FlowRunsHistorySurface, AgentActivity, AgentsSurface.
- Continuidade: continuity_controller.go, internal/control/handoff e docs/architecture/session-continuity.md. Native resume NÃO é garantido.
- Memória legada: internal/nexus/intelligence_memory.go; Maestro context já possui adaptador somente-leitura internal/control/handoff/maestro_context.go.
- Uso/custos: handlers_telemetry.go, handlers_cost.go, internal/core/telemetry, cost ledger e web/features/usage.
- Browser: internal/browsercontrol/browser_runtime_session.go e launcher/browser_auto_mcp.go; há provisão lazy do browser e capability por runtime.
- docs/product/agent-workspace-v1-acceptance.md marca 48 obrigações, muitas parciais e sem evidência final; nenhuma presença de arquivo equivale a certificação.

## Auditoria de issues/PRs

No momento da inspeção: **46 issues abertas, 3 issues fechadas** (#53, #48, #6) no Nexus. Existem muitos PRs merged na release, inclusive #145/#143/#140/#131 (attention), #130 (capability metadata), #149 (quota classification), #128 (auditoria), mas issues de produto permanecem abertas. PR #58 (split contextual) foi fechado **sem merge**; não confundir sua implementação potencial em outros commits com merge daquele PR. PR experimental #150 permanece aberto em draft e não é dependência deste plano.

Matriz de reaproveitamento, pendência e caminho:

| Tema | Código encontrado | Issue canônica | Decisão |
|---|---|---|---|
| Director / vibe UX | Lead + ExecutionRole + MissionTeams | #152 (nova); #104 #110 #85 | UI e comando por capability, sem subagent runtime novo |
| Papéis/modelos por workflow | mission_team.go / execution_roles.go / scheduler | #104 #110 #87 #113 | Unificar catálogo e preferências; não recriar modelos fast/good |
| Tarefas de fundo | mission_execution.go, notifications, FlowRunsHistory | #85 #100 #114 | Atividade observável por execução, sem job store duplicado |
| Histórico em árvore | RuntimeGeneration/Agent lineage e eventos | #104 #114 #86 | Árvore operacional Nexus; decisões semânticas Maestro |
| Fork conversacional | Comparison Lab e handoff previstos | #108 #110 #153 | Separar clone de contexto, branch Git e sessão nativa |
| Restauração | continuity_controller e handoff | #85 #111 #114 | Classificar attach/resume/handoff/restart com provas |
| Checkpoints | internal/control/handoff/checkpoint.go | #86 #105 #114 | Snapshot operacional; não salvar LLM memory |
| Rewind | Timeline / Git / compare previstos | #108 #114 | Navegação/seleção de versão; escrita/revert exigem autorização |
| Compaction + context notes | Maestro context brief; legado memória Nexus | Maestro #41; Nexus #86 #113 | Implementar na autoridade Maestro e expor projeção |
| retain / recall / reflect | Maestro memory.js; Nexus intelligence_memory.go | Maestro #41; Nexus #86 #113 | Migrar e descontinuar escrita semântica no Nexus |
| Aprendizado -> skill | Maestro skill registry + router | Maestro #41; Nexus #86 #110 | Proposta e revisão no Maestro, nunca auto-promover |
| Importação sessões | SessionHost/continuity/handoff parcial | #153 (nova), #108 #111 | Read-only metadata, provenance, native resume verificado |
| TTFT, latência, erros/traces | telemetry e uso existentes | #87 #114 | Instrumentação observável por provider/turn |
| Tokens, custos e cache | Cost ledger, usage UI | #87 | Dashboard com fontes e incerteza |
| Browser CDP | browsercontrol, Browser MCP | #83 #107 | Melhorar engine existente, sem segundo navegador |
| Gates | agent-workspace-v1 acceptance | #27 #81 #115 | E2E, segurança, plataformas, SHA real |

## Execução estritamente sequencial (cada fase libera a seguinte)

### 00 — Contratos e migração de autoridade, antes de mexer em UX
**Issues:** Maestro #41; Nexus #86 #113. **PRs:** 00A contrato e capability negotiation no Maestro; 00B API/adapter Nexus; 00C migração legacy memory (backup/dry-run/apply/rollback).
- Inventariar cada operação de IntelligenceMemory e dados persistidos, separar hábitos/decisões/notas do estado operacional, definir fonte canônica e precedência. Não apagar original antes de verificação.
- Definir contratos memória read/search/record, origem, escopo, digest e autorização, sem exportar JSONL cru por padrão. Garantir compatibilidade de context.brief.
- Reconciliar ADR antigo "Maestro opcional" versus obrigação arquitetural de Maestro gerenciado, mantendo Direct estável quando indisponível e sinalizando DEGRADED.
**Aceite:** 2 projetos/branches isolados, registro e recall via Maestro, redaction, round-trip Bridge, rollback e ausência de novos writes semânticos Nexus. Testes nos **dois** repos e migração reversível. Sem esse gate não implementar history/compaction/skills semânticos.

### 01 — Workflows por papéis, capacidades e modelos
**Issues:** #104 #110 #87 #113. **PRs:** normalização/capabilities; UI seletor; testes.
- Projetar catálogo de roles existente do Nexus (build/investigate/review/verify/architect/security) ao catálogo Maestro, mantendo identidade de AgentSpec vs ExecutionRole temporária.
- Definir templates por atividade (bugfix, revisão, feature, release), requisitos de skill e resource preferences PIN/PREFER/AUTO. Backend (ResourceScheduler e Maestro owner) decide; não criar segundo motor nem fixar modelos "fast/good".
- Exibir por etapa: papel, conta/modelo selecionado, motivo, custo estimado/UNKNOWN e política de escrita.
**Aceite:** executar workflow de 3 papéis sobre Codex/OpenCode simulados e 2 perfis, sem mudar AgentRevision por troca transitória; PIN fail-closed.

### 02 — Jobs e tarefas em segundo plano, observabilidade
**Issues:** #85 #100 #104 #114 #81. **PRs:** projeção job+generation; entrega de eventos; UI fila.
- Normalizar apenas a projeção de status (QUEUED/RUNNING/NEEDS_USER/SETTLED/FAILED/CANCELLED/UNKNOWN) em cima de MissionRun/Run do owner. Não criar segunda JobRunner nem id de agente concorrente.
- Maestro v1 não fornece runs.subscribe: optar por adapter de eventos versionado no Maestro ou leitura reconciliada com polling limitado, cache/version cursor, jamais inventar streaming nativo.
- Fila mostra ID, geração, worker, provider, worktree, tempo, última atividade, próxima ação, exceções e URL de evidência.
- Cancelamento, retries e respostas exigem correlation, owner e idempotency key; quando não há capability, mostrar "Abrir terminal".
**Aceite:** 2 runs concorrentes, 2 clients, refresh/restart e cancel seguro; único envio de resultado e notificações deduplicadas.

### 03 — Director Mode (adaptação de /vibe sem copiar runtime)
**Issues:** #152 #104 #105 #110 #85. **PRs:** comandos tipados; Director screen; steering/cancel guard; E2E.
- Tela de direção com workers existentes, objetivos, dependências, tarefas, seleção por papel e critérios de aceite; comando explícito para atribuir/continuar tarefa.
- Command contract: DIRECTIVE/INSPECT/WAIT/QUEUE_NEXT/STEER/CANCEL. Reconhecer que STEER ao vivo só existe com API suportada; caso contrário, enfileirar **somente** pelo owner que conhece a fila ou abrir o terminal, sem prometer steer.
- Propagar orientação em snapshots de execução/WorkReceipts e contexto Maestro; nunca copiar transcrição nativa para um novo diretório de memória Nexus.
**Aceite:** 2 workers independentes, mudar prioridade/role durante a jornada, concluir/falhar/cancelar com rastreabilidade; nenhuma escrita em checkout compartilhado.

### 04 — Histórico em árvore navegável e Timeline
**Issues:** #104 #114 #86 #100. **PRs:** projeção com IDs estáveis; timeline/árvore UI; filtros.
- Árvore de Agent -> RuntimeGeneration -> Run/WorkPackage -> evento/resultado, por fonte/geração. Marcadores de decisões/lessons são referências consultadas no Maestro (não duplicação semântica).
- Timeline destaca spawn, owner, detach/attach, interventão, retry, tests, worktree/commit e links de evidência; eventos de provider somente quando comprovados.
**Aceite:** árvore estável após refresh, 2 worktrees isolados, transições UNKNOWN, visão 390px/teclado, sem dados cross-project.

### 05 — Fork de exploração e comparação
**Issues:** #108 #110 #153 #81. **PRs:** contrato fork; contexto; UI comparar.
- Separar (A) fork de navegação/hipótese no Composer/Flow, (B) novo Agent/Run com ContextCapsule do Maestro, (C) fork nativo do provider quando comprovado. A e B não garantem C.
- Novo worktree/branch somente com autorização; lineage com base commit e scope; diffs, testes, custo e decisões humanas lado a lado.
**Aceite:** comparar duas alternativas sem alterar a original, impedir dois writers no mesmo checkout, sem merge/replay automático.

### 06 — Checkpoints operacionais e snapshots
**Issues:** #105 #114 #86. **PRs:** schema+idempotência; restore da UI; UI checkpoint.
- Checkpoint = runtime IDs/generation, owner, worktree/base SHA, progress receipt, layout, status, artifacts digests e refs Maestro; dados semânticos permanecem no Maestro.
- Diferenciar VISUAL_SNAPSHOT, RUNTIME_CHECKPOINT, MAESTRO_CONTEXT_REVISION e PROVIDER_NATIVE_SNAPSHOT; últimos dois apenas por APIs testadas.
**Aceite:** snapshot/restore visual sem spawn; runtime morto não aparece vivo; migração do schema e stale detection.

### 07 — Restauração de tarefas interrompidas
**Issues:** #85 #111 #114 #153. **PRs:** classificador; preflight; recuperação guiada.
- Estados e operações: REATTACH_SAME_RUNTIME, NATIVE_RESUME_VERIFIED, NEW_RUN_WITH_CONTEXT, HUMAN_RESTART, UNSUPPORTED. Não reciclar session ID entre providers.
- Registrar última ação externa, evidência idempotente, bloqueio de efeitos incertos e confirmação antes de repetir trabalho; Maestro-owned restaurado via Runtime Maestro.
**Aceite:** kill/restart em pontos críticos, não duplicar comando, mesma geração só se processo vivo; statuses confirmados nos dois clients.

### 08 — Rewind e escolha de versões
**Issues:** #108 #114 #109. **PRs:** navegação temporal read-only; restauração de contexto sob demanda; git revert separado.
- "Voltar" no histórico muda a seleção e mostra estado/diffs/decisões anteriores; não apaga branch de observações nem zera o runtime.
- Voltar código exige comparação/novo worktree e aprovação explícita. Contexto recuperado vem do Maestro com source/digest; provider session rewind só se suportado de forma nativa.
**Aceite:** 3 checkpoints/branches, consultar passado sem mutação, operações Git protegidas.

### 09 — Compaction e context notes via Maestro
**Issues:** Maestro #41; Nexus #86 #113. **PRs:** capability read; notes/summary no Maestro; projeção na UI.
- Exibir orçamento, digest, origem e sinais de stale do context.brief atual; notas editáveis, seleção e consolidação pertencem ao Maestro.
- Context minimization precisa ser mensurada (tokens antes/depois sem alegar custo real), auditada e reversível.
- Não interferir na compaction do próprio Codex/Claude/OMP; não carregar árvores inteiras automaticamente.
**Aceite:** duas sessões recebem brief consistente com versão/escopo, privacy, rehydrate sob demanda, falha de Bridge explícita.

### 10 — Retain / Recall / Reflect canônicos
**Issues:** Maestro #41; Nexus #86 #113. **PRs:** read/search; write approvals; reflection UI read-only.
- Adapter do Nexus para observações do Maestro, com busca contextual e evidência. Reflection não gera verdade canônica sem verificação e promoção aprovadas.
- Migrar habits legados e bloquear shadow writes; dedupe, escopo e status verificado. Não introduzir banco de embeddings no Nexus.
**Aceite:** decisões por branch não vazam a outra, recuperação e retenção do Maestro com aprovação, rollback sem perda.

### 11 — Aprendizado para skills
**Issues:** Maestro #41 + catálogo Maestro; Nexus #86 #110. **PRs:** contrato de proposta Maestro; preview; validação.
- Converter descobertas verificadas em candidato de skill contendo problema, gatilhos, restrições, custo, critérios, origem, confiança e testes.
- O catálogo/roteamento/arquivos de skill pertencem ao Maestro. O Nexus mostra proposta, diff e resultado da validação, sem criar SKILL.md silenciosamente.
**Aceite:** candidato rejeitado não entra em produção, candidato aceito preserva autoria/revisão e não sobrescreve skill do usuário.

### 12 — Importação de sessões externas
**Issues:** #153 #111 #108 #86. **PRs:** adapter read-only por provider; lineage; UX.
- Metadados autorizados e provas de compatibilidade; distinguir READ_ONLY_IMPORT, VERIFIED_NATIVE_RESUME, CONTEXT_HANDOFF e ATTACH.
- Sem armazenar transcrição completa no Nexus por padrão. Capturas semânticas opt-in entram pelo Maestro.
**Aceite:** duas fontes reais, idempotência, version mismatch e sanitização, nenhum native resume fictício.

### 13 — Traces por provider: TTFT, latência, erros
**Issues:** #87 #114 #100. **PRs:** envelope; instrumentação; consulta.
- Tipos mínimos: provider/profile/runtimeGeneration/turn, start, first-observable-token (quando estruturado), end, cancellation, retries, cache evidence, error category, clock source e timestamp.
- TTFT só é exibido quando o protocolo fornece limites suficientes; PTY com streaming sem turn boundary robusta deve reportar UNKNOWN/UNSUPPORTED.
- Traces locais com sampling, retenção, privacidade e redaction; nenhum prompt/code/token secreto nos spans. Distinguir provider-reported e measured.
**Aceite:** latência real no harness estruturado testado, erro/timeout/cancel diferenciados, custo de instrumentação controlado e métricas ausentes transparentes.

### 14 — Dashboard de tokens, custo e cache
**Issues:** #87 #104 #114. **PRs:** agregação sobre CostLedger; UI filtros; export.
- Reaproveitar Usage/Cost Intelligence e suas políticas de identidade, conta, quota LIVE/CACHED/ESTIMATED/UNKNOWN.
- Métricas por Agent/Run/Provider/conta/worktree/projeto: input/output/cached tokens somente com fonte, TTFT, duração, erro/retry, equivalência API e custo faturado separadamente.
- Não somar estimativa com valor faturado, nem tratar quota opaca como 0; permitir CSV com escopo e redaction.
**Aceite:** fixture com 2 providers, cache de um deles indisponível, filtros e export consistentes, totalizações sem double count.

### 15 — Nexus Browser/CDP sem segundo navegador
**Issues:** #83 #107 #111 #112. **PRs:** tabs/sessões runtime; richer inspection; QA segurança.
- Preservar internal/browsercontrol + lazy browser_open + MCP scoped grant. Inspirar-se no OMP tab supervisor para navegação de múltiplas abas/surfaces, não transplantar backend.
- Oferecer inspector de DOM, console/network sanitizados, viewport e visual diff somente nas permissões existentes; takeover humano protegido.
- Browser pertence ao Core; se owner Maestro, negociar grants específicos. Proibido delegar arbitrariamente acesso a qualquer site interno/arquivo.
**Aceite:** dois agents, dois clients, browser separado por runtime/projeto, MFA sem vazamento, revoke, SSRF/cross-project negatives, Chromium real no SHA.

## Entregas de cada fase

Cada fase é uma cadeia de PRs curtos: (1) contrato/backend; (2) apresentação/UX; (3) testes/E2E/docs. Cada PR contém esquema e migração compatível, provas de operação/segurança, atualização de announcement/tour se UI visível, i18n PT/EN/ES, WCAG 2.2 AA, 390px/tablet/desktop e qualidade Go/TS. Não passar à fase seguinte sem integrar a fase anterior na release e validar dependências/contratos no SHA resultante.

Gates transversais:
1. Dois agentes, dois worktrees, dois clientes e interrupções reais nos recursos concorrentes.
2. Owner único, leases, permissionamento, revogação, fallback explícito e ausência de autoaprovação.
3. Tests unit/integration/race quando pertinentes; Playwright/Chromium; Linux/Windows/macOS quando requer PTY/ConPTY/HUD.
4. Evidence Bundle do SHA, versão de Maestro/CLI, screenshots reais, logs sanitizados, testes com exit code.
5. **NÃO fechar issue, publicar anúncio ou marcar SUPPORTED com base somente em arquivo, PR merged ou mock.**
6. PRs direcionados exclusivamente a release/v1.0.0; não mexer em main; OMP integration #150 fora deste fluxo.

## Ordem operacional resumida

00 contrato Maestro/migração → 01 papéis/modelos → 02 jobs/atividade → 03 Director → 04 árvore/timeline → 05 fork/comparação → 06 checkpoints → 07 recuperação → 08 rewind → 09 compaction/context notes → 10 retain/recall/reflect → 11 learned skills → 12 importação de sessões → 13 TTFT/traces → 14 Usage/custos/cache → 15 Browser/CDP.

**Estimativas de esforço (planejamento, não prazos):** 00 L, 01 M, 02 M, 03 L, 04 M, 05 L, 06 M, 07 L, 08 M, 09 M, 10 L, 11 M, 12 M, 13 M, 14 M, 15 L. Cada letra considera risco e testes nos dois repositórios; não fornece compromisso temporal.

## Referências

- https://github.com/can1357/oh-my-pi/blob/main/docs/vibe-mode.md
- https://github.com/can1357/oh-my-pi/blob/main/docs/agent-hub.md
- https://github.com/can1357/oh-my-pi/blob/main/docs/session.md
- https://github.com/can1357/oh-my-pi/blob/main/docs/compaction.md
- https://github.com/can1357/oh-my-pi/blob/main/docs/memory.md
- https://github.com/can1357/oh-my-pi/blob/main/docs/tools/browser.md
- https://github.com/IAPro-Community/Orquestrador-Maestro/blob/v1/runtime/bridge/bridge.js
- https://github.com/IAPro-Community/Orquestrador-Maestro/blob/v1/orquestrador/bin/memory.js
- docs/product/agent-workspace-v1-acceptance.md
- docs/design/universal-agent-harness-support.md
- docs/architecture/session-continuity.md
- docs/product/nexus-browser-auto-mcp.md

## Complemento: padrões concretos extraídos do source OMP

- **Direção e recuperação:** vibe/runtime.ts mantém worker owner+parentSessionId, eventos de spawn/turn/tombstone em vibe/lifecycle.ts, filas para next-turn quando steering em andamento não é suportado. No Nexus, owner/generation fence e fila tipada devem substituir qualquer tentativa de transcrever o worker em memória própria.
- **Jobs confiáveis:** async/job-manager.ts possui estados tipados, max running, retry/backoff de entrega e descarte/cleanup após consumo. A evolução Nexus aproveita essas **invariantes** em Mission/Run/notification existentes, sem importar o JobManager OMP.
- **Browser lifecycle:** tools/browser/tab-supervisor.ts diferencia criador da aba de quem a reutilizou, controla sessões vivas/mortas, cancelamento de operações pendentes quando uma aba fecha, lifetime/idle cleanup. Nexus precisa preservar esses atributos em seus próprios grants/leasing por runtime, evitando que um viewer encerre aba de outro owner.
- **Telemetria:** packages/agent/src/telemetry.ts utiliza campos tipados compatíveis com OpenTelemetry para input/output/cache-read/cache-write, erro e custo estimado, além de atributos próprios e distinção de cache de gateway versus cache de prompt. Na evolução Nexus, adaptar a semântica, não a implementação, mantendo unknown em dados ausentes e nenhuma equivalência API confundida com cobrança.
- **Ordem de validação:** primeiro contrato e evidência de source; depois UI; depois E2E, com PRs menores e guard rails de permissões. Fonte source pin OMP main@d33a0edb.
