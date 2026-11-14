# Runbook — Recuperação de infra CI/Docs (2026-11-14)

Contexto: o restart do servidor de 14/11 removeu `python3`, `/usr/bin/google-chrome` e o **runner self-hosted `nexus-ci`**. Dois workflows ficaram vermelhos por causas 100% administrativas (não são regressões de código). Diagnóstico completo em `DEV/WORKLOG.md` (seções de 14/11).

## 1. CI principal em "queued" eterno — runner self-hosted sumiu

**Sintoma:** o run de `ci.yml` fica `queued` para sempre; job "Sanitize self-hosted workspace" sem runner. Todos os jobs de `ci.yml` exigem `[self-hosted, Linux, X64, nexus-ci]`.

**Ação do maestro (admin):**

```bash
# Na máquina que hospedará o runner (anterior não sobreviveu ao restart):
mkdir actions-runner && cd actions-runner
# pegar o link atual em: GitHub → repo → Settings → Actions → Runners → New self-hosted runner → Linux X64
curl -o actions-runner-linux-x64.tar.gz -L <URL_DO_PAINEL>
tar xzf actions-runner-linux-x64.tar.gz
./config.sh --url https://github.com/IAPro-Community/nexus \
  --token <TOKEN_DO_PAINEL> \
  --labels nexus-ci
./svc.sh install && ./svc.sh start   # serviço persistente
```

- O label `nexus-ci` é obrigatório (matches `[self-hosted, Linux, X64, nexus-ci]`).
- `setup-go`/`setup-bun` instalam toolchains no runner automaticamente; não pré-instalar nada.
- Segurança (ver `docs/community/ADMIN_SETUP.md`): runner **nunca** exposto a PRs de fork; `ci.yml` é push/dispatch only.

## 2. "Publish Nexus documentation" vermelho — environment bloqueia a branch

**Sintoma:** job falha em 2–3s, antes do primeiro step. Histórico: 151 runs, 1 único sucesso — na branch `docs/nexus-1.0-portal-20261022`. Causa: deployment branch policy do environment `github-pages` não inclui `release/v1.0.0`.

**Ação do maestro (admin):**

1. GitHub → **Settings → Environments → github-pages → Deployment branches and tags** → adicionar `release/v1.0.0` (ou "No restriction").
2. **Actions → Publish Nexus documentation → run vermelho mais recente → Re-run failed jobs.**

Pré-requisitos já satisfeitos por mim: `npm audit --audit-level=high` verde (2 highs corrigidos em `555b16e8`), `check:docs` (100 páginas) e `build` validados localmente.

## 3. "Refresh synthetic documentation captures" vermelho — comportamento correto

**Nada a fazer.** As duas falhas recentes são a guarda anti-drift do step 12 ("Source branch advanced during capture: refusing stale write"): a captura completa com sucesso (steps 1–11 pass), mas o push é recusado porque a branch avançou durante a janela de captura no Windows. O workflow se auto-cura no próximo push que toque `web/src/**`. Regra operacional: **evitar push ao Nexus durante a janela de captura** (~5–10 min após um push de `web/src`).

## Nota de ambiente local (pós-restart)

- `python3`, `/usr/bin/google-chrome`, `gh` não existem mais. Usar `node` para JSON, `~/.cache/ms-playwright/chromium-1243` (instalado) para Playwright/Chromium, e a API REST do GitHub via `node --input-type=module -e "fetch(...)"` para telemetria de CI.
- Firefox 155 do playwright instalado (`~/.cache/ms-playwright/firefox-1543`).
