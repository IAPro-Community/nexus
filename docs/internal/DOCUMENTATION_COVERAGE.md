# Cobertura de documentação

Auditoria atualizada em **2026-09-22** para a linha
`feat/nexus-1.0-closure`.

“Documentada” significa que existe uma página pública canônica, orientada a
uso ou referência, ligada ao mapa de documentação. Isso **não** significa
certificação operacional de toda plataforma/provider nem autorização da
release estável.

## Cobertura revisada

| Área | Cobertura pública | Estado |
| --- | --- | --- |
| Visão geral e jornada do produto | README + `docs/product/how-it-works.md` | Coberto |
| Projects, workspaces e Direct | Getting Started + Product | Coberto |
| Agents e modelo de identidade | Product + arquitetura | Coberto |
| Providers, profiles, quota/capacity | Product + referência | Coberto |
| Composer / PromptArtifact | Product | Coberto |
| WorkPlan / Flow / GO | Product + arquitetura | Coberto |
| MissionRunner / lifecycle / recovery | Product + arquitetura | Coberto |
| Coordenação dinâmica entre Agents | Product + spec interna | Coberto |
| ContextCapsule / WorkReceipt / evidence | Product + arquitetura | Coberto |
| AgentMatcher / ResourceScheduler | Arquitetura + modelo mental | Coberto |
| Maestro | Product + arquitetura | Coberto como opcional |
| Web / API local | Docs de API + arquitetura | Coberto; API interna |
| CLI / automação | Referência CLI | Coberto |
| Desktop / plataformas | Operação + matriz | Coberto com ressalva de certificação |
| Segurança / guardrails | Security + arquitetura | Coberto |
| Instalação / packages / update | Getting Started + operação | Coberto |
| Troubleshooting | Operação | Coberto |
| Release truth | `DEV/validation/current/RELEASE_STATUS.md` | Autoridade separada |
| Diagramas conceituais | `docs/assets/diagrams/` | Coberto |
| Screenshots reais | `docs/assets/screenshots/` + manifesto | Coberto |

## Documentos públicos canônicos

Entradas principais:

- `README.md`;
- `docs/README.md`;
- `docs/product/how-it-works.md`;
- `ARCHITECTURE.md`;
- `docs/reference/feature-matrix.md`;
- `docs/operations/platform-support.md`.

Páginas especializadas devem aprofundar esses contratos, não redefini-los.

## Separação público x interno

`DEV/` permanece ledger de engenharia, specs, worklogs e evidência histórica.
Relatórios antigos podem conter termos como “final”, “current” ou números de
beta válidos apenas para a revisão em que foram produzidos.

Para o estado corrente:

- produto/uso → `README.md` e `docs/`;
- arquitetura → `ARCHITECTURE.md` e `docs/architecture/`;
- release → `DEV/validation/current/RELEASE_STATUS.md`;
- fechamento v1 → `DEV/SPECS/NEXUS_1_0_CLOSURE.md`.

## Mídia

Dois grupos têm semânticas diferentes:

1. `assets/screenshots/`: screenshots automatizados de fixture sintética;
2. `assets/diagrams/`: diagramas conceituais SVG.

Nenhum deles substitui evidência nativa de plataforma/provider.

## Lacunas honestas

1. Não existe compromisso de API/SDK público estável; a API local é interna.
2. A release v1 estável depende de certificação same-SHA e das plataformas
   declaradas.
3. Providers autenticados e quota live dependem do ambiente/perfil real.
4. Signing/notarização devem ser descritos conforme a evidência do release
   correspondente.
5. Documentos históricos em `DEV/` não foram reescritos artificialmente; o
   índice e as fontes canônicas deixam explícita a precedência.

## Regra de manutenção

Ao alterar uma capacidade pública:

1. atualizar a página de produto correspondente;
2. atualizar a matriz de funcionalidades;
3. atualizar o evidence ledger quando o contrato mudar;
4. atualizar diagramas se a arquitetura mudar;
5. atualizar release status somente com evidência correspondente;
6. rodar o verificador de documentação e os gates aplicáveis.
