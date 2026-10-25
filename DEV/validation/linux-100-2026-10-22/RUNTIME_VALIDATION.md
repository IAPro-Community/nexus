# Runtime, agentes e terminal — Linux

## Desktop e Core embutido

- `nexus-desktop` Linux AMD64 foi iniciado em uma sessão D-Bus real do GNOME 49/Wayland; a janela Wails `IAPro Nexus` foi observada via AT-SPI com página de onboarding e extents 1332×889.
- O Core embutido respondeu HTTP 200 (814 bytes) no smoke do build corrigido.
- Desktop e HUD permaneceram vivos juntos; o Desktop iniciou `nexus-hud --linked`. Isso prova início do processo, não prova a apresentação visual do HUD.
- Uma amostra de repouso após a execução inicial mediu Desktop ~223 MB e HUD ~194 MB, CPU ~0,5%/~0,4%. Isso não substitui um soak test.
- O build corrigido também iniciou como Wails e respondeu no Core. A interação nativa permanece bloqueada pelo ambiente de captura/input.

## E2E browser

Os comandos foram executados com o CLI construído a partir da mesma base de release, Playwright Chromium local e dados temporários.

| Suíte | Resultado | Falha observada |
|---|---|---|
| `test:e2e` | PASS | — |
| `test:e2e:capacity` | PASS | — |
| `test:e2e:hud` | PASS | Teste browser; não certifica HUD nativo |
| `test:a11y` | PASS | Axe reportou zero violações minor no cenário executado |
| `test:visual` | PASS | Sete viewports headless e verificações de menus/tema/densidade |
| `test:e2e:composer` | FAIL | Timeout do botão `Nova elaboração`; a página ficou na visão overview. Log também capturou 403 em `/wails/custom.js`. |
| `test:e2e:surface` | FAIL | Abas observadas: `overview`, `terminals`; a aba de work product esperada não apareceu. |
| `test:e2e:composer-spine` | FAIL | Timeout do botão `existing prompt` / `prompt existente`. |
| `test:e2e:composer-vague-spine` | FAIL | Timeout no placeholder do campo de ideia/objetivo. |
| `test:e2e:composer-themes` | FAIL | 20/20 temas não chegaram a `.nx-composer-surface`; não certifica os visuais do Composer. |

## Agentes e continuidade

- A suíte Go e fixtures/mocks do repositório passaram, incluindo as regressões de roteamento focadas.
- A solicitação explícita em português `peça ao Codex para revisar isso` era classificada como implementer; a correção local encaminha os termos `revisar`, `revise`, `revisão` e `revisao` para reviewer. Teste de regressão passou.
- A inspeção de disponibilidade listou CLIs instaladas; a consulta de perfis não confirmou conta autenticada. Nenhuma chamada live a provedor externo foi feita. Delegação real, failover, retomada, cancelamento e continuidade entre agentes ficam BLOCKED.

## Terminal e resiliência

Não foi possível operar terminal/PTY na janela nativa por falta de input confiável. Não foram validados manualmente ANSI, Unicode, resize, stdin/stdout, cancelamento, processo órfão, múltiplas sessões nem recuperação após crash. A suíte Go automatizada passou, mas não é evidência de homologação desses fluxos na UI.

Crashes, processo longo, múltiplas instâncias, configuração corrompida, serviço/porta ocupada e recuperação não foram provocados nesta campanha; status NOT_RUN/BLOCKED conforme a matriz.

## Reprodução dos E2E falhos

```bash
bun install --cwd web --frozen-lockfile
make build
bun run --cwd web test:e2e:composer
bun run --cwd web test:e2e:surface
bun run --cwd web test:e2e:composer-spine
bun run --cwd web test:e2e:composer-vague-spine
bun run --cwd web test:e2e:composer-themes
```

Logs integrais: `logs/test-e2e-*-final.log`; status da campanha: `logs/e2e-status-final.tsv`.
