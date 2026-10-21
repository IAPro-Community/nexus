# Troubleshooting

Comece sempre por `nexus doctor` (ou `./nexus doctor` quando estiver no
repositório) e use `--json` ou `--bundle` quando precisar anexar um diagnóstico
sanitizado. Não publique tokens da URL de bootstrap, cookies, API keys ou paths
pessoais.

## A Web não abre ou pede autenticação

**Causa provável:** a URL de bootstrap expirou, foi aberta em outro host, ou o
browser não preservou a sessão local.

**Solução:** execute `nexus web` novamente e abra a URL exibida no mesmo host.
Por padrão o servidor usa loopback; não copie o token da URL para tickets.

**Como validar:** a página deve carregar projetos e a sessão deve aparecer em
`GET /api/v1/session` apenas para o browser autenticado.

## O provider não está disponível ou não autentica

**Causa provável:** a CLI oficial não está instalada, o perfil não concluiu o
login, ou sua quota/saúde impede a seleção.

**Solução:** use `nexus providers status --json`, depois
`nexus providers register <provider> --profile <nome>` ou
`nexus login <provider> <nome>`. Consulte [providers e quota](../product/providers-and-usage.md).

**Como validar:** `nexus profiles --json` deve mostrar o perfil esperado sem
tratar `UNKNOWN` como capacidade disponível.

## Um terminal supervisionado não conecta

**Causa provável:** o runtime terminou, o shell não está acessível na
plataforma ou a sessão precisa ser recuperada.

**Solução:** liste com `nexus ps --json`, use `nexus attach <runtime-id>` se
ele ainda existir e execute `nexus doctor` antes de recriar a sessão. Preserve
somente a parte sanitizada da saída para análise.

**Como validar:** o runtime aparece ativo e o terminal aceita um comando simples
no projeto correto.

## O teclado ficou com setas estranhas ou Ctrl+C não recuperou a sessão

**Causa provável:** o terminal recebeu modos privados de teclado de uma TUI
interrompida, ou o runtime persistido pertence a uma geração antiga do Nexus.

**Solução:** execute `nexus doctor terminal` para ver os modos, a geração e a
compatibilidade do runtime. Para emitir o reset seguro imediatamente, use
`nexus doctor terminal --recover`; depois anexe novamente. Se o diagnóstico
marcar a sessão como incompatível, recrie-a com `nexus codex --yolo`. O Nexus
não encerra automaticamente o processo antigo.

**Como validar:** as setas, `Ctrl+C`, colagem e texto UTF-8 devem chegar ao
provider sem relatórios de foco/CPR; `nexus doctor terminal --json` deve mostrar
o runtime como `current-and-compatible` antes do attach.


### Se ainda aparecer `[A`, `[C`, `OA`, `[<...M` ou outro lixo no prompt

Ative temporariamente `NEXUS_TERMINAL_TRACE=1`, recrie o runtime e reproduza
o problema. O Nexus grava um JSONL por runtime em
`<NEXUS_DATA_DIR>/terminal-trace/<runtime-id>.jsonl`.

O trace não persiste texto bruto: conteúdo imprimível comum é mascarado, dígitos
viram `#` e apenas a estrutura necessária para diagnosticar protocolos de
terminal permanece visível, por exemplo `<1b>[A`, `<03>` e `<0d>`.

Estágios úteis incluem:

- `web.xterm.ondata`: bytes produzidos pelo xterm antes do filtro Web;
- `web.xterm.filtered`: saída do filtro Web;
- `web.websocket.input`: payload recebido pelo servidor;
- `web.scrubber.output`: saída do scrubber Web;
- `cli.stdin` e `cli.scrubber.output`: caminho equivalente do attach CLI;
- `host.raw_attach.input`: fragmentos recebidos no stream bruto do SessionHost;
- `host.cmd_input`: frames de input RPC;
- `host.scrubber.output`: bytes efetivamente liberados pela última barreira
  antes do PTY do provider.

Compare os estágios na mesma janela de tempo. Se `host.raw_attach.input` mostrar
`<1b>`, `[`, `A` separados mas `host.scrubber.output` mostrar
`<1b>[A`, a fragmentação foi recuperada corretamente. Se o último estágio
também estiver fragmentado ou faltar `<1b>`, o ponto exato da perda fica
identificado.

O arquivo tem limite de **8 MiB por runtime**; assinaturas inválidas enviadas pelo
navegador são descartadas antes da persistência. Ao atingir o limite, novas
amostras deixam de ser registradas. Remova o JSONL antigo após coletar a
evidência necessária e reinicie a sessão para uma nova captura.

Desative a variável após a coleta; esse modo é somente diagnóstico.

## Desktop não inicia

**Causa provável:** o build nativo ou a dependência da plataforma não foi
validado naquele sistema.

**Solução:** teste primeiro `nexus web`, depois consulte a [matriz de
plataformas](platform-support.md) e o guia [Desktop](../desktop/overview.md).

**Como validar:** trate Web e Desktop como superfícies distintas; sucesso no
browser não certifica o shell nativo.

## A execução planejada precisa de intervenção

**Causa provável:** uma Mission encontrou uma dependência, gate ou permissão que
não pode ser resolvida de forma autônoma.

**Solução:** abra a Mission/Flow, leia a solicitação de atenção e retome apenas
depois de fornecer a ação necessária. Consulte [Missions](../product/missions.md)
e [atenção humana](../product/interaction-model.md).

Ao abrir um issue, informe sistema operacional, versão (`nexus version --json`),
comando, trecho de erro sanitizado e passos de reprodução.


## A Mission bloqueou uma coordenação por replay não comprovado

**Sintoma:** a Mission entra em `BLOCKED_NEEDS_USER` com motivo como
`COORDINATION_REPLAY_UNPROVEN` ou informa que não é seguro repetir a package.

**Por que acontece:** um Agent pediu informação/delegação depois de um provider
turn, mas o Nexus não conseguiu provar simultaneamente que o workspace permaneceu
inalterado e que o contrato proibia side effects externos relevantes.

**O que fazer:**

1. abra a intervenção da Mission;
2. revise alterações do workspace e Git HEAD;
3. confirme se houve qualquer operação externa (push, deploy, rede, serviço pago
   ou acesso a secret);
4. não use “retry” cegamente;
5. escolha uma opção de retomada explícita somente depois de reconciliar o
   estado real.

**Como validar:** depois da resolução, a Mission deve continuar por uma transição
persistida; não deve aparecer uma segunda execução silenciosa da mesma operação
externa.

Veja [Coordenação entre Agents](../product/agent-coordination.md) e
[Guardrails](../architecture/guardrails.md).

## Uma child package de coordenação não apareceu ou não concluiu

**Causa provável:** o pedido foi resolvido por memória/artifact existente,
falhou na validação, excedeu limites de fan-out/payload, ou a child está
aguardando dependências/recursos.

**Solução:** abra o Flow Run e verifique request, lineage, status da child,
resource allocation e WorkReceipt. Um pedido resolvido localmente pode não gerar
novo nó de trabalho, por design.

**Como validar:** requests que exigem trabalho devem ter `ChildPackageID`;
requests concluídas por child devem apontar para o `ResultReceiptID`.

## Quota desconhecida ou conta errada

Verifique `nexus providers status --json`, `nexus profiles --json` e `nexus usage --json`. Compare escopo de conta, fonte, janela e timestamp. `UNKNOWN` não equivale a limite esgotado ou capacidade plena; `UNSUPPORTED` também não significa falha do provider. Veja [providers e quota](../product/providers-and-usage.md).

## HUD mostra custo ou quota antiga

Confira timestamp, `LIVE/CACHED/ESTIMATED`, origem do custo, monitor e conexão com o Core. O HUD não executa novos probes nem inventa custo. Veja [HUD](../product/hud.md).

## HUD nativo não arrasta ou abre no tamanho errado

Arraste pela alça de grip à esquerda da cápsula, não pelo corpo: o corpo é o botão que abre e fecha o resumo. A alça também aceita setas (16 px; `Shift` + seta = 64 px). Para mudar o tamanho, abra o popover e use **Compacto**, **Confortável** ou **Grande**; o preset é persistido e preserva a âncora inferior direita. Se a janela abrir em 190 × 48, o primeiro show não aplicou a geometria persistida; reinicie o HUD e verifique `hud-window-state.json`. Se o arraste continuar falhando, confirme que o binário é o HUD novo e não uma instância antiga com outro ID de instância.

## Túnel ou convite remoto inválido

Gerar convite novo via **Settings → Acesso remoto**, validar HTTPS e os limites da sessão. Não desabilitar `Secure` dos cookies para contornar HTTP. Veja [acesso remoto](../product/remote-access.md).

## Capturas e documentação

Se `docs-verify` apontar SHA visual antigo, a captura é `STALE`, não PASS; siga [recaptura](visual-evidence.md).
