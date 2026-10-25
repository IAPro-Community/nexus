# Distribuição e atualização — Linux AMD64

## Artefatos locais produzidos

- GoReleaser v2.18.0 snapshot local do CLI: `nexus_Linux_x86_64.tar.gz` e `nexus_1.0.0_linux_amd64.deb`.
- A execução de snapshot recebeu `NEXUS_VERSION=1.0.0`; foi uma substituição local para validar empacotamento e não foi publicada.
- O `.deb` foi inspecionado com `dpkg-deb --info`, extraído para root temporário e o binário executou `nexus version --json` como `1.0.0`, `amd64`, commit `f62fdec0…`. Não instalamos no host.
- Os archives nativos Desktop e HUD foram montados com os nomes emitidos pela CI (`nexus-desktop_Linux_x86_64.tar.gz`, `nexus-hud_Linux_x86_64.tar.gz`), extraídos e colocados lado a lado num diretório temporário. Ambos eram executáveis e o Desktop iniciou seu HUD irmão.
- `file` confirmou ELF x86_64; `ldd` não encontrou bibliotecas ausentes.

Checksums registrados em `logs/goreleaser-checksums.log`, `logs/linux-artifact-checksums.log` e `logs/patched-binary-checksums.log`. Os binários temporários ficam fora do Git no worktree de validação `/tmp/nexus-linux-100-f62fdec0/`; não foram adicionados ao repositório.

## Estado da publicação oficial

A fonte da branch permanece `1.0.0-beta.10`. A consulta à API do GitHub encontrou previews `prerelease` e unsigned, inclusive a tag preview mais recente em SHA diferente do homologado. Não foi encontrada release/tag estável `v1.0.0`, manifesto estável assinado ou artefatos estáveis para instalação. Não houve publicação, tag ou mudança de release.

## Teste do instalador

O script real foi executado com `--version=1.0.0 --no-path` e `NEXUS_INSTALL_DIR` temporário. A URL do manifesto estável retornou HTTP 404; o script encerrou com exit 1 e não deixou `nexus` instalado. Esse comportamento fail-closed é esperado sem release, mas impede homologar a instalação/atualização válida, integridade com assinatura publicada, recuperação após download interrompido e preservação de configuração.

## Ajustes locais

`install.sh` antes baixava o Desktop sem instalar `nexus-hud`, embora o Desktop procure o HUD como binário irmão. A correção local agora exige que `nexus-hud_Linux_x86_64.tar.gz` esteja listado no manifesto assinado, verifica assinatura/digest pelo fluxo existente e instala o HUD ao lado do Desktop. Também removemos orientação RPM que não correspondia à configuração de artefatos. `bash -n`, testes do mapa de chaves do manifesto e smoke isolado de layout passaram. O caminho válido ainda precisa de uma release estável real para end-to-end.

## Limitações

- `goreleaser release --snapshot` cobriu CLI e DEB configurados; os arquivos nativos Desktop/HUD foram montados pelas convenções da CI, sem criar formato novo.
- Nenhum pacote foi instalado no sistema e nenhum artefato foi publicado.
- ShellCheck não estava disponível; `make format-check`, `bash -n` e regressões relevantes foram executados.
