# Correções e melhorias aplicadas

## FIX-001 — redimensionamento do terminal PTY

**Área:** Terminal / workspace  
**Comportamento esperado:** a grade xterm e o PTY devem acompanhar mudanças de largura do painel mesmo sem resize da janela principal.  
**Reprodução antes da correção:** abrir terminal no Desktop, executar stty size, arrastar o separador do painel e consultar stty size de novo.  
**Resultado antes:** o PTY permaneceu em 21 linhas × 79 colunas após o painel crescer.  
**Causa:** TerminalPane observava apenas o evento window.resize; o resize do painel não gera esse evento.  
**Correção:** ResizeObserver no container, ajuste coalescido em requestAnimationFrame e cancelamento/desconexão no unmount.  
**Regressão:** dois testes em terminalPaneResize.test.ts passaram. Teste nativo observado: 21×79 → 25×91 após arrastar o painel.  
**Evidência:** screenshots/native-terminal-resize-before-fix-xwayland.png e screenshots/native-terminal-resize-after-fix-xwayland.png.  
**Severidade:** P2, corrigido no overlay local.  
**Commit:** nenhum; a alteração permanece não commitada na árvore local.

## FIX-002 — direção de produto: mascote NEX único

**Área:** HUD / produto  
**Comportamento esperado:** mostrar um mascote oficial NEX bem acabado e funcional; Barra e Bandeja são apresentações do HUD, sem personagem Companion 2D separado.  
**Auditoria da release:** `NexusHud` usa uma apresentação de `GuardianMascot`, que renderiza `NexPixelSprite`. `GuardianModeSelector` contém Barra, Mascote e Bandeja. Não há `NexusGuardian.tsx`, atlas de Companion nem modo visual 2D separado na branch testada. `companionModel` projeta estados/atenção para o mesmo NEX. PR #55 não foi mesclada.  
**Correção:** removidas referências de produto que apresentavam outro mascote/modo; docs atuais descrevem a identidade NEX única.  
**Regressão:** `GuardianMascot.test.tsx` exige exatamente uma arte NEX por renderização; `GuardianModeSelector.test.tsx` proíbe Companion 2D no seletor. Testes focados e frontend quality completo passaram.  
**Evidência:** `screenshots/native-hud-exact-final-xwayland.png`; `web/src/features/hud/GuardianMascot.test.tsx`; `web/src/features/hud/GuardianModeSelector.test.tsx`; `COMPANION_VALIDATION.md`.  
**Severidade:** P1 de direção de produto, resolvido no caminho ativo; runtime real de atenção ainda BLOCKED.  
**Commit:** nenhum; as alterações permanecem locais, sem commit/push.

## Limite de integração

Não foram aplicadas correções adicionais ao updater ou à execução de agentes porque esses fluxos não tinham uma reprodução real com provedor autenticado. Não foi feito merge de PR nem alteração de tag.

## FIX-003 — argumento de perfil duplicado no comando de login

**Área:** Contas / login de provedor  
**Comportamento esperado:** para o perfil padrão cujo nome é igual ao provedor, mostrar `nexus login <provider>` uma única vez.  
**Resultado observado:** o modal de conta exibia `nexus login codex codex`; o painel de recursos também montava o comando com os dois argumentos repetidos.  
**Causa:** havia dois formatadores independentes e ambos sempre acrescentavam o nome do perfil.  
**Correção:** um formatador comum em `web/src/lib/providerLoginCommand.ts` agora omite o perfil quando ele repete o provedor. Os dois fluxos da interface usam o mesmo helper; perfis nomeados continuam explícitos.  
**Regressão:** quatro testes de unidade e wiring passaram. O gate frontend completo passou com 103 arquivos e 633 testes; o bundle Desktop/HUD foi recompilado.  
**Evidência:** `logs/frontend-quality-final-login-fix.log` e `logs/build-native-login-fix-final.log`.  
**Severidade:** P2, corrigido no overlay local.  
**Commit:** nenhum; a alteração permanece não commitada na árvore local.

## FIX-004 — importação da sessão Gemini sem campo `email`

**Área:** Provedores / contas / OAuth Gemini  
**Comportamento esperado:** reconhecer e importar a sessão OAuth válida do Gemini CLI, mesmo quando a identidade não vem como campo `email` separado no arquivo de tokens.  
**Reprodução antes da correção:** o arquivo instalado `~/.gemini/oauth_creds.json` continha `access_token`, `refresh_token`, `expiry_date`, `id_token`, `scope` e `token_type`, mas sem `email`; o serviço retornava `NO_SESSION`.  
**Causa:** o validador e `InspectAuth` exigiam `email` no arquivo, que não faz parte do formato de credenciais observado.  
**Correção:** autenticação passa a ser determinada pela presença de token OAuth, validade temporal e marcador de revogação. E-mail permanece metadado opcional, resolvido dos dados de conta já permitidos; token expirado com refresh é reconhecido como autenticado degradado e token expirado sem refresh é rejeitado. A cópia continua atômica, sem sobrescrita e com permissão `0600`.  
**Regressão:** formato Gemini sem email, expiração, falta de refresh, JSON inválido e token revogado cobertos em `gemini_auth_test.go`; importação isolada e serviço real `ImportHostSessions` passaram com a sessão instalada. Os logs não exibem material OAuth.  
**Evidência:** `logs/gemini-live-host-import-after-fix.log` e `logs/live-host-import-service-after-fix.log`.  
**Severidade:** P1, corrigido no overlay local.  
**Commit:** nenhum; a alteração permanece não commitada na árvore local.

## FIX-005 — registros de descoberta isolados por instância

**Área:** Core / descoberta do `nexus web`
**Comportamento esperado:** encerrar um Core não deve apagar a descoberta de outra instância saudável no mesmo perfil.
**Reprodução antes da correção:** iniciar dois Cores com o mesmo `NEXUS_DATA_DIR` em portas diferentes; encerrar a segunda instância; consultar a descoberta e a saúde da primeira.
**Resultado antes:** as instâncias compartilhavam `web-auth/listen.json`; o segundo processo sobrescrevia o primeiro e removia o registro ao encerrar, mesmo com o primeiro HTTP 200.
**Correção:** cada PID agora grava `listen-<pid>.json`; leitores também aceitam o arquivo legado, preferem registros vivos e saudáveis e ignoram processo morto. Encerramento remove apenas o estado próprio.
**Regressão:** `TestReadListenStateFallsBackToAnotherLiveServerAfterClose` e `TestReadListenStateSupportsLegacySingleFile`; reteste real com portas 13241/13242, encerramento fora de ordem e health HTTP 200 aprovado.
**Evidência:** `logs/core-multi-instance-state-final-build.log`; `logs/make-quality-full-after-resilience-fixes-retry.log`.
**Severidade:** P1, corrigido no overlay local.
**Commit:** nenhum.

## FIX-006 — shutdown HTTP gracioso retorna sucesso

**Área:** Core / encerramento de servidor
**Comportamento esperado:** `http.Server.Close` provocado por shutdown normal não encerra a CLI com exit 1.
**Resultado antes:** `Server.Start()` propagava `http: Server closed` como erro fatal durante SIGTERM.
**Correção:** `Server.Start()` trata `http.ErrServerClosed` como encerramento normal e retorna `nil`.
**Regressão:** `TestServerStartReturnsNilAfterGracefulShutdown`; dois processos finais receberam SIGTERM e retornaram exit 0; testes/race da suíte completa passaram.
**Evidência:** `logs/core-multi-instance-state-final-build.log`; `logs/make-quality-full-after-resilience-fixes-retry.log`.
**Severidade:** P2, corrigido no overlay local.
**Commit:** nenhum.
