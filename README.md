<p align="center"><img src="nexus-logo.png" alt="IAPro Nexus" width="340"></p>

<h1 align="center">IAPro Nexus</h1>

<p align="center"><strong>Uma workstation local para operar coding agents.</strong></p>

<p align="center">Projetos, agentes persistentes, terminais reais, providers, worktrees e automação em uma experiência integrada de Web, Desktop e CLI.</p>

<p align="center"><a href="README.en.md">English</a> · <a href="README.es.md">Español</a> · <a href="docs/README.md">Documentação</a></p>

![IAPro Nexus workspace](docs/assets/screenshots/workspace-overview.png)

## TL;DR

IAPro Nexus é uma workstation local para trabalhar com coding agents. Ele organiza projetos, agentes persistentes, sessões, terminais reais, providers, worktrees e uso/quota em um único Core. O caminho mais simples é: abrir um projeto, criar uma AI Session, escolher um provider e trabalhar no terminal. Composer, Flow, Mission e Maestro são camadas progressivas; não são pré-requisitos para o trabalho direto.

## Por que Nexus?

- **Trabalho direto primeiro:** projeto → AI Session → provider → terminal.
- **Agentes persistentes:** a identidade do agente permanece mesmo quando uma geração de runtime termina.
- **Operação local:** processos, filesystem, sessões e workspaces ficam sob o controle do Nexus.
- **Automação gradual:** use Composer, Flow e Mission quando planejamento, dependências, paralelismo ou verificação justificarem a complexidade.
- **Honestidade operacional:** uso, continuidade, disponibilidade de provider e suporte de plataforma só são apresentados como verificados quando há evidência.

## Comece em poucos minutos

Requisitos: Go 1.25+ e Bun 1.3.9+ para compilar do código-fonte.

```bash
git clone https://github.com/kivervinicius/ai-cli.git
cd ai-cli
bun --cwd web install --frozen-lockfile
make build
./nexus doctor
./nexus web
```

Abra a URL de bootstrap exibida por `nexus web`. Ela usa loopback por padrão e autentica a sessão no navegador.

1. Abra ou adicione um projeto.
2. Escolha **AI Session** em **+ New**.
3. Escolha um provider disponível localmente.
4. Abra o terminal e trabalhe normalmente.

Não é necessário abrir Composer, Flow ou Mission para esse caminho.

## As três superfícies

| Superfície | Melhor uso |
| --- | --- |
| **Nexus Web** | Controle, observação e administração no navegador local. |
| **Nexus Desktop** | Shell nativo Wails quando o build e a plataforma tiverem evidência correspondente. |
| **Nexus CLI** | Automação, scripting, diagnóstico, execução headless e operação no terminal. |

Todas apontam para o Core compartilhado quando a capacidade está disponível. Consulte o [status por plataforma](docs/operations/platform-support.md) antes de tratar uma capacidade nativa como suportada.

## Capacidades principais

| Capacidade | Estado documental |
| --- | --- |
| Projects, sessões e agentes | Implementado no Core e na Web; consulte [Agentes](docs/product/agents.md). |
| Terminal local | Implementado com xterm na Web e backends de terminal do sistema; evidência varia por plataforma. |
| Providers e usage/quota | Implementado com estados honestos como `LIVE`, `CACHED`, `ESTIMATED` e `UNKNOWN`. |
| Composer | Superfície de refinamento e compilação de prompt; veja [Composer](docs/product/composer.md). |
| Flow | Superfície de plano e execução; veja [Flow](docs/product/flow.md). |
| Mission/Autopilot | Capacidade avançada em evolução; consulte a evidência antes de tratar como certificada. |
| Maestro | Integração opcional de método, skills, risco e gates; veja [Maestro](docs/product/maestro.md). |

## Veja o produto

- [Tour visual](docs/product/visual-tour.md)
- [Caminho direto](docs/product/direct.md)
- [Agentes e terminais](docs/product/agents.md)
- [Composer, Flow e Mission](docs/product/composer.md)

As imagens são capturas automatizadas do produto real e possuem origem registrada no [manifesto visual](docs/assets/screenshots/manifest.json). Uma imagem não substitui a matriz de suporte nativo.

## Documentação

- [Começar](docs/getting-started/installation.md)
- [Mapa por intenção](docs/README.md)
- [Modelo mental](docs/product/mental-model.md)
- [Providers e quota](docs/product/providers-and-usage.md)
- [Desktop](docs/desktop/overview.md)
- [Arquitetura](docs/architecture/overview.md)
- [Troubleshooting](docs/operations/troubleshooting.md)
- [Contribuição](CONTRIBUTING.md)

## Maestro, sem dependência obrigatória

O Nexus é o produto operacional: projetos, agents, sessions, runtimes, providers, workspaces e UX. O [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro) contribui com método, skills, risco, processo, revisão e políticas de verificação. O caminho Direct continua funcionando sem Maestro; quando a integração estiver indisponível, o estado deve ser explicitamente degradado, nunca inventado.

## Segurança e comunidade

Leia [SECURITY.md](SECURITY.md), [SUPPORT.md](SUPPORT.md), [GOVERNANCE.md](GOVERNANCE.md) e [ROADMAP.md](ROADMAP.md). O CHANGELOG registra mudanças entregues; o ROADMAP registra apenas trabalho futuro.

## Licença

Distribuído sob a [licença MIT](LICENSE).
