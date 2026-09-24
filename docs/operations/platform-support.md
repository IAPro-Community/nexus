# Suporte de plataformas

A release estável **v1.0.0** permanece congelada em **Linux amd64**. A expansão
multiplataforma é implementada na linha seguinte e não altera a evidência da
v1.0.0.

A linha pós-v1 é cumulativa e prioriza Windows:

1. **v1.0.0** — Linux amd64;
2. **v1.1.0-beta.1** — Linux amd64 + Windows amd64;
3. **v1.1.0-beta.2** — Linux amd64 + Windows amd64 + macOS arm64;
4. **v1.1.0** — as três plataformas, após certificação estável.

Windows amd64 e macOS arm64 deixam de depender de cross-build como prova. Uma
plataforma só pode ser publicada quando houver certificação **nativa, assinada
e exact-SHA** no mesmo commit promovido pela release.

| Capacidade | Linux amd64 | Windows amd64 | macOS arm64 |
| --- | --- | --- | --- |
| CLI/Core | CI + GoReleaser | build/test nativo | build/test nativo |
| Terminal/runtime | PTY + UDS | ConPTY + Named Pipe | PTY + UDS |
| Desktop Wails | build nativo | build nativo | build nativo |
| Assinatura do SO | manifesto Ed25519 | Authenticode + timestamp | Developer ID |
| Gatekeeper/notarização | N/A | N/A | notarytool + stapling + spctl |
| Instalador principal | DEB / install.sh | NSIS assinado | PKG assinado e notarizado |
| Lifecycle smoke | install/version/doctor/uninstall | NSIS install/version/doctor/uninstall | PKG install/version/doctor/uninstall |
| Updater | manifesto Ed25519 | manifesto Ed25519 | manifesto Ed25519 |
| Evidência | exact-SHA | exact-SHA nativa | exact-SHA nativa |

## Nexus HUD — matriz de certificação

O HUD é uma projeção pós-v1 e acompanha a mesma expansão de plataformas desta
linha. Implementação em source não equivale a suporte público: cada gadget
nativo precisa passar pelo gate exact-SHA da plataforma correspondente.

| HUD | Linux amd64 | Windows amd64 | macOS arm64 | Web |
| --- | --- | --- | --- | --- |
| Projection/Core | implementado; CI pendente | source compartilhado | source compartilhado | implementado; CI pendente |
| Gadget nativo | implementado; certificação nativa pendente | **UNVERIFIED** | **UNVERIFIED** | N/A |
| Frameless | source Wails v2 | **UNVERIFIED** | **UNVERIFIED** | N/A |
| Always-on-top | compositor/WM dependent | **UNVERIFIED** | **UNVERIFIED** | N/A |
| Posicionamento global | X11: alvo; Wayland: limitado | **UNVERIFIED** | **UNVERIFIED** | viewport |
| Multi-monitor / DPI | **UNVERIFIED** | **UNVERIFIED** | **UNVERIFIED** | browser |
| Web HUD | N/A | N/A | N/A | implementado |
| Certificação final | **PENDING** | **UNVERIFIED** | **UNVERIFIED** | **PENDING** |

`NEXUS_HUD_CROSS_PLATFORM` permanece **NO_GO** enquanto qualquer target nativo
obrigatório estiver sem certificação. Em Wayland, posicionamento absoluto e
always-on-top podem ser limitados pelo compositor; a capability matrix do HUD
deve degradar de forma explícita, sem transformar limitação do compositor em
estado saudável.

## Regra de publicação

A presença de código ou um cross-build não caracteriza suporte. O workflow
`CI` aceita `native_platform=windows|macos|all`, sempre vinculado a
`native_target_sha`.

A política de publicação é fail-closed por versão:

| Release | Plataformas publicadas | Evidência nativa obrigatória |
| --- | --- | --- |
| v1.0.0 | Linux amd64 | baseline Linux já congelada |
| v1.1.0-beta.1 | Linux + Windows | `Windows native certification` |
| v1.1.0-beta.2 | Linux + Windows + macOS | Windows + macOS + `Seal native certification evidence` |
| v1.1.0 | Linux + Windows + macOS | Windows + macOS + seal + `Stable Certification` |

Versões futuras desconhecidas usam a política mais restritiva por padrão. O
`Signed Release` promove apenas artefatos nativos certificados no SHA exato;
ele não reconstrói Windows/macOS em Linux e não aceita jobs skipped como PASS
para uma plataforma requerida.

## Windows amd64

O runner deve ter os labels:

```text
self-hosted, Windows, X64, nexus-windows
```

Dependências do runner:

- Go 1.25.14;
- Bun 1.3.9;
- Windows SDK com `signtool.exe`;
- NSIS com `makensis.exe`;
- WebView2 Runtime para smoke do Desktop.

O environment GitHub `native-signing` deve fornecer:

- `WINDOWS_SIGNING_PFX_BASE64`;
- `WINDOWS_SIGNING_PFX_PASSWORD`.

A certificação gera e valida:

- `nexus_Windows_x86_64.zip`;
- `nexus-desktop_Windows_x86_64.zip`;
- `nexus-hud_Windows_x86_64.zip`;
- `nexus-setup-Windows_x86_64.exe` (inclui Desktop + HUD);
- evidência Authenticode;
- `nexus doctor --json` antes e depois da instalação.

O NSIS registra a aplicação no perfil do usuário, adiciona o diretório ao PATH
e remove ambos durante o uninstall.

## macOS arm64

O runner deve ter os labels:

```text
self-hosted, macOS, ARM64, nexus-macos
```

Ele deve ser um runner dedicado de certificação, com Xcode command line tools e
`sudo -n` habilitado somente para o lifecycle smoke.

O environment `native-signing` deve fornecer:

- `APPLE_APP_CERT_P12_BASE64`;
- `APPLE_APP_CERT_PASSWORD`;
- `APPLE_INSTALLER_CERT_P12_BASE64`;
- `APPLE_INSTALLER_CERT_PASSWORD`;
- `APPLE_NOTARY_KEY_P8_BASE64`;
- `APPLE_NOTARY_KEY_ID`;
- `APPLE_NOTARY_ISSUER_ID`.

A certificação gera:

- `nexus_Darwin_arm64.tar.gz`;
- `nexus-desktop_Darwin_arm64.zip`;
- `nexus-hud_Darwin_arm64.zip`;
- `nexus-installer_Darwin_arm64.pkg` (inclui Desktop + HUD).

O CLI e o app são assinados com Developer ID. O app e o PKG passam por
notarização, stapling e validação Gatekeeper antes de serem aceitos. O PKG é
instalado no runner, executa `nexus version` e `nexus doctor --json`, e é
removido por `nexus-uninstall`, preservando os dados do usuário.

## Manifesto e updater

O manifesto Ed25519 é validado conforme a matriz da release. Em
`v1.1.0-beta.1`, são obrigatórias as chaves Linux e Windows:

```text
linux_amd64
desktop_linux_amd64
hud_linux_amd64
windows_amd64
desktop_windows_amd64
hud_windows_amd64
windows_amd64_nsis
```

A partir de `v1.1.0-beta.2`, também são obrigatórias:

```text
darwin_arm64
desktop_darwin_arm64
hud_darwin_arm64
darwin_arm64_pkg
```

A release falha se faltar qualquer chave requerida pela sua política.

## Compatibilidade da v1.0.0

A evidência histórica da v1.0.0 continua válida e Linux-only. Este trabalho não
retroativamente declara Windows/macOS como certificados naquela versão. A
promoção dessas plataformas acontece somente em uma release posterior cujo SHA
tenha passado pela cadeia nativa acima.
