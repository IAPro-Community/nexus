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

## 4. Toolchain do runner `nexus-ci` no container (idempotente)

O runner `nexus-ci` roda dentro do container `vpn-dev-workspace-terminal-1`. O container é efêmero: cada recriação apaga a toolchain nativa e quebra `Linux E2E`, `Security` e a captura visual (Chromium sem libs GTK/WebKit, `pkg-config` ausente).

**Recuperar:**

```bash
docker exec -u root vpn-dev-workspace-terminal-1 bash -c \
  'apt-get -o APT::Sandbox::User=root -o Dir::Cache=/tmp/aptcache update && \
   apt-get -o APT::Sandbox::User=root -o Dir::Cache=/tmp/aptcache install -y --no-install-recommends \
   pkg-config python3 build-essential jq zip unzip xdotool xvfb dbus-x11 imagemagick \
   libglib2.0-dev libgtk-3-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev libcairo2-dev \
   libpango1.0-dev libx11-dev libxtst-dev libxkbcommon-dev libgbm-dev libasound2-dev \
   libatk1.0-dev libatspi2.0-dev libnss3 libnspr4 libatk-bridge2.0-0 libdrm2 \
   libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libgbm1 libasound2t64 libatspi2.0-0 \
   libxshmfence1 libcups2t64 fonts-liberation fonts-dejavu-core'
```

Ou, a partir de um clone do repositório:

```bash
docker cp scripts/ci/provision-linux-runner-container.sh vpn-dev-workspace-terminal-1:/tmp/provision.sh
docker exec -u root vpn-dev-workspace-terminal-1 bash /tmp/provision.sh
```

**Por que `-o APT::Sandbox::User=root -o Dir::Cache=/tmp/aptcache`:** o `apt` roda com usuário sandbox diferente de root e o cache padrão não é gravável; sem essas opções a instalação falha com `seteuid 42 failed` e `archives/partial: Permission denied`.

**Verificação:**

```bash
docker exec -u root vpn-dev-workspace-terminal-1 bash -c \
  'pkg-config --modversion gtk+-3.0 webkit2gtk-4.1 libsoup-3.0 gio-unix-2.0 && ldd $(which chromium 2>/dev/null || echo /usr/bin/chromium) | grep "not found" || true'
```

Sem `not found` e com os quatro modulos `pkg-config` presentes, `Linux E2E`, `Security` e `Documentation visual capture evidence` passam.

## Nota de ambiente local (pós-restart)

- `python3`, `/usr/bin/google-chrome`, `gh` não existem mais. Usar `node` para JSON, `~/.cache/ms-playwright/chromium-1243` (instalado) para Playwright/Chromium, e a API REST do GitHub via `node --input-type=module -e "fetch(...)"` para telemetria de CI.
- Firefox 155 do playwright instalado (`~/.cache/ms-playwright/firefox-1543`).
