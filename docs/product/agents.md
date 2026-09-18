# Agents

## Modelo canônico

```text
Agent (identidade persistente)
├── AgentSpec (especialização comportamental da revisão)
├── AgentRevision (snapshot imutável da configuração)
└── RuntimeGeneration (incarnação executável)
    └── ProviderSession (sessão do provider)
```

Um Agent é uma identidade operacional persistente. Ele mantém nome, projeto,
especialização, histórico e continuidade através de múltiplas
`RuntimeGeneration`s. Codex, Claude e Gemini são providers/executores; trocar
provider não troca a identidade do Agent.

`AgentRevision` é o snapshot imutável da configuração efetiva. O JSON da
revisão contém dados de execução e, quando configurado, o `AgentSpec`.
Agents legados que só têm `Agent.Role` são normalizados para um `AgentSpec`
mínimo com o mesmo role e sem instruções inventadas.

`RuntimeGeneration` é uma incarnação concreta do processo. Seu `RevisionID`
identifica a revisão que originou a execução; `ProviderSession` é uma
referência separada à sessão do executor.

`AgentSpec.role` é a especialização persistente (por exemplo, `senior
developer`). `WorkPackage.role` é o papel assumido naquela tarefa (por
exemplo, `reviewer`). O contexto efetivo é composto como:

```text
Senior Developer acting as Reviewer for this WorkPackage
```

O compilador preserva as duas origens (`agent` e `task`) para diagnóstico.
Direct, Automated e Orchestrated compartilham contexto, provider adapter,
runtime e observabilidade; Maestro adiciona método e gates apenas quando
habilitado.

Prompt/contexto é composto por seções tipadas e provenance; não é persistido
um system prompt gigante como substituto do modelo.

Use Agents para tarefas recorrentes ou papéis especializados. Use AI Session
para uma interação concreta. `UNKNOWN` e `UNVERIFIED` não significam sucesso.
