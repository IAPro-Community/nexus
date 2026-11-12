# Focus Queue — implementação local — 2026-11-09

## O que foi implementado

- `web/src/app/focusQueueModel.ts`: fila única de pendências com severidade
  primeiro (`needs_user` antes de `error`) e depois episódio mais antigo;
  deduplicação por episódio (`attention_fingerprint`, fallback runtime);
  exclusão de episódios já respondidos (`attention_submitted_revision >=
  attention_revision`), espelhando o claim one-shot durável do servidor.
- `web/src/notifications/FocusQueueSection.tsx` + `.module.scss`: seção "Fila
  de pendências" no Radar da Central de Atenção, com rank, rótulo de
  severidade, tempo de espera via `Intl.RelativeTimeFormat` e foco no runtime
  exato pelo mesmo caminho do radar (`planFocusAttention`).
- Descoberta de produto: anúncio `focus-queue` (revision 1) na nova superfície
  `notifications`, passo de tour único com âncora `data-tour="focus-queue"`,
  escopo `data-feature-tour-scope` no drawer, i18n PT-BR/EN/ES (notifications,
  discovery e tour). Reutiliza FeatureTour/ProductTour compartilhados.
- `web/src/types.ts`: campo `updated_at?: string` no contrato do cliente
  (backend já serializa; base temporal da fila com fallback `started_at`).
- Lacuna corrigida em código já commitado: `ComparisonLabPanel.tsx` agora
  normaliza `checks`/`changed_files` com `asArray`/`asStringArray` — o gate
  `null-arrays` voltou a PASS (falha pré-existente, não introduzida por esta
  fatia).

## Verificação executada

```text
npm --prefix web run test -- --run src/app/focusQueueModel.test.ts        # 5/5
npm --prefix web run test -- --run (announcements, tour, tourTargets,     # 31/31
  featureReveal, focusQueueModel)
node scripts/verify-agent-workspace-evidence-map.mjs                      # 94+2/96, 0 unmapped, 0 certificados
node --test scripts/verify-agent-workspace-evidence-map.test.mjs          # 4/4
cd web && node scripts/verify-report.mjs                                  # 10/10 gates PASS (135 arquivos/850 testes)
```

`make quality-full` e `make build` foram executados sobre o estado final desta
árvore; resultado registrado em `DEV/VERIFY.md`.

## Correções colaterais nesta árvore (pré-existentes, bloqueavam o gate)

- `ComparisonLabPanel.tsx`: `checks`/`changed_files` agora normalizados com
  `asArray`/`asStringArray` — gate `null-arrays` do `web-verify` voltou a PASS.
- `internal/nexus/maestro.go`: removido código morto do contrato CLI antigo
  (`querySkillsList`, `normalizeMaestroSkillList`, `mergeMaestroSkillMetadata`);
  `queryCapabilitiesFromDir` retorna somente a capability (unparam/unused do
  lint-go).
- `internal/nexus/maestro_test.go`: teste do contrato antigo reescrito para o
  Bridge v1 (`TestMaestroCapabilitiesNegotiatesBridgeV1Contract`) e motivo
  degradado do teste version-only atualizado.
- Hermeticidade de testes: `TestGetAccountInfo` (internal/profile) e
  `setupTestEnvironment` (internal/app) fixam `NEXUS_REAL_HOME` temporário;
  sem isso, `Create` adota as sessões reais do host (`AdoptHostAuth`) e os
  testes passam a depender do estado local das contas AGY/Codex.

## Limites (não homologação)

- Testes do modelo e de catálogo/tour são unitários; não há E2E visual do
  drawer com fila povoada por runtimes reais nem execução no app distribuído.
- A fila é projeção cliente do Registry; a idempotência durável da resposta
  continua sendo o claim one-shot do servidor (`attention_claim.go`), já
  coberto por testes Go. Nenhum provider foi chamado.
- `focus-queue` permanece `verification: pending`, `sourceStatus: partial`,
  sem aceite. Zero das 48 capacidades está certificada.
