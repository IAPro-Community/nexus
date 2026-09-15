# Nexus v1.0.0-beta.1 — Release Candidate

> **Status**: RELEASE CANDIDATE
> **SHA**: `be6c793`
> **Tag**: `v1.0.0-beta.1`
> **Generated**: 2026-09-14

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

## Verification

| Gate | Status |
|---|---|
| Version contract | PASS ✓ |
| Format check | PASS ✓ |
| ESLint | PASS ✓ |
| Stylelint | PASS ✓ |
| TypeScript | PASS ✓ |
| Go lint | PASS ✓ |
| Go tests | PASS ✓ |
| Frontend tests | PASS ✓ |
| Vet | PASS ✓ |
| Race | PASS ✓ |
| Attention E2E | PASS ✓ |
| Overnight smoke | PASS ✓ |
| Security | PASS ✓ |
| Build | PASS ✓ |
| Binary install | PASS ✓ |

## Known UNVERIFIED

- 8h soak (not a merge requirement)
- Heterogeneous multi-provider Mission (requires live providers)
- Native Windows/macOS execution (CI evidence pending)

## Installation

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/IAPro-Community/nexus/main/scripts/install.sh | bash

# Or from source
git clone https://github.com/IAPro-Community/nexus
cd nexus
git checkout v1.0.0-beta.1
make install-local
```

## Git Tag

```bash
git fetch --tags
git checkout v1.0.0-beta.1
```
