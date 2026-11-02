# Capturas e evidências visuais — Nexus 1.0

O [manifesto](../assets/screenshots/manifest.json) contém screenshots gerados com fixture sintética. A recaptura associada ao PR #181 foi executada sobre `a3fbd4478bbcdf143d62fdad03e534c821270850`, com `bun run docs:capture` e `node scripts/docs-verify.mjs` em CI; a evidência nativa Desktop (`VIS-011`) continua classificada como **HISTORICAL**, não PASS da versão atual. Imagens Web sintéticas não certificam homologação Desktop ou release 1.0.

## Como recapturar

1. Congelar `release/v1.0.0` em um SHA imutável e garantir worktree limpo.
2. Instalar dependências Web, Playwright e, quando aplicável, Wails e dependências nativas.
3. Usar somente fixture sintética, nunca contas, tokens ou arquivos reais.
4. Executar os comandos abaixo na raiz do checkout:

```bash
git rev-parse HEAD
cd web
bun install --frozen-lockfile
bun run docs:capture
cd ..
node scripts/docs-verify.mjs
```

5. Confirmar `VIS-001` a `VIS-011`, caminhos, classificação `SYNTHETIC`, SHA, viewports e marcadores do terminal. Não substituir `source_sha` sem gerar novas capturas.
6. Validar separadamente Desktop/HUD nativo, teclado, scroll, foco, resize, instalação/desinstalação e acessibilidade em Linux, Windows e macOS, no SHA exato.

Uma captura Web não certifica um shell Desktop. Falha de captura = `FAIL`; SHA desatualizado = `STALE`; runner indisponível = `UNVERIFIED`. Não promover baseline de screenshots automaticamente para obter PASS.

Veja [suporte](platform-support.md), [release](release.md) e [tour visual](../product/visual-tour.md).
