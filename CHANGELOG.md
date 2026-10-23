# Changelog

## 1.0.0-beta.10 — adendo de source — 2026-10-22 (não homologado)

Esta atualização documenta a reconciliação de 27 commits entre
`622aa933` e `e98c711` na `release/v1.0.0`, sem criar outra beta ou tag.

### Código integrado

- Scroll interno dos painéis Web, preservando o layout nativo do Desktop.
- Entrega correta de JS/CSS no attach do Desktop antes do fallback SPA.
- Entrada do terminal Web/Windows por bytes diretos no stream anexado,
  sem envelopar cada tecla como `CmdInput` RPC.
- `STICKY_LEAD` com exceção restrita de `DELEGATE` explícito e read-only
  em `OFF`, sem autorizar Mission automática (TTY e Lead interativo Web).
- `ContinuityController` acionado por quota observada, preferência por
  outra conta do mesmo provider, exclusão de capacidade upstream compartilhada,
  hop limit, eventos de substituição e reattach do Lead.
- `ExecutionTarget`, `CapacityDomain`, `ModelCatalog` e preferências de
  especialistas sem converter preferências em `PIN` ou inventar modelos.

### Documentação e validação

- Guias de terminal, continuidade, routing, Desktop e eventos atualizados.
- Matriz funcional regenerada com **36** recursos rastreáveis no source.
- Referência de código: `e98c711e86ffd578111cfca454513722198bacbe`.
- [Homologação Windows exact-SHA #39870167188](https://github.com/IAPro-Community/nexus/actions/runs/39870167188): **FAIL** em `format:check` (`GuardianMascot.test.tsx`); `docs-verify` passou no run anterior.
- Correção de estilo registrada posteriormente no commit `43c48376`;
  sem afirmar resultado de nova CI para o HEAD dos documentos.
- Registro de evidência e gates: [reconciliação beta.10](DEV/validation/current/BETA10_GAP_CLOSURE_20261022.md).

**Decisão:** código presente, sem certificação nativa same-SHA; release ainda
**NO-GO**. Não distribuir como stable v1 por este adendo.


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


## 1.0.0-beta.7 — 2026-10-08 — Windows validation continuation

Windows AMD64 automated gates passed locally on source SHA `d840797bbd975a5c7edc54ebe23f3dfef56ff947`. This is a local technical PASS; official publication remains NO-GO until exact-SHA native CI, signing/trust-root and remaining manual release gates close.

### Adicionado

- Suporte a ConPTY e transporte terminal desacoplado no Windows.
- Internacionalização nativa da CLI e seleção dinâmica de localidade, incluindo pt-BR.
- Oito suítes browser E2E aprovadas em Windows AMD64, mais as tarefas de acessibilidade e visual.

### Corrigido

- Conformidade estática do host ao processar entrada de terminal e compatibilidade do terminal tracing no Windows.
- Testes de terminal tracing para permissões NTFS e escape de HTML/JSON.
- Teste de cabeçalho de uso da CLI para saída localizada, incluindo Uso:.
- Flakiness de testes Go sob carga Windows e escopo correto das funções de uninstall NSIS.
- Contrato estático de release atualizado para a função un namespaced do uninstaller NSIS.

### Validação Windows

- Go fmt, vet, govulncheck, golangci-lint e suíte completa sem cache: PASS no SHA 52f35c04181b85379219ab55fc3192c63a5440a8.
- Bun frozen-lockfile e quality:full: PASS, 82 arquivos e 487 testes unitários.
- Oito suítes Playwright browser E2E: PASS.
- `bun install --cwd web --frozen-lockfile` e `bun run --cwd web quality:full`: PASS (Bun 1.3.9, Node 22.14.0, 82 arquivos/487 testes).
- Go formatting, vet, golangci-lint, govulncheck, suíte `go test -count=1 ./...`, regressões control/runtime e Agent Interop: PASS.
- Builds CLI/Desktop/HUD e smokes nativos isolados Desktop/WebView2 e HUD: PASS; `nexus version --json` reporta 1.0.0-beta.7 e commit d840797b.
- Binários e ZIPs locais, com checksums: `DEV/validation/local-beta/1.0.0-beta.7/d840797bbd975a5c7edc54ebe23f3dfef56ff947/windows-AMD64/distribution-artifacts/CHECKSUMS.md`.
- Todos os executáveis são `NotSigned`; o host não tem `signtool.exe`, certificado/secrets Authenticode nem NSIS. A chave pública de update está versionada; os secrets para assinar manifestos não estão configurados localmente.

### Estado da publicação

O remoto confirmou `origin/release/v1.0.0` em `52f35c04181b85379219ab55fc3192c63a5440a8` antes desta continuidade; o candidato local `d840797b` é descendente do SHA KIVER `d14154d2`. A tag Beta 7 não existe no remoto. O estado atual de GitHub Actions e do inventário de Releases não foi consultado. O smoke do instalador exato não foi executado porque `makensis.exe` não está disponível. Não criar tag nem publicar enquanto esses gates permanecerem abertos.
## 1.0.0-beta.6 — 2026-09-29 — Usage/Capacity V2 and single release line

- Composer Interview Engine: added per-dimension clarity/ambiguity, information-gain question priority, bounded QUICK/STANDARD/DEEP profiles and a single pressure-test pass for hidden trade-offs. Repository/memory-inferred facts are no longer surfaced by the web workspace as open questions.
- Usage semantics: introduced provider-boundary `UsageObservation`, explicit `USED_PERCENT` / `REMAINING_PERCENT` semantics and independent `SUBSCRIPTION` / `API` resource channels without changing existing `UsageSnapshot` consumers.
- Codex/AGY: percentage inversion/preservation now goes through one normalization function; Codex `0% used` is contract-tested as `100% remaining`, while AGY remaining percentages remain remaining percentages.
- Multi-account routing: when every usable profile only has `ESTIMATED` quota evidence, Nexus ranks the measured estimates instead of discarding them and falling back to unrelated LRU. `LIVE/CACHED` evidence still outranks any estimate.
- Capacity API: `quota_remaining` is again literal remaining capacity of the best usable pool (0..1). Multi-pool breadth remains an internal scheduler signal and is no longer encoded into the public quota field.
- Provider compatibility: scheduler regressions cover Codex, AGY, Claude, OpenCode, Cursor and Gemini generically; AGY independent pool semantics remain covered separately.
- Release engineering: `release/v1.0.0` is the only active release line. Release publication now fails closed unless the candidate SHA is exactly the canonical branch HEAD; CI no longer treats arbitrary `release/**` branches as active release targets.
- Distribution remains a single rolling prerelease entry (`channel-beta`); historical beta branches/tags are evidence only and cannot authorize a new publication.

## 1.0.0-beta.5 — 2026-09-28 — terminal paste & mouse isolation

- Terminal paste atomic: preserved bracketed paste mode (`\x1b[?2004h`) in `TerminalOutputScrubber` so multiline text pasted into AGY and other provider TUIs enters atomically as a single block instead of submitting line-by-line with premature execution.
- Terminal mouse tracking isolation: hardened `KeyboardModeReset` and `terminalKeyboardModeReset` across all TTY control and runtime recovery paths to explicitly disable normal (`?1000`), button-event (`?1002`), any-event (`?1003`), SGR extended (`?1006`), and urxvt (`?1015`) mouse tracking modes upon session exit, interrupt, or error.
- Web terminal input sanitization: filtered raw and ESC-prefixed SGR mouse reports (`\x1b[<...M` and `[<...M`) in xterm.js forwarder and embedded metadata stripper.
- LineShadow promotion cleanup: ensured delimiter stripping in `flushPasteBuf` does not leave residual leading spaces from replaced escape bytes.

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

## Aditamento Windows — 2026-10-07 — worktree local

- Corrigido invalid origin no WebView Wails: somente o Host exato wails.localhost passa quando uma origem desktop é reconhecida; hosts lookalike continuam bloqueados.
- Corrigido o layout do Project Hub: erros ocupam toda a largura do formulário e o botão Adicionar Projeto mantém o rótulo em uma linha.
- Acrescentados testes de segurança de origem e regressão visual E2E no viewport Windows 775x483.
- Os gates Go e Web locais passaram. As oito suítes E2E passaram no worktree Windows isolado com as mesmas alterações.
- O Desktop foi recompilado para validação, mas o reteste automatizado encerrou com código 0 antes de oito segundos; o binário de baseline repetiu o resultado. Exigir smoke nativo interativo na próxima certificação exact-SHA.
- Os novos executáveis e checksums são somente evidência local: NotSigned, sem trust root, sem commit e sem publicação.
