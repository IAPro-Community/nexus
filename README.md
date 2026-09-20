<p align="center"><img src="nexus-logo.png" alt="IAPro Nexus" width="340"></p>

> **Beta fechado v1.0.0-beta.2 (release candidate)** — build local verificada;
> a tag/release pública correspondente ainda não foi confirmada neste branch.
> Verifique se a tag `v1.0.0-beta.2` e o GitHub Release existem antes de
> baixar binários (`git ls-remote --tags origin v1.0.0-beta.2`).
> Consulte [Beta Terms](docs/legal/BETA_TERMS.md) e
> [Privacy](docs/legal/PRIVACY.md).

<p align="center"><img src="docs/assets/promo/promo-nexus-hero.png" alt="IAPro Nexus — workstation local para coding agents" width="720"></p>

<h1 align="center">IAPro Nexus</h1>

<p align="center"><strong>Uma workstation local para operar coding agents.</strong></p>

<p align="center">Projetos, agentes persistentes, sessões de terminal supervisionadas, providers, worktrees e automação em uma experiência integrada de Web, Desktop e CLI.</p>

<p align="center"><a href="README.en.md">English</a> · <a href="README.es.md">Español</a> · <a href="docs/README.md">Documentação</a></p>

![IAPro Nexus workspace](docs/assets/screenshots/workspace-overview.png)

## TL;DR

IAPro Nexus é uma workstation local para trabalhar com coding agents. Ele organiza projetos, agentes persistentes, sessões, terminais supervisionados, providers, worktrees e uso/quota em um único Core. O caminho mais simples é: abrir um projeto, criar uma AI Session, escolher um provider e trabalhar no terminal. Composer, Flow, Mission e Maestro são camadas progressivas; não são pré-requisitos para o trabalho direto.

Com ele você pode:

- iniciar uma sessão de IA no projeto sem montar uma pilha de orquestração;
- manter a identidade do agente enquanto o runtime e o provider mudam;
- observar disponibilidade e quota sem transformar ausência de dados em sucesso;
- usar planejamento, dependências e verificação somente quando o objetivo pedir.

→ [Quick Start de 5–10 minutos](docs/quick-start.md)

## O que este produto melhora?

| Antes / problema | Com o Nexus |
| --- | --- |
| Cada CLI de coding agent tem contexto, perfil e terminal próprios. | Projetos, perfis, sessões e workspaces ficam visíveis em uma workstation local única. |
| Trocar de provider pode perder o contexto operacional. | O Agent é uma identidade persistente; provider/modelo são recursos de execução roteados separadamente. |
| Acompanhar quota ou falha exige consultar ferramentas diferentes. | Estados como `LIVE`, `CACHED`, `ESTIMATED` e `UNKNOWN` deixam explícita a qualidade da evidência. |
| Todo trabalho precisa de um fluxo pesado de planejamento. | O caminho Direct abre uma sessão imediatamente; Composer, Flow e Mission são opt-in progressivos. |

## Por que Nexus?

- **Trabalho direto primeiro:** projeto → AI Session → provider → terminal.
- **Agentes persistentes:** a identidade do agente permanece mesmo quando uma geração de runtime termina.
- **Operação local:** processos, filesystem, sessões e workspaces ficam sob o controle do Nexus.
- **Automação gradual:** use Composer, Flow e Mission quando planejamento, dependências, paralelismo ou verificação justificarem a complexidade.
- **Honestidade operacional:** uso, continuidade, disponibilidade de provider e suporte de plataforma só são apresentados como verificados quando há evidência.

## Comece em poucos minutos

Caminho recomendado (**zero-toolchain**): baixe um binário de release. Não precisa de Go, Bun nem Node.

**Linux / macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/IAPro-Community/nexus/main/install.sh | bash -s -- --version=v1.0.0-beta.2
nexus doctor
nexus web
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/IAPro-Community/nexus/main/install.ps1 -OutFile install.ps1
.\install.ps1 -Version v1.0.0-beta.2
nexus doctor
nexus web
```

`--version=v1.0.0-beta.2` baixa o GitHub Release pinado e valida `checksums.txt` (integridade) — exige que tag e release existam; se ainda não publicados, compile do fonte abaixo. `--version=latest` resolve a tag mais nova depois que a release existir. O repositório precisa ser público (ou use `GH_TOKEN`) para o download anônimo.

Pré-requisitos do instalador de release: `curl` ou `wget`, `tar` e `sha256sum`/`shasum` (Unix) ou PowerShell (Windows). Desktop nativo pode exigir WebView2 (Windows) ou WebKitGTK (Linux).

### Desenvolvedores (compilar do fonte)

Requisitos: Go 1.25+ e Bun 1.3.9+.

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun --cwd web install --frozen-lockfile
make build
./nexus doctor
./nexus web
```

Ou: `./install.sh --build-from-source [--yes]` / `.\install.ps1 -BuildFromSource`.

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

Guias de distribuição **separados**:

- [Quick Start](docs/quick-start.md)
- [Funcionalidades por objetivo](docs/features/index.md)
- [Documentação para usuários](docs/user/README.md)
- [Referência da CLI](docs/cli/README.md)
- [Visuais (ajuda/divulgação)](docs/getting-started/visuals.md)
- [Getting Started (índice)](docs/getting-started/README.md)
- [Instalação nativa](docs/getting-started/installation.md)
- [Pacotes DEB/RPM/NSIS](docs/getting-started/packages.md)
- [Docker (compatibilidade)](docs/getting-started/docker-compat.md)
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

Software proprietário — distribuído como binário sob a [licença beta](LICENSE).
Consulte também [Beta Terms](docs/legal/BETA_TERMS.md),
[Privacy](docs/legal/PRIVACY.md) e
[Third-Party Notices](docs/legal/THIRD_PARTY_NOTICES.md).
