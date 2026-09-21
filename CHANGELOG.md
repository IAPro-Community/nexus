# Changelog

## Unreleased — source v1.0.0-beta.4 — Nexus 1.0 closure

- Added the Composer → PromptArtifact → Flow → revision-bound Execution Preview North Star, with no Agent/runtime/Mission side effects during preview.
- Productized persistent specialist Agent Modes and canonical AgentMatcher taxonomy for senior development, investigation, architecture, security review and performance investigation.
- Kept Capacity collection on the canonical quota path while exposing production observation probes without a second poller.
- Hardened Skills v1 Flow round-trip so canonical `skill_ids` preserve the legacy alias without making Maestro a second catalog authority.
- Added a local observational Cost Intelligence ledger with explicit evidence kinds; unknown tokens/cost stay unknown rather than zero.
- Added persisted soft/hard/emergency cost policy, comparable-sample anomaly detection and Settings/Usage surfaces.
- Added cost-aware Execution Preview and evidence-bound confirmation before starting new Mission work under hard/emergency or unknown hard-budget headroom.
- Restored the frontend format gate to a real repository-wide Prettier check.


## 1.0.0-beta.3 — 2026-09-20 — Surface isolation and Flow recovery

- Mount only the active product surface; inactive ones unmount while
  TerminalsHost stays alive (no PTY recreation, no history loss).
- Flow drafts persist across surface switches; stale refreshes are dropped
  by a latest-wins guard.
- BLOCKED_NEEDS_USER resolves through the run intervention instead of a
  misleading Resume/Return (Resume remains for PAUSED only).
- Flow Run UX hierarchy with progressive disclosure, localized in
  pt-BR/en/es; deep-review findings fixed (duplicate restore ids, focus
  lifecycle, per-plan draft keys).

## 1.0.0-beta.2 — 2026-09-19 — Runtime attach reliability

- Preserve the authoritative supervised-host PID, generation, and binary
  identity during launcher registration so Codex and AGY sessions attach
  reliably after startup.
- Restart the matching Web process after local installation and recover stale
  hosts left by older binaries.

## 1.0.0-beta.1 — 2026-09-15 — Community Preview

Primeira versão pública candidata do IAPro Nexus: workstation local para
coding agents (CLI, Web e Desktop). Isto **não** é um 1.0.0 estável.

### Added

- Composer e Flow como caminho de planejamento na Web (brief, skills, Turn into Flow).
- Superfície de uso com paridade CLI (`login`, `logout`, `use`, `register`, `explain`).
- Telemetria autenticada (`/api/v1/telemetry/stats`, `/api/v1/telemetry/history`).
- Conductor na CLI (`nexus conductor policy`, `nexus conductor dispatch`).
- Pacotes zero-toolchain: binários, DEB, RPM, NSIS com alias `ai.exe`.
- Builds Desktop Wails para Linux, Windows e macOS no CI.

### Changed

- Caminho Web de `auto_plan` foi aposentado: o planejamento passa por Composer → Flow.
- Rotas HTTP `/api/v1/missions/*` e `/api/v1/conductor/*` não fazem parte desta beta; o Conductor permanece na CLI.
- Module path canônico: `github.com/IAPro-Community/nexus`.

### Fixed

- Attach de contexto no Composer rejeita symlink fora do projeto, limita leitura e recusa sessão `FINALIZED`.
- Helper de teste Windows para readiness de terminal sem timeout de PowerShell.
- Filtro de terminal remove rajadas de navegação CSI sem ESC (`[B[A…`) sem engolir o literal `[A`.
- URLs de instalação apontam para `IAPro-Community/nexus`.

### Limits

- Community Preview: Authenticode e notarização Apple não estão nesta versão.
- Updater assinado exige environment GitHub `beta-signing` e secrets `NEXUS_UPDATE_*`.
- Install zero-toolchain precisa de GitHub Release publicado e repositório público (ou `GH_TOKEN`).
- Mission autenticada multi-provider permanece evidência local, não gate desta tag.

O SHA de freeze e a evidência de CI same-SHA ficam em
[`DEV/validation/current/RELEASE_STATUS.md`](DEV/validation/current/RELEASE_STATUS.md).
