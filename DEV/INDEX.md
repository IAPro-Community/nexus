# DEV Master Index

Guia e índice de navegação estruturada da documentação de engenharia e governança do IAPro Nexus.

**Current release status for HEAD:** [`validation/current/RELEASE_STATUS.md`](validation/current/RELEASE_STATUS.md)

Reports under `validation/FINAL_*.md` are historical unless they live in `validation/current/` and carry matching `GIT_SHA`.

Release evidence uses `GIT_SHA` for the validated code revision and, when a
documentation-only evidence commit is authorized, `EVIDENCE_COMMIT` for that
commit. A dirty worktree is recorded as `WORKTREE_FROZEN: NO`; it must not be
represented as a frozen release candidate. The current real-provider attempt
and its honest status live in [`validation/current/real-provider-mission/`](validation/current/real-provider-mission/).

O código canônico utiliza o binário `nexus`, mantendo `ai` como alias transparente de compatibilidade.

---

## 1. Governança e Estado Ativo

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
Decision record: [`DEV/DECISIONS/NEXUS_TERMINAL_CONTINUITY.md`](DECISIONS/NEXUS_TERMINAL_CONTINUITY.md)
- [Nexus 1.0 delivery meta](NEXUS_1_DELIVERY_META.md)
