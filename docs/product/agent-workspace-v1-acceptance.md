# Nexus 1.0 — Contrato obrigatório de aceite do Agent Workspace

> **Escopo aprovado:** todas as experiências e capacidades listadas aqui pertencem à linha única `release/v1.0.0`, e não a uma 1.1/1.2. Incluir no escopo não significa que o código está concluído. O release permanece **NO-GO** enquanto houver item sem prova no SHA final.

**Fonte de verdade estruturada:** [`docs/reference/agent-workspace-v1.json`](../reference/agent-workspace-v1.json).

## Regra arquitetural permanente

Nexus controla Workspace OS, PTY/ConPTY, sessões que inicia, leases, quotas, browser e apresentação. O Maestro funciona independentemente do Nexus, e é o núcleo de memória/contexto/skills/metodologia obrigatório na arquitetura alvo do Nexus. `executionOwner=NEXUS|MAESTRO` determina quem executa, cancela e reinicia cada Run. Não criar segundo scheduler/runtime nem segundo banco semântico. Preservar migração reversível do legado.

## Definição transversal de pronto

1. Código integrado diretamente na `release/v1.0.0` e refletido na distribuição exata, com testes unit/integration/E2E pertinentes.
2. Cenário real com pelo menos **2 Agents e 2 clientes** quando há interação multiagente ou remota; validar isolamento de projeto/worktree, credenciais, input lease e reconexão.
3. Acessibilidade WCAG 2.2 AA, PT/EN/ES, 390px/tablet/desktop, temas e redução de movimento quando aplicáveis.
4. Linux amd64, Windows amd64 e macOS arm64 certificados no **mesmo SHA imutável**, inclusive PTY/ConPTY, browser, Desktop/HUD e instalador.
5. Evidência objetiva: logs sanitizados, comandos e exit codes, screenshots reais com proveniência, run IDs, PR/commit e resultados vinculados ao SHA. Não aceitar testes só escritos, claims de modelo ou documentação como prova.
6. Release só pode avançar se o gate de escopo, os workflows native exact-SHA e os gates legais/security da publicação passarem.

## Matriz de obrigações

Legenda: **present** indica código-base localizado; **partial** significa lacuna de integração/UX; **missing** significa funcionalidade ainda não localizada como experiência completa. Todos os itens têm `verification=pending` até prova externa.

### As 7 experiências do Workspace

| ID | Requisito da V1 | Source | Issue | Evidência mínima |
| --- | --- | --- | --- | --- |
| `terminal-studio` | Terminal Studio | present | [#78](https://github.com/IAPro-Community/nexus/issues/78) | Native CLI em tab/split sem relançar provider; Control lease, Ctrl+C, setas, scrollback e resize PTY/ConPTY com dois clientes |
| `agent-command-center` | Agent Command Center (Orbit) | partial | [#104](https://github.com/IAPro-Community/nexus/issues/104) | Cartões de todos os Agents por projeto com estados observados e seleção direta; Trabalhando/aguardando/erro/concluído sem inferir estado do texto do CLI |
| `review-studio` | Review Studio | partial | [#109](https://github.com/IAPro-Community/nexus/issues/109) | Diff, teste, evidência e terminal no mesmo workspace; Review read-only por padrão com autorização explícita para aplicar mudanças |
| `browser-studio` | Browser Studio | partial | [#107](https://github.com/IAPro-Community/nexus/issues/107) | BrowserSurface integrada a terminal, diff e Agent Inspector; Permissões de CDP/MCP, takeover humano e sessão de projeto isolada |
| `comparison-lab` | Comparison Lab | missing | [#108](https://github.com/IAPro-Community/nexus/issues/108) | Duas alternativas paralelas com worktrees e execução isolados; Comparador de testes/diffs/custo e decisão humana sem merge automático |
| `remote-companion` | Remote Companion | partial | [#111](https://github.com/IAPro-Community/nexus/issues/111) | Mobile Inbox e terminal remoto vinculados ao host; Troca de dispositivo sem reiniciar sessão, com lease e revogação |
| `mission-observatory` | Mission Observatory | partial | [#85](https://github.com/IAPro-Community/nexus/issues/85) | Visualização read-only de Missions, dependências, agentes e provas; ExecutionOwner controla mutações, retries e cancelamento |

### Organização e navegação (6)

| ID | Requisito da V1 | Source | Issue | Evidência mínima |
| --- | --- | --- | --- | --- |
| `smart-layouts` | Smart Layouts | partial | [#105](https://github.com/IAPro-Community/nexus/issues/105) | Layouts sugeridos por atividade sem spawn implícito de processo; Preservação de surface IDs, foco, splits e restauração |
| `workspace-snapshots` | Workspace Snapshots | partial | [#105](https://github.com/IAPro-Community/nexus/issues/105) | Salvar e restaurar estado visual versionado por projeto; Não ressuscitar processo morto nem duplicar runtime |
| `agent-groups` | Agent Groups | missing | [#105](https://github.com/IAPro-Community/nexus/issues/105) | Agrupar Agents por tarefa, worktree e feature; Filtrar e reordenar preservando identidade e sessão |
| `quick-switcher` | Quick Switcher | present | [#84](https://github.com/IAPro-Community/nexus/issues/84) | Command Palette pesquisa projetos, Agents, ações e superfícies; Atalhos e foco confiáveis em desktop e mobile |
| `follow-agent` | Follow Agent | missing | [#105](https://github.com/IAPro-Community/nexus/issues/105) | Acompanhar visualização selecionada de Agent com pausa explícita; Não adquirir lease de input sem consentimento |
| `floating-inspector` | Floating Inspector | partial | [#104](https://github.com/IAPro-Community/nexus/issues/104) | Inspector destacável com status, identidade, evidência e lineage; Somente estado observado e informação já autorizada |

### Execução e supervisão (6)

| ID | Requisito da V1 | Source | Issue | Evidência mínima |
| --- | --- | --- | --- | --- |
| `agent-parking` | Agent Parking | missing | [#114](https://github.com/IAPro-Community/nexus/issues/114) | Diferenciar ocultar superfície de suspender runtime quando suportado; UI não diz que processo está pausado se só ficou oculto |
| `attention-radar` | Attention Radar | partial | [#100](https://github.com/IAPro-Community/nexus/issues/100) | Consolidar aprovações, perguntas, erros e blockers de CLIs heterogêneos; Ações direcionadas a geração e provider corretos com dados verificáveis |
| `focus-queue` | Focus Queue | partial | [#100](https://github.com/IAPro-Community/nexus/issues/100) | Fila de pendências priorizadas por severidade e tempo; Deduplicação, idempotência e resposta sem reabrir ações resolvidas |
| `agent-timeline` | Agent Timeline | partial | [#114](https://github.com/IAPro-Community/nexus/issues/114) | Linha de tempo de launch, attach, quota, handoff e recovery; Eventos com timestamp, source, runtimeGeneration e correlação |
| `resource-recommendations` | Resource Recommendations | partial | [#87](https://github.com/IAPro-Community/nexus/issues/87) | Recomendações com elegibilidade e motivo para conta/modelo; LIVE/CACHED/UNKNOWN explícitos; respeitar PIN e quota |
| `runtime-health` | Runtime Health | partial | [#114](https://github.com/IAPro-Community/nexus/issues/114) | Diagnosticar lease, PTY, heartbeat, PID e provider capability; Erros e UNKNOWN visíveis sem afirmar recuperação falsa |

### Multiagentes e revisão (6)

| ID | Requisito da V1 | Source | Issue | Evidência mínima |
| --- | --- | --- | --- | --- |
| `agent-compare` | Agent Compare | missing | [#108](https://github.com/IAPro-Community/nexus/issues/108) | Comparar duas implementações executadas em worktrees independentes; Testes e custo com evidência e critérios explícitos |
| `review-handoff` | Review Handoff | partial | [#110](https://github.com/IAPro-Community/nexus/issues/110) | Enviar diff de Agent para revisão independente delimitada; Resposta e proof vinculados ao dono do Run sem duplicar execução |
| `consult-another-agent` | Consult Another Agent | partial | [#110](https://github.com/IAPro-Community/nexus/issues/110) | Solicitar segunda opinião com contexto/scope do Maestro; Evitar loops, auto-delegação e ações externas não autorizadas |
| `change-radar` | Change Radar | partial | [#109](https://github.com/IAPro-Community/nexus/issues/109) | Arquivos e hunks alterados agrupados por Agent/worktree; Aviso de colisão com base commit e revisão humana |
| `test-matrix` | Test Matrix | partial | [#109](https://github.com/IAPro-Community/nexus/issues/109) | Exibir resultados reais dos comandos de teste por worktree; Exit code, SHA, timestamp e logs; nenhuma declaração LLM vira PASS |
| `release-desk` | Release Desk | missing | [#109](https://github.com/IAPro-Community/nexus/issues/109) | Vincular evidências, PRs, issues, diff e gates de uma release; Sem merge/publicação sem aprovação e checks verificáveis |

### Navegador e testes visuais (6)

| ID | Requisito da V1 | Source | Issue | Evidência mínima |
| --- | --- | --- | --- | --- |
| `browser-pane` | Browser Pane | present | [#107](https://github.com/IAPro-Community/nexus/issues/107) | Abrir Browser como superfície real do Workspace OS; Sem iframe como substituto de browser automation |
| `design-inspect` | Design Inspect | missing | [#107](https://github.com/IAPro-Community/nexus/issues/107) | Selecionar elemento e coletar DOM/estilos/screenshot sanitizados; Encaminhar contexto apenas a Agent explicitamente escolhido |
| `live-preview` | Live Preview | partial | [#107](https://github.com/IAPro-Community/nexus/issues/107) | Exibir app web local autorizada vinculada ao processo host; Não expor host arbitrário nem segredos de sessão |
| `browser-automation` | Browser Automation | partial | [#83](https://github.com/IAPro-Community/nexus/issues/83) | MCP e CDP com capacidades verificáveis e grant temporário; Intervenção humana para MFA, revoke, isolamento e auditoria |
| `visual-diff` | Visual Diff | missing | [#107](https://github.com/IAPro-Community/nexus/issues/107) | Comparar antes/depois por screenshot determinística; Região e limiar declarados, revisão e artefatos rastreáveis |
| `device-preview` | Device Preview | partial | [#107](https://github.com/IAPro-Community/nexus/issues/107) | Viewports de telefone, tablet e desktop na mesma browser session; Escalas, foco, acessibilidade e redimensionamento correto |

### Remoto e personalização (6)

| ID | Requisito da V1 | Source | Issue | Evidência mínima |
| --- | --- | --- | --- | --- |
| `mobile-inbox` | Mobile Inbox | partial | [#111](https://github.com/IAPro-Community/nexus/issues/111) | Perguntas/approvals e jobs no celular vinculados à sessão; Responder uma vez, inclusive em reconexões ou falhas de rede |
| `remote-terminal` | Remote Terminal | partial | [#111](https://github.com/IAPro-Community/nexus/issues/111) | Conectar ao terminal existente sem criar novo runtime; Controle único com token scoped, CSRF, expiry e revoke |
| `workspace-sharing` | Workspace Sharing | missing | [#112](https://github.com/IAPro-Community/nexus/issues/112) | Compartilhar observação por projeto/Agent sob RBAC; Viewer-only inicial; cross-project leakage negado e revoke imediato |
| `voice-actions` | Voice Actions | partial | [#114](https://github.com/IAPro-Community/nexus/issues/114) | Ação de voz exige controle explícito e identifica Agent de destino; Não inserir Enter automaticamente nem ler segredos do prompt |
| `contextual-hud` | Contextual HUD | partial | [#114](https://github.com/IAPro-Community/nexus/issues/114) | HUD exibe apenas estado/sinal real de Agent e atenção; Não bloquear janelas; reduced-motion e smoke nativo |
| `nex-companion` | Nex Operational Companion | partial | [#114](https://github.com/IAPro-Community/nexus/issues/114) | Mascote abre a ação operacional correspondente a evento confirmado; Respeitar NEX identity lock, alto contraste e motion reduction |

### Diferenciais adicionais (4)

| ID | Requisito da V1 | Source | Issue | Evidência mínima |
| --- | --- | --- | --- | --- |
| `workspace-recipes` | Workspace Recipes | missing | [#105](https://github.com/IAPro-Community/nexus/issues/105) | Recipes Frontend/Backend/Review/Debug/Release são layouts reutilizáveis; Receita não inicia Agents ou ações automaticamente |
| `parallel-challenge` | Parallel Challenge | missing | [#108](https://github.com/IAPro-Community/nexus/issues/108) | Distribuir uma especificação a múltiplos Agents isolados e compará-los; Sem merge automático, custo UNKNOWN não vira zero |
| `change-collision-detector` | Change Collision Detector | missing | [#109](https://github.com/IAPro-Community/nexus/issues/109) | Detectar alterações conflitantes por base commit/hunks; Informar incerteza e permitir revisão antes do merge |
| `device-handoff` | Seamless Device Handoff | partial | [#111](https://github.com/IAPro-Community/nexus/issues/111) | Continuar no mobile a sessão do host sem duplicar execução; Lease, reattach, estados reconciliados e histórico auditável |

### Dependências obrigatórias e certificação (8)

| ID | Requisito da V1 | Source | Issue | Evidência mínima |
| --- | --- | --- | --- | --- |
| `maestro-managed-dependency` | Maestro Mandatory Managed Dependency | missing | [#113](https://github.com/IAPro-Community/nexus/issues/113) | Nexus localiza/instala versão compatível e negocia capacidades; Maestro permanece standalone sem dependência do Nexus |
| `maestro-canonical-memory` | Maestro Canonical Semantic Memory | partial | [#113](https://github.com/IAPro-Community/nexus/issues/113) | Migrar IntelligenceMemory com backup, dry-run, idempotência e rollback; Nexus retém somente checkpoint operacional e nunca segunda store semântica |
| `single-execution-owner` | Single Execution Owner | partial | [#113](https://github.com/IAPro-Community/nexus/issues/113) | Cada Run tem executor único NEXUS ou MAESTRO; Retry, cancel e terminal input seguem o owner; não há duplo supervisor |
| `composer-flow-onboarding` | Composer, Flow e tour da release | partial | [#102](https://github.com/IAPro-Community/nexus/issues/102) | Concluir Composer/Flow e tours por componente reutilizável; E2E na distribuição final, PT/EN/ES, teclado e cenários reais |
| `remote-generated-outputs` | Remote Generated Outputs | partial | [#94](https://github.com/IAPro-Community/nexus/issues/94) | Listar e visualizar imagens, vídeo e arquivos gerados no host; ACL por projeto, MIME/hash, TTL e não expor filesystem indiscriminadamente |
| `two-client-real-e2e` | Two-Client Real Agent Homologation | missing | [#81](https://github.com/IAPro-Community/nexus/issues/81) | Dois clientes simultâneos e dois Agents em worktrees distintos; Leases, conflitos, handoffs, browsers e reconexões autenticados |
| `cross-platform-exact-sha` | Cross-Platform Exact-SHA Certification | missing | [#27](https://github.com/IAPro-Community/nexus/issues/27) | Linux/Windows/macOS mesma revisão em release candidate; PTY/ConPTY, Browser, Desktop, HUD, install e artefatos rastreáveis |

## Integração com o processo de distribuição

`node scripts/verify-agent-workspace-v1.mjs` valida estrutura/integridade do escopo (deve passar no CI, sem fingir aceite).

`node scripts/verify-agent-workspace-v1.mjs --release --sha <SHA> --evidence <ARQUIVO>` é **fail-closed** e requer uma evidência externa e imutável do SHA alvo com cada capacidade aprovada e certificações por plataforma. Não existe bypass para status pendente.

O artifact de evidências deverá ser produzido pelos jobs E2E reais no CI; até essa entrega, o Signed Release não deve publicar nenhuma beta/estável da v1 com escopo incompleto. **Não** colocar um SHA fixo em um JSON versionado — um commit que altera esse JSON modifica seu próprio SHA.

Veja também [Nexus 1.0 — linha única de release](https://github.com/IAPro-Community/nexus/issues/27), [matriz atual](../reference/feature-matrix.md), [Maestro](maestro.md) e [publicação](../PUBLICATION_CHECKLIST.md).
