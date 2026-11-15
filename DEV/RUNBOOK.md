# Runbook — Recuperação de infra CI/Docs (2026-11-14)

Contexto: o restart do servidor de 14/11 removeu `python3`, `/usr/bin/google-chrome` e o **runner self-hosted `nexus-ci`**. Dois workflows ficaram vermelhos por causas 100% administrativas (não são regressões de código). **Status 2026-11-15: AMBOS RESOLVIDOS** (ver seções para o que sobrou opcional).

## 1. CI principal em "queued" eterno — runner self-hosted sumiu ✅ RESOLVIDO 2026-11-15

**Resolvido pelo orquestrador (token do credential helper, admin:true):** runner `Dev-Web-Kiver` registrado em `~/actions-runner` (v2.338.2, labels `[self-hosted, Linux, X64, nexus-ci]`), rodando como **processo do usuário (nohup)**. Correções de ambiente aplicadas: `BUN_INSTALL` privado no `.env`, cache `_tool` limpo, e `~/.bun/bin/bun` alinhado ao 1.3.9 pedido pelo CI (rename sobre binário busy — TUI do maestro intocado).

**Opcional (persistência após reboot) — única ação com sudo:**

```bash
cd ~/actions-runner && sudo ./svc.sh install && sudo ./svc.sh start
```

- O label `nexus-ci` é obrigatório (matches `[self-hosted, Linux, X64, nexus-ci]`).
- `setup-go`/`setup-bun` instalam toolchains no runner automaticamente; não pré-instalar nada.
- Segurança (ver `docs/community/ADMIN_SETUP.md`): runner **nunca** exposto a PRs de fork; `ci.yml` é push/dispatch only.

## 2. "Publish Nexus documentation" vermelho — environment bloqueia a branch ✅ RESOLVIDO 2026-11-15

**Resolvido pelo orquestrador:** `release/v1.0.0` adicionada às deployment branch policies do environment `github-pages` via API (`POST /environments/github-pages/deployment-branch-policies`). Run de validação por dispatch → **success**. A crônica de 151 runs encerrou-se; nada mais a fazer.

## 3. "Refresh synthetic documentation captures" vermelho — comportamento correto

**Nada a fazer.** As duas falhas recentes são a guarda anti-drift do step 12 ("Source branch advanced during capture: refusing stale write"): a captura completa com sucesso (steps 1–11 pass), mas o push é recusado porque a branch avançou durante a janela de captura no Windows. O workflow se auto-cura no próximo push que toque `web/src/**`. Regra operacional: **evitar push ao Nexus durante a janela de captura** (~5–10 min após um push de `web/src`).

## Nota de ambiente local (pós-restart)

- `python3`, `/usr/bin/google-chrome`, `gh` não existem mais. Usar `node` para JSON, `~/.cache/ms-playwright/chromium-1243` (instalado) para Playwright/Chromium, e a API REST do GitHub via `node --input-type=module -e "fetch(...)"` para telemetria de CI.
- Firefox 155 do playwright instalado (`~/.cache/ms-playwright/firefox-1543`).
