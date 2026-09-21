# Slice: Capacity Monitor Revival — 2026-09-15

Reativação do monitor proativo de capacidade (`feat/capacity-monitor-notifications`,
05-09, 17 commits, arquivado em `archive/2026-09-15/feat/capacity-monitor-notifications`)
sobre a árvore atual (`feat/nexus-maximum-delivery` @ `1f42644`).

## Estratégia

Reimplementação adaptativa, NÃO merge/rebase: a base original (`c9a2c8c`) precede a
consolidação Core (application services, rotas splitadas, `AccountScope`, module path
`github.com/IAPro-Community/nexus`). O código arquivado é referência; cada task é
reescrita contra as seams atuais. Branch: `feat/capacity-monitor-revival`
(worktree `../nexus-capacity-worktree`). Sem push sem pedido.

## Seams atuais (não duplicar)

- `internal/nexus/quota_monitor.go`: `QuotaDropMonitor` maturo (thresholds, milestones,
  failover `RecommendResources`, `AccountScope`, event bus, persistência JSON). O revival
  ADICIONA: scheduler adaptativo, estados com histerese, persistência SQLite, preferências.
- `internal/core/quota/`: `quota.go`, `view.go`, `view_builder.go`, `eligibility.go`.
  NÃO tem abstração `Probe` — Task 1 a cria.
- `internal/nexus/store/migrations/`: última `0020_*` → próxima do slice: `0021_*`.
- Web: `web/src/nexus/api.ts`, `web/src/types.ts`, `InAppNotificationCenter`,
  `SettingsSurface`. UI arquivada é vintage 05-09: adaptar, não copiar.
- Spec/plan originais (ainda válidos como contrato):
  `docs/superpowers/specs/2026-09-05-capacity-monitor-notifications-design.md`,
  `docs/superpowers/plans/2026-09-05-capacity-monitor-notifications-implementation.md`.

## Fora do escopo

Tray desktop (incremento separado, agora desbloqueado via Wails — spec própria futura);
push remoto; providers novos; qualquer mudança em `QuotaDropMonitor` existente
(só composição sobre ele).

## Tasks

- [x] T0 — branch/worktree/slice doc (esta).
- [x] T1 — `internal/core/quota/probe.go`: `Probe` read-only + `ProbeDescriptor.Validate`
      + helpers de atribuição; `probe_test.go`. Gate: `go test ./internal/core/quota/`.
- [x] T2 — persistência SQLite: `migrations/0021_capacity_monitor.sql` + store
      (observações, leases, transições, preferências); teste de migração idempotente.
- [x] T3 — `internal/nexus/capacitymonitor/` enxuto: estados com histerese + scheduler
      adaptativo (jitter) com loop próprio sobre ProbeRegistry (standalone; o bridge
      para o dispatch do `QuotaDropMonitor` é T4); sem activity/power
      detectors na v1 (reavaliar depois). FEITO: per-window, repeat-tick read-only,
      lease single-leader via store, prefs como thresholds, eventos
      QUOTA_LOW/EXHAUSTED/MONITOR_RECOVERED no bus (delivery/dedup = T5).
      Deep review (6 red/15 yellow/3 q, revisor independente): tudo endereçado —
      lease atômico, account_scope, NaN/Inf, jitter limitado, sem com ctx,
      backoff por pass totalmente falho, TTL adaptativo, fail-closed em erro
      de store, repeat read-only, nil-registry explícito.
- [x] T4 — bridge de recomendação sem ciclo de import (adapter sobre `RecommendResources`).
  FEITO: `capacitymonitor/recommendation.go` (`ResourceKey/Recommendation/Recommender`),
  `internal/nexus/capacity_recommendation_bridge.go` (`CapacityRecommender` sobre o
  `RecommendResources` existente, advisory-only, sem migração automática) + teste.
- [x] T5 — delivery: dedup por fingerprint + receipts via `internal/control/events`.
  FEITO: `ResetCycleKey` (reset real no fingerprint), `ShouldDeliver` (re-alerta só em
  mudança de estado/janela/recurso/recovery/repeat explícita), `Put/FindCapacityEvent`,
  `ListCapacityEvents`, `PutDeliveryReceipt`, `capacity_alert_states` (ack/snooze) com
  enforcement no delivery, evento persistido ANTES de publicar (restart-safe) + testes.
- [x] T6 — API web autenticada: status/history/preferences/ack/snooze + testes.
  FEITO: `CapacityApplicationService` + `handlers_capacity.go`
  (`/api/v1/capacity/status|history|preferences|ack|snooze`, auth+envelope atuais,
  UNKNOWN nunca vira saudável, sem paths/credentials) + `handlers_capacity_test.go`.
- [x] T7 — UI: `CapacityMonitorSettings` + `CapacityAlertCard` adaptados ao
  `SettingsSurface`/notification center atuais (SCSS Modules, i18n, `t(...)`).
  FEITO: aba `capacity` em `SettingsSurface`, `CapacityMonitorSettings.tsx`,
  `capacityAlertFromEvent` (id por fingerprint, recovered, recomendação) ligado no
  `InAppNotificationCenter` com fallback legado, faixa de monitor no
  `UsageAccountCard` (estado/transição/ack), execution summary no `FlowRunSurface`
  (skills/reasoning/isolation/routing mode+why). `make web-verify` 10/10.
- [x] T8 — autostart Linux consent-gated + testes + doc de evidência beta.
  FEITO (escopo mínimo decidido pelo Maestro): `Start/StopCapacityMonitor` hospedado
  no processo Nexus (padrão `StartQuotaMonitor`), só com `Enabled+BackgroundEnabled`
  (default OFF), idempotente, lease-gated, `Shutdown` desliga; sem daemon novo, sem
  battery detector. Registro de probes de produção: `emptyProbeRegistry` (UNKNOWN
  visível, nunca alerta) — probes reais em
  `DEV/FUTURE_EVOLUTION_FROM_OPENCODEX_OVERCLOCK.md`.
- [x] T9 — gate final: `make quality`, `go test ./internal/nexus/... ./internal/core/quota/...`,
  `git diff --check`; relatório em `DEV/validation/CAPACITY_MONITOR_REVIVAL_REPORT.md`.
  FEITO: `go test ./internal/...` verde, `-race` nos pacotes concorrentes tocados,
  `go vet` limpo, `make build`, `make web-verify` 10/10, `make test-e2e` verde,
  `git diff --check` limpo. Relatório: `DEV/validation/INCREMENTAL_EVOLUTION_REPORT.md`.

## Verificação por task

RED antes de GREEN quando houver contrato novo; `go vet` nos pacotes tocados;
`git diff --check`. Proibido `any`, CSS inline estático, `!important`, texto
hardcoded na UI.
