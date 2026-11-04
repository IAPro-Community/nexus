# Agent routing

Agent selection is provider-independent. `internal/nexus/agent_matching.go`
owns persistent-Agent eligibility, taxonomy normalization, deterministic
score, confidence, explainability and rejection reasons.
`ResourceScheduler` chooses the provider/profile/model resource.

Legacy role-only Agents remain readable through `NormalizeAgentSpec`;
normalization does not invent capabilities.

## STICKY_LEAD e roteamento interativo

`STICKY_LEAD` preserva a identidade do Lead e inicia com
`DelegationMode=OFF`. Complexidade, decomposição heurística e menções ao
provider atual não autorizam Mission nem troca de provider. A Mission só
começa quando o usuário a solicita explicitamente. Um pedido explícito para
outro provider registrado pode executar um turno único sem criar Mission.

`ResolveExplicitProviderDelegate` em
`internal/nexus/registered_provider_routing.go` exige uma instrução de
roteamento ligada ao nome de um provider registrado. O provider ativo é
excluído dos destinos. Escritas delegadas exigem que a sessão de origem esteja
em YOLO e que o provider destino ofereça contrato verificado de escrita no
workspace sem rede; Safe e Plan bloqueiam escritas. Revisões exigem capacidade
read-only verificada. Em falhas, a sessão recebe a causa e pergunta ao usuário
como prosseguir, sem trocar provider nem executar silenciosamente a solicitação
por outro agente.

A entrada do Lead é analisada em `internal/app/control_cmd.go` (TTY) e,
quando a etiqueta explícita `nexus.interactive_lead=true` está presente, em
`internal/control/web/handler_terminal.go` (Web). Esse exame de linha altera a
superfície de risco do teclado; não equivale a certificação de transparência do
provider para todos os controles.

## Preferências e evidência

As preferências de execução dos especialistas usam
`SpecialistExecutionPreferences` e o catálogo observado de modelos; um
`DefaultProvider` legado não é um `PIN`. A escolha precisa obedecer
capacidade, requisitos, conta e política de recurso. Consulte
[resource routing](resource-routing.md).

Testes do source: `internal/nexus/registered_provider_routing_test.go`,
`internal/nexus/promotion/promoter_test.go`,
`internal/nexus/single_turn_delegate_test.go` e
`internal/control/driver/driver_test.go`.
**Pendente:** verificação com Codex/Claude/AGY reais no mesmo SHA, incluindo
Enter, menus e comandos nativos.
