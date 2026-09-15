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
- [ ] T1 — `internal/core/quota/probe.go`: `Probe` read-only + `ProbeDescriptor.Validate`
      + helpers de atribuição; `probe_test.go`. Gate: `go test ./internal/core/quota/`.
- [ ] T2 — persistência SQLite: `migrations/0021_capacity_monitor.sql` + store
      (observações, leases, transições, preferências); teste de migração idempotente.
- [ ] T3 — `internal/nexus/capacitymonitor/` enxuto: estados com histerese + scheduler
      adaptativo (jitter) compondo `QuotaDropMonitor.CheckAccount`; sem activity/power
      detectors na v1 (reavaliar depois).
- [ ] T4 — bridge de recomendação sem ciclo de import (adapter sobre `RecommendResources`).
- [ ] T5 — delivery: dedup por fingerprint + receipts via `internal/control/events`.
- [ ] T6 — API web autenticada: status/history/preferences/ack/snooze + testes.
- [ ] T7 — UI: `CapacityMonitorSettings` + `CapacityAlertCard` adaptados ao
      `SettingsSurface`/notification center atuais (SCSS Modules, i18n, `t(...)`).
- [ ] T8 — autostart Linux consent-gated + testes + doc de evidência beta.
- [ ] T9 — gate final: `make quality`, `go test ./internal/nexus/... ./internal/core/quota/...`,
      `git diff --check`; relatório em `DEV/validation/CAPACITY_MONITOR_REVIVAL_REPORT.md`.

## Verificação por task

RED antes de GREEN quando houver contrato novo; `go vet` nos pacotes tocados;
`git diff --check`. Proibido `any`, CSS inline estático, `!important`, texto
hardcoded na UI.
