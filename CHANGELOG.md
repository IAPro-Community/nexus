# Changelog

## 1.0.0 — 2026-09-23 — Nexus v1

> Esta seção descreve o **source final candidato**. A tag/release pública só pode
> ser criada depois da certificação estável do mesmo SHA.

### Adicionado

- Fluxo canônico Composer → PromptArtifact → WorkPlan → Flow → Execution Preview
  → GO explícito → MissionRunner, sem side effects durante preview.
- Agent Modes persistentes e taxonomia canônica do AgentMatcher para
  desenvolvimento, investigação, arquitetura, segurança, QA e revisão.
- Skills v1 com catálogo/provenance e integração opcional com Maestro.
- Cost Intelligence observacional, budgets e detecção de anomalia sem converter
  custo desconhecido em zero.
- Coordenação tipada entre Agents com `NEED_INFO`, `ARTIFACT_REQUEST`,
  `REVIEW_REQUEST` e `DELEGATE`.
- Resolução de coordenação por memory/receipts/artifacts antes de materializar
  child work.
- Child packages duráveis com lineage, capabilities para o scheduler e
  WorkReceipt de resposta.
- Continuação segura da package de origem sob a mesma identidade persistente de
  Agent, condicionada a prova positiva de replay safety.
- Limites de fan-out e payload para coordenação gerada por provider.
- Documentação v1 em PT-BR, incluindo diagramas canônicos versionados em
  `docs/assets/diagrams/`.

### Alterado

- A matriz estável da v1 foi fechada em **Linux amd64**; Windows/macOS permanecem
  implementados no source, porém deferidos e não publicados como suporte certificado.
- O CI/release foi simplificado para não depender de minutos pagos e para produzir,
  validar e assinar somente os artifacts suportados da v1.
- `WorkPlan` permanece a fonte de verdade; Flow é explicitamente sua projeção,
  não um segundo planner.
- AgentMatcher escolhe **quem**; ResourceScheduler escolhe **com qual recurso**.
- Preferência de provider em coordenação permanece preferência e não herda
  autoridade MANUAL/PIN do trabalho de origem.
- Mission/coordenação passaram a ser documentadas separando
  **implementação no source candidato** de **certificação de release**.
- Capacity continua no caminho canônico de quota/observação, sem segundo poller.
- O gate de frontend voltou a usar verificação Prettier real em todo o
  repositório.

### Segurança e confiabilidade

- Replay de coordenação falha fechado quando workspace/fingerprint não comprova
  ausência de mutação ou quando o contrato autoriza side effects externos.
- Dispatch externo com resultado desconhecido continua sem reexecução cega.
- UNKNOWN permanece UNKNOWN em quota, custo, suporte e evidência.
- Push/deploy continuam condicionados a autorização explícita.


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
