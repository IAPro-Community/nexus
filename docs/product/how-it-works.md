# Como o Nexus v1 funciona

> **TL;DR:** o Nexus transforma um objetivo em um plano versionado, permite revisar
> esse plano visualmente, exige uma autorização explícita para iniciar execução e
> só conclui uma Mission quando o trabalho possui evidência suficiente para atingir
> `COMPLETED_VERIFIED`.

![Fluxo ponta a ponta do Nexus v1](../assets/diagrams/nexus-v1-flow.svg)

## 1. O modelo em uma frase

```text
Objetivo → Composer → WorkPlan → Flow → GO → MissionRunner → Agents → Evidências → COMPLETED_VERIFIED
```

O Nexus é o **control plane e runtime operacional**. Ele mantém projetos,
identidades de Agent, runtimes, providers, workspaces, planejamento, execução,
evidências e recuperação.

O [Maestro](maestro.md) é uma integração opcional para método, skills, risco e
gates. Ele **não** substitui o runtime do Nexus e não cria uma segunda máquina de
estados.

## 2. Três profundidades de uso

O produto não obriga toda tarefa a passar pela orquestração completa.

| Necessidade | Caminho recomendado | Custo de coordenação |
| --- | --- | --- |
| alteração pequena, debugging, exploração | **Direct** | mínimo |
| objetivo ainda precisa de clareza | **Composer** | baixo |
| dependências, múltiplos agentes, longa duração, gates | **Flow + Mission** | controlado |

### Direct

```text
Project → AI Session → Provider → Terminal
```

É o caminho mais rápido. Não exige Composer, Flow, Mission ou Maestro.

### Composer

Composer refina o objetivo e produz um `PromptArtifact` estruturado. Ele
combina contexto explícito do usuário, inteligência do projeto e memória
durável sem transformar tudo em um prompt gigante.

### Flow

Flow é a projeção visual do **WorkPlan canônico**. Ele permite revisar fases,
packages, dependências, paralelismo e alocação de Agents. Não existe um
“segundo plano” escondido na UI.

### Mission

Depois do **GO explícito**, a revisão aprovada é congelada e o
`MissionRunner` executa o plano de forma durável.

## 3. Fontes de verdade

| Objeto | Responsabilidade |
| --- | --- |
| `PromptArtifact` | intenção refinada e contexto compilado |
| `WorkPlan` | plano semântico e versionado |
| `Flow` | projeção visual/editável do WorkPlan |
| `MissionRun` | instância de execução durável |
| `PackageRun` | unidade efetivamente executada |
| `ContextCapsule` | handoff limitado de contexto |
| `WorkReceipt` | resultado e evidência de uma package |
| Evidence Bundle | conjunto de evidências da Mission |

## 4. O que acontece ao pressionar GO

1. O Nexus valida que a revisão aprovada pode ser executada.
2. A revisão do plano fica associada à Mission.
3. Cada package recebe dependências, requisitos, gates e política de recurso.
4. O `ResourceScheduler` escolhe provider/perfil/modelo elegíveis.
5. O Agent recebe um prompt versionado e contexto limitado.
6. O runtime executa, testa, revisa e verifica.
7. Falhas podem produzir retry, remediation, replan ou intervenção humana.
8. O estado é persistido entre reinícios.
9. A Mission só termina quando o Definition of Done global é satisfeito.

## 5. Agents não são providers

![Componentes do sistema](../assets/diagrams/nexus-v1-components.svg)

Um **Agent** é uma identidade persistente. Provider e modelo são recursos de
execução.

```text
Agent
 ├─ AgentSpec
 ├─ AgentRevision
 └─ RuntimeGeneration
      └─ ProviderSession
```

Isso permite preservar papel, histórico e continuidade operacional mesmo quando
um novo runtime é criado ou o scheduler seleciona outro recurso compatível.

## 6. Coordenação dinâmica

Durante uma Mission um Agent pode precisar de informação, artefato, revisão ou
trabalho delegado. A comunicação não vira uma malha peer-to-peer livre.

Operações v1:

- `NEED_INFO`
- `ARTIFACT_REQUEST`
- `REVIEW_REQUEST`
- `DELEGATE`

Toda solicitação passa pelo `MissionRunner`. O Nexus tenta reutilizar memória,
artifacts e receipts antes de criar nova execução. Quando trabalho novo é
necessário, ele vira uma child package rastreável.

Leia [Coordenação entre Agents](agent-coordination.md).

## 7. Evidência antes de sucesso

O Nexus não considera uma tarefa concluída somente porque um modelo disse
“done”.

A conclusão pode depender de:

- comandos de verificação;
- testes automatizados;
- revisão independente;
- artifacts esperados;
- critérios de aceite;
- ausência de intervenção pendente;
- Definition of Done global.

`UNKNOWN` não é convertido em sucesso.

## 8. Recovery e restart

Estado crítico é durável. Após restart, o Nexus reidrata a Mission, receipts,
capsules, dependências e lineage antes de decidir o próximo passo.

Operações externas com resultado desconhecido são tratadas de forma
**fail-closed**. O Nexus não repete cegamente um dispatch que pode ter produzido
side effects.

## 9. Segurança

Os limites de autonomia são definidos pelo `AutonomyContract`, worktrees,
isolamento do provider e validação pós-execução.

Exemplos de controles:

- padrões de arquivos permitidos;
- proibição de Git destrutivo;
- push/deploy negados por padrão;
- rede, secrets e paid services negados quando o contrato exige isolamento;
- reviewers/investigators read-only;
- retries e no-progress limitados;
- intervenção humana tipada.

Leia [Guardrails e segurança da execução](../architecture/guardrails.md).

## 10. Quando usar Maestro

Use Maestro quando quiser adicionar ao Nexus:

- metodologia;
- skills reutilizáveis;
- classificação de risco;
- critérios e gates;
- políticas de revisão/verificação.

Sem Maestro, Direct, Agents, Composer, Flow e Mission continuam pertencendo ao
Nexus e devem permanecer operacionais.

## Próximos passos

- [Quick Start](../quick-start.md)
- [Modelo mental](mental-model.md)
- [Composer](composer.md)
- [Flow](flow.md)
- [Missions](missions.md)
- [Coordenação entre Agents](agent-coordination.md)
- [Arquitetura](../architecture/overview.md)
