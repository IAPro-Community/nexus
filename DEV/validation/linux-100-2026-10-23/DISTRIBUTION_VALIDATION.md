# Validação de distribuição e atualização Linux

Resultado: artefatos locais CLI/Desktop/HUD compilados e instalados; pacote Linux e atualização oficial não homologados.

## Build e instalação local

O commit-base remoto era bf6adf0c80ac3531d7291844512a14a017ab0002 e VERSION é 1.0.0-beta.10. Os alvos make build, make build-desktop e make build-hud foram executados. Os binários foram instalados por substituição atômica nas pastas ~/.local/bin e no bin/.local do perfil Codex kivergmail. Os dois conjuntos são idênticos por SHA-256; hashes finais estão em `logs/install-final-go1.26.10-bf6adf0c.log`. Cópias anteriores preservadas em `/home/desenvolvedor/.local/state/ai-manager/nexus-update-backups/20261023T203808Z-go1.26.10`.

A instalação informa beta.10, Linux amd64 e o commit-base bf6adf0c. O binário inclui alterações locais não commitadas, portanto a string de commit não substitui os hashes do artefato como proveniência.

## Distribuição oficial

- O projeto declara Debian/DEB como pacote Linux no GoReleaser; nenhum DEB/tarball do SHA e worktree testados foi gerado, inspecionado ou instalado.
- O diretório dist já continha artefatos antigos beta.7 e foi preservado. O harness scripts/homologate-beta.sh chama GoReleaser com --snapshot --clean, exige um checkout limpo e Bun 1.3.9; o ambiente local tem Bun 1.4.2. Esse harness e seu snapshot ficaram BLOCKED para não apagar os artefatos existentes.
- Não foi publicada release estável 1.0.0. Assinatura, checksum publicado, manifesto do GitHub Releases, instalação/desinstalação de pacote e compatibilidade de arquitetura não foram validados.

## Atualizador

Não foi executada atualização A→B, download de GitHub Release, verificação de assinatura/checksum, interrupção de rede, recuperação, rollback nem preservação de configuração após atualização. Esses cenários permanecem NOT_RUN. Nenhuma atualização destrutiva foi aplicada à instalação local.

## Resultado

O estado instalado é beta.10, não uma release estável homologada. Distribuição oficial Linux permanece NO-GO até validar o pacote DEB exato, o manifesto/artefatos da GitHub Release e a cadeia de assinatura no SHA final limpo.
