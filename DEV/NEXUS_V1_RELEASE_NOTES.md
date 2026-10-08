# IAPro Nexus 1.0.0-beta.8 — Release Notes (candidato)

Data: 8 de outubro de 2026

SHA homologado localmente no Windows AMD64: `ab397c5449cc76dfd07b44e335f45de6762a94f4`.

Estado: gates automatizados locais Windows passaram neste SHA. A release oficial continua NO-GO até CI nativo exact-SHA, assinatura/trust root e smokes manuais restantes.

## Destaques

- Sessões supervisionadas do Codex usam ConPTY automaticamente no Windows, sem exigir `AI_CLI_ENABLE_CONPTY=1`; outros providers mantêm standard pipes por padrão.
- A seleção de backend do terminal e o capability report agora refletem o provider. `AI_CLI_ENABLE_CONPTY=1` segue como override global, e `AI_CLI_DISABLE_CONPTY=1` permite desativá-lo.
- Oito suítes browser E2E passaram no Windows AMD64, além das verificações de acessibilidade e visual.
- Go fmt/vet/lint/security/testes, qualidade Web e builds Windows CLI/Desktop/HUD passaram no mesmo SHA.
- Smokes nativos isolados do Desktop/Wails/WebView2 e HUD passaram no Windows (13 s e 8 s); eles confirmam inicialização e criação do WebView2, sem substituir a revisão visual e os fluxos manuais pendentes.

## Evidência Windows do terminal Codex

O teste de integração `TestCodexSessionHostDefaultsToConPTYAndForwardsArrowInput` confirmou console não redirecionado e encaminhamento de seta como `DownArrow`. O smoke interativo com perfil isolado abriu a tela de boas-vindas/login do Codex v0.160.1 usando o padrão do produto, sem login nem prompt, e encerrou limpo com Ctrl+C.

## Estado da release

A branch `fix/windows-beta8-homologation-20261008` foi enviada e está vinculada por PR draft #45 a `release/v1.0.0`; ainda não foi integrada. CLI, Desktop e HUD foram reconstruídos para `ab397c54`; seus checksums estão em `outputs/windows-beta8-distribution-ab397c54-20261008/` e os binários são `NotSigned`. O NSIS installer e seu smoke de `7d2ef8aa` pertencem ao candidato anterior; `makensis.exe` não está disponível neste host para reconstrução atual. CI Windows exact-SHA, Authenticode/trust root e smokes manuais de provider/Mission/Voice continuam pendentes. Nenhuma tag foi criada. Esta continuidade executou somente homologação Windows.

---

## Histórico — IAPro Nexus 1.0.0-beta.7

Data: 8 de outubro de 2026

SHA validado localmente no Windows AMD64: `d840797bbd975a5c7edc54ebe23f3dfef56ff947`.

Estado: os gates automatizados locais de Windows passaram neste SHA. A release oficial permanece NO-GO até CI nativo exact-SHA, assinatura de binários/manifesto e gates manuais pendentes.

### Destaques

- Suporte a ConPTY e transporte de terminal desacoplado no Windows.
- Internacionalização nativa da CLI e seleção dinâmica de localidade.
- Oito suítes browser E2E aprovadas em Windows AMD64, além de acessibilidade e visual.
- Conformidade estática do host e correções de terminal tracing para Windows.
- Desktop Wails/WebView2 e HUD passaram smokes de inicialização com perfis isolados.

### Correções desta continuação Windows

- Wails v3 usa o Host `wails.localhost` para o Core embutido. A política aceita somente esse Host exato com origem desktop reconhecida; origens externas e hosts lookalike continuam rejeitados.
- O Project Hub apresenta erros em toda a largura do formulário e mantém Adicionar Projeto em uma linha.
- A identidade Git do contexto limita subprocessos que travam e falha fechado como incompleta; o homologador usa um `TEMP` curto isolado para evitar limites de caminho em fixtures NTFS.
- Builds CLI, Desktop e HUD foram produzidos para Windows AMD64 com metadados do commit `d840797b`. Checksums: `DEV/validation/local-beta/1.0.0-beta.7/d840797bbd975a5c7edc54ebe23f3dfef56ff947/windows-AMD64/distribution-artifacts/CHECKSUMS.md`.

### Estado de release

PASS local: PowerShell/docs, Bun frozen install e `quality:full` (82 arquivos/487 testes), Go formatting/vet/lint/security/tests, regressões de control/runtime e Agent Interop, oito suítes browser E2E, acessibilidade/visual, builds CLI/Desktop/HUD e smokes nativos Desktop/HUD.

Os executáveis são `NotSigned`. A chave pública de update está versionada e foi usada no build; este host não tem `signtool.exe`, certificado/secrets Authenticode, NSIS ou os secrets `NEXUS_UPDATE_PRIVATE_KEY`/`NEXUS_UPDATE_KEY_ID` para assinar manifestos. O CI nativo exact-SHA não foi revalidado e o instalador final não foi construído. Smokes interativos de providers/Mission/Voice e captura visual de renderização Desktop ainda não foram feitos neste SHA. O diagnóstico local também sinalizou que o Gemini CLI instalado usa Node 18.12.1 e não iniciou; isso não é contabilizado como provider smoke aprovado.

O remoto confirmou `origin/release/v1.0.0` em `52f35c04181b85379219ab55fc3192c63a5440a8` antes desta continuidade; o candidato local é descendente de `d14154d2`. O código e os relatórios foram enviados em fast-forward; o primeiro HEAD remoto confirmado foi `f1928d0f75cf49a0968f248bd7dc9b8e54655dcb`. A tag `v1.0.0-beta.7` continua ausente. Não criar tag nem publicar enquanto os gates externos e manuais permanecerem abertos.

---

## Registro histórico — notas anteriores v0.5.0-beta.4
**Release**: `v0.5.0-beta.4`  
**Commit**: `50bf05d` (synchronized with source HEAD)  
**Status histórico**: release candidate descrito na data deste registro

## Key Highlights
1. **Nexus Intelligence Layer**: Provider-neutral intent analysis, ambiguity evaluation (BLOCKING/IMPORTANT/LOW_IMPACT), context assembly, and deterministic prompt compilation.
2. **Structured WorkPlans & Prompt Compilation**: Hierarchical decomposition (`Mission -> Phase -> WorkPackage -> Task`), versioned immutable `PlanRevision`s, and `ExecutionSnapshot`s backed by SQLite.
3. **Visual Plan Builder**: Interactive Workspace OS interface for editing, reordering, and compiling prompts for structured work packages.
4. **Autonomous Mission Runner**: Durable state machine (`READY -> ALLOCATE -> COMPILE -> EXECUTE -> TEST -> REVIEW -> VERIFY`), bounded retries, and independent review verification loop.
5. **Resource Recommendation Service**: Multi-criteria explainable scoring (`BALANCED`, `PRESERVE_QUOTA`, `PREFER_PROVIDER`, `MANUAL`) based on real-time quotas and health.
6. **Hardened Security & Single-Writer Authority**: Unified terminal writer leases, loopback/private interface enforcement, session expiration/rotation, and Git branch protection.
7. **Web-First CLI Experience**: Default web launch (`nexus` / `ai`) with non-interactive subcommands (`plan`, `agents`, `projects`, `doctor`, `status`).
