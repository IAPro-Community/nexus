# Fechamento da documentação — IAPro Nexus v1

## Objetivo

Esta revisão consolida a documentação pública da v1 em PT-BR e estabelece uma
hierarquia clara de fontes de verdade para produto, arquitetura, operação e
release.

O objetivo não é reescrever todo o histórico de engenharia em `DEV/`, e sim
garantir que alguém consiga:

1. entender o que o Nexus é;
2. instalar e obter o primeiro resultado;
3. entender Direct, Composer, WorkPlan, Flow, GO e Mission;
4. entender Agents, routing e coordenação;
5. entender segurança, evidence e recovery;
6. localizar detalhes técnicos sem encontrar contratos contraditórios;
7. distinguir source funcional de release certificada.

## Entradas canônicas

- [README](README.md)
- [Portal de documentação](docs/README.md)
- [Como o Nexus funciona](docs/product/how-it-works.md)
- [Arquitetura](ARCHITECTURE.md)
- [Matriz funcional](docs/reference/feature-matrix.md)
- [Status de release](DEV/validation/current/RELEASE_STATUS.md)

## Fluxo documentado

```text
Objetivo
  → Composer
  → PromptArtifact
  → WorkPlan
  → Flow
  → GO
  → MissionRunner
  → Agents
  → WorkReceipts / Evidence
  → Global DoD
  → COMPLETED_VERIFIED
```

## Documentação criada no fechamento

### Produto

- `docs/product/how-it-works.md`
- `docs/product/agent-coordination.md`

### Arquitetura

- `docs/architecture/mission-runtime.md`
- `docs/architecture/guardrails.md`

### Diagramas versionados

- `docs/assets/diagrams/nexus-v1-overview.svg`
- `docs/assets/diagrams/nexus-v1-flow.svg`
- `docs/assets/diagrams/nexus-v1-components.svg`
- `docs/assets/diagrams/nexus-v1-agent-coordination.svg`
- `docs/assets/diagrams/nexus-v1-mission-lifecycle.svg`
- `docs/assets/diagrams/nexus-v1-guardrails.svg`

Esses arquivos são `SYNTHETIC_DIAGRAM`: explicam contratos, mas não são
evidência de plataforma/provider.

## Documentação consolidada

Foram alinhados, entre outros:

- `README.md`;
- `docs/README.md`;
- `ARCHITECTURE.md`;
- `docs/product/overview.md`;
- `docs/product/mental-model.md`;
- `docs/product/missions.md`;
- `docs/product/autonomous-delivery.md`;
- `docs/features/index.md`;
- `docs/reference/feature-matrix.md`;
- `docs/GLOSSARY.md`;
- `docs/faq.md`;
- `docs/whats-new.md`;
- `docs/assets/README.md`;
- `SUPPORT.md`;
- `CHANGELOG.md`;
- coverage/evidence ledgers.

## Contratos estabilizados pela documentação

### Nexus x Maestro

```text
Nexus   = execução, estado, Agents, resources, workspaces, evidence e UX
Maestro = método, skills, risco, gates e política de verificação
```

Maestro é opcional. Não existe segundo MissionRunner ou scheduler.

### AgentMatcher x ResourceScheduler

```text
AgentMatcher      → QUEM
ResourceScheduler → COM QUAL RECURSO
```

Provider/model não definem identidade de Agent.

### WorkPlan x Flow

```text
WorkPlan = fonte de verdade
Flow     = projeção visual/editável
```

### Mission

Sucesso é evidence-based. `COMPLETED_VERIFIED` exige packages verificadas,
DoD global e ausência de intervenção pendente.

### Coordenação

Operações fechadas:

- `NEED_INFO`;
- `ARTIFACT_REQUEST`;
- `REVIEW_REQUEST`;
- `DELEGATE`.

Coordenação passa pelo MissionRunner, tenta contexto durável primeiro e só cria
child work quando necessário.

### Segurança

Replay automático exige prova positiva. Outcome externo desconhecido,
workspace alterado ou side effect não provado levam a fail-closed/intervenção.

## Screenshots e mídia

Screenshots reais da aplicação continuam em `docs/assets/screenshots/` com
manifesto de fixture sintética.

Diagramas conceituais ficam em `docs/assets/diagrams/`.

Marketing fica separado em `docs/assets/promo/`.

## Documentos históricos

`DEV/` contém muitos relatórios datados. Eles não foram apagados porque são
evidência útil de decisões, regressões e campanhas.

Precedência:

1. código/testes atuais;
2. docs públicas canônicas;
3. `DEV/validation/current/RELEASE_STATUS.md`;
4. specs ativas;
5. relatórios históricos.

## Estado da release

A documentação descreve o **source candidato da v1**. Isso não autoriza declarar
uma release estável.

A decisão de release permanece em:

[`DEV/validation/current/RELEASE_STATUS.md`](DEV/validation/current/RELEASE_STATUS.md)

Enquanto os gates finais não estiverem satisfeitos, preserve o verdict
`NEXUS_1_0_CANDIDATE = NO_GO`.

## Validação exigida para o fechamento

No SHA final da documentação, executar os gates aplicáveis:

```bash
node scripts/docs-verify.mjs
git diff --check
make quality
make web-verify
```

Além disso, o CI do PR deve ser interpretado no contexto correto: falha nativa
antes de qualquer step não é evidência de regressão funcional, mas também não
é certificação da plataforma.

Este documento não marca esses comandos como PASS antecipadamente; o resultado
deve vir do SHA final.
