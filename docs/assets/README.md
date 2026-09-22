# Assets visuais da documentação

Os assets são classificados para evitar que uma ilustração conceitual seja
confundida com evidência de runtime.

## Diagramas canônicos (`diagrams/`)

SVGs editáveis, sem dados de usuário, usados para explicar o funcionamento da
v1.

| Arquivo | Uso principal |
| --- | --- |
| [nexus-v1-overview.svg](diagrams/nexus-v1-overview.svg) | README e visão geral |
| [nexus-v1-flow.svg](diagrams/nexus-v1-flow.svg) | fluxo ponta a ponta |
| [nexus-v1-components.svg](diagrams/nexus-v1-components.svg) | arquitetura lógica |
| [nexus-v1-agent-coordination.svg](diagrams/nexus-v1-agent-coordination.svg) | coordenação dinâmica |
| [nexus-v1-mission-lifecycle.svg](diagrams/nexus-v1-mission-lifecycle.svg) | state machine da Mission |
| [nexus-v1-guardrails.svg](diagrams/nexus-v1-guardrails.svg) | segurança/guardrails |

Classificação: **`SYNTHETIC_DIAGRAM`**. Explica contratos, não certifica
plataforma, provider ou release.

## Screenshots reais (`screenshots/`)

Capturas automatizadas da aplicação em fixture sintética isolada.

O [manifesto](screenshots/manifest.json) registra cenário e origem. Essas
capturas demonstram uma superfície específica e também não substituem a matriz
de plataforma.

Tour: [product/visual-tour.md](../product/visual-tour.md).

## Ajuda / instalação (`help/`)

| Arquivo | Documento |
| --- | --- |
| [help-install-overview.png](help/help-install-overview.png) | [Getting Started](../getting-started/README.md) |
| [help-install-native.png](help/help-install-native.png) | [Instalação](../getting-started/installation.md) |
| [help-install-packages.png](help/help-install-packages.png) | [Pacotes](../getting-started/packages.md) |
| [help-install-docker.png](help/help-install-docker.png) | [Docker](../getting-started/docker-compat.md) |

## Divulgação (`promo/`)

Assets de marca/marketing. Não usar como evidência técnica.

## Regra editorial

Ao adicionar uma imagem:

1. use alt text específico;
2. classifique como screenshot, diagrama ou marketing;
3. não inclua secrets/dados reais;
4. prefira SVG para diagramas;
5. atualize este catálogo quando o asset for canônico.
