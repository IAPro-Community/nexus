# IAPro Nexus — inventário preliminar de dependências (v1 source)

> **NÃO AUDITADO COMO NOTICE JURÍDICO FINAL.** Este inventário foi extraído de `go.mod` e `web/package.json` do source candidato. Ele não valida, por si só, licenças SPDX, textos completos, assets ou o SBOM dos binários. É obrigação da release executar a revisão jurídica/supply-chain.

## Origem de verdade

- `go.mod` e `go.sum`: dependências Go.
- `web/package.json` e `web/bun.lock`: dependências JavaScript/transitivas.
- Artefatos de release e seu SBOM: somente eles confirmam a composição **efetivamente distribuída**.
- `LICENSE` governa o código próprio do Nexus nos termos aplicáveis; dependências de terceiros preservam licenças separadas.

## Go — dependências diretas declaradas

| Módulo | Versão |
| --- | --- |
| `github.com/Microsoft/go-winio` | `v0.6.2` |
| `github.com/Xuanwo/go-locale` | `v1.1.3` |
| `github.com/charmbracelet/bubbles` | `v0.21.1` |
| `github.com/charmbracelet/bubbletea` | `v1.3.10` |
| `github.com/charmbracelet/lipgloss` | `v1.1.1-0.20250404203927-76690c660834` |
| `github.com/creack/pty` | `v1.1.24` |
| `github.com/gorilla/websocket` | `v1.5.3` |
| `github.com/nicksnyder/go-i18n/v2` | `v2.6.1` |
| `github.com/wailsapp/wails/v3` | `v3.0.0-beta.25` |
| `golang.org/x/image` | `v0.45.0` |
| `golang.org/x/sys` | `v0.47.0` |
| `golang.org/x/term` | `v0.45.0` |
| `golang.org/x/text` | `v0.41.0` |
| `modernc.org/sqlite` | `v1.57.0` |

**Wails atual:** `github.com/wailsapp/wails/v3 v3.0.0-beta.25` (não Wails v2). O conjunto de bibliotecas transitorias, bindings nativos e demais obrigações deve ser revisto a partir do lock/build do SHA final.

## Web — dependências de runtime declaradas

| Pacote | Versão declarada |
| --- | --- |
| `@assistant-ui/react` | `^0.15.20` |
| `@radix-ui/react-alert-dialog` | `^1.1.23` |
| `@radix-ui/react-dialog` | `^1.1.23` |
| `@radix-ui/react-dropdown-menu` | `^2.1.24` |
| `@radix-ui/react-tooltip` | `^1.2.16` |
| `@wailsio/runtime` | `3.0.0-beta.25` |
| `@xyflow/react` | `^12.11.6` |
| `i18next` | `^26.4.0` |
| `i18next-browser-languagedetector` | `^8.2.1` |
| `lucide-react` | `^0.475.0` |
| `react` | `^19.0.0` |
| `react-dom` | `^19.0.0` |
| `react-i18next` | `^17.0.12` |
| `react-qr-code` | `^2.2.0` |
| `react-router-dom` | `^7.18.3` |
| `sonner` | `^2.0.8` |
| `xterm` | `^5.3.0` |
| `xterm-addon-fit` | `^0.8.0` |

Bibliotecas utilizadas apenas em build/desenvolvimento também podem impor avisos nas distribuições, conforme composição dos artefatos. Não assumir que todas as dependências de desenvolvimento ficam excluídas do binário sem inspeção real.

## Gate obrigatório antes de publicar uma release

1. Gerar SBOM CycloneDX/SPDX para **cada artefato** do exato SHA.
2. Resolver a árvore transitiva, incluindo dependências Go, Bun, Wails, WebView/NSIS, assets, fontes, logos e binários baixados de terceiros (ex.: `cloudflared`).
3. Conferir licenças, avisos, atribuições e redistribuição, incluindo compatibilidade com a licença **atual PolyForm Strict** e com a licença **alvo Apache-2.0**, além de obrigações/copyleft que continuem aplicáveis; obter revisão jurídica quando necessária.
4. Distribuir os textos de licença e notices exigidos no próprio pacote, além do documento legível.
5. Anexar logs/checksums e auditoria de dependências à evidência da release; `UNKNOWN` não é aprovado.

Regenerar inventários e executar os verificadores quando `go.mod`, `go.sum`, `web/package.json` ou `web/bun.lock` mudarem. Veja [publicação](../PUBLICATION_CHECKLIST.md) e [certificação](../operations/release.md).

## Abertura comunitária planejada

O inventário continua preliminar. A [migração responsável](OPEN_SOURCE_TRANSITION.md)
não autoriza presumir que logos, fontes, binários empacotados, código de terceiros
ou assets estejam sob Apache-2.0. Cada componente deve manter licença e atribuição
próprias; gerar SBOM e notices para o exato artefato distribuído. O estado
jurídico do source atual permanece conforme [LICENSE](../../LICENSE).
