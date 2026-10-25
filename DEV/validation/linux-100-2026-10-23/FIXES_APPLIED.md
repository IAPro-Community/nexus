# Correções locais exercitadas durante a homologação

Nenhum commit ou push foi criado. As alterações abaixo já estavam no worktree local e foram preservadas; a compilação e os testes incluem o conjunto local sobre o SHA remoto bf6adf0c80ac3531d7291844512a14a017ab0002.

## Correção aplicada nesta campanha

- **Makefile / segurança:** o fallback de `make security` passava `GOTOOLCHAIN=auto` e, sem `govulncheck` instalado, escolhia Go 1.25.14 apesar de o projeto fixar Go 1.26.10. A menor mudança passou a fixar o fallback em `GOTOOLCHAIN=go1.26.10`. Regressão: `make security` passou e o gate `make quality-full` foi repetido nessa toolchain. Diff permanece local e sem commit.

## Autenticação e seleção de provedor

- Importação de sessão existente do host para um perfil temporário do Nexus foi exercitada pela interface nativa em ambiente isolado. O perfil Codex criado ficou autenticado, auth.json foi gravado com modo 0600 e o arquivo de origem não mudou. Dados temporários foram removidos após o teste. Evidência: logs da sessão de importação descritos no relatório principal.
- Normalização do comando de login evita repetir o provedor como nome de perfil padrão. A UI nativa e o clipboard retornaram exatamente “nexus login codex”, sem duplicar codex. Captura: screenshots/native-login-command-codex-xwayland-bf6adf0c.png. Regressão: providerLoginCommand.test.ts, incluída nos 612 testes frontend aprovados.
- A mensagem de sucesso/resultado da importação não apareceu de forma visível na janela capturada, embora o perfil temporário esteja autenticado. Registrar como defeito P2; a validação da autenticação não aprova o feedback da interface.

## Layout

- Corrigido o espaçamento entre título e descrição da ação de adicionar repositório no ProjectHub. A tela real mostra a descrição em linha separada. Regressão frontend coberta por firstRunReadiness.test.ts e pelo conjunto de testes.

## Outros ajustes locais presentes

O worktree também contém ajustes em adaptadores de provedores, isolamento, handlers/control plane, HUD e preferências de primeira execução. Os testes Go e frontend executados cobrem o build local inteiro; não foi feita atribuição de commit individual porque essas alterações não foram commitadas nesta homologação.

## Limite

A existência e renderização do GuardianMascot não integram o Nexus Companion 2D da PR #55. Nenhuma correção local substituiu ou simulou essa integração. O gate permanece NO-GO — COMPANION_2D_NOT_INTEGRATED.
