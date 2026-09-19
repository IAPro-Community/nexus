# Changelog

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
