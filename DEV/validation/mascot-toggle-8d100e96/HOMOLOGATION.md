# Homologação — toggle do popup no topo do painel (8d100e96)

Código: `7eca10d4` (feature) + `8d100e96` (docs/visual). HUD: `nexus-hud.exe` SHA-256
`9b78fb107248e09102aed76baf238342fc2990f11ae99ae1648b27f1d60f17f4`,
build `production`, Go `1.26.10`, commit injetado `8d100e96`.

## Mudança

O controle `Popup independente: Sim/Não` saiu do rodapé do painel e passou para
logo abaixo do cabeçalho (`.popupSettingRow`), com a dica contextual
`Mascote disponível` / `Necessário para o Mascote`. Em perfis com vários provedores
o rodapé exigia rolagem; agora o controle nasce na primeira dobra, tanto no painel
inline (`Nexus HUD`) quanto na janela destacada (`Nexus HUD Popup`).

## Harness

`scripts/homologate-hud-mascot-toggle-windows.ps1` com `-ExpectedSha` do build,
duas sessões e estado isolado: **PASS**, `failed_checks: []`.

- Sessão A: `Popup independente: Não` → habilitar (`detachedPopover=true`, popup
  oculto criado, cápsula `220x60`) → expandir (popup `334x496`) → Mascote
  (`136x112`, cápsula renderiza o mascote) → popup oculto.
- Sessão B (reinício): abre em `MASCOT` `136x112` com popup persistido → reabrir
  pelo mascote → `Popup independente: Sim` → desabilitar → `BAR` `220x60`, popup
  oculto, painel inline reabre com o toggle em `Não`.

Evidência: `RESULT.json`, `mascot.png`, `bar-restored.png`, logs das duas sessões.

## Gates

- `npm run quality`: 97 arquivos / 599 testes, lint, stylelint, prettier e typecheck OK.
- `gofmt` limpo, `go vet ./internal/desktop ./cmd/nexus-hud`, `go test` dos mesmos pacotes OK.
- `node scripts/docs-verify.mjs`: 192 Markdown / 33 assets OK no commit `8d100e96`.
- Manifesto visual recapturado em `7eca10d4` (`captured 10 isolated synthetic surface(s)`).
