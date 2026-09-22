# Glossário técnico do Nexus v1

| Termo | Significado |
| --- | --- |
| **Project** | raiz de código gerenciada pelo Nexus |
| **Agent** | identidade persistente de um trabalhador |
| **AgentSpec** | especialização tipada: role, instruções, capacidades e constraints |
| **AgentRevision** | snapshot imutável da configuração efetiva de um Agent |
| **AgentMatcher** | autoridade que escolhe **quem** deve executar uma tarefa |
| **RuntimeGeneration** | encarnação concreta de execução de um Agent |
| **Provider** | CLI/serviço que fornece capacidade de IA |
| **ProviderSession** | sessão do executor; não é a identidade do Agent |
| **Profile** | configuração/autenticação local de um provider |
| **ResourceScheduler** | autoridade que escolhe provider/profile/model para uma execução |
| **Resource policy** | preferência, pin ou regras de seleção de recurso |
| **Worktree** | checkout Git isolado |
| **Direct** | caminho Project → AI Session → Provider → Terminal |
| **Composer** | refinaria de intenção/contexto |
| **PromptArtifact** | saída estruturada e versionada do Composer |
| **WorkPlan** | plano canônico e versionado do trabalho |
| **Flow** | projeção visual/editável do WorkPlan |
| **GO** | autorização explícita de uma revisão do plano |
| **Mission** | trabalho coordenado executado a partir de plano aprovado |
| **MissionRun** | instância durável da execução de uma Mission |
| **Package / PackageRun** | unidade de trabalho / sua instância de execução |
| **PhaseRun** | projeção durável de execução agrupada por fase |
| **ContextCapsule** | handoff limitado de contexto entre etapas |
| **WorkReceipt** | resultado/evidência persistida de uma package |
| **Artifact** | saída durável referenciada pela Mission |
| **Evidence Bundle** | conjunto agregado de evidências da execução |
| **Definition of Done / DoD** | critérios que precisam ser satisfeitos para conclusão |
| **COMPLETED_VERIFIED** | estado terminal após packages + DoD global verificados |
| **BLOCKED_NEEDS_USER** | estado em que decisão/intervenção humana é necessária |
| **Dispatch boundary** | ponto em que uma operação externa é considerada enviada ao provider |
| **Replay safety** | prova de que uma execução pode ser repetida sem duplicar side effects |
| **Coordination request** | solicitação tipada de um Agent ao MissionRunner |
| **NEED_INFO** | pedido de informação para continuar |
| **ARTIFACT_REQUEST** | pedido/reuso de artefato |
| **REVIEW_REQUEST** | pedido de revisão independente |
| **DELEGATE** | pedido de trabalho filho delimitado |
| **IntelligenceMemory** | memória limitada, visível e com provenance |
| **Maestro** | integração opcional de método, skills, risco e gates |
| **Usage / Quota / Capacity** | evidência observada de consumo/disponibilidade |
| **UNKNOWN** | ausência de evidência suficiente; nunca equivale a sucesso |
| **Continuity** | evidência de como contexto/execução foi retomado |
| **PlatformBridge** | fronteira entre Core e capacidades nativas |
