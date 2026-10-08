# IAPro Nexus 1.0.0-beta.7 — Release Notes

Data: 7 de outubro de 2026

SHA-base homologado: 52f35c04181b85379219ab55fc3192c63a5440a8

Estado: validação local Windows; publicação NO-GO até congelamento do source, CI nativo e assinatura.

## Destaques

- Suporte a ConPTY e transporte de terminal desacoplado no Windows.
- Internacionalização nativa da CLI e seleção dinâmica de localidade.
- Oito suítes browser E2E aprovadas em Windows AMD64.
- Conformidade estática do host e correções de terminal tracing para Windows.

## Correções desta continuação Windows

- Wails v3 usa o Host wails.localhost para o Core embutido. A política aceita somente esse Host exato com origem desktop reconhecida; origens externas e hosts lookalike continuam rejeitados.
- O Project Hub apresenta erros em toda a largura do formulário e mantém Adicionar Projeto em uma linha.
- Builds locais de CLI, Desktop e HUD foram produzidos com a marca 52f35c04-dirty. São artefatos de validação, não de distribuição.

## Estado de release

A árvore de trabalho contém alterações locais sem commit. Os gates Go e Web passaram; as oito suítes E2E passaram no worktree Windows isolado com as mesmas alterações. O HUD permaneceu ativo no smoke de oito segundos. A repetição automatizada do Desktop encerrou com código 0 antes de oito segundos tanto no build atual quanto no binário de baseline, e precisa ser repetida numa sessão Windows interativa.

Os binários são NotSigned e não têm trust root de atualização. O workflow exact-SHA aguarda runner Windows e faltam certificado/secrets Authenticode. Não há tag v1.0.0-beta.7 ou GitHub Release. Não publicar antes de fechar essas pendências.

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
