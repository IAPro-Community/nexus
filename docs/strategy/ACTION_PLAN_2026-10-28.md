# Nexus 1.0 — Plano de ação de implementação a partir da release (28/10/2026)

> **Documento de execução proposto, não relatório de conclusão.** Diagnóstico rastreável: [Auditoria de implementação dos 48 requisitos](RELEASE_IMPLEMENTATION_AUDIT_2026-10-28.md), fonte no SHA `1c67e3b6509cdfdd9d84cc79e3dfcef547b63828`. Branch canônica `release/v1.0.0`. O runtime deve sempre ser reavaliado contra HEAD antes de cada PR, para reconciliar desenvolvimento concorrente.
>
> **Importante:** este plano trata **implementação restante, UX e integração**. A homologação formal exact-SHA, assinaturas e releases são trilha própria e **não** podem ser consideradas executadas por finalizar estes PRs. As 48 obrigações continuam marcadas obrigatórias pelo contrato v1.0. A intenção do roadmap de adoção de diferir parte delas para marcos posteriores **não substitui** decisão explícita de escopo. Até resolução, não descartar itens de v1 nem promover versão estável.

## 0. Regra de engenharia para todos os PRs

1. Fazer `fetch` de `release/v1.0.0` e `compare` com baseline antes de editar. Inventariar APIs, modelos e stores existentes; **portar seletivamente** diferenças úteis; não aplicar PR antigo inteiro.
2. Reusar: `SessionHost` / `Runtime Registry` / eventos, `WorkspacePresentationState`, `WorkspaceLayoutService`, `ResourceScheduler`, `MissionRunner`, `WorkPlan`, `BrowserControl`, `artifacts`, `Maestro Bridge`. Não introduzir segundo PTY, process manager, workflow engine, quota source, memória semântica ou artifact registry.
3. Um PR deve entregar **vertical slice** utilizável e estados de erro/UNKNOWN — API, UI, teste, documentação, métricas se aplicável. Nenhuma feature flag ligada por padrão sem capacidade versionada.
4. Segurança por padrão: projeto/runtime/identidade explícitos, grants com TTL e revogação, CSRF/auth, fail-closed para writes, segredo fora de logs/artefatos, testes cross-project e de race. Nunca fazer merge/push, iniciar agente, enviar credenciais ou aprovar permissões automaticamente sem autorização.
5. Dois níveis de evidência independentes: (a) **implementação** com testes determinísticos/fakes, UX e diff revistos; (b) **homologação** live de provider e ambiente exact-SHA. O primeiro não finge o segundo.
6. Atualizar `docs/reference/agent-workspace-v1.json` e catálogo de claims apenas quando código e testes sustentarem a mudança; `docs/release-status.md` e gates de assinatura continuam fontes de certificação.
7. Se o recurso já existir, preferir **refatoração de integração** e remoção de caminhos redundantes; não reabrir épico como implementação do zero.

## 1. Ordem executável: 7 ondas, 29 fatias de PR

Cada fatia cabe idealmente em 1 PR revisável; algumas dependem de refatoração prévia e podem ser partidas. P0 = funcionalidade estruturante; P1 = diferencial de uso; P2 = ampliação. Tamanho S/M/L é complexidade relativa, **não cronograma**.

### Onda A — Inventário confiável, primeiro uso e estado universal [P0]

**Objetivo de saída:** iniciar Direct em projeto/provider escolhido; visualizar estado e intervenções corretos, sem duplicar runtime nem disparar side effects.

| PR | Issues | Mudança implementável, sem retrabalho | Aceite específico |
| --- | --- | --- | --- |
| A1 — contrato de capacidades/claims | #76, #79, #97–#101 | Confrontar `features.json`, `agent-workspace-v1.json`, `driver.Registry` e `EffectiveCapabilities`; distinguir TUI experimental de ACP/RPC realmente negociado. Corrigir docs/UX que dizem `NOT IMPLEMENTED` para infraestrutura presente, sem dizer `CERTIFIED`. | Teste de contrato impede UI de anunciar controle não testado; docs geradas e status não contraditórios |
| A2 — onboarding guiado existente | #77, #79 | Evoluir `WelcomeModal`, `ProviderReadinessBanner`, `firstRunReadiness`, `DirectSessionLauncher`: projeto → provider/conta → start → interação; versão/login UNKNOWN, reintento e rollback. Não recriar wizard separado. | Instalação com/sem provider mostra diagnóstico; dois perfis isolados; cancelamento sem Agent ou PTY órfão |
| A3 — projeção única de atenção | #100, #104, #78 | Ligar `agentwire/state` + eventos `structured_backend` + OSC7501/PTY ao Registry e `GlobalAttentionRadar`, com `runtimeId+epoch+interactionId` e origem `AUTHORITATIVE/SIGNALLED/INFERRED`. | Perguntas idênticas de 2 runtimes não se misturam; ACK ≠ concluído; input não confirma permission sozinho; UI aponta ao runtime certo |
| A4 — fluxo de aprovação e resposta | #100, #111 | Estender intervenções tipadas na mesma autorização existente: answer/select/reject ou `abrir terminal` quando capability desconhecida. Sem segundo inbox store. | Retentativa idempotente, autorização por sessão, revogação, erros stale/cross-project; um answer não resolve outro Agent |

**Dependências:** A1 precede A3/A4; A2/A3 podem ocorrer em paralelo em arquivos separados. `#78` mantém testes de recuperação contínuos em cada fatia.

### Onda B — Providers universais sem falsas promessas [P0/P1]

**Objetivo de saída:** suportar novos harnesses por capacidades reais, não só botões ou modelos associados.

| PR | Issues | Mudança | Aceite |
| --- | --- | --- | --- |
| B1 — ACP/JSONL contratos restantes | #97 | Completar version negotiation, limites/backpressure, cancelamento, resposta de permission e vínculo sessão/perfil no `agentwire.Bridge` + `SessionHost`. Preservar PTY padrão. | Fakes com respostas fora de ordem, frames grandes, process exit, startup fail, stale approvals e no double-run |
| B2 — Pi/OMP integrações verificáveis | #98 | Evoluir `PreviewHarnessDriver` e `NewAgentModal`, modo `pi_rpc`, extension UI e estado `agent_settled`; revisar sintaxe/versionamento por instalação. | Dois perfis, duas sessões; pergunta/resposta, stop/recover e versão não suportada fail-closed |
| B3 — MiMo/DSH ACP | #99 | Usar mesma bridge, probe `mimo acp`/`dsh acp`; DSH não pode fingir terminal PTY. Capabilities reais e fallback explícito. | Lifecycle, auth, sessão, cancelamento e respostas estruturadas sem fabricar `SUPPORTED` |
| B4 — compatibilidade/discovery | #101 | Doctor/version probe/status para demais CLIs; abrir implementação nova apenas quando demanda + protocolo testados. | Matriz `INSTALLED / COMPATIBLE / EXPERIMENTAL / UNSUPPORTED`, sem driver dummy no catálogo |

**Nota:** `ACP`, `JSONLines`, `PiRPC` e `structuredBackend` **já estão no source**; B1–B3 são fechamento de interoperabilidade, não criação do transporte inicial. Executar B2/B3 progressivamente; não bloquear o Workspace principal por todos os harnesses.

### Onda C — Workspace multiagente com ownership de verdade [P0/P1]

**Objetivo de saída:** dois Agents trabalhando em paralelo em Git worktrees independentes, operação fácil pelo Workspace e layout que sobrevive a refresh.

| PR | Issues | Mudança | Aceite |
| --- | --- | --- | --- |
| C1 — Command Center + Inspector | #104, #80 | Compor `AgentsSurface`, runtime registry/projection e `WorkspaceSurfaceHost` em lista global de Agents com filtros, foco e inspector. Exibir provider/profile, status observado/UNKNOWN, generation, worktree, owner e quota com provenance. | Agente selecionado abre runtime correto; troca de projeto e refresh não inicia processo |
| C2 — worktrees operacionais | #80, #81 | Consumir `handlers_git_worktrees.go` no frontend; associar `ensureAgentWorktree` a Agent e exibir branch/dirty/locked. Criar/remover apenas com autorização e proteção de processo/changes. | Dois worktrees reais, checkout em caminhos distintos, limpeza negativa com dirty/active |
| C3 — snapshots, groups e Follow Agent | #105, #80, #84 | Reusar `WorkspacePresentationState`, receitas já existentes e `WorkspaceLayoutService`; adicionar snapshots nomeados, grupos virtuais e follow com pause. Não confundir snapshot interno de modo com snapshot do usuário. | Restaurar layout não recria PTY; split ratio/active surface preservados; teclado e viewport 390px |
| C4 — colisões e cenário multiagente | #81, #108, #109 | Detectar alterações simultâneas de arquivos/hunks por worktree; antes de permitir Compare/Release, resolver conflitos e conservar ownership. | Dois providers, dois clientes autenticados, um frontend desmontado/reconectado, sem writer duplicado |

**Dependências:** C1/C2 antes de C3/C4. O backend worktree e os 4 modos `TABS/DESKTOP/MOSAIC/SPLIT` já existem.

### Onda D — Do trabalho ao resultado revisável [P1]

**Objetivo de saída:** diff+test+evidência+feedback disponíveis no mesmo lugar antes de uma ação Git.

| PR | Issues | Mudança | Aceite |
| --- | --- | --- | --- |
| D1 — Review Studio | #82, #109 | Projetar `runner.review` e `EvidenceBundle` em arquivos/hunks/testes; comentários por path/line com referência ao run/revision. | Read-only inicial, arquivo/teste em até duas interações; revisão persiste refresh |
| D2 — Test Matrix e Change Radar | #109, #82 | WorkReceipts com `command/exit code/sha/ambiente/logs`, seleção por Agent; collision detector do C4. | Texto do LLM nunca vira teste PASS; regressão/unknown explícitos |
| D3 — Mission Observatory + Review Handoff | #85, #110, #102 | Integrar `FlowRunSurface`, MissionRunner, EvidenceBundle e delegação; exibir estado por attempt, retries, proof e parecer do agente revisor. | Scenario success/failure, restart idempotente, `COMPLETED_VERIFIED` só com DoD real |
| D4 — issue→PR/Release Desk | #89, #109 | Integração GitHub de escopo mínimo: issue/Agent/worktree/commit/PR/check com preview de diff e confirmação de push/merge. | Revogação de token bloqueia ações novas; nenhum autopush; read-only sem token de escrita |
| D5 — Comparison Lab | #108 | Reusar C2/C4 e D1/D2: executar 2 soluções isoladas, comparar diffs/testes/custo, escolher final manualmente. | 2 agentes mesmo objetivo em worktrees isolados; escolher/descartar sem merge tácito |

**Dependências:** C2, C4 → D1/D2; D1/D2 → D3/D4/D5. Composer/Flow recebem adequações em cima do sistema existente, não novo orquestrador.

### Onda E — Browser Studio + mobilidade real [P1]

| PR | Issues | Mudança | Aceite |
| --- | --- | --- | --- |
| E1 — browser inspect e visual diff | #83, #107 | Sobre `BrowserSurface`/CDP, implementar captura seletor/DOM/estilos/crop redigidos e diff visual com baseline/threshold; ligar ao Agent escolhido. | 1 screenshot/elemento com autoria e sem segredos; diff com threshold explícito; handoff humano preservado |
| E2 — device preview e browser review | #107, #84 | Browser pane lateral a terminal/diff, viewport real (mobile/tablet/desktop), URL e session lease estáveis. | Mesmo Browser ID, dois clientes não controlam juntos; login/MFA humano |
| E3 — Outputs: completar, não refazer | #94, #88 | Evoluir `artifacts` e `OutputsSurface`: provenance automática `agentId/runtimeId/runId`, limites/TTL de blobs, MIME e streaming maior; downloads/versões e galeria já existem. | Artefato produzido por Agent aparece no celular; original verificável e sem path traversal |
| E4 — Mobile Inbox + device handoff | #111, #100 | Reusar radar/Registry/auth; aprovações remotas idempotentes, transferir lease de input sem relaunch do runtime, alertar sobre conexão offline. | Celular e desktop revezam input com owner único; autenticação/CSRF/revogação |
| E5 — Workspace Sharing/RBAC | #112, #91 | Threat model e modelo de grants `observer read-only`; implementar somente quando projeto/identidade forem isoláveis. Browser grants **não são** workspace RBAC. | Duas identidades com access control, cross-project denials, revogação WS e logs sanitizados |

**Dependências:** E4 depende de A4/C1; E3 depende de isolamento por projeto já presente; E5 depende de autorização explicitamente desenhada, nunca usar o túnel público como RBAC.

### Onda F — Maestro canônico e retenção de contexto [P0/P1]

| PR | Issues | Mudança | Aceite |
| --- | --- | --- | --- |
| F1 — contrato e dependência gerenciada | #86, #113 | Negociar Bridge `context.brief`, versão/capabilities, provenance e scope. Instalador gerencia Maestro somente onde autorizado; preserva instalação independente. | Instalação, atualização, modo degradado e rollback sem perder Direct/contas |
| F2 — migração reversível de IntelligenceMemory | #86, #113 | Export, backup, mapear hábito/fato/projeto, backfill idempotente por digest e confirmação; gravar semântico **só no Maestro** ao final. | Fixture com dados legados: migra→reconecta→rollback sem perda nem duplicação |
| F3 — único dono de execução | #113, #110, #85 | Fencing de Run owner `NEXUS/MAESTRO`, impedir dual dispatch, tipar handoffs de contexto e parecer. | Restart/crash e retry não reexecutam side effect com estado desconhecido |

**Decisão arquitetural:** Maestro continua independente; Nexus não assume memória semântica própria nem adiciona fallback silencioso. Pode executar F1 paralelo às ondas C/D; F2/F3 exigem contratos nos dois repositórios.

### Onda G — acabamento, UX/Docs e lançamento Community [P1/P2]

| PR / trabalho | Issues | Mudança | Aceite |
| --- | --- | --- | --- |
| G1 — Composer/Flow UX e tours | #102, #103, #79 | Fechar 6 estados de tour existentes com âncoras/i18n; avaliar Driver.js/Joyride antes de ampliar spotlight manual. | Sem segundo engine; cenários vazios/erro/preview/GO com navegação de teclado e temas |
| G2 — timeline, saúde, voz e HUD | #114, #63 | Alimentar HUD/NEX com estado real do Core; health/timeline e foco acessível; voice requer confirmação. | Reduced motion, sem alertas espúrios, consumo limitado e erros explícitos |
| G3 — conteúdo e experiência de adoção | #76, #79, #93 | Atualizar Quick Start/site/docs por capacidades verificadas; coletar fricções de uso, concorrência e tempo até primeira sessão. | Casos Direct, dois Agents e Mission demonstráveis com status correto; nenhuma claim de certificação sem fonte |
| G4 — segurança e licenciamento Community | #65, #74, #90, #96 | Administrador habilita private vulnerability reporting e rulesets, review de propriedade/PRs/assets e autoriza eventual mudança para Apache-2.0. | Autorizações documentadas fora do repo público; política de PR externo segura; `LICENSE` só muda após clearance |

**Não antecipar** #92 (workers SSH/containers) e #91 (enterprise) como dependência do MVP local; estudar após existência de isolamento/autorização. NEX visual #119/#120 deve ser desenvolvido sem criar segundo mascote ou mudar o lock de identidade.

## 2. Mapa explícito de responsabilidades e issues sobrepostas

| Dono único existente | Issues a reconciliar | Não duplicar |
| --- | --- | --- |
| `artifacts` + `OutputsSurface` | #94 e #88 | Artifact registry/storage e galeria paralela |
| `maestro.go` / Bridge + `IntelligenceMemory` legado | #86 e #113 | Banco semântico, processo Maestro próprio no Nexus |
| `WorkspacePresentationState` + Worktree Core | #80, #81, #104, #105 | LayoutService, Agent scheduler, PTY |
| `runner.review` + `EvidenceBundle` | #82, #85, #89, #109, #110 | Ledger de review ou runner de Mission |
| `BrowserControl` / Browser MCP | #83, #107 | Outra instância padrão de Chromium e grant policy |
| `Registry` / `agentwire` / `GlobalAttentionRadar` | #78, #97–#100, #111 | Read model de status paralelo e permissões inferidas |

## 3. Sequência crítica e paralelização segura

```text
A1 (claims/capabilities) ── A2 (primeiro uso)
        │
        ├── B1 ── B2 (Pi/OMP) ── B3 (MiMo/DSH) ── B4
        │
        └── A3 (estados) ── A4 (intervenções) ───── E4 (Mobile Inbox)
                             │
C1 (Command Center) ── C2 (worktrees) ── C4 (colisões)
         └──────────── C3 (snapshots / grupos / follow)
C2/C4 ── D1 (review) ── D2 (tests/changes) ── D3 (Missions)
                                ├── D4 (GitHub/Release Desk)
                                └── D5 (Comparison Lab)
Browser core ── E1 (inspect/diff) ── E2 (responsive)
Outputs core ── E3 (proveniência/download) ── E4
Auth grants ─────────────────────────── E5 (Sharing/RBAC)
F1 (Maestro contract) ── F2 (migration) ── F3 (execution owner)
G1/G2/G3/G4 acontecem após owners das respectivas interfaces estabilizarem
```

**Primeiros 4 PRs prioritários:** A1 (claims + capability), A3 (reconciliação universal), C1 (Command Center) e C2 (worktrees). A2 pode ocorrer em paralelo ao C1 se modificar arquivos disjuntos. B1 é relevante, mas não precisa bloquear adoção por Codex/OpenCode já existentes. F1 precisa começar cedo porque a migração entre repositórios pode ser demorada.

## 4. Critérios concretos de pronto para implementação (além de `CI verde`)

1. **Direct:** instalar/abrir, detectar provider/conta, iniciar sessão, input/resize, stop/recover, nenhum processo órfão em cenários negativos; sem credenciais pedidas na UI.
2. **Workspace:** dois Agents/2 worktrees com processos independentes; split persistente; refresh sem second launch; conflitos visíveis e controlados; estado UNKNOWN preservado.
3. **Atenção:** perguntas/aprovações são vinculadas a `runtimeGeneration` e `interactionId`, confirmação atômica e observação do resultado; nada é marcado resolvido por texto de terminal/Enter.
4. **Browser:** grants e lease limitados; intervenção manual para login; outro cliente não assume controle simultâneo; inspect redigido; screenshot diffs versionados.
5. **Review/Missions:** diff, comando/exit code e WorkReceipt rastreáveis; nenhum LLM-text-as-PASS; git push/merge apenas com confirmação.
6. **Outputs/mobile:** galeria existente preservada; origem agent/run quando conhecida; autorização negativa entre projetos; download original, limite MIME/tamanho e arquivo não modificado.
7. **Maestro:** contexto canônico por projeto/branch/worktree, versão/digest; export/import/rollback demonstrados; isolamento e 1 owner por Run.
8. **Community:** licença vigente honesta, contribuidor externo sem runner self-hosted e canal de segurança privado ativo.
9. **UX/a11y:** desktop + viewport 390px, i18n PT/EN/ES conforme escopo, teclado/foco, contraste e reduced motion sem CSS ad hoc descontrolado.

## 5. Conflito documental de escopo — decisão necessária

Os dois documentos atuais são incompatíveis quanto ao que é obrigatório antes de declarar 1.0 pronta:
- [Contrato de aceite](../product/agent-workspace-v1-acceptance.md) inclui 48 capacidades como requisito `releaseRequired`, incluindo Comparison Lab, Visual Diff, Sharing e Maestro gerenciado.
- [Roadmap de adoção](NEXUS_ADOPTION_EXECUTION_PLAN_2026.md) organiza muitas dessas entregas em marcos M1/M2/M3 e explicitamente afirma que um backlog futuro não vira gate automático da v1.0.

**Proposta conservadora:** preservar as 48 como escopo obrigatório da versão 1.0 até decisão expressa do mantenedor; manter implementação em fatias que convivam com feature flags/estado parcial e sem desativar gates. Se a 1.0 puder lançar sem algum item, formalizar um ADR com versão-alvo, feature IDs exatos, impacto no usuário, rollback e alteração explícita do contrato e documentação comercial. Não alterar o contrato silenciosamente.

## 6. Acompanhamento de execução

- Cada fatia recebe PR com rótulo da issue principal + referências às secundárias, owner, risco, links para código/antes/depois, impacto de migração e critérios automáticos de teste. **Não criar issues duplicadas** das #76–#115.
- Cada PR passa por review de auth/project boundaries e pelo `Community PR Checks` aplicável. Se falhar por dependência anterior, registrar como bloqueio, nunca mascarar resultado.
- O progresso é medido em fatias realmente concluídas e capacidades do contrato atualizadas com prova de código, não em quantidade de commits, imagens ou issues fechadas.
- A homologação nativa exact-SHA, evidência para os 48 itens e publicação/assinatura continuam sob #27/#115, **trilha separada após cada integração funcional**.
- Revisar o plano quando o HEAD da release mover mais do que docs/builds: conferir `git diff` entre base e novo HEAD e remover tarefas já incorporadas.

**Próximo passo técnico direto:** iniciar A1 com mapeamento automatizado dos caminhos/claims e registrar current CapabilityEvidence para providers; em paralelo preparar C1 desenhando contrato de inspector sem criar nova fonte de estado.
