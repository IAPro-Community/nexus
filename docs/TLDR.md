# Nexus v1 — TL;DR

**Nexus** é uma workstation local e um control plane para coding agents: une projetos, agentes persistentes, CLIs/providers, terminais supervisionados, quota, roteamento de recursos e execução com evidências.

> **Estado:** branch `release/v1.0.0`, source `1.0.0-beta.10` observado em 18/10/2026. **Não é uma release 1.0 estável publicada ou homologada.** Ver [status](release-status.md) e [suporte por plataforma](operations/platform-support.md).

## Em 30 segundos

- **Direct:** projeto → AI Session → provider → terminal; não requer Maestro.
- **Composer:** transforma intenção em WorkPlan; **Flow:** mostra e permite revisar o plano.
- **GO:** autoriza uma revisão concreta; **Mission:** coordena execução, worktrees, validações e evidências.
- **Agents** são identidades persistentes, não sinônimos de provider ou modelo.
- **Quota/HUD:** visibilidade operacional sem inventar disponibilidade quando o estado é `UNKNOWN`.
- **Maestro:** integração opcional para método, skills e gates.

## Experimentação local a partir do source

Requisitos do fluxo Unix documentado: Go 1.25+, Bun 1.3.9+, Git e Make.

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun install --cwd web --frozen-lockfile
make build
./nexus doctor
./nexus web
```

O repositório de source está público; usar o source exige respeitar a licença PolyForm Strict. A publicação do código **não** prova que há instalador assinado ou v1 estável. Após abrir a UI: **Projects → + New → AI Session → provider → terminal**. A disponibilidade/autenticação dos providers depende da instalação local.

**Não use links presumidos de installers.** Veja a [instalação](getting-started/installation.md) para saber quais binários foram efetivamente verificados.

## Comandos úteis

```bash
nexus version
nexus doctor --json
nexus providers status --json
nexus status
nexus ps --json
```

## Licença

**Vigente: source-available, PolyForm Strict 1.0.0; não é open source OSI.** A autorização geral para modificar, redistribuir ou utilizar comercialmente o código **ainda não está concedida**. A decisão estratégica é migrar para **Apache-2.0 após clearance jurídico e de segurança**, sem tornar Composer, Flow ou Agents premium. Consulte a [licença vigente](../LICENSE), o [guia legal](legal/LICENSING.md), o [plano de abertura](legal/OPEN_SOURCE_TRANSITION.md) e as [orientações de contribuição](../CONTRIBUTING.md).

## Próximo passo

[Quick Start](quick-start.md) · [Documentação completa](README.md) · [Arquitetura](../ARCHITECTURE.md) · [CLI](cli/README.md) · [FAQ](faq.md) · [Suporte](../SUPPORT.md)
