# DEV Master Index

> **Documentação pública canônica:** [README](../README.md) →
> [docs/README](../docs/README.md) →
> [Como funciona](../docs/product/how-it-works.md) →
> [Arquitetura](../ARCHITECTURE.md).
>
> `DEV/` é ledger de engenharia, specs e evidência. Relatórios históricos não
> substituem a documentação pública nem o
> [status corrente de release](validation/current/RELEASE_STATUS.md).

## Fechamento beta.10 — 22/10/2026

- [Reconciliação de código, documentação e evidências](validation/current/BETA10_GAP_CLOSURE_20261022.md):
  delta de 27 commits, contratos públicos, seis recursos adicionados ao catálogo,
  CI/Windows exact-SHA, falha de formatação e lista de gates pendentes.
- [Changelog público](../CHANGELOG.md) e [status de release](../docs/release-status.md)
  não significam aprovação de plataforma; a certificação depende de um HEAD congelado.

## Contratos públicos v1

- [Como o Nexus funciona](../docs/product/how-it-works.md)
- [Coordenação entre Agents](../docs/product/agent-coordination.md)
- [Runtime da Mission](../docs/architecture/mission-runtime.md)
- [Guardrails](../docs/architecture/guardrails.md)
- [Modelo mental](../docs/product/mental-model.md)
- [Matriz de funcionalidades](../docs/reference/feature-matrix.md)


Guia e índice de navegação estruturada da documentação de engenharia e governança do IAPro Nexus.

**Current release status for HEAD:**

**Matriz base da homologação Linux AMD64 (2026-10-25):** [relatório](validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md) · [matriz](validation/linux-100-2026-10-25/TEST_MATRIX.md) · [resumo JSON](validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_SUMMARY.json) · [validação do mascote único](validation/linux-100-2026-10-25/MASCOT_VALIDATION.md) · [escopo confirmado](validation/linux-100-2026-10-25/COMPANION_VALIDATION.md). Resultado NO_GO_LINUX (63 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN; snapshot de overlay `516b706a…`, base `bf6adf0c…`). Produto: um único mascote NEX; não há modo ou renderer Companion 2D. O balão nativo e “Adiar” passaram com fixture, sem substituir a validação pendente de atenção gerada por agente real. [`validation/current/RELEASE_STATUS.md`](validation/current/RELEASE_STATUS.md)

Follow-up nativo do Attention Center (2026-10-25): [validação de filtros, navegação para decisão e layout](validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md). O cartão agora abre a missão no workspace; a evidência vem de fixture isolada e inclui teste SQLite após reopen. Fingerprint do overlay: `7368531977b7422eccdf00e7c676f57d440e8d736d9e114bcdc071b9d5885fe6`; o veredito geral continua NO_GO_LINUX.

Follow-ups da mesma campanha: [PRs #57/#58 e testes Linux](validation/linux-100-2026-10-25/PR578_FOLLOWUP.md) · [PR #61, busca do portal e contrato do mascote](validation/linux-100-2026-10-25/PR61_DOCS_SITE_VALIDATION.md).

Latest Windows beta.7 evidence: [BETA7_WINDOWS_HOMOLOGATION_20261007.md](validation/current/BETA7_WINDOWS_HOMOLOGATION_20261007.md). NSIS Path-state smoke: [BETA7_NSIS_PATH_STATE_SMOKE_20261007.md](validation/current/BETA7_NSIS_PATH_STATE_SMOKE_20261007.md).

Reports under `validation/FINAL_*.md` are historical unless they live in `validation/current/` and carry matching `GIT_SHA`.

Release evidence uses `GIT_SHA` for the validated code revision and, when a
documentation-only evidence commit is authorized, `EVIDENCE_COMMIT` for that
commit. A dirty worktree is recorded as `WORKTREE_FROZEN: NO`; it must not be
represented as a frozen release candidate. The current real-provider attempt
and its honest status live in [`validation/current/real-provider-mission/`](validation/current/real-provider-mission/).

O código canônico utiliza o binário `nexus`, mantendo `ai` como alias transparente de compatibilidade.

---

## Agent Interop incorporado à linha v1 beta

- [`SPECS/NEXUS_V1_AGENT_INTEROP.md`](SPECS/NEXUS_V1_AGENT_INTEROP.md): spec canônica da V1 para ExecutionRole, Agent Projection, capability truth e interoperabilidade de agents.
- [`docs/architecture/ADR-agent-interop-projection.md`](../docs/architecture/ADR-agent-interop-projection.md): decisão de projetar o modelo canônico de Agents para hosts externos sem criar um segundo runtime/orquestrador.

A decisão de release de 24/09/2026 incorporou esse escopo ao candidato
`v1.0.0-beta.3`. Os nomes de arquivo “1_1” são preservados por rastreabilidade,
não representam mais uma linha de release separada. O status operacional segue
`docs/release-status.md`.

---

## 1. Governança e Estado Ativo

- [`SPECS/NEXUS_CONDUCTOR.md`](SPECS/NEXUS_CONDUCTOR.md): Spec técnica do Nexus Conductor — orquestração nativa de Goals → Missions.
- [`SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md`](SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md): Plano executável Luna para CI, quota, pools, terminal supervisionado e continuidade nos três SOs.
- [`HANDOFF.md`](HANDOFF.md): Estado atual de execução e próxima ação recomendada.
- [`CONTEXT.md`](CONTEXT.md): Arquitetura corrente, runtime, drivers e restrições operacionais.
- [`SPECS/ACTIVE.md`](SPECS/ACTIVE.md): Contrato ativo de trabalho e critérios de aceitação vigentes.
- [`VERIFY.md`](VERIFY.md): Evidências e histórico consolidado de validações e gates.
- [`NEXUS_1_FINAL_ACCEPTANCE.md`](NEXUS_1_FINAL_ACCEPTANCE.md): ledger executável da aceitação RC do Nexus 1.0.
- [`WORKLOG.md`](WORKLOG.md): Diário de bordo detalhado de todas as implementações e mudanças.
- [`../DOCUMENTATION_COMPLETE.md`](../DOCUMENTATION_COMPLETE.md): entrega de documentação pública; inventário e ledger estão em `docs/internal/`.

---

## 2. Arquitetura, Interoperabilidade e Especificações Canônicas

- [`docs/architecture/quota/account-isolation.md`](../docs/architecture/quota/account-isolation.md): fronteira canônica de estado por conta.
- [`docs/architecture/quota/identity-scoping.md`](../docs/architecture/quota/identity-scoping.md): identidade persistida e versionamento de credencial.
- [`docs/architecture/providers/registration-lifecycle.md`](../docs/architecture/providers/registration-lifecycle.md): separação entre instalação e registro.
- [`docs/architecture/agent-routing.md`](../docs/architecture/agent-routing.md): matching canônico de Agents persistentes.
- [`docs/architecture/resource-routing.md`](../docs/architecture/resource-routing.md): seleção de recursos e políticas do scheduler.
- [`docs/architecture/mission-teams.md`](../docs/architecture/mission-teams.md): projeção de Mission Teams sobre Flow, WorkPlan e MissionRunner.
- [`docs/architecture/session-continuity.md`](../docs/architecture/session-continuity.md): estados e limites de continuidade.
- [`docs/architecture/ADR-module-path-iapro-community.md`](../docs/architecture/ADR-module-path-iapro-community.md): decisão do module path canônico.
- [`docs/architecture/quota/legacy-cache-migration.md`](../docs/architecture/quota/legacy-cache-migration.md): tratamento não destrutivo de cache legado.

- [`NEXUS_V1_ARCHITECTURE.md`](NEXUS_V1_ARCHITECTURE.md): Arquitetura mestre do Workspace OS, runtimes e store.
- [`NEXUS_CANONICAL_ALIGNMENT.md`](NEXUS_CANONICAL_ALIGNMENT.md): Alinhamento canônico entre CLI, Web e drivers de provedores.
- [`NEXUS_CAPABILITY_PRESERVATION.md`](NEXUS_CAPABILITY_PRESERVATION.md): Matriz de preservação honesta de capacidades.
- [`NEXUS_V1_AGENT_MODEL.md`](NEXUS_V1_AGENT_MODEL.md): Modelo de agentes duráveis e persistentes.
- [`NEXUS_V1_MAESTRO_INTEGRATION.md`](NEXUS_V1_MAESTRO_INTEGRATION.md): Integração profunda e governança com o Orquestrador Maestro.
- [`BRANDING_AND_ASSETS.md`](BRANDING_AND_ASSETS.md): Identidade visual, catálogo de assets, favicon, script de extração e propostas de design.

---

## 3. Matrizes de Plataforma, Provedores e Segurança

- [`FINAL_PROVIDER_MATRIX.md`](FINAL_PROVIDER_MATRIX.md): Tabela de suporte e comportamento de Codex, AGY, Claude, OpenCode, Gemini e Cursor.
- [`FINAL_PLATFORM_MATRIX.md`](FINAL_PLATFORM_MATRIX.md): Compatibilidade entre Linux, macOS e Windows.
- [`FINAL_SECURITY_REPORT.md`](FINAL_SECURITY_REPORT.md) & [`NEXUS_MAXIMUM_DELIVERY_SECURITY.md`](NEXUS_MAXIMUM_DELIVERY_SECURITY.md): Isolamento de processos, sandbox, redação de credenciais e integridade de D-Bus/keyrings.

---

## 4. Frontend e Validação Canônica

- [`validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md): Último gate do frontend (`make web-verify`).
- [`validation/FRONTEND_HISTORY.md`](validation/FRONTEND_HISTORY.md): Histórico compacto dos gates de frontend.
- [`validation/CURRENT_STATE_AUDIT.md`](validation/CURRENT_STATE_AUDIT.md): Inventário evidence-backed da campanha atual.
- [`validation/FINAL_CLOSURE_REALITY_AUDIT.md`](validation/FINAL_CLOSURE_REALITY_AUDIT.md) e [`validation/FINAL_CLOSURE_CHECKPOINTS.md`](validation/FINAL_CLOSURE_CHECKPOINTS.md): auditoria e checkpoints resumíveis da consolidação final.
- [`validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md`](validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md): Relatório final de plataforma, Desktop, Update e release.
- [`../docs/product/autonomous-delivery.md`](../docs/product/autonomous-delivery.md): experiência de entrega autônoma para usuários.
- [`../docs/operations/mission-teams.md`](../docs/operations/mission-teams.md): operação, recuperação e evidências de Mission Teams.
- [`../docs/testing/mission-teams.md`](../docs/testing/mission-teams.md): cenários e gates de teste do pipeline.
- Após alterações em `web/`, execute `make web-verify` (ou `npm --prefix web run verify`).
- Em caso de inconsistência visual na Web, faça o rebuild completo do binário com `make build` e reinicie o processo.

---

## 5. Histórico e Relatórios de Release

- [`FINAL_RELEASE_REPORT.md`](FINAL_RELEASE_REPORT.md): Relatório de fechamento de releases estáveis.
- [`NEXUS_V0_FINAL_AUDIT.md`](NEXUS_V0_FINAL_AUDIT.md) & [`NEXUS_V1_FINAL_ENGINEERING_REPORT.md`](NEXUS_V1_FINAL_ENGINEERING_REPORT.md): Auditorias técnicas de consolidação.
- [`validation/production-go-no-go/FINAL_REPORT.md`](validation/production-go-no-go/FINAL_REPORT.md): red team independente — `PRODUCTION_NO_GO`.
- [`validation/overnight-production-certification/FINAL_REPORT.md`](validation/overnight-production-certification/FINAL_REPORT.md): aborto `ABORTED_PRECONDITION_FAILED` da certificação overnight.
- [`validation/production-finalization/FINAL_REPORT.md`](validation/production-finalization/FINAL_REPORT.md): evidência da campanha de fechamento do release candidate.
- [`DEV/validation/`](validation/): Logs brutos, capturas de tela e artefatos de testes automatizados e manuais.
Decision record:
- [ADR-20261007-WAILS-LOCALHOST-ORIGIN.md](DECISIONS/ADR-20261007-WAILS-LOCALHOST-ORIGIN.md): allowlist exata do host virtual Wails no Desktop Windows.
- [Release Notes beta.7 Windows](NEXUS_V1_RELEASE_NOTES.md): destaques e status local de publicação. [`DEV/DECISIONS/NEXUS_TERMINAL_CONTINUITY.md`](DECISIONS/NEXUS_TERMINAL_CONTINUITY.md)
- [Nexus 1.0 delivery meta](NEXUS_1_DELIVERY_META.md)
