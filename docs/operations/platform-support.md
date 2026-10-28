# Suporte de plataformas

A linha **Nexus 1.0 ainda está em beta**. O trabalho que estava separado como
“1.1” foi incorporado à linha de desenvolvimento da v1. A versão-fonte corrente
é declarada em `VERSION` e `web/package.json`; a situação de homologação
vigente está em [status da release](../release-status.md). A presença de uma
versão beta no source não constitui certificação exact-SHA.

O registro histórico da última beta anterior menciona `v1.0.0-beta.1` (Linux amd64); a disponibilidade pública atual dos artefatos exige verificação independente.
O source candidato seguinte integra Linux amd64, Windows amd64 e macOS arm64,
mas nenhuma dessas plataformas deve ser anunciada como certificada para a nova
beta antes da evidência exact-SHA correspondente.

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
| Certificação do source atual | PENDING | PENDING | PENDING |

## Política de publicação da v1

A presença de código, cross-build ou um merge não caracteriza suporte. A
publicação continua fail-closed e vinculada ao SHA promovido.

| Release | Plataformas pretendidas | Evidência obrigatória |
| --- | --- | --- |
| `v1.0.0-beta.1` | Linux amd64 | evidência histórica já publicada |
| `v1.0.0-beta.7+` | Linux + Windows + macOS | CI exact-SHA + Windows + macOS + seal |
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
.\scripts\homologate-beta.ps1
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
| Certificação source atual | PENDING | PENDING | PENDING | PENDING |

O source atual adiciona alça de arraste nativa e três presets de tamanho
(`compact`, `comfortable`, `large`). Em 2026-10-21 o fluxo foi exercitado em
Windows 11 sobre o worktree baseado em `f6098b8c684a97bf7e8e56804109ca491c78cfa2`
(UIA + arraste real + reinício), mas isso **não** certifica o SHA final nem
Linux/macOS. `NEXUS_HUD_CROSS_PLATFORM` continua **NO_GO** como claim de
certificação até o mesmo SHA passar a matriz nativa exigida.

## Toolchains registrados no source

| Gate | Go | Referência |
| --- | --- | --- |
| CI Linux | `1.26.10` | `.github/workflows/ci.yml` |
| Windows beta.10 (sem signing) | `1.26.10` | `.github/workflows/windows-beta10-homologation.yml` |
| Certificação nativa assinada | **confirmar runner/workflow efetivamente executado** | `.github/workflows/ci.yml` e `.github/workflows/release.yml` |

`go.mod` exige Go 1.25 como baseline da linguagem, **não** obriga todos os runners a usar o mesmo patch. Uma campanha só certifica o SHA e a toolchain que realmente executou. Divergências entre harness/CI devem ser reconciliadas no relatório antes de marcar PASS.

## Windows amd64

O runner de certificação deve usar:

```text
self-hosted, Windows, X64, nexus-windows
```

Dependências:

- Go 1.26.10 no workflow Windows beta.10 dedicado e também no CI Linux/macOS atual. Validar toolchain efetiva antes de executar; não misturar resultados de versões Go divergentes sem registrar a evidência;
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

A partir de `v1.0.0-beta.7`, o manifesto assinado exige a matriz completa:

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

## Publicação no próprio repositório (21/10/2026)

Código, instaladores e binários oficiais são disponibilizados em
[GitHub Releases do IAPro-Community/nexus](https://github.com/IAPro-Community/nexus/releases).
Os instaladores e o updater usam os canais assinados `channel-beta` e
`channel-stable` no próprio repositório. `NEXUS_RELEASE_REPO` permanece
apenas como override explícito para forks e desenvolvimento.

O pipeline `.github/workflows/release.yml` usa `GITHUB_TOKEN` para
publicar, sem precisar de `NEXUS_DIST_TOKEN` ou `NEXUS_DIST_REPO`.
O marcador `channel-beta` é um ponteiro móvel intencional, alinhado ao
SHA do candidato antes da publicação; tags de versões finais permanecem
imutáveis. A atualização automática continua exigindo manifesto assinado
Ed25519 e verificação SHA-256.

A mudança de repositório não dispensa Authenticode no Windows nem
Developer ID/notarização no macOS para os gates de distribuição
certificada. Builds unsigned podem ser homologados localmente, mas
não serão tratados como assinados ou certificados.

## Prévia portátil sem assinatura Microsoft (opt-in)

A `release/v1.0.0` também contém
[`unsigned-preview.yml`](../../.github/workflows/unsigned-preview.yml),
um workflow que compila nos runners GitHub-hosted Linux amd64, Windows
amd64 e macOS arm64/Intel e publica os ZIP/tar.gz em uma
[GitHub Release de prévia](https://github.com/IAPro-Community/nexus/releases)
**com tag imutável `preview-<SHA completo>`**, somente se todos os quatro
ambientes (Linux amd64, Windows amd64, macOS arm64 e macOS Intel) passarem. Não usa
certificado Microsoft, conta Apple Developer, token de espelho nem chave
privada do updater; é uma prévia manual **sem instalador assinado**.

Apenas atualizar a branch **não** publica a prévia. Como o workflow ainda
está somente na branch `release/v1.0.0`, a maneira estável de dispará-lo
é enviar um commit intencional cujo título contenha `[unsigned-preview]`:

```bash
git fetch origin
git switch release/v1.0.0
git pull --ff-only origin release/v1.0.0
git commit --allow-empty -m "ci(preview): build unsigned assets [unsigned-preview]"
git push origin release/v1.0.0
```

Isso cria uma execução de `Unsigned Source Release Preview` no Actions.
Nenhum release de prévia é publicado se algum dos jobs falhar.
Cada execução aprovada produz uma tag imutável vinculada ao SHA de entrada;
commits paralelos na release não fazem o workflow trocar o SHA compilado. O
arquivo `SHA256SUMS.txt` acompanha os oito archives: Linux CLI/Desktop/HUD,
Windows CLI/Desktop/HUD e dois conjuntos macOS (Apple Silicon e Intel),
cada um com CLI + dois bundles `.app`.

**Não** use `install.sh --version=latest` ou `install.ps1 -Version
latest` para tentar instalar essa prévia: esses instaladores exigem o
manifesto Ed25519 oficial, que não é produzido nas tags `preview-*`.
Instale os arquivos de prévia manualmente, num diretório isolado e sem
substituir instalações existentes. O macOS Gatekeeper e o Windows
SmartScreen podem bloquear apps sem assinatura; não desative a proteção
do sistema para contornar o bloqueio. A prévia não substitui a
certificação de distribuição estável.

## Atualizações a partir do GitHub Releases

A GUI consulta o Update Service ao iniciar, ao recuperar o foco e, durante
uso contínuo, a cada hora. Um update **só é anunciado** se o manifesto
`update-manifest.json` do canal oficial for autenticado por Ed25519 e
validado para a plataforma. Pré-releases unsigned sob tags `preview-*` não
constituem canais automáticos e nunca participam desse processo.

Para instalações beta, quando não há beta mais nova, o Desktop também
consulta a versão estável assinada. Builds beta com o mesmo número de
versão só são considerados novos quando o `source_sha` assinado é
**descendente comprovado** do commit embutido no executável local,
consultando a API pública de comparação de commits do GitHub. SHA
diferente por si só não significa versão nova. Se a API estiver
indisponível, se os históricos divergirem ou se o build local não
contiver commit válido, a atualização de mesma versão não é oferecida
(falha segura). Atualizações com SemVer superior seguem a validação
criptográfica normal. `nexus update` retorna "atualizado" sem
reinstalar um binário de mesma versão quando não houver upgrade elegível.

Na aba **Configurações > Atualizações**, a ação do Nexus é separada do
botão **Atualizar Maestro**. Quando um pacote nativo verificado está
disponível, **Baixar instalador oficial do Nexus** abre o arquivo
publicado no repositório; quando não há instalador compatível, a ação
abre a página da release para orientações manuais.

O Desktop e o HUD **não** fazem auto-replace de seus próprios
executáveis: o manifesto `goos_goarch` contém o artefato de CLI e
nunca deve substituir um binário Wails. NSIS, PKG, DEB, RPM e outros
métodos gerenciados permanecem na rota de instalação nativa. Não há
instalação sem consentimento nem reinício automático durante sessões
ativas. Para CLIs standalone, use:

```bash
nexus update check
nexus update
```

Caso deseje mudar explicitamente da beta para a estável na CLI:
`nexus update check --channel=stable` e
`nexus update --channel=stable`. A interface de Desktop oferece o
pacote estável verificado quando elegível, mas a instalação ainda
precisa ser confirmada e concluída pelo instalador do sistema.

A publicação de uma release assinada só é considerada concluída quando
os gates exact-SHA, chaves/certificados e o manifesto gerado tiverem
sido validados. A detecção local não comprova a instalação final.
