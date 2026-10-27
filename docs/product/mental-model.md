# Modelo mental

| Termo | Significado |
| --- | --- |
| **Project** | raiz de código e contexto operacional |
| **Agent** | identidade persistente de um trabalhador |
| **AgentSpec** | especialização tipada do Agent |
| **AgentRevision** | snapshot imutável da configuração efetiva |
| **RuntimeGeneration** | processo concreto que executa um Agent |
| **Provider** | CLI/serviço de IA usado como recurso de execução |
| **ProviderSession** | sessão pertencente ao executor; não é a identidade do Agent |
| **Profile** | configuração/autenticação local de um provider |
| **Worktree** | checkout Git isolado |
| **Composer** | refinaria de intenção/contexto |
| **PromptArtifact** | saída estruturada e versionada do Composer |
| **WorkPlan** | fonte de verdade semântica do plano |
| **Flow** | projeção visual/editável do WorkPlan |
| **GO** | autorização de uma revisão específica do plano |
| **Mission** | execução coordenada de um plano aprovado |
| **MissionRun** | instância durável dessa execução |
| **PackageRun** | unidade executável dentro de uma Mission |
| **ContextCapsule** | handoff limitado de contexto |
| **WorkReceipt** | resultado/evidência de uma package |
| **Evidence Bundle** | evidência agregada da execução |
| **IntelligenceMemory** | memória semântica legada do Nexus; preservada até migração para o Maestro |
| **ResourceScheduler** | escolhe provider/profile/model conforme policy/capacidade |
| **AgentMatcher** | escolhe a identidade de Agent adequada |
| **Maestro** | integração opcional na beta atual; núcleo independente e obrigatório na arquitetura alvo (não implementada) |
| **COMPLETED_VERIFIED** | Mission concluída após DoD global e evidência |

## Regra mais importante

```text
AgentMatcher      = QUEM executa
ResourceScheduler = COM QUAL RECURSO executa
MissionRunner     = QUAL É O ESTADO DA EXECUÇÃO
WorkPlan          = O QUE FOI AUTORIZADO
```

Essas responsabilidades não devem ser fundidas.
