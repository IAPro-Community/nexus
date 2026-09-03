# Handoff

## Canonical hardening branch handoff — 2026-09-02

The active isolated branch is feat/nexus-canonical-hardening and changes are
intentionally uncommitted. Validation reports document the implemented
attention/arrangement foundation and remaining NO-GO scope. Do not describe
global recovery, VAPID persistence, provider discovery/resume/fork, or the
complete Golden Journey as shipped until implemented and revalidated.

## Current state

Nexus was rebuilt from the current source and installed at `/home/desenvolvedor/.local/bin/nexus`; the `ai` alias points to the same binary. Both report version `0.5.0-beta.10`.

## Verification

Fresh `zsh` verification and focused Nexus Go tests passed. No `nexus web` process is currently running.

`nexus usage` foi recompilado para mostrar a capacidade completa por modelo e
os resets de cada janela (5h/semanal), sem reduzir a saída ao bottleneck.

Sessões Web rotacionam automaticamente antes do vencimento. Se o servidor for
reiniciado ou a sessão ficar ociosa além do limite, a interface explica como
abrir o novo Bootstrap exibido por `nexus web`.

O uso de recursos é apresentado em tabela no CLI e em grid responsivo no
dashboard, com capacidade/reset por grupo e status sem sobreposição visual.
Para localizar contas e grupos no terminal, use `nexus usage` e
pressione `/` para filtrar.

O banner de atenção do agente também foi corrigido: ele não ocupa mais a área
central do dashboard e a mensagem padrão `? for shortcuts` não abre aviso
falso. O binário instalado contém esses ajustes.

Conclusões de tarefa e erros usam agora um componente de notificações
transitórias próprio; perguntas e aprovações continuam visíveis no banner por
exigirem resposta.

## Next action

Run `nexus usage` para conferir as janelas por modelo; depois `nexus web` para
iniciar o local Web Workspace OS e abrir a URL Bootstrap de uso único.
