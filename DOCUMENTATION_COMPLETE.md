# Documentação completa — IAPro Nexus

## Resumo

Esta entrega consolida a documentação existente em uma jornada de produto:
entender o valor, ver o produto, iniciar uma sessão Direct, descobrir recursos
por objetivo e aprofundar em referência, operação ou arquitetura. Afirmações
de produto foram ligadas a implementação/testes no
[evidence ledger](docs/internal/DOCUMENTATION_EVIDENCE.md).

## O que foi encontrado no produto

- workstation local com Web, Desktop e CLI;
- projetos, workspaces, agents persistentes e runtimes supervisionados;
- providers/perfis com observabilidade de quota e estados honestos;
- Direct, Composer, WorkPlan/Flow, Mission e integração Maestro opcional;
- API local autenticada para as superfícies do produto;
- update, doctor, security, tunnel e scripts de qualidade/captura.

O status por funcionalidade está no
[inventário](docs/internal/FEATURE_INVENTORY.md) e na
[matriz](docs/reference/feature-matrix.md).

## Documentação criada ou consolidada

- [Quick Start](docs/quick-start.md), [funcionalidades](docs/features/index.md)
  e entradas por [usuário](docs/user/README.md),
  [desenvolvimento](docs/development/README.md) e
  [operação](docs/operations/README.md);
- referências para [CLI](docs/cli/README.md), [API local](docs/api/README.md),
  [configuração](docs/reference/configuration.md),
  [FAQ](docs/faq.md), [glossário](docs/glossary.md) e
  [o que mudou](docs/whats-new.md);
- visão de arquitetura com Mermaid, troubleshooting baseado em operações reais,
  inventário, cobertura e ledger de evidência.

## Visuals e demonstrações

O repositório já possui screenshots automatizados em
`docs/assets/screenshots/`, manifesto sintético e a demonstração
`docs/assets/demos/direct-workflow.webm`. Veja o
[tour visual](docs/product/visual-tour.md). Não foram criados GIFs ou vídeos
adicionais: a demonstração WebM existente é reproduzível e evita duplicar mídia
pesada no Git.

## Screenshots criados ou atualizados

O capturador regenerou `workspace-overview`, `projects`, `direct-session`,
`provider`, `terminal`, `agents`, `usage-quota`, `composer`, `flow` e
`mission`, além da demo Direct. A revisão visual manual confirmou que a
captura de workspace mostra somente a fixture `Nexus Demo Workspace`, sem dados
de projeto ou credenciais reais.

## Quick Start, arquitetura, API e CLI

- O [Quick Start](docs/quick-start.md) foi executado contra diretórios de dados
  temporários: listagem de projetos, registro do repositório e listagem final.
- A [arquitetura](docs/architecture/overview.md) ganhou Mermaid e uma distinção
  explícita entre desenho e certificação de plataforma.
- A [API local](docs/api/README.md) foi classificada como interna; não promete
  um SDK público.
- A [referência da CLI](docs/cli/README.md) foi confrontada com
  `go run ./cmd/nexus --help` e encaminha flags específicas para o help da
  versão instalada.

## Troubleshooting e problemas encontrados

- Troubleshooting agora cobre bootstrap Web, provider/perfil, terminal,
  Desktop e intervenção de Mission com comandos de validação reais.
- Nesta versão de Bun, `bun --cwd web run docs:capture` abre a ajuda; a forma
  validada é `bun run --cwd web docs:capture`.
- O gate de qualidade revelou um wrapper de lock Codex sem uso e dois literais
  de contrato de instalador ausentes. O wrapper redundante foi removido e os
  instaladores voltaram a declarar a verificação fail-closed.

## Validação final

| Verificação | Resultado |
| --- | --- |
| `bun run --cwd web docs:capture` | PASS — fixture sintética, 10 superfícies + demo |
| `node scripts/docs-verify.mjs` / `make docs-verify` | PASS — 149 Markdown, 22 ativos obrigatórios |
| Quick Start isolado (projetos) | PASS |
| `go test ./internal/release -count=1` | PASS |
| `make quality` | PASS |
| `make web-verify` | PASS — 10 gates de frontend |
| `make security` | PASS — nenhuma vulnerabilidade encontrada |
| `git diff --check` | PASS |

## Como atualizar as mídias

```bash
bun run --cwd web docs:capture
node scripts/docs-verify.mjs
```

O capturador usa uma fixture isolada e deve falhar se detectar paths ou dados
locais não sintéticos. Revise o manifesto antes de aceitar novas imagens.

## Como publicar e validar

As páginas são Markdown estático e podem ser publicadas pelo mecanismo já usado
pelo repositório. Antes de publicar, execute:

```bash
node scripts/docs-verify.mjs
make web-verify
make quality
```

## Limites e débitos restantes

- A API local não é um SDK público.
- Certificação de provider autenticado, Mission em ambiente real e Desktop por
  plataforma continua condicionada à evidência operacional.
- A documentação técnica/histórica em `DEV/` permanece a memória de engenharia
  e não foi duplicada como manual público.

Consulte a [cobertura](docs/internal/DOCUMENTATION_COVERAGE.md) para as lacunas
classificadas e o [troubleshooting](docs/operations/troubleshooting.md) para
problemas comuns.
