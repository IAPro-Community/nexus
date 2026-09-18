> **HISTORICAL — NOT CURRENT RELEASE EVIDENCE**
>
> Canonical current status: `DEV/validation/current/RELEASE_STATUS.md`.

# Revisão técnica atual — Nexus

Data da execução: 2026-09-03/04 (horário local/UTC)

## Parecer

**Aprovado para uso local individual em loopback, com riscos remotos e de
confiança explicitamente aceitos.** O projeto compila, os testes Go e Web
passam e o servidor foi recompilado/reiniciado. Isso não equivale a aprovação
para exposição pública, uso multiusuário ou execução de repositórios não
confiáveis.

## Correções aplicadas nesta revisão

- `web/src/app/useNexusData.ts`: refresh global single-flight, sem requisições
  sobrepostas, pausado em abas ocultas e atualizado ao retornar à aba.
- `web/src/nexus/AgentTerminal.tsx`: removido estado React obsoleto que gerava
  warning de lint.
- `internal/control/web/handler_terminal.go`: encerramento idempotente do
  WebSocket/IPC quando o runtime morre, liberando assinatura de eventos e
  evitando handler preso aguardando indefinidamente o navegador.
- `internal/control/web/handler_terminal.go`: resize usa fila de capacidade um,
  mantendo apenas a última dimensão enquanto um RPC está em andamento.
- Arquivos Go modificados anteriormente no escopo principal foram normalizados
  com `gofmt`.

## Evidência executada

- `go test -count=1 ./...`: PASS.
- `go vet ./...`: PASS.
- `go test -race -count=1 ./internal/control/... ./internal/nexus/... ./internal/core/security/... ./internal/core/session/...`: PASS.
- `go test -race -count=1 ./internal/control/web`: PASS após a correção do
  encerramento do terminal.
- `make web-verify`: PASS — typecheck, lint, null-arrays, 188 testes, build,
  sincronização do bundle embutido e marcadores de UI.
- `npm audit --offline --omit=dev --audit-level=high`: 0 vulnerabilidades.
- Busca de credenciais: somente fixtures intencionais de testes/redaction.
- Build de `cmd/nexus` para `linux/amd64`, `darwin/amd64` e `windows/amd64`:
  PASS (compilação cruzada; não substitui execução no host).
- `git diff --check`: PASS.
- Smoke local: `/api/v1/health` retornou 200, bootstrap retornou 302 com
  cookie HttpOnly/SameSite e CSP, e `nexus version` reportou `0.5.0-beta.23`.

## Riscos e gaps restantes

### Alto — remoto por HTTP não fornece confidencialidade

`internal/control/web/bind.go:15-42` permite endereço privado somente com
`--remote`, mas o servidor continua HTTP. O cookie de sessão é emitido em
`internal/control/web/server.go:144-150` e `:206` sem `Secure`, necessário para
HTTP funcionar. Em uma LAN hostil, bootstrap, cookie, comandos digitados e
saída do terminal podem ser interceptados. O bind público continua recusado.

Este risco foi aceito para o escopo escolhido (Safari em LAN privada via HTTP),
mas a recomendação segura continua sendo loopback, túnel SSH ou VPN privada.
TLS/identidade deve ser requisito antes de exposição fora de uma rede confiável.

### Alto — verificações são comandos shell confiados

`internal/nexus/runner/verifier.go:16-54` executa comandos via `/bin/sh -c` ou
`cmd.exe /c`; os comandos podem vir de `verification_commands` em
`internal/control/web/handlers_nexus.go:1385-1409` e scripts detectados no
workspace. Isso é coerente com o produto local, porém um repositório malicioso
ou contrato gerado por fonte não confiável pode executar comandos com a
identidade do usuário. Os guards de PATH em
`internal/nexus/autonomyguard/process_guard.go` são política de conveniência,
não sandbox contra código hostil.

Antes de suportar projetos não confiáveis ou multiusuário, é necessário sandbox
de SO/container, allowlist explícita e confirmação para comandos fora da política.

### Médio — perda deliberada de saída em consumidor lento

`internal/control/host/fanout.go:59-71` usa filas limitadas e descarta chunks
quando o consumidor está atrasado, protegendo o processo supervisionado contra
bloqueio. O ring buffer de 128 KiB (`internal/control/host/host.go:74`) limita
memória, mas um terminal muito ativo pode ter lacunas visuais. A recuperação
deve usar reconexão/histórico; falta um marcador/telemetria explícita de overflow.

### Médio — validação multiplataforma ainda é compilação, não execução

Linux foi exercitado localmente. Windows e macOS foram compilados por
`GOOS/GOARCH`; os jobs reais de ConPTY, Named Pipe, PTY, permissões e installer
estão definidos em `.github/workflows/ci.yml`, mas não foram executados neste
ambiente. Não há aplicativo nativo iOS: o caminho suportado é Web/Safari, que
também não foi executado em dispositivo Safari nesta rodada.

### Baixo — modelo comercial/multi-tenant não está implementado

O domínio atual é local individual: regras de projeto, Agente, geração de
runtime, quota e missão ficam em `internal/nexus`, com sessão Web local. Não há
isolamento de tenant, RBAC, billing ou entitlement SaaS. Isso é adequado à
premissa atual, mas é obrigatório antes de transformar o produto em serviço
compartilhado.

## Desempenho e memória

As estruturas críticas têm limites: scrollback xterm 5000 linhas, ring buffer
128 KiB, fanout 256 itens por cliente, frames IPC/JSON limitados e snapshots de
contexto limitados. O race detector passou nos módulos de concorrência. O
polling global foi coalescido e pausado em background; resize e encerramento de
WebSocket também foram limitados.

Não foi executado ensaio real de 20 projetos/10 terminais, heap profile ou soak
test de horas. Portanto não há base para afirmar ausência absoluta de leaks ou
meta de latência sob carga; o próximo gate deve medir CPU, RSS, goroutines,
sockets, perda de fanout e tempo de reconexão nesse envelope.

## Segurança positiva verificada

- bind loopback por padrão e rejeição de wildcard/público;
- bootstrap aleatório, sessão com expiração absoluta/idle e CSRF para mutações;
- validação de Origin no HTTP/WebSocket;
- CSP, `nosniff`, `frame-ancestors`, `Referrer-Policy` e Permissions-Policy;
- respostas sanitizadas não expõem `Env`, argumentos ou binário do runtime;
- filesystem host-wide bloqueado em modo remoto;
- remotes Git redigidos e snapshots/contexto limitados;
- testes de segurança locais e race do núcleo passaram.

## Ferramentas indisponíveis

`govulncheck`, `gitleaks`, `semgrep`, `trivy` e `actionlint` não estão
instalados. O `npm audit` online não concluiu por indisponibilidade de rede;
foi usado o modo offline, que reportou zero vulnerabilidades no cache local.
