# Configuração administrativa do GitHub para abrir o Nexus Community

Estas etapas **não são configuráveis apenas por um commit de Markdown/YAML**. Precisam de alguém com permissão de administrador em `IAPro-Community/nexus`. O objetivo é receber propostas e, após o clearance da licença, contribuições de código, **sem expor runners self-hosted nem segredos de release**.

## 1. Regras de branches

No GitHub: **Repository → Settings → Rules → Rulesets → New branch ruleset**.

Aplique pelo menos às branches `main` e `release/v1.0.0` (ou use regras específicas se seus ritos diferirem). Recomendações:

- exigir PR antes de merge, pelo menos 1 aprovação qualificada, review CODEOWNERS e resolução de conversas;
- impedir force push, exclusão e bypass indiscriminado;
- exigir checks de PR da pipeline **Community PR Checks / Community source quality** e **Community PR Checks / Community diff hygiene**, **depois que esses checks tiverem rodado e o identificador estiver disponível**;
- não configurar como obrigatório um check exclusivo de `push` da CI self-hosted em PR: ele não será produzido para forks;
- proteger tags de distribuição e restringir quem pode acionar ambientes `beta-signing` e `native-signing`;
- revisar apps com permissão write, tokens legados, membros e capacidade de bypass. Escolher o GitHub App esperado como emissor de status quando houver suporte.

As regras devem ser testadas com PR de fork e PR de mantenedor. Não presumir que rulesets estejam ativos a partir desta documentação.

Referências oficiais:
- https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository
- https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets

## 2. GitHub Actions seguro para terceiros

- `.github/workflows/community-pr.yml`: eventos `pull_request`, runners hospedados efêmeros `ubuntu-24.04`, `contents: read`, checkout sem credenciais persistentes, sem secrets de assinatura.
- `.github/workflows/ci.yml`: **push/dispatch**, infraestrutura self-hosted. Não reintroduzir `pull_request` sem isolamento rigoroso.
- `.github/workflows/release.yml`: somente publicação com gates exatos, aprovações de ambiente e material de assinatura restrito; não expor chaves em PRs ou issues.
- Verificar que nenhum job com `pull_request_target` faça checkout/execução de código de fork. Usar revisão cuidadosa de Actions externas e, quando viável, pin por SHA.

Se o runner self-hosted atualmente tiver sido exposto a PRs de forks, revisar o histórico de execuções, credenciais locais e isolamento do host antes de ativar contribuições externas em escala.

## 3. Reporte privado de vulnerabilidades

No repositório: **Settings → Security and quality → Advanced Security → Private vulnerability reporting → Enable** (menus podem mudar).

Depois verificar em **Security → Advisories** a presença de **Report a vulnerability**. Atualizar `SECURITY.md` se a organização disponibilizar contato privado formal. Não pedir reprodução de vulnerabilidades em issue pública.

Referência: https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository

## 4. Direitos e moderação

- Confirmar os titulares antes do commit de Apache-2.0 conforme [plano legal](../legal/OPEN_SOURCE_TRANSITION.md). O admin GitHub não é automaticamente o titular de todo o source.
- Preparar DCO após a licença aberta e determinar quem revisa PRs de terceiros. Não ativar bloqueios que exijam um bot inexistente.
- Manter `CODEOWNERS`, `CODE_OF_CONDUCT.md` e templates de issues/PR atualizados; criar labels `good first issue` e `help wanted` somente quando houver tarefas reais.

## 5. Patrocínios e transparência

Quando a conta real de GitHub Sponsors estiver habilitada para recebimento e revisada pelo titular, adicionar `.github/FUNDING.yml` com o identificador verificado. Até lá, **nenhum link de pagamento é anunciado como ativo**. Não condicionar funcionalidades Community à contribuição financeira.

## 6. Critérios para assinar a abertura

Registre no histórico da decisão: titular autorizado, inventário de terceiros/assets, auditoria privada de segredos, commit de licença, SHA com CI e links dos rulesets/configurações habilitadas, confirmação de teste de PR de fork e canal de vulnerabilidades. Somente após isso chamar a distribuição de "Nexus open source Apache-2.0".

Consulte também [SECURITY](../../SECURITY.md), [PUBLICATION_CHECKLIST](../PUBLICATION_CHECKLIST.md), [GOVERNANCE](../../GOVERNANCE.md) e [guia de revisão](REVIEW_GUIDE.md).

## Implementação preliminar nas duas branches (27/10/2026)

A separação de CI foi aplicada **também à branch padrão `main`**, que até então
continuava executando `pull_request` no CI self-hosted. Atualmente tanto
`main` quanto `release/v1.0.0` possuem a pipeline isolada `community-pr.yml`,
com `ci.yml` reservado a push/dispatch e o teste de regressão
`internal/release/community_workflow_test.go`.

O fechamento **administrativo** dessa fronteira está acompanhado em
[issue #96](https://github.com/IAPro-Community/nexus/issues/96).
Habilitar os rulesets, testar com fork real e confirmar required checks ainda
são tarefas manuais do administrador; não estão marcadas como concluídas.
O [reporte privado](https://github.com/IAPro-Community/nexus/issues/65)
continua dependente da habilitação no GitHub Settings.
