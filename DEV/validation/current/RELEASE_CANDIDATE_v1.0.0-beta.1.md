# Nexus v1.0.0-beta.1 — Release Candidate

> **Status**: RELEASE CANDIDATE (FINAL = GO)
> **SHA**: `2b9c95693be7bf73a463c6395ec1ea9f131a4c9c`
> **Tag**: `v1.0.0-beta.1`
> **Generated**: 2026-09-15
> **Hosted CI**: [Run 34924168564](https://github.com/IAPro-Community/nexus/actions/runs/34924168564) (14/14 PASS)

## What Changed Since 0.5.0-beta.23

- **Nexus Conductor** — orquestração nativa Goals → Missions com dispatch único CLI/Web
- **read_only_review** — capability para revisão sem edits do agent
- **Worker efêmero** — agent descartável para execução de goals com reconciliação automática
- **Integração opt-in** — fast-forward ff-only merge com receipt
- **Instrumentação** — AgentInvocation em event metadata
- **API Conductor** — `GET/PUT /api/v1/conductor/policy`, `POST /api/v1/conductor/dispatch`
- **CLI Conductor** — `nexus conductor policy get/set`, `nexus conductor dispatch`
- **FS browse fix** — allowed roots check antes de stat (403 para paths fora da boundary)
- **Documentação usuário final** — `docs/product/conductor.md` e links
- **Usage Surface Redesign** — grid adaptativo de contas, modais de login/registro/explicação do scheduler, paridade completa com comandos CLI
- **Terminal Windows E2E Hardening** — helper de teste autocontido para eliminar timeouts de inicialização de PowerShell em ambientes virtualizados
- **Zero-Toolchain Packaging** — binários standalone, pacotes .deb, .rpm, instalador Windows NSIS com alias transparente `ai.exe`

## Verification Matrix

| Gate | Status | Evidence |
|---|---|---|
| Version contract | PASS ✓ | `make version-check` |
| Format check | PASS ✓ | `make format-check` |
| ESLint | PASS ✓ | `npm --prefix web run lint` |
| Stylelint | PASS ✓ | `npm --prefix web run lint:styles` |
| TypeScript | PASS ✓ | `npm --prefix web run typecheck` |
| Go lint | PASS ✓ | `make lint-go` (0 issues) |
| Go tests | PASS ✓ | `go test ./...` (100% pass) |
| Frontend tests | PASS ✓ | `vitest run` (73 suites, 388 tests pass) |
| Vet & Race | PASS ✓ | `go vet` + `go test -race` |
| Attention E2E | PASS ✓ | `make attention-e2e` |
| Overnight smoke | PASS ✓ | `make overnight-smoke` |
| Security | PASS ✓ | `govulncheck` (0 vulnerabilities) |
| Documentation & Visual Manifest | PASS ✓ | `make docs-verify` (159 docs, 11 captures) |
| Hosted Windows E2E | PASS ✓ | CI Run 34924168564 |
| Hosted macOS E2E | PASS ✓ | CI Run 34924168564 |
| Hosted Linux E2E | PASS ✓ | CI Run 34924168564 |
| Hosted Desktop Builds (Win/Mac/Linux) | PASS ✓ | CI Run 34924168564 |
| Hosted Package Smoke (DEB/RPM/NSIS) | PASS ✓ | CI Run 34924168564 |

## Deferred Items (Non-Blocking)

- 8h overnight soak (bounded soak harness PASS)
- Heterogeneous multi-provider Mission (requires live 3rd party API quotas)

## Installation

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/IAPro-Community/nexus/main/scripts/install.sh | bash

# Or from source
git clone https://github.com/IAPro-Community/nexus
cd nexus
git checkout feat/nexus-maximum-delivery
make install-local
```
