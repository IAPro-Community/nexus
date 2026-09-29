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


## 1.0.0-beta.5 — 2026-09-29 — Codex Windows convergence

- Windows/Agent resources: new agents may be created without inventing a fake `provider:default` binding; the selected provider is retained only as allocation preference until the Resource Picker persists the real account.
- Safe Apply: stopped agents can persist specialization/configuration while unassigned; running agents still fail closed if provider identity is missing.
- Regression guard: the Windows patch was ported selectively onto the canonical beta line instead of merging its stale base, preserving Beta 4 terminal transparency, explicit Mission promotion, daemon supervision, OpenCode and custom Docker/WSL/SSH target fields.
- UI tests cover preservation of custom target configuration while provider/profile remain unassigned.

**Release state:** source convergence complete on the Beta 5 candidate; exact-SHA runtime validation remains required before claiming platform certification.

## 1.0.0-beta.4 — 2026-09-27 — unified beta hardening

- Release engineering: a single canonical branch, `release/v1.0.0-beta`, now carries future 1.0 beta iterations. Beta 1/2/3 remain historical checkpoints rather than parallel release lines.
- Distribution: prerelease publishing targets one rolling public entry, **Nexus 1.0 Beta** (`channel-beta`); the exact internal SemVer and source SHA remain in the signed update manifest.
- Native shell: Desktop and HUD are ported to Wails v3.0.0-beta.25 without importing stale Mission/ExecutionRole/OpenCode code from the old migration branch.
- Windows lifecycle: the browser helper is copied instead of hard-linked to the running executable, so cleanup can remove `ai-browser.exe` while the test/runtime process remains alive.
- Windows tests: symlink-sensitive tests detect actual OS capability instead of failing merely because Developer Mode/admin privilege is unavailable.
- Conversation tests: provider history fixtures isolate HOME/USERPROFILE/APPDATA/LOCALAPPDATA/XDG paths so real Codex history cannot contaminate synthetic test data.
- Web tests: the process-wide Nexus singleton/SQLite store is closed before temporary data removal; optional hosts-file mutation can be disabled on locked-down runners.
- OpenCode enforcement: effective-permission introspection is accepted as valid read-only evidence without fabricating a nonexistent `edit` tool event.
- CI/homologation: Wails v3 native builds, exact framework evidence, release-branch push coverage and version-aware local evidence paths are wired into the beta gates.

**Certification state:** source remediation/integration in progress; native same-SHA Linux/Windows/macOS evidence remains required before publishing Beta 4.

## 1.0.0-beta.3 — 2026-09-25 — V1 functional closure

### Agent Interop e execução

- ExecutionRole consolidado para Build, Investigate, Review, Verify, Architect e Security.
- OpenCode Agent Projection com renderer version-aware, ownership fail-closed e receipts duráveis.
- Discovery nativo sem LLM e activation probe explícito com confirmação de custo/quota.
- Enforcement read-only certificado separadamente: Nexus exige prova do toolset/permission efetivo do OpenCode ou rejeição explícita de write/edit/patch, sempre com canário ausente antes de persistir `ENFORCED`.
- Troca de role no terminal tenta projection OpenCode automaticamente sem chamada ao modelo; fallback
  permanece Nexus overlay quando native projection não é possível.
- Controle **Certify** exige confirmação e executa activation + enforcement somente para roles
  read-only/verify-only.
- Command Palette expõe as roles do terminal ativo e `Ctrl/Cmd+Shift+A` foca o selector sem
  capturar `Shift+Tab` do TUI hospedado.
- Providers sem adapter nativo continuam usando o mesmo ExecutionRole por Nexus overlay; a beta.3
  só anuncia OpenCode como native projection.
- Custom execution targets para Docker/VPN Workspace, Docker Compose, WSL, SSH e wrappers, mantendo
  provider lógico separado do target.
- Custom target não exige provider instalado no host e suporta `{cwd}`, `{target_cwd}`,
  `{provider}`, `{profile}`, `{profile_home}` e `{args}`.

### Voz e acessibilidade

- Nexus Voice Accessibility integrado à V1.
- Groq / Whisper Large V3 Turbo como STT primário e Cloudflare Workers AI / Whisper como fallback.
- Entrada de voz global e no terminal; transcrição entra no PTY sem Enter automático.
- Settings com providers reordenáveis, secrets mantidos no backend e fallback tipado.
- Browser capture por MediaRecorder; fallback Linux por PipeWire/ALSA quando disponível; integração
  WebView2 no Windows e WKWebView/macOS preparada para permissão nativa.

### UX e confiabilidade

- Mount only the active product surface; inactive ones unmount while TerminalsHost stays alive
  (no PTY recreation, no history loss).
- Flow drafts persist across surface switches; stale refreshes are dropped by a latest-wins guard.
- BLOCKED_NEEDS_USER resolves through the run intervention instead of a misleading Resume/Return.
- Flow Run UX hierarchy with progressive disclosure, localized in pt-BR/en/es.

### Estado de release

- `1.0.0-beta.3` é o source candidato da V1.
- Implementação não substitui homologação exact-SHA. CI remoto segue indisponível enquanto a conta
  GitHub Actions estiver bloqueada por créditos; homologação local/native continua obrigatória.

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
