# Suporte de plataformas

A linha **Nexus 1.0 ainda está em beta**. O trabalho que estava separado como
“1.1” foi incorporado ao candidato `v1.0.0-beta.2`.

A tag histórica `v1.0.0-beta.1` existe no repositório-fonte, mas não há
GitHub Release pública correspondente. O source candidato `v1.0.0-beta.2`
integra Linux amd64, Windows amd64 e macOS arm64; nenhuma plataforma deve ser
anunciada como publicada/certificada para essa beta antes da evidência
exact-SHA correspondente.

| Capacidade | Linux amd64 | Windows amd64 | macOS arm64 |
| --- | --- | --- | --- |
| CLI/Core | integrado | integrado | integrado |
| Terminal/runtime | PTY + UDS | ConPTY + Named Pipe | PTY + UDS |
| Desktop Wails | integrado | integrado | integrado |
| Nexus HUD | integrado | integrado | integrado |
| Assinatura do SO | manifesto Ed25519 | Authenticode + timestamp | Developer ID |
| Gatekeeper/notarização | N/A | N/A | notarytool + stapling + spctl |
| Instalador principal | DEB / install.sh | NSIS assinado | PKG assinado e notarizado |
| Lifecycle smoke | install/version/doctor/uninstall | NSIS install/version/doctor/uninstall | PKG install/version/doctor/uninstall |
| Updater | manifesto Ed25519 | manifesto Ed25519 | manifesto Ed25519 |
| Certificação atual do beta.2 | PENDING | PENDING | PENDING |

## Política de publicação da v1

A presença de código, cross-build ou um merge não caracteriza suporte. A
publicação continua fail-closed e vinculada ao SHA promovido.

| Release | Plataformas pretendidas | Evidência obrigatória |
| --- | --- | --- |
| `v1.0.0-beta.1` | Linux amd64 | tag histórica no source; sem GitHub Release pública |
| `v1.0.0-beta.2+` | Linux + Windows + macOS | CI exact-SHA + Windows + macOS + seal |
| `v1.0.0-rc.*` | Linux + Windows + macOS | CI exact-SHA + Windows + macOS + seal |
| `v1.0.0` | Linux + Windows + macOS | tudo acima + Stable Certification |

Versões futuras desconhecidas usam a política mais restritiva por padrão.

## Homologação local quando GitHub Actions estiver indisponível

O repositório contém um harness local que executa os gates automatizáveis e
registra evidência por SHA:

```bash
./scripts/homologate-beta.sh
```

Windows:

```powershell
.\scripts\homologate-beta.ps1 -ExpectedSha (git rev-parse release/v1.0.0-beta.2)
```

Isso não substitui assinatura/notarização nem native certification remota.
Serve para impedir que a indisponibilidade temporária do GitHub Actions bloqueie
todo o ciclo de correção e homologação beta.

## Nexus HUD

O HUD usa a mesma linha de release da v1. Ele é uma projeção do Core, não um
segundo Capacity Monitor.

| HUD | Linux amd64 | Windows amd64 | macOS arm64 | Web |
| --- | --- | --- | --- | --- |
| Projection/Core | integrado | integrado | integrado | integrado |
| Gadget nativo | build disponível | build disponível | build disponível | N/A |
| Autostart | adapter Linux | adapter Windows | adapter macOS | N/A |
| Frameless/always-on-top | compositor/WM dependent | source implementado | source implementado | N/A |
| Multi-monitor / DPI | UNVERIFIED | UNVERIFIED | UNVERIFIED | viewport |
| Certificação beta.2 | PENDING | PENDING | PENDING | PENDING |

`NEXUS_HUD_CROSS_PLATFORM` continua **NO_GO** como claim de certificação até
o mesmo SHA passar a matriz nativa exigida.

## Windows amd64

O runner de certificação deve usar:

```text
self-hosted, Windows, X64, nexus-windows
```

Dependências:

- Go 1.25.14;
- Bun 1.3.9;
- Windows SDK com `signtool.exe`;
- NSIS com `makensis.exe`;
- WebView2 Runtime.

O environment `native-signing` fornece:

- `WINDOWS_SIGNING_PFX_BASE64`;
- `WINDOWS_SIGNING_PFX_PASSWORD`.

A certificação produz e valida:

- `nexus_Windows_x86_64.zip`;
- `nexus-desktop_Windows_x86_64.zip`;
- `nexus-hud_Windows_x86_64.zip`;
- `nexus-setup-Windows_x86_64.exe`;
- Authenticode;
- HUD launch smoke;
- `nexus doctor --json`;
- install/uninstall real.

## macOS arm64

O runner deve usar:

```text
self-hosted, macOS, ARM64, nexus-macos
```

O environment `native-signing` fornece:

- `APPLE_APP_CERT_P12_BASE64`;
- `APPLE_APP_CERT_PASSWORD`;
- `APPLE_INSTALLER_CERT_P12_BASE64`;
- `APPLE_INSTALLER_CERT_PASSWORD`;
- `APPLE_NOTARY_KEY_P8_BASE64`;
- `APPLE_NOTARY_KEY_ID`;
- `APPLE_NOTARY_ISSUER_ID`.

A certificação produz:

- `nexus_Darwin_arm64.tar.gz`;
- `nexus-desktop_Darwin_arm64.zip`;
- `nexus-hud_Darwin_arm64.zip`;
- `nexus-installer_Darwin_arm64.pkg`;
- Developer ID;
- notarização/stapling/Gatekeeper;
- HUD launch smoke;
- install/uninstall real.

## Manifesto e updater

A partir de `v1.0.0-beta.2`, o manifesto assinado exige a matriz completa:

```text
linux_amd64
desktop_linux_amd64
hud_linux_amd64
windows_amd64
desktop_windows_amd64
hud_windows_amd64
windows_amd64_nsis
darwin_arm64
desktop_darwin_arm64
hud_darwin_arm64
darwin_arm64_pkg
```

A release falha se faltar qualquer chave exigida. Jobs skipped, CI ausente ou
artefatos sem assinatura nunca contam como PASS.
