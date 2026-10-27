# Login do Claude Code no Windows — 2026-10-26

## Diagnóstico e correção

O adapter do Claude no Nexus chamava `claude login`. No Claude Code 2.1.310 instalado nesta máquina, esse não é o comando de autenticação: o CLI mostra a ajuda genérica da sessão interativa, que pode deixar o terminal preto e sem iniciar o OAuth. A autenticação oficial é `claude auth login`.

`internal/core/provider/adapters/claude/claude.go` agora passa `auth login` ao CLI. O comando do Nexus continua sendo `nexus login claude [perfil]`.

## Verificação

- PASS — `gofmt` no adapter.
- PASS — `go build -o %TEMP%\nexus-claude-login-check.exe ./cmd/nexus`.
- PASS — `go vet ./internal/core/provider/adapters/claude`.
- PASS — `claude auth login --help` na instalação local 2.1.310 lista “Sign in to your Anthropic account”.
- Diagnóstico reproduzido sem autenticar: `claude login --help` mostra a ajuda genérica de `claude`, não a tela de autenticação.
- Nenhum login OAuth foi iniciado e nenhuma credencial foi alterada.

## Estado de implantação

A aplicação instalada beta.11 segue aberta em `127.0.0.1:13000` (HTTP 200), com processos Nexus ativos. O binário verificado foi compilado em `%TEMP%`; a instalação e o reinício não foram feitos para preservar os runtimes em execução. Portanto, a correção está no checkout, mas só será usada pela aplicação depois de atualizar e reiniciar a instalação local.
