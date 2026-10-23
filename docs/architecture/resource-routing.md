# Resource routing

`ResourceScheduler`/`RecommendResources` continua sendo o alocador canônico
de provider/profile. `AgentMatcher` decide **quem** executa; o scheduler
decide **com qual recurso**. O novo `ExecutionTarget` é uma projeção desse
processo, não uma segunda implementação de scheduler.

## Elegibilidade e evidência de capacidade

`quota.EvaluateEligibility` aplica os gates de autenticação, saúde,
cooldown, indisponibilidade, rate limit e exaustão conhecida. Quota e
capacidade operacional são dimensões distintas:

- `LIVE`/`CACHED` com medida confiável podem sustentar disponibilidade.
- `UNKNOWN` exige tratamento explícito e não pode virar porcentagem zero.
- `UNSUPPORTED` permite um recurso saudável e autenticado elegível com
  capacidade `OPAQUE`, sem fabricar porcentagens.
- `PIN` é um requisito do usuário, não uma preferência flexível; `PREFER`
  admite alternativas elegíveis.
- Só selecionar provider mais barato **depois** de satisfazer especialidade,
  requisitos e risco de execução.

Sem quota mensurável, recursos elegíveis seguem seleção limitada e
determinística, não uma pontuação de capacidade inventada.

## ExecutionTarget, CapacityDomain e ModelCatalog (beta.10)

`internal/nexus/execution_target.go` acrescenta:

| Estrutura | Responsabilidade |
| --- | --- |
| `ExecutionTarget` | Projeção do backend, profile, identidade de conta, modelo, capabilities, saúde e evidência de capacidade |
| `CapacityDomain` | Identificar a capacidade upstream compartilhada, preferindo `AccountScope` verificado a labels de profile |
| `ModelCatalog` | Catálogo de modelos observados/configurados, com origem, validade e deduplicação; **não descobre modelos por suposição** |
| `ResolveExecutionTarget` | Resolver preferências pela política de routing existente e retornar a projeção selecionada |

Um nome de modelo solicitado com `AffinityPin` que não existe no
`ModelCatalog` resulta em erro explícito. Na ausência de modelos observados,
é possível resolver um recurso elegível **sem inventar nome de modelo**.

`BuildExecutionTargets` materializa as opções para contas e modelos conhecidos.
Contas distintas que compartilham o mesmo `AccountScope` verificado devem ser
tratadas como o mesmo domínio de capacidade para prevenir failover ilusório.

## Preferências de especialistas

`internal/nexus/agent_modes.go` representa
`SpecialistExecutionPreferences` com destinos `TargetPreference`, traits
desejados, preferência de custo e nível de raciocínio. O
`DefaultProvider` legado é convertido em preferência suave (`PREFER`), não
em `PIN`. Sugestões de modelo específicas de frontend só são consideradas
quando existe evidência no catálogo; não são garantia de disponibilidade.

A escolha não pode sobrescrever `PIN` explícito. A presença da API de
`ModelCatalog` no source **não prova** descoberta automática de todos os
modelos em cada provider ou integração em todas as superfícies.

## Evidência e homologação

- Código: `internal/nexus/execution_target.go`,
  `internal/nexus/agent_modes.go`, `internal/nexus/runtime_routing.go`.
- Testes cadastrados: `internal/nexus/execution_target_test.go`,
  `internal/nexus/agent_modes_test.go`.
- Pendente: testes executados no SHA final, provider/model catalog real,
  roteamento sem regressão com contas indisponíveis e smokes nativos.

Veja [Agent routing](agent-routing.md),
[quota](usage-quota.md) e
[continuidade](session-continuity.md).
