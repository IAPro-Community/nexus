# Matriz funcional — Nexus 1.0

> Gerada a partir de [features.json](features.json). **Presença do código e dos testes não significa que tenham passado no SHA atual.** A versão estável continua sujeita à [homologação](../operations/platform-support.md).
>
> Fonte dos recursos: [catálogo auditável](features.json). Atualize o catálogo e execute `node scripts/verify-feature-catalog.mjs --write` para regenerar esta tabela.

O [contrato integral do Agent Workspace Nexus 1.0](../product/agent-workspace-v1-acceptance.md)
é uma **matriz de aceite de release separada** deste inventário de código:
engloba 48 obrigações (incluindo as 30 capacidades UX) e bloqueia promoção
enquanto faltar implementação e prova exact-SHA. Não interpretar
`implemented` abaixo como `VERIFIED`.

<!-- FEATURE_CATALOG_START -->
| ID | Recurso e documentação | Superfícies no source | Estado no source | Evidência de testes | Certificação nativa |
| --- | --- | --- | --- | --- | --- |
| projects | [Projetos e workspaces](../product/overview.md) | `core`, `web`, `cli`, `api` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| direct | [Direct e AI Session](../product/direct.md) | `web`, `cli`, `core` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| agents | [Agents persistentes](../product/agents.md) | `core`, `web`, `cli`, `api` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| terminals | [Terminal supervisionado e continuidade](../product/terminals.md) | `core`, `web`, `cli`, `desktop` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| providers | [Providers e perfis](../product/providers-and-usage.md) | `core`, `web`, `cli` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| account-handoff | [Troca de conta e handoff](../product/account-switching.md) | `core`, `cli` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| quota | [Quota e elegibilidade](../product/providers-and-usage.md) | `core`, `cli`, `web`, `api` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| capacity-monitor | [Monitor de capacidade](../product/providers-and-usage.md) | `core`, `web`, `api`, `hud` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| pace | [Previsão de consumo de quota](../architecture/quota/pace-intelligence.md) | `core`, `web`, `hud`, `api` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| cost | [Cost Intelligence](../architecture/quota/EXTERNAL_USAGE_AND_PUBLIC_STATUS.md) | `core`, `web`, `api`, `hud` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| external-usage | [Importação manual de uso Claude](../architecture/quota/EXTERNAL_USAGE_AND_PUBLIC_STATUS.md) | `core`, `web`, `api` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| public-status | [Status público de serviços](../architecture/quota/EXTERNAL_USAGE_AND_PUBLIC_STATUS.md) | `web`, `api` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| hud | [HUD compacto e desktop](../product/hud.md) | `core`, `web`, `desktop`, `hud`, `api` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| composer | [Composer](../product/composer.md) | `core`, `web`, `api` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| flow | [WorkPlan e Flow](../product/flow.md) | `core`, `web`, `cli`, `api` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| missions | [MissionRunner durável](../product/missions.md) | `core`, `web`, `cli`, `api` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| coordination | [Coordenação de Agents](../product/agent-coordination.md) | `core`, `web`, `api` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| conductor | [Conductor](../product/conductor.md) | `core`, `web`, `api` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| scheduler | [ResourceScheduler](../architecture/resource-routing.md) | `core`, `cli`, `web` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| maestro | [Integração Maestro](../product/maestro.md) | `core`, `web`, `cli` | optional | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| voice | [Entrada e controle por voz](../product/voice-accessibility.md) | `web`, `desktop`, `core` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| themes | [Temas e aparência](../product/themes.md) | `web`, `desktop` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| remote | [Acesso remoto autenticado](../product/remote-access.md) | `web`, `cli`, `api` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| auth | [Sessões, autenticação e CSRF](../security.pt-BR.md) | `core`, `web`, `api` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| updates | [Atualização e integridade](../architecture/updates.md) | `core`, `cli`, `api` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| doctor | [Diagnósticos](../operations/troubleshooting.md) | `core`, `cli` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| desktop | [Desktop nativo](../desktop/overview.md) | `desktop`, `core` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| api-local | [API local](../api/README.md) | `api`, `web`, `desktop` | internal | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| cli | [CLI e automação](../cli/README.md) | `cli`, `core` | implemented | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| installer | [Instalação e desinstalação](../getting-started/installation.md) | `cli`, `desktop` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| browser-panel-scroll | [Rolagem do workspace Web](../architecture/WORKSPACE_STYLES_MODULARIZATION.md) | `web` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| windows-terminal-input | [Transporte de entrada Web para PTY/ConPTY](../product/terminals.md) | `core`, `web`, `desktop` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| sticky-lead-explicit-review | [STICKY_LEAD: revisão explícita sem Mission](../product/interactive-control.md) | `core`, `cli`, `web` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| runtime-quota-continuity | [Continuidade por esgotamento de quota](../architecture/session-continuity.md) | `core`, `cli` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| execution-target-catalog | [ExecutionTarget, CapacityDomain e ModelCatalog](../architecture/resource-routing.md) | `core` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
| desktop-static-assets | [Entrega de assets no attach Desktop](../desktop/overview.md) | `desktop`, `core` | experimental | SOURCE_TEST_PRESENT_NOT_RUN | PENDING_EXACT_SHA |
<!-- FEATURE_CATALOG_END -->

## Interpretação

- `implemented`: há implementação e teste identificados, **não** garantia de execução real ou release certificada.
- `experimental`: integração/limitação ainda precisa de homologação do fluxo completo.
- `optional`: integração não necessária ao funcionamento básico.
- `internal`: contrato do produto, não API pública estável.
- `SOURCE_TEST_PRESENT_NOT_RUN`: testes existem, mas não foram executados nesta revisão documental.
- `PENDING_EXACT_SHA`: plataforma/superfície ainda não certificada no SHA da release em análise.

Para detalhes de implementação e limites, consulte [features.json](features.json). Suporte real exige smoke e evidências específicas do mesmo commit. O código listado não autoriza afirmar que todas as superfícies funcionam em Linux, Windows ou macOS.
