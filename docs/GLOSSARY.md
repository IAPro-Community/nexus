# Glossário

| Termo | Significado |
| --- | --- |
| Project | Raiz de código gerenciada pelo Nexus. |
| Agent | Identidade persistente de trabalho. |
| AgentSpec | Especialização persistente e tipada de um Agent: role, instruções, capacidades, restrições e política de verificação. |
| AgentRevision | Snapshot imutável da configuração efetiva de um Agent. |
| Runtime | Processo/ambiente que hospeda uma execução. |
| Runtime generation | Execução específica de um Agent. |
| Session | Contexto persistente de interação com um agent/provider. |
| Provider | CLI/serviço que executa a IA. |
| ProviderSession | Identificador de sessão mantido pelo executor; não é a identidade do Agent. |
| Account/Profile | Conta ou perfil selecionado para um provider. |
| WorkPlan | Plano canônico de trabalho. |
| WorkPackage role | Papel assumido por um Agent em uma tarefa específica; compõe com `AgentSpec.role` e não o substitui. |
| Worktree | Checkout isolado para trabalho Git. |
| Composer | Superfície para descobrir intenção e compilar prompt. |
| PromptArtifact | Resultado estruturado e rastreável do Composer. |
| Flow | Projeção visual/editável de um plano ou handoff. |
| Mission | Execução coordenada de trabalho com etapas e verificação. |
| Maestro | Integração opcional de método, skills, risco e gates. |
| Usage / Quota | Consumo e limite observado de provider/profile. |
| Continuity | Evidência de como uma sessão foi retomada. |
| PlatformBridge | Fronteira entre Core e recursos nativos. |
