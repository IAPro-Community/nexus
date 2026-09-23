# IAPro Nexus

**A local workstation for operating coding agents.**

Projects, persistent agents, supervised terminal sessions, providers, worktrees and automation in one Web, Desktop and CLI experience.

<p align="center"><a href="README.md">Português</a> · <strong>English</strong> · <a href="README.es.md">Español</a></p>

![IAPro Nexus workspace](docs/assets/screenshots/workspace-overview.png)

<p align="center"><img src="docs/assets/promo/promo-nexus-hero.png" alt="IAPro Nexus — local workstation for coding agents" width="720"></p>

## TL;DR

IAPro Nexus is a local workstation for working with coding agents. It organizes projects, persistent agents, sessions, supervised terminals, providers, worktrees and usage/quota around one Core. The simplest path is: open a project, create an AI Session, choose a provider and work in the terminal. Composer, Flow, Mission and Maestro are progressive layers, not prerequisites for direct work.

## Start in minutes

The stable Nexus v1 support scope is **Linux amd64**. Windows and macOS remain in the
source tree but are deferred until native certification is completed.

Recommended path: install a signed prebuilt release. No Go, Bun, or Node toolchain is
required; Ed25519 verification currently requires Python 3 with `cryptography`.

```bash
curl -fsSL https://github.com/IAPro-Community/nexus-releases/releases/download/v1.0.0-beta.1/install.sh -o install.sh
bash install.sh --version=v1.0.0-beta.1
nexus doctor
nexus web
```

The pinned installer is downloaded from the public binaries-only distribution repository.
It refuses unsigned releases, verifies the Ed25519 manifest, checks the artifact SHA-256,
and confirms the embedded binary version before replacement. `--version=latest` means
GitHub's latest non-prerelease release; while the project is beta, use the explicit beta
tag shown above.

Release prerequisites: Linux amd64, `curl` or `wget`, `tar`, SHA-256 tooling,
Python 3 with `cryptography`, and WebKitGTK only when using the Linux Desktop.

### Developers (build from source)

Requirements: Go 1.25+ and Bun 1.3.9+.

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun --cwd web install --frozen-lockfile
make build
./nexus doctor
./nexus web
```

For development from source, platform-specific build paths may still exist, but they are
not part of the supported v1 release contract.

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

[Security](SECURITY.md) · [Support](SUPPORT.md) · [Governance](GOVERNANCE.md) · [Roadmap](ROADMAP.md) · [Changelog](CHANGELOG.md) · [Beta License](LICENSE)
