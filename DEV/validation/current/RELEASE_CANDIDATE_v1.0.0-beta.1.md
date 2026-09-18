# Nexus v1.0.0-beta.1 — Release Candidate

> **Status**: CLOSURE_READY_PENDING_FREEZE
> **Prior GO SHA**: `2b9c95675d1727008eec86c58cd6be9e8a44a788`
> **Tag alvo**: `v1.0.0-beta.1`
> **Generated**: 2026-09-15
> **Hosted CI (prior freeze)**: [Run 34924168564](https://github.com/IAPro-Community/nexus/actions/runs/34924168564) (14/14 PASS)

O SHA `2b9c95693be7…` citado na versão anterior deste arquivo não existe no git.

## What Changed Since 0.5.0-beta.23

- **Nexus Conductor (CLI)** — `nexus conductor policy get/set`, `nexus conductor dispatch`
- **Composer → Flow** — caminho Web de planejamento; `auto_plan` HTTP aposentado
- **read_only_review** — capability para revisão sem edits do agent
- **Worker efêmero** — agent descartável para execução de goals com reconciliação automática
- **Integração opt-in** — fast-forward ff-only merge com receipt
- **Instrumentação** — AgentInvocation em event metadata
- **FS browse fix** — allowed roots check antes de stat (403 para paths fora da boundary)
- **Documentação usuário final** — `docs/product/conductor.md` e links
- **Usage Surface Redesign** — grid adaptativo de contas, modais de login/registro/explicação do scheduler, paridade completa com comandos CLI
- **Terminal Windows E2E Hardening** — helper de teste autocontido para eliminar timeouts de inicialização de PowerShell em ambientes virtualizados
- **Zero-Toolchain Packaging** — binários standalone, pacotes .deb, .rpm, instalador Windows NSIS com alias transparente `ai.exe`

## Verification Matrix

A matriz 14/14 abaixo é do **freeze anterior** `2b9c95675d17` (run 34924168564). O SHA desta campanha de fechamento ainda **não** tem CI hosted same-SHA — anotar depois do commit autorizado.

| Gate | Status | Evidence |
|---|---|---|
| Version contract | local, revalidar no freeze | `make version-check` |
| Hosted 14/14 (prior GO) | PASS no SHA `2b9c95675d17` | [Run 34924168564](https://github.com/IAPro-Community/nexus/actions/runs/34924168564) |
| Hosted same-SHA deste fechamento | PENDING | exige commit + push |
| Linux golangci no HEAD `82f5447` | FAIL (corrigido neste worktree) | misspell PT + SA1019 |
| Windows TUI lock busy | FAIL no HEAD (`>1s`); folga 3s neste worktree | `start_failure_test.go` |

## Deferred Items (Non-Blocking)

- 8h overnight soak (bounded soak harness PASS)
- Heterogeneous multi-provider Mission (requires live 3rd party API quotas)

## Installation

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/IAPro-Community/nexus/main/install.sh | bash -s -- --version=v1.0.0-beta.1

# Or from source
git clone https://github.com/IAPro-Community/nexus
cd nexus
make install-local
```
