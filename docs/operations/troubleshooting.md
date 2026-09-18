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
