# Troubleshooting

1. Execute `./nexus doctor` e preserve a saída sem tokens ou credenciais.
2. Confirme que o provider escolhido está instalado e autenticado localmente.
3. Para a Web, use a URL de bootstrap exibida por `./nexus web` e não exponha o
   token da URL.
4. Para problemas de terminal, confirme permissões e o shell da plataforma.
5. Para Desktop, reproduza no build nativo e consulte a matriz de plataformas.

Ao abrir um issue, informe sistema operacional, versão, comando, trecho de
erro sanitizado e passos de reprodução. Não inclua cookies, tokens, API keys,
paths pessoais ou repositórios privados.
