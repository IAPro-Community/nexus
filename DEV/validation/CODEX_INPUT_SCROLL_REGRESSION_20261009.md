# Nexus Codex — regressão de scroll, setas e menus interativos

Data: 2026-10-09  
Linha: `release/v1.0.0` (`1.0.0-beta.10` no momento da remediação)  
Estado: **código corrigido; smoke interativo nativo ainda obrigatório**.

## Defeito reproduzido conceitualmente no parser

O Codex exibe escolhas após planejamento e em prompts de aprovação. O
caminho de entrada interpretava relatórios de mouse X10 (`ESC [ M Cb Cx Cy`)
como `CSI M` finalizado, liberando bytes de botão/coordenadas no PTY.
Relatórios URXVT (`ESC [ b;x;yM`) também não eram descartados. Uma parte
das entradas nativas passava por dois scrubbers temporizados (cliente CLI e
SessionHost), amplificando atrasos entre `ESC` e `[A`/`[B`.
O reset externo também não desativava `DECSET 1007` (alternate scroll), que
pode converter roda do mouse em setas enquanto um TUI está na tela alternativa.

## Contratos corrigidos

1. Somente o **SessionHost** faz coalescência e saneamento de entrada antes do
   PTY para `nexus codex`. O cliente CLI é byte-transparent.
2. Mouse SGR, X10 e URXVT é filtrado no SessionHost; X10 incompleto é retido
   e descartado em timeout/flush, nunca encaminhado como `ESC[M`.
3. Reset do TTY desabilita `?1007` e modos mouse `?1005`/`?1016` além dos
   modos já existentes. O filtro de saída também bloqueia essas negociações.
4. **Não remover** setas reais CSI/SS3, Enter, Ctrl+C, Kitty/CSI-u ou texto
   arbitrário somente porque contém `[A`/`[B`.
5. O Web também remove relatórios X10/URXVT completos. Entradas Web
   fragmentadas continuam sendo resolvidas no SessionHost.

## Gates automatizados

```bash
go test -count=1 ./internal/runtime ./internal/control/terminal ./internal/control/web ./internal/app
go test -race -count=1 ./internal/runtime ./internal/control/terminal ./internal/control/web ./internal/app
cd web
bun install --frozen-lockfile
bun run test
bun run typecheck
bun run build
```

Estes comandos ainda precisam ser executados no **SHA exato** do candidato.
Regras: nenhuma falha de regressão; `go test -race` sem disputa;
nenhum reordenamento de bytes CSI/SS3; mouse X10 fragmentado nunca
deve causar um `ESC` ou sequência residual no provider.

## Gate permanente no CI

O workflow `.github/workflows/ci.yml` possui passos nomeados
`Codex input safety gate (mouse, selectors and fragmented PTY traffic)`
no Linux e `Codex terminal input safety gate (Windows ConPTY)` no Windows nativo.

- O Linux executa os testes de terminal e TTY com `-race -count=1` a cada
  CI regular da release, além da suíte Go completa.
- O Windows executa o mesmo conjunto relevante com `-count=1` no workflow
  `workflow_dispatch` de certificação nativa; esse job **não** executa
  automaticamente em cada push. O gate de publicação exige certificação
  Windows/Linux aplicável no mesmo SHA, conforme a política da release.
- As etapas PowerShell agora verificam `$LASTEXITCODE` após cada comando:
  um teste de Codex com falha não poderá ser encoberto pelo sucesso de
  um teste de delegação executado depois.
- Caso o runner esteja `queued`, **não há PASS**. Uma suíte que nunca
  iniciou não certifica a correção.
- Para impedir merge direto com checks vermelhos, os administradores devem
  configurar proteção do galho/regras de repositório com as verificações de
  CI obrigatórias; esta integração não tem permissão de leitura/edição dessa
  configuração.

### Checagem local sem administrador — PowerShell

Na raiz do checkout correspondente ao SHA que será distribuído:

```powershell
git rev-parse HEAD
go test -count=1 ./internal/runtime -run '^(TestTerminalInputScrubber|TestTerminalOutputScrubber|TestTerminalInputEscape)'
if ($LASTEXITCODE -ne 0) { throw 'Regressão de entrada Codex' }
go test -count=1 ./internal/control/terminal -run '^(TestKeyboardModeReset|TestResetKeyboardModes|TestTTYController)'
if ($LASTEXITCODE -ne 0) { throw 'Falha no reset de modos do terminal' }
```

Exige Go instalado no perfil do usuário, mas não exige privilégios de
administrador. Se o Go não estiver instalado, usar binário/resultado
produzido pelo CI para o mesmo commit; **não** declarar aprovação sem
essa evidência. Executar testes não substitui o smoke de mouse real.

### Critério reproduzível para o menu do Codex

Em Windows e Linux, com o **binário compilado no SHA validado**, executar
ao menos 30 ciclos de abertura de menu de escolha em uma sessão descartável.
Em cada ciclo: movimentar roda de mouse para cima/baixo, setas rápidas,
alternar foco e confirmar uma opção apenas uma vez. Critério: zero
cancelamentos não solicitados, zero `[A`/`[B`/`[<` residuais, nenhuma
confirmação duplicada. Repetir no terminal Web/Desktop quando aplicável.
Guardar SHA, versão do Codex, backend PTY (ConPTY/Unix PTY), sistema,
resultado e log mascarado em evidência privada de homologação.

## Smoke interativo obrigatório (Windows e Linux)

Usar uma sessão de teste com Codex autenticado, sem interromper trabalho real:

1. Iniciar `nexus codex` e solicitar um planejamento que exija escolher uma
   opção em menu (não usar prompt existente de valor).
2. Com o menu aberto, rolar a roda para cima e para baixo várias vezes; a
   conversa e o menu não devem ser cancelados nem receber caracteres literais.
3. Navegar pelas opções com **setas reais** do teclado, incluindo sequências
   repetidas e rápidas. Pressionar Enter uma vez: exatamente uma opção deve
   ser confirmada; não devem aparecer `[A`, `[B`, `[<...`, `ESC[M`.
4. Abrir outro prompt de escolha. Usar PgUp/PgDn/scrollback e retomar a
   seleção. Confirmar que não existe cancelamento espontâneo.
5. Repetir com janela redimensionada, alternância de foco, sessão reconectada
   e opções em Web/desktop, sem duplicar o prompt ou submeter Enter extra.
6. Pressionar Escape e Ctrl+C **intencionalmente** em sessão descartável;
   verificar que continuam funcionando e não são filtrados indevidamente.
7. Validar Codex em ConPTY do Windows (sem fallback silencioso para pipes) e
   Linux PTY com `nexus doctor --json`; registrar mecanismo e versão.

Para diagnóstico sem conteúdo de prompts, ativar antes do Nexus:

PowerShell:
```powershell
$env:NEXUS_TERMINAL_TRACE = '1'
nexus codex
```

Bash:
```bash
NEXUS_TERMINAL_TRACE=1 nexus codex
```

Os traces ficam no data dir Nexus em `terminal-trace/<runtime_id>.jsonl`,
limitados a 8 MiB, com texto comum mascarado; manter os arquivos privados
porque podem conter metadados de sessão. Investigar sequências nos stages
`cli.stdin`, `cli.input` e `host.scrubber.output`. Em Web, observar
`web.xterm.ondata` e `web.xterm.filtered` se habilitados.

## Critério de fechamento

**NO-GO** enquanto não houver PASS automated + smoke Windows/Linux
no SHA exato. Não fazer novo bump público apenas pelo merge deste patch.
