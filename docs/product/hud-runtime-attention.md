# Nexus Guardian — intervenções de terminais e agentes

**Status:** código integrado na `release/v1.0.0`; testes automatizados adicionados. A certificação GUI nativa em Windows, Linux e macOS ainda depende das sessões reais do mesmo SHA.

## Contrato funcional

O Guardian deve chamar atenção mesmo com a janela principal do Nexus minimizada ou a cápsula configurada para **Somente bandeja**, desde que `nexus-hud` e o Core estejam em execução:

1. CLI/Agent/Mission entra em uma espera real por confirmação, escolha, revisão ou texto;
2. O SessionHost/provider adapter registra `needs_user` no **Runtime Registry** (ou a Mission registra `BLOCKED_NEEDS_USER` no estado canônico);
3. A projeção `GET /api/v1/hud/state` inclui `runtime_attention`, sem depender do título do terminal;
4. O Mascote/BARRA indica a contagem, e o popup lista o provedor, projeto e tipo de intervenção, com **Abrir terminal**;
5. O monitor **nativo Go** (`cmd/nexus-hud/attention_monitor.go`) verifica o mesmo Registry em intervalos de 5 segundos, inclusive quando o WebView está oculto, e envia **um aviso do sistema por transição**;
6. Quando o usuário envia uma resposta ao PTY supervisionado, a espera anterior é removida. Se surgir outra pergunta, ela volta a ser exibida.

Não é permitido autoaprovar comandos, responder perguntas por heurística, expor prompts privados em toasts, criar polling adicional dos provedores ou duplicar o MissionRunner.

## Como as fontes são detectadas

Ordem de confiança:

- **1. Eventos estruturados** do provedor, quando há API/hook comprovado: aprovação, pergunta, intervenção. Não varrer stdout se esses eventos forem autoritativos.
- **2. PTY supervisionado** pelo Nexus: reconhecer controles explícitos (`[y/N]`, seletores, `ask_question`, `Question N of M`, `Select all that apply`, `Press Enter to select`), capturando o estado real da sessão. Alternância de tela, ANSI e quebras de linha justificam uma janela de contexto um pouco maior apenas para formulários.
- **3. Título OSC/statusline** do provedor: sinal auxiliar opcional, nunca pré-requisito para alertas e nunca evidência suficiente por si só.
- **4. Adaptador específico** para CLI cuja interface não emite pergunta observável no PTY. Neste caso é indispensável uma API/hook suportado pelo CLI ou outra fonte explícita: silêncio do terminal não permite deduzir seguramente que existe uma pergunta.

### AGY / Antigravity CLI

A CLI do Antigravity usa telas `ask_question` e seleção por teclado. O detector foi ajustado para reconhecer formulários multicampo e o rodapé `Press Enter to select`, mesmo quando o título permanece estático.

A preferência `/title` do AGY pode tornar títulos mais úteis, mas **não é requisito**. Não usar `notifications=true` como substituto de aprovação: essa opção da CLI cobre notificações do próprio provedor, não é um contrato de eventos de perguntas verificável pelo Nexus.

**Limite explícito:** se o AGY renderiza um diálogo exclusivamente em uma superfície fora do PTY ou não emite nenhum evento, o parser não tem como observá-lo. O suporte pleno exigirá implementar/adaptar um canal de eventos do AGY, com fixtures reais e prova de origem por runtime.

## Privacidade, falsos positivos e estado

- `runtime_attention` não inclui o texto da pergunta, histórico de saída, argumentos de execução ou tokens.
- Só estados `WAITING/APPROVAL` com contexto explícito, espécie de pergunta válida e runtime vivo são considerados pendências.
- Não confundir `? for shortcuts`, logs explicativos ou perguntas em markdown com aprovações.
- Movimentação de setas não encerra uma pendência; submissão via Enter ou resposta Y/N confirmada pela entrada supervisionada remove o estado anterior.
- A notificação nativa é deduplicada por runtime e fingerprint, e não é emitida novamente a cada polling; uma nova espera após resolução volta a notificar.
- O monitor Go não depende de `document.visibilityState`, do browser ou de timers do React. O badge tem o mesmo conteúdo canônico, podendo atualizar com menor frequência que o aviso nativo.
- O toque em **Abrir terminal** abre o projeto/Agent relacionado na instância Nexus disponível.

## Homologação

Executar cenários AGY `ask_question`, seleção multietapa, formulário, prompt por teclado e Codex/Claude/OpenCode em um terminal, em um Agent persistente e com a janela principal minimizada. Repetir para Barra, Mascote e Bandeja em Windows/macOS/Linux.

Comprovar: transição novo prompt → badge e toast; setas → aviso permanece; Enter → pendência desaparece; próximo prompt → nova notificação; ausência de pergunta → ausência de aviso; clique → terminal correto; ausência de vazamento de dados; acesso via Core local ao vivo.

Nenhum teste automatizado de detector pode provar uma interface AGY que não forneça eventos nem bytes de terminal. Esse gap deve ser documentado por versão do AGY e plataforma, nunca marcado como PASS genérico.
