# Nexus — Plano de adoção e execução (2026)

> **Status: backlog proposto / não homologado.** Criado em 27/10/2026 a partir da análise competitiva e da inspeção de `release/v1.0.0`. Este documento orienta priorização e aferição de valor; **não** substitui `docs/release-status.md`, `docs/operations/platform-support.md`, os gates same-SHA nem uma decisão de mantenedor sobre o escopo de cada release.
>
> **Princípio:** não aumentar o número de features por si só; maximizar trabalho útil finalizado, rastreável, seguro e reutilizável por desenvolvedores que já usam Codex, Claude, OpenCode e outras CLIs.

## Problema de mercado e aposta

O Nexus deve justificar adoção não como um clone de IDE/Orca/N45/Cursor, mas como **workstation local + control plane independente** que reúne execução de agentes diferentes, isolamento de worktrees, supervisão, recursos/custos e entrega com provas verificáveis.

A adoção não exige abandonar o agente atual: o caminho de entrada é Direct; orquestração só aparece quando a complexidade a justifica. **Não** prometer interoperabilidade perfeita de transcript de provider, segurança multiusuário, disponibilidade de conta/modelo sem evidência ou certificação nativa não realizada.

### Contratos arquiteturais preservados

- Um `MissionRunner`, um `WorkPlan`, um `ResourceScheduler`, um terminal supervisionado e um gerenciador de layout. UI, CLI e Desktop são projeções do Core existente.
- `AgentMatcher` escolhe **quem**; `ResourceScheduler` escolhe **com qual recurso**; `MissionRunner` controla **estado de execução**; `WorkPlan` define **o que foi autorizado**.
- Maestro permanece produto independente. A arquitetura-alvo é Maestro como autoridade de memória semântica/skills/método; Nexus mantém checkpoints operacionais. Na beta, integração ainda é opcional e migração não foi concluída.
- Execução com políticas fail-closed; push, merge, deploy, rede, secrets e mudanças destrutivas demandam autorização explícita e gates próprios.
- Licença vigente **PolyForm Strict 1.0.0 (source-available)**; Apache-2.0 é apenas intenção condicionada ao clearance jurídico. Community completa não deverá depender de assinatura artificial quando a abertura for aprovada.
- `release/v1.0.0` segue candidata; nenhuma tarefa futura deste plano é automaticamente um gate novo para 1.0. Preservar freeze e separar desenvolvimento pós-v1 de estabilização.
- Desktop e sistemas operacionais: a matriz contemporânea é `docs/operations/platform-support.md`. **Nunca** promover plataforma sem certificação no SHA exato.

## Ordem executável, sem datas artificiais

Usar *marcos por evidência*, não estimativas cronológicas. P0 = bloqueia um caminho de adoção fundamental; P1 = entrega de diferenciação; P2 = expansão/escala; P3 = pesquisa/arquitetura futura. O tamanho S/M/L é relativo e não é SLA (S: PR pequeno; M: ~1–3 PRs; L: épico que precisa ser fatiado).

### M0 — Fundamentos confiáveis e primeiro uso (convergir com v1.0 sem ampliar escopo sem aprovação)

**Meta verificável:** instalar/abrir Nexus e iniciar Direct num projeto com um provider elegível, detectar falhas e recuperar sessões, com documentação honesta. O último SHA deve passar os gates já definidos antes de anunciar uma release.

| Issue | Prioridade | Entrega concreta | Dependência / observação |
| --- | --- | --- | --- |
| [#27](https://github.com/IAPro-Community/nexus/issues/27) | gate existente | Fechar homologação e linha única de release | não duplicar |
| [#76](https://github.com/IAPro-Community/nexus/issues/76) | P0 · S | Inventário auditável de claims, testes e certificações | #27 |
| [#77](https://github.com/IAPro-Community/nexus/issues/77) | P0 · M | Primeira execução fluida de CLI e conta conhecida | #76, #27 |
| [#78](https://github.com/IAPro-Community/nexus/issues/78) | P0 · M | Confiabilidade Direct/PTY, lifecycle e reanexação | #27 |
| [#79](https://github.com/IAPro-Community/nexus/issues/79) | P0 · S | Quick Start, tour e limitações alinhados com código | #76 |

**Critério de saída:** 5 instalações limpas em ambiente preparado, mediana <= 3 minutos entre primeira abertura e sessão interativa; Codex e OpenCode testados em pelo menos dois perfis distintos; 20 ciclos supervisionados de início/fechamento/reattach sem órfãos ou duplicação; evidências com SO/versão/commit. O objetivo de três minutos é **meta a testar**, não dado alcançado. Qualquer desvio vira bug com reprodução, não `PASS`.

### M1 — Um workspace que realmente reduz atrito

**Meta verificável:** dois coding agents reais trabalhando em paralelo com worktrees separados, status consistente entre dois clientes autenticados, revisão e controle de browser sem sobrescrever mudanças.

| Issue | Prioridade | Entrega concreta | Dependência / observação |
| --- | --- | --- | --- |
| [#80](https://github.com/IAPro-Community/nexus/issues/80) | P1 · M | Worktrees visíveis, splits e layout persistente | #78; PR #58 foi fechada **sem merge** |
| [#81](https://github.com/IAPro-Community/nexus/issues/81) | P1 · M | Prova E2E com dois agentes e dois clientes | #80 |
| [#82](https://github.com/IAPro-Community/nexus/issues/82) | P1 · M | Diff/review/feedback com confirmação de ações | #81 |
| [#83](https://github.com/IAPro-Community/nexus/issues/83) | P1 · M | Nexus Browser MCP, lease e takeover humano | #78, #81 e security gates |
| [#84](https://github.com/IAPro-Community/nexus/issues/84) | P1 · M | Responsive/accessibility: workspace, mobile, teclado | #80–#83; pode começar por auditoria |

**Critério de saída:** Codex + OpenCode, dois worktrees e dois clientes com E2E reproduzível; nenhuma gravação cruzada ou merge não autorizado; browser com um único controlador por lease, credenciais sanitizadas, fail-closed; fluxo desktop/mobile com teclado, foco visível e axe sem violações críticas.

### M2 — Entrega verificável e operações assistidas

**Meta verificável:** um projeto evolui desde tarefa aprovada até entrega inspecionável com testes, review, DoD, dados de custo diferenciados por proveniência e continuidade responsável. Artifacts podem ser visualizados remotamente apenas por autorização.

| Issue | Prioridade | Entrega concreta | Dependência / observação |
| --- | --- | --- | --- |
| [#85](https://github.com/IAPro-Community/nexus/issues/85) | P1 · L | Mission + Evidence Bundle navegado no próprio Workspace | #81, #82 |
| [#86](https://github.com/IAPro-Community/nexus/issues/86) | P1 · L | Contexto canônico via Maestro com migração reversível | contrato cross-repo |
| [#87](https://github.com/IAPro-Community/nexus/issues/87) | P2 · M | Recursos/quotas/custos explicáveis, UNKNOWN preservado | #85, scheduler existente |
| [#88](https://github.com/IAPro-Community/nexus/issues/88) | P2 · M | Imagens e outros artifacts acessíveis do mobile com ACL | #83, #84, segurança |
| [#89](https://github.com/IAPro-Community/nexus/issues/89) | P2 · M | Issue → Agent → commit → PR sob aprovação explícita | #82, #85 |

**Critério de saída:** dois cenários live de Mission (sucesso e bloqueio/falha); conclusão `COMPLETED_VERIFIED` somente após prova; restart sem replay perigoso; relatório distingue custo real, estimado e desconhecido; artifact gerado no desktop visualizado no mobile autorizado, com testes negativos de vazamento de dados; vínculo GitHub não faz push automático.

### M3 — Comunidade, diferenciação e enterprise responsável

**Meta verificável:** validar demanda antes de ampliar a plataforma; abrir o código sob licença alvo somente com clearance legal; planejar operações empresariais sem enfraquecer Community.

| Issue | Prioridade | Entrega concreta | Dependência / observação |
| --- | --- | --- | --- |
| [#90](https://github.com/IAPro-Community/nexus/issues/90) | P2 · M | Checklist jurídico para eventual Apache-2.0 | [#74](https://github.com/IAPro-Community/nexus/issues/74), [#65](https://github.com/IAPro-Community/nexus/issues/65), #27 |
| [#91](https://github.com/IAPro-Community/nexus/issues/91) | P3 · S | RFC segurança/RBAC/autorização por projeto/equipe | #83, #88, #89, #65 |
| [#92](https://github.com/IAPro-Community/nexus/issues/92) | P3 · S | RFC e spike isolado para SSH/containers/remoto | #91 |
| [#93](https://github.com/IAPro-Community/nexus/issues/93) | P2 · M | Pesquisa e benchmark de adoção, 5 devs e 2 projetos | #77, #81, #85 |

**Critério de saída:** relatório de benchmark com baselines, falhas e decisões de backlog; matriz de autorização e threat model antes de qualquer funcionalidade empresarial; abertura jurídica atestada antes de editar `LICENSE`; recursos de Team/Remote somente após RFC, financiamento e risco aceitos.

### Gates existentes que não devem gerar issues duplicadas

- [#27 — homologação v1.0](https://github.com/IAPro-Community/nexus/issues/27): autoridade do release, exact-SHA e plataforma.
- [#63 — Guardian click-through nativo](https://github.com/IAPro-Community/nexus/issues/63): componente independente da jornada de adoção; não promover UX de HUD sem seu gate.
- [#65 — canal privado para reportar vulnerabilidades](https://github.com/IAPro-Community/nexus/issues/65): antes de qualquer oferta externa de Browser/artifacts/Team.
- [#74 — direitos das contribuições externas](https://github.com/IAPro-Community/nexus/issues/74): condição jurídica, não simples review técnico.

## Métricas do produto

| Indicador | Como coletar / guardrail | Meta inicial proposta |
| --- | --- | --- |
| Tempo até primeiro Agent interativo | Instalações preparadas, cronômetro first-open → input aceito pelo PTY; separar falha de auth | mediana <= 3 min, n >= 5 |
| Estabilidade do terminal | 20 ciclos / SO + Provider, process tree e estado pós-close; não contar Windows como Linux | 0 runtimes órfãos e 0 duplicações no cenário |
| Trabalho paralelo | 2 agents + 2 worktrees + 2 browsers autenticados, hashes e Git status antes/depois | 0 escrita cruzada ou merge implícito |
| Browser takeover | Lease concorrente, revogação, falhas de conexão, dois devices | 1 controlador efetivo; 0 ações sem grant |
| Qualidade de entrega | 1 caso sucesso + 1 falha, comparação com critérios de aceite e checks reais | 0 `COMPLETED_VERIFIED` falso |
| Exposição de artifacts | Testes positivos e negativos de ACL, MIME, path traversal, tokens | 0 acesso não autorizado em E2E |
| Decisão de provider | Validar logs e prova de quotas, identidade de conta e motivo de fallback | 100% das recomendações exibidas com origem ou UNKNOWN |
| Valor percebido | 5 devs externos, 2 repos reais; pesquisa após tarefas comparáveis | identificar 5 principais fricções e uma razão observada de reutilização |

As metas orientam testes; nenhum número desta seção é alegação de desempenho atual. Os resultados devem registrar denominador, plataforma, provider, versão e SHA. Dados pessoais/transcrições não entram em telemetria de produto sem consentimento.

## Sequenciamento e proteção contra escopo infinito

1. **Agora:** triage #27 + #76–#79; consertar falhas antes de fazer demos. P0 aqui não autoriza desrespeitar freeze da v1.0 — só blockers aprovados entram na release.
2. **Depois da base estável:** #80 → #81 → #82; #83 pode rodar em paralelo depois de estabelecer as bases de segurança. #84 acompanha UX.
3. **Com workspace confiável:** #85 como vertical slice prioritário, enquanto #86 começa por RFC/contrato e migração conservadora. #87–#89 entram quando os owners básicos estiverem comprovados.
4. **Paralelo não bloqueante:** jurídico/comunidade (#74, #90), pesquisa (#93), RFCs (#91–#92). Sem modificar o runtime crítico só para demonstrar paridade com concorrentes.
5. **Todo marco:** demo real, cobertura de erros, testes de autorização e validação pelo menos em dois clientes. Sem screenshots sintéticas apresentadas como evidência nativa atual.

### Definition of Done transversal

- [ ] Há caso de uso, dono arquitetural, fluxo UX, critérios e risco documentados no issue.
- [ ] A mudança reutiliza contratos existentes em vez de criar segundo Core/scheduler/memória/layout/runtime.
- [ ] Tests automatizados e smokes correspondentes passam **no SHA da alteração**; se não, status BLOCKED/UNVERIFIED.
- [ ] Negativas: sem credenciais em logs/artefatos, sem escalada de autorização, sem acesso cruzado de worktree/projeto.
- [ ] Responsivo, foco por teclado, i18n e tema preservados; código segue `AGENTS.md` e padrões de engenharia.
- [ ] Docs e evidências atualizadas; release status e feature matrix não alterados sem lastro.
- [ ] A CI de PR documental valida links e contratos sem fabricar evidência visual; mudanças no código/assets seguem a verificação de proveniência completa, e a certificação de release depende de gates próprios same-SHA.
- [ ] PR revisado pelo maintainer; mudanças de arquitetura, licença, segurança e APIs passam por RFC/ADR quando aplicável.

## Registro de decisões e anti-metas

- **Não** reimplementar Cursor/VS Code, editor completo, terminal host, plataforma de billing, memória semântica paralela ou runtime remoto com privilégio amplo.
- **Não** lançar marketplace de agentes/skills/MCP nem dezenas de adapters antes de haver usuários reais com problema comprovado.
- **Não** tratar `PR closed` como código mergeado. Reconciliar explicitamente branch/contextual-workspace e histórico Orca antes de portar.
- **Não** medir sucesso por número de recursos, agents disparados, tokens consumidos ou demos geradas.
- **Não** prometer prazo de entrega até ter owners, disponibilidade, baseline e capacidade de CI/native runners.
- **Sim:** medição de tempo de tarefa, qualidade real, retrabalho, estabilidade de sessão e autonomia com intervenção humana compreensível.

## Responsabilidade e acompanhamento

Backlog mantido por issues numeradas acima. Recomenda-se revisão quinzenal pelo maintainer para revisar prioridades, dependências, blockers e evidências; isso é um processo sugerido, não um agendamento criado. Cada issue fechada deve apontar PR, commit, matriz de testes, plataformas verificadas e alterações de documentação. `ROADMAP.md` permanece contexto histórico de fechamento da v1, enquanto este documento registra a nova trilha de **adoção**, sem substituir políticas de release.

**Links canônicos:** [Feature matrix](../reference/feature-matrix.md) · [Release status](../release-status.md) · [Platform support](../operations/platform-support.md) · [Maestro](../product/maestro.md) · [Open source transition](../legal/OPEN_SOURCE_TRANSITION.md).
