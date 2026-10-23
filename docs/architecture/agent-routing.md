# Agent routing

Agent selection is provider-independent. `internal/nexus/agent_matching.go`
owns persistent-Agent eligibility, taxonomy normalization, deterministic
score, confidence, explainability and rejection reasons.
`ResourceScheduler` chooses the provider/profile/model resource.

Legacy role-only Agents remain readable through `NormalizeAgentSpec`;
normalization does not invent capabilities.

## STICKY_LEAD e pedido explícito de revisão (beta.10)

`STICKY_LEAD` preserva identidade de Lead e permanece com
`DelegationMode=OFF` por padrão: isso **desabilita promoção automática a
Mission**, mas o source agora admite uma exceção explícita e restrita.

`ResolveExplicitReadOnlyProviderDelegate` em
`internal/nexus/registered_provider_routing.go` analisa um pedido textual
direcionado a provider registrado e verifica a classificação de papel
read-only. Apenas o pedido que satisfaz os gates pode seguir como um turno
`DELEGATE` sem Mission. Prompts comuns, comandos nativos, pedidos de
implementação e simples menções a providers devem permanecer `KEEP`.

A entrada do Lead é analisada em
`internal/app/control_cmd.go` (TTY) e, quando a etiqueta explícita
`nexus.interactive_lead=true` está presente,
`internal/control/web/handler_terminal.go` (Web).
Esse exame de linha altera a superfície de risco do teclado; não equivale a
certificação de transparência do provider para todos os controles.

**Atenção:** classificar uma tarefa como read-only não constitui sandbox
técnico nem comprova que o processo alvo está impossibilitado de escrever
arquivos. A política de autorização/isolamento segue obrigatória.

## Preferências e evidência

As preferências de execução dos especialistas usam
`SpecialistExecutionPreferences` e o catálogo observado de modelos; um
`DefaultProvider` legado não é um `PIN`. A escolha precisa obedecer
capacidade, requisitos, conta e política de recurso. Consulte
[resource routing](resource-routing.md).

Testes do source: `internal/nexus/registered_provider_routing_test.go`,
`internal/nexus/agent_modes_test.go`.
**Pendente:** testes efetivamente executados e verificação com
Codex/Claude/AGY reais no mesmo SHA, incluindo Enter, menus e comandos nativos.
