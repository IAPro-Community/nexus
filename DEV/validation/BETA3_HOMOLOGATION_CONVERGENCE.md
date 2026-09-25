# Nexus 1.0.0-beta.3 — convergência da homologação Linux + Windows

Data: 2026-09-25

## Baseline

- candidato beta.3: `release/v1.0.0-beta.3@61b87409ba93f307ce2f95f7ca76348a310e3d95`
- evidência Linux beta.2: `fix/beta2-linux-autopilot-20260925@e4574e4cfd4676032e2284f8a1b2a3c5885e287e`
- remediação Windows beta.2: `fix/beta2-windows-autopilot-20260925-1039`
- head Windows revisado: `a9f6dda3ff063b7b46d2260680a75d09cfa661c3`

Esta branch porta os achados para a beta.3 por semântica. Ela não faz merge da
linha beta.2 e preserva o enforcement/ExecutionRole/Voice introduzido no PR #23.

## Correções incorporadas

- proteção contra self-kill do launcher standalone;
- expectativa de schema derivada da maior migration embutida, sem constante 26/27/28;
- OpenCode golden e normalização de versão alinhados;
- assertions NSIS corrigidas e release policy alinhada à beta.3;
- regras de lint/error strings convergidas;
- `clip: rect(...)` removido do visually-hidden;
- `.gitattributes` para LF determinístico e scripts Windows em CRLF;
- helper E2E cross-platform com `nexus.exe` no Windows;
- fixtures E2E sem dependência obrigatória de GNU Make;
- cleanup Windows por caminho nativo quando necessário;
- docs capture portável;
- helper de recaptura documental exact-SHA para Windows;
- homologador Windows com:
  - SHA esperado;
  - AMD64;
  - Go 1.25.14;
  - Bun 1.3.9;
  - golangci-lint v2.12.2;
  - govulncheck v1.7.0;
  - HOME/profile/data isolados;
  - trust root canônico;
  - browser matrix completa;
  - Desktop + HUD Wails;
- homologador Linux com pin de SHA e gofmt limitado a arquivos rastreados;
- comandos Bun normalizados para execução cross-platform;
- `VERSION`, package Web, Desktop Wails e HUD Wails alinhados em `1.0.0-beta.3`;
- documentação localizada e ledger corrente alinhados à beta.3.

## Validação estrutural executada no GitHub

Confirmado na branch:

- nenhuma referência `local-beta2` no homologador Windows;
- `ExpectedSha` presente no homologador Windows;
- browser matrix contém Composer spine/vague/themes, Axe e visual;
- helper E2E resolve `nexus.exe` no Windows;
- fixtures portadas não dependem de `make build`;
- assertions de release não contêm a antiga política `v1.1.0-beta.1`;
- assertion beta.3 presente;
- SCSS não contém mais `clip: rect`;
- teste de schema não usa número fixo;
- Web/Desktop/HUD reportam `1.0.0-beta.3`.

## Boundary de evidência

Isto é remediação e validação estrutural de source. Não é PASS nativo Windows.

Antes de publicar uma beta que anuncie Windows:

1. checkout limpo desta branch/commit em Windows AMD64;
2. recaptura documental exact-SHA;
3. execução de `scripts/homologate-beta.ps1` no mesmo SHA;
4. smokes manuais de Codex selector/terminal, OpenCode Certify/enforcement, HUD e Voice;
5. packaging/NSIS/AuthentiCode no mesmo SHA conforme a política de release.

Linux também deve ser re-homologado no mesmo SHA final. macOS permanece
UNVERIFIED até certificação nativa do mesmo SHA se for anunciado pela release.
