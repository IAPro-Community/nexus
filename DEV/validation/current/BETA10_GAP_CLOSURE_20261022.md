# Reconciliação da beta.10 — 22/10/2026

**Branch:** `release/v1.0.0`  
**Versão do source:** `1.0.0-beta.10` (sem criar nova tag de release)  
**Baseline anterior:** `622aa93333bc1431486d5d799d6427d2d7d3d119`  
**Source code auditado após a intervenção:** `e98c711e86ffd578111cfca454513722198bacbe`  
**Delta da intervenção:** 27 commits, 21 arquivos; [comparação dos commits](https://github.com/IAPro-Community/nexus/compare/622aa93333bc1431486d5d799d6427d2d7d3d119...e98c711e86ffd578111cfca454513722198bacbe).

> Este é um ledger de engenharia, **não** um certificado de release. O SHA
> acima é o source auditado antes da presente revisão documental. Os commits
> de documentação e a correção de estilo posteriores geram outro HEAD;
> precisam de novos gates exact-SHA. Não transferir PASS entre SHAs.

## Inventário das alterações

| Escopo | Implementação | Teste no source | Contrato canônico | Estado |
| --- | --- | --- | --- | --- |
| Desktop attach — JS/CSS antes do fallback SPA | `internal/control/web/desktop_attach.go` | `internal/control/web/desktop_attach_test.go` | [Desktop](../../../docs/desktop/overview.md) | CÓDIGO PRESENTE / NATIVE UNVERIFIED |
| Scroll interno do workspace Web | `web/src/app/workspace-styles/03-workspace-layout.css`, `web/src/index.tsx` | `web/src/app/workspaceSurfaceStyles.test.ts`, `web/scripts/e2e-surface-isolation.mjs` | [Modularização](../../../docs/architecture/WORKSPACE_STYLES_MODULARIZATION.md) | CÓDIGO PRESENTE / E2E PENDENTE |
| Entrada de teclado Web/Windows diretamente no stream PTY | `internal/control/web/handler_terminal.go` | `internal/control/web/handler_terminal_input_test.go` | [Terminais](../../../docs/product/terminals.md) | CÓDIGO PRESENTE / CONPTY NATIVE UNVERIFIED |
| Revisão read-only explícita com STICKY_LEAD e Mission OFF | `internal/nexus/registered_provider_routing.go`, `internal/app/control_cmd.go`, `internal/control/web/handler_terminal.go` | `internal/nexus/registered_provider_routing_test.go` | [Controle interativo](../../../docs/product/interactive-control.md) | CÓDIGO PRESENTE / TUI NATIVE UNVERIFIED |
| Continuidade por quota e reattach do Lead | `internal/nexus/continuity_controller.go`, `internal/nexus/quota_monitor_service.go`, `internal/nexus/nexus.go`, `internal/app/control_cmd.go` | `internal/nexus/continuity_controller_test.go` | [Continuidade](../../../docs/architecture/session-continuity.md) | CÓDIGO PRESENTE / HANDOFF REAL PENDENTE |
| AccountScope/labels no handoff | `internal/control/handoff/account.go`, `internal/control/handoff/context.go`, `internal/control/handoff/service.go` | testes de continuidade e handoff existentes | [Troca de contas](../../../docs/product/account-switching.md) | CÓDIGO PRESENTE / GATE NATIVO PENDENTE |
| ExecutionTarget, CapacityDomain, ModelCatalog e preferências | `internal/nexus/execution_target.go`, `internal/nexus/agent_modes.go` | `internal/nexus/execution_target_test.go`, `internal/nexus/agent_modes_test.go` | [Resource routing](../../../docs/architecture/resource-routing.md) | CÓDIGO PRESENTE / CATALOGO REAL NÃO CERTIFICADO |
| Evento de substituição de runtime | `internal/control/events/events.go` | testes do controlador e eventos existentes | [Eventos](../../../docs/architecture/events.md) | CÓDIGO PRESENTE / ENTREGA FIM A FIM PENDENTE |

A [matriz pública](../../../docs/reference/feature-matrix.md) e seu
[catálogo](../../../docs/reference/features.json) foram atualizados de 30
para 36 entradas, incluindo seis contratos específicos da intervenção.
Todos permanecem com evidência
`SOURCE_TEST_PRESENT_NOT_RUN` / `PENDING_EXACT_SHA` até a certificação
independente — **testes versionados não significam testes aprovados**.

## Resultado exato da homologação Windows do source e98c711

- Workflow: [Windows release 1.0.0 exact-SHA homologation #39870167188](https://github.com/IAPro-Community/nexus/actions/runs/39870167188).
- `checkout`, verificação do SHA e arquitetura, setup, preflight, captura
  visual e checagem de outputs: **etapas concluídas com sucesso**.
- Verificação documental: `docs verification passed: 193 Markdown files,
  33 required assets/files` **nesse run**.
- Gate de homologação completa: **FAIL** na tarefa
  `bun run format:check` (`prettier --check`).
- Causa exibida no log: `src/features/hud/GuardianMascot.test.tsx`
  não estava conforme o formatador; script abortou com exit code 1 antes
  de completar a sequência de validações.
- Artefatos do mesmo run:
  [evidência Windows](https://github.com/IAPro-Community/nexus/actions/runs/39870167188/artifacts/12465484923) e
  [resumo de first-run](https://github.com/IAPro-Community/nexus/actions/runs/39870167188/artifacts/12465474816).
- A [CI geral #39870167164](https://github.com/IAPro-Community/nexus/actions/runs/39870167164)
  foi **cancelada**, não aprovada.
- A pipeline de unsigned preview foi **SKIPPED**, não publicou
  um candidato validado no mesmo SHA.

O arquivo `web/src/features/hud/GuardianMascot.test.tsx` recebeu
formatação pontual no commit `43c48376bc39c55d2bc888b23ee03b163831a011`.
Esse commit **não é evidência de que o Prettier passou**: é necessário
reexecutar a pipeline após integrar todas as mudanças documentais.

## Complemento de merge seletivo: classificação de falhas

A revisão posterior detectou que a branch histórica
`fix/v1.0.0-gap-closure-20260930` continua divergente e não pode ser
mesclada integralmente: há mudanças já incorporadas no release, arquivos
refatorados após o ponto de origem e pelo menos um uso de
`watchRuntimeFailures` sem implementação localizada na branch histórica.

A parte preservável foi isolada em
`fix/v1.0.0-runtime-failure-evidence-20261022`:
`SessionHost` passa a publicar `failure_kind` tipado e a descrição sanitizada
quando um runtime falha, com teste para rate limit, quota, autenticação, rede,
falha desconhecida, cooldown e ausência de vazamento de output.
O listener automatizado para eventos `RUNTIME_FAILED` **não** foi copiado
da branch antiga e segue fora deste merge; não inferir novo failover
automático a partir desse enriquecimento de eventos.

A integração via pull request deve ser validada no SHA efetivamente mergeado,
com `go test` do pacote host e os gates completos; um merge não é PASS nativo.

## Gates para sair de NO-GO

1. Congelar um único HEAD após os commits de documentação. Registrar `GIT_SHA`
   exato, `VERSION`, `EVIDENCE_COMMIT` e os links dos runs.
2. Rodar `node scripts/verify-feature-catalog.mjs` e
   `node scripts/docs-verify.mjs`. Confirmar contagem de 36 recursos e
   nenhum link/path inválido.
3. Executar `cd web && bun run format:check && bun run lint &&
   bun run lint:styles && bun run typecheck && bun run test && bun run build`.
4. Executar `go test ./internal/nexus/... ./internal/control/...` e
   depois `go test ./...` com a toolchain exigida por `go.mod`.
5. Executar `cd web && bun run test:e2e:surface`; verificar scroll
   em viewport móvel/laptop, além de foco e ausência de overflow horizontal.
6. Testar Web/Desktop em Windows com Codex real: abrir janela, carregar bundles,
   digitar e receber eco, Enter, setas/CSI/SS3, `Esc`, `Ctrl+C`,
   colagem Unicode, scroll, menus e reconexão. Registrar capturas sanitizadas.
7. Testar STICKY_LEAD `OFF` com prompt comum (`KEEP`), revisão explicitamente
   direcionada (`DELEGATE`, sem Mission), e pedido de implementação
   (`KEEP` ou fluxo de autorização explicitamente selecionado).
8. Testar quotas com contas distintas e mesmo upstream, sem inventar quota
   em `UNKNOWN`; executar handoff real, rollback, limites de salto e
   reattach sem duplicação de writer/efeitos.
9. Realizar homologação native same-SHA nas plataformas pretendidas e publicar
   uma decisão de release `GO` ou `NO_GO`, apontando todas as evidências.
10. Recapturar as imagens para o SHA congelado. O
    `docs/assets/screenshots/manifest.json` anterior tem
    `source_sha=9407f18b...`, **não** o SHA de intervenção.

**Veredito documental deste registro: CÓDIGO INTEGRADO; DOCS EM RECONCILIAÇÃO;
RELEASE NO-GO.** Não alterar para PASS sem logs/smokes do mesmo SHA.
