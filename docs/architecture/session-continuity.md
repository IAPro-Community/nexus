# Session continuity

A continuidade de uma sessão supervisionada **não** significa que dois providers
compartilham a mesma conversa nativa. O Nexus distingue reattach do mesmo runtime,
resume nativo verificável, handoff de conta, handoff de contexto e abertura de
nova sessão. Um processo iniciado não transforma `NATIVE_RESUME_UNVERIFIED` em
continuidade comprovada.

## Controlador de continuidade (beta.10, código de 22/10/2026)

`internal/nexus/continuity_controller.go` adiciona um
`ContinuityController` para substituir o recurso de um **runtime supervisionado
existente**, sem criar uma Mission. O ponto de entrada é
`ContinueRuntime(ctx, runtimeID, reason)`; o monitor de quota é conectado pelo
`wireQuotaContinuity` em `internal/nexus/nexus.go`.

A sequência implementada é:

1. Localizar o runtime na registry e verificar o contexto da chamada.
2. Limitar a quatro saltos de continuidade (`nexus.continuity_hops`).
3. Consultar recursos; excluir o provider/profile atual, os já tentados e
   profiles que compartilhem o mesmo `AccountScope` upstream verificado.
4. Priorizar outra conta elegível do **mesmo** provider; na falta dela,
   considerar outra conta/provider elegível.
5. Registrar `nexus.continuity_pending` e publicar
   `QUOTA_FAILOVER_REQUESTED`.
6. Invocar o serviço de handoff existente: `AccountHandoff` para o mesmo
   provider ou `ContextHandoff` para outro. O serviço de handoff é responsável
   por preflight, transferência e regras de writer único.
7. Registrar labels de rastreabilidade e a nova geração; atualizar a identidade
   lógica do Agent, emitir `RUNTIME_REPLACED` e
   `QUOTA_FAILOVER_COMPLETED`. Em falha, emitir `QUOTA_FAILOVER_FAILED`.

Os labels de origem são clonados para a nova sessão, sem aliasing de mapas.
O `AccountScope` de destino é fornecido no launch. O Lead interativo da CLI
possui caminho de reattach para a geração substituta.

## Quando a troca automática pode acontecer

`internal/nexus/quota_monitor_service.go` aciona o controlador apenas quando
`QuotaDropMonitor.CheckAccount` retorna um **evento concreto de esgotamento**
vinculado a um runtime ativo. Uma leitura de quota ausente não é convertida
artificialmente em zero. `UNKNOWN`, `UNSUPPORTED` e falhas de leitura não são
prova de esgotamento. A ausência de métrica não autoriza a troca automática.

`ContinuityReasonForFailure` mapeia causas tipadas (quota, rate limit,
autenticação, indisponibilidade de provider/rede), mas a existência desse
mapeamento **não significa** que todos os eventos de erro do runtime estejam
ligados ao monitor nem que todos disparem recuperação automática.

## Invariantes e limites

- Agent, Project, workspace, geração e lineage precisam continuar rastreáveis.
- Dois profiles da mesma conta upstream não constituem fallback independente.
- `PIN` explícito não pode ser silenciosamente reinterpretado como `AUTO`.
- Um handoff entre providers usa uma cápsula de contexto limitada, **não** um
  session ID nativo de outro provider.
- Status `UNVERIFIED` e falha de attach devem permanecer visíveis.
- O controlador não autoriza replay de ações externas não comprovadamente
  idempotentes, nem dispensa autorização de execução de Mission.

## Implementação, testes e gates

- Código: `internal/nexus/continuity_controller.go`,
  `internal/nexus/quota_monitor_service.go`,
  `internal/control/handoff/account.go`,
  `internal/control/handoff/context.go`,
  `internal/app/control_cmd.go`.
- Testes registrados: `internal/nexus/continuity_controller_test.go`
  (seleção, account scope compartilhado, alternância e falhas tipadas).
- **Ainda não certificado:** failover com duas contas autenticadas reais,
  integridade do terminal/PTY durante handoff, rollback sob falha, continuidade
  no Web/Desktop e smoke Windows/macOS/Linux do mesmo SHA.

Veja [contas](../product/account-switching.md),
[resource routing](resource-routing.md) e
[status da release](../release-status.md).
