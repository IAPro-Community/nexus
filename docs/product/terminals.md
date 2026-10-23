# Terminal, PTY/ConPTY e sessões supervisionadas

O Nexus deve ser **transparente ao protocolo interativo do provider**. Em uma sessão Codex, Claude ou outra TUI, o provider é quem controla seletores, setas, Enter, mouse, bracketed paste e atalhos nativos. O Nexus controla ciclo de vida, sessão, attach, permissões e observabilidade — **não transforma teclas em prompts**.

## Uso rápido

```bash
nexus doctor
nexus providers status --json
nexus codex
```

Na interface Web/Desktop, use **Projects → + New → AI Session → Provider → Terminal**. Para uma sessão supervisionada, o Core possui identidade de runtime e pode oferecer attach/stop/recovery conforme o estado. Consulte `nexus --help` e a [CLI](../cli/README.md) para comandos disponíveis no build usado.

## Modos de execução

- **Direct** (`--direct`): inicia o provider fora do transporte supervisionado.
- **Supervised** (`--supervised`): Nexus supervisiona processo, terminal e runtime, sem assumir o teclado do provider.
- **STICKY_LEAD** em TTY supervisionado: mantém contexto/identidade de Lead quando possível, com delegação `OFF` por padrão.
- **ASK/AUTO**: delegação precisa de ativação explícita; nenhum evento do terminal equivale a autorizar uma Mission.
- **GO**: somente o fluxo formal de planejamento autoriza executar uma revisão específica.

Os aliases `nexus codex` e equivalentes devem continuar úteis mesmo se metadados opcionais de Lead falharem; a sessão nativa não deve ser encerrada apenas por essa falha. Veja [controle interativo](interactive-control.md).

## Entrada de teclado e mouse

```text
Teclado/mouse → xterm ou terminal nativo → transporte → PTY/ConPTY → provider
```

Setas/CSI/SS3 e sequências de mouse devem chegar intactas ao processo. O Nexus pode filtrar relatórios de foco/CPR que seu protocolo explicitamente identifica, sem reinterpretar seleção de menus ou texto digitado. **Scroll não pode cancelar uma etapa de planejamento.**

Antes de certificar uma versão, testar: setas, scroll, cliques, `Ctrl+C`, `Esc`, colagem Unicode, Enter, resize, reconexão, prompt de aprovação e menus nativos do Codex.

## Alterações do transporte Web/Windows e rolagem (22/10/2026)

Na implementação beta.10, o WebSocket do terminal envia bytes de entrada ao
stream anexado da sessão pela função `writeAttachedTerminalInput`, sem criar
uma moldura `CmdInput` a cada tecla. O `SessionHost` continua sendo a
fronteira de autorização de writer e normalização de PTY/ConPTY.

O payload de `Esc`, CSI/SS3, Unicode, Enter e `Ctrl+C` não deve ser
remontado ou confundido com JSON de controle. Foram adicionados testes para
preservação de bytes e escrita curta em
`internal/control/web/handler_terminal_input_test.go`. Esses testes **não
substituem** uma sessão real do Codex no Windows.

A interface Web distingue `data-nexus-platform="web"` no elemento raiz e
permite scroll nos painéis ativos quando o conteúdo ultrapassa o viewport.
A alteração fica em `web/src/app/workspace-styles/03-workspace-layout.css`;
o Desktop Wails mantém o layout nativo. Testes estáticos e uma verificação de
rolagem com viewport móvel/laptop foram adicionados em
`web/src/app/workspaceSurfaceStyles.test.ts` e
`web/scripts/e2e-surface-isolation.mjs`. É preciso executar o E2E e
verificar zoom/teclado/scroll reais antes de declarar correção homologada.

Em `STICKY_LEAD` com `OFF`, existe a exceção de um pedido de revisão
**explicitamente direcionado** a outro provider registrado, sem Mission.
Consulte [controle interativo](interactive-control.md) e
[roteamento](../architecture/agent-routing.md); a detecção da submissão é
uma fronteira sensível a regressões de TUI, ainda pendente de homologação.

## Recuperação segura

Se setas começarem a inserir `[A`, `OA` ou caracteres de mouse, siga o diagnóstico; não apague a sessão ou dados de perfil como primeira medida:

```bash
nexus doctor terminal
nexus doctor terminal --recover
nexus ps --json
```

Uma sessão de geração antiga pode precisar de novo runtime. A recuperação do terminal não prova que uma operação externa já foi concluída e não autoriza replay automático. Para investigação de bytes/protocolo, existe `NEXUS_TERMINAL_TRACE=1`; o trace deve ser inspecionado e sanitizado antes de compartilhamento.

## Plataformas e segurança

Linux/macOS usam mecanismos PTY Unix; Windows usa ConPTY/Named Pipes onde suportado. Build cross-platform **não certifica** o uso interativo nativo.

O shell e os agentes podem modificar arquivos com as permissões do usuário do sistema operacional. Worktree de Git e profile isolado **não são sandbox de segurança**. Não cole credenciais no terminal e não exponha sessões via túnel a pessoas sem confiança.

Veja [troubleshooting](../operations/troubleshooting.md), [segurança](../security.pt-BR.md) e [suporte de plataformas](../operations/platform-support.md).
