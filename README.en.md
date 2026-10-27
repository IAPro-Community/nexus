# IAPro Nexus

**A local workstation for operating coding agents.**

Projects, persistent agents, supervised terminal sessions, providers, worktrees and automation in one Web, Desktop and CLI experience.

<p align="center"><a href="README.md">Português</a> · <strong>English</strong> · <a href="README.es.md">Español</a></p>

![IAPro Nexus workspace](docs/assets/screenshots/workspace-overview.png)

<p align="center"><img src="docs/assets/promo/promo-nexus-hero.png" alt="IAPro Nexus — local workstation for coding agents" width="720"></p>

## TL;DR

IAPro Nexus is a local workstation for working with coding agents. It organizes projects, persistent agents, sessions, supervised terminals, providers, worktrees and usage/quota around one Core. The simplest path is: open a project, create an AI Session, choose a provider and work in the terminal. Composer, Flow, Mission and Maestro are progressive layers, not prerequisites for direct work.

## Start in minutes

The source candidate on `release/v1.0.0` declares **`1.0.0-beta.11`** as checked on 2026-10-27; this is **not** a certified or published stable 1.0. The source repository is public; history and third-party licensing review remain open release gates. Platform support requires exact-SHA validation. Do not derive binary download URLs from historical tags or source versions.

### Developers (build from source)

Requirements: Go 1.25+ and Bun 1.3.9+.

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun install --cwd web --frozen-lockfile
make build
./nexus doctor
./nexus web
```

For development from source, platform-specific build paths exist, but source
integration alone is not a public support or certification claim.

Then follow **+ New → AI Session → provider → terminal**. You do not need Composer, Flow or Mission for direct work.

## Web, Desktop and CLI

| Surface | Best for |
| --- | --- |
| **Nexus Web** | Local browser control, observation and administration. |
| **Nexus Desktop** | Native Wails shell when the target platform has matching build/runtime evidence. |
| **Nexus CLI** | Scripting, diagnostics, headless operation and terminal workflows. |

See the [platform support matrix](docs/operations/platform-support.md) before treating a native capability as supported.

## Explore

- [Documentation map](docs/README.md)
- [Getting started index](docs/getting-started/README.md)
- [Signed native install](docs/getting-started/installation.md)
- [Packages and native artifacts](docs/getting-started/packages.md)
- [Docker compatibility](docs/getting-started/docker-compat.md)
- [Visual tour](docs/product/visual-tour.md)
- [Direct workflow](docs/product/direct.md)
- [Composer](docs/product/composer.md)
- [Flow](docs/product/flow.md)
- [Desktop](docs/desktop/overview.md)
- [Architecture](docs/architecture/overview.md)
- [Troubleshooting](docs/operations/troubleshooting.md)

The [Maestro integration](docs/product/maestro.md) is optional for the Direct path. Claims about support, usage and continuity are evidence-bound; planned or experimental capabilities are labeled as such.

## Community

[TL;DR](docs/TLDR.md) · [Docs](docs/README.md) · [Security](SECURITY.md) · [Support](SUPPORT.md) · [Governance](GOVERNANCE.md) · [Roadmap](ROADMAP.md) · [Changelog](CHANGELOG.md) · [PolyForm Strict License](LICENSE) · [Licensing guide](docs/legal/LICENSING.md)

**Current license:** [PolyForm Strict 1.0.0](LICENSE), source-available, **not OSI open source**. It does not grant general rights to modify or redistribute the project. **Target license:** Apache-2.0, **not yet effective**. Relicensing requires verified ownership, third-party and asset rights, and a private security/history review. Public source visibility is not permission to publish modified forks.

**Community direction:** a fully functional, local Nexus Community with Composer, Flow, agents and Missions, funded voluntarily and eventually through optional managed services and enterprise support—not paywalls on core functionality. External AI providers may charge their own fees.

[Open-source transition plan](docs/legal/OPEN_SOURCE_TRANSITION.md) · [Current licensing terms](docs/legal/LICENSING.md) · [Contributing](CONTRIBUTING.md) · [Forks and upstream](docs/community/FORKS_AND_UPSTREAM.md) · [Trademark/branding](docs/legal/TRADEMARK_POLICY.md) · [Funding](docs/community/FUNDING.md) · [Third-party notices](docs/legal/THIRD_PARTY_NOTICES.md).

Stable release certification remains **NO-GO** pending exact-SHA validation.
