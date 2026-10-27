# Forks, colaboração e melhorias upstream

**Importante:** a licença atual na branch é a [PolyForm Strict 1.0.0](../../LICENSE). No estado atual, o repositório público não concede permissão geral para modificar, distribuir ou publicar derivados. O fluxo abaixo se tornará aplicável a contribuições de código após a aprovação e publicação da migração descrita em [OPEN_SOURCE_TRANSITION](../legal/OPEN_SOURCE_TRANSITION.md). Até lá, questões e propostas de melhoria são bem-vindas em issues/discussões; não presumir autorização geral para enviar patches.

## Como queremos trabalhar quando a licença abrir

Forks, aplicações derivadas e serviços comerciais serão permitidos pelos termos da licença adotada. **Nenhum fork será obrigado a contribuir de volta por uma política do projeto.** O objetivo é tornar o upstream atraente com integração ágil, documentação e manutenção confiável.

Para contribuir:
1. Leia [CONTRIBUTING](../../CONTRIBUTING.md) e a política de segurança.
2. Confira issues existentes ou abra uma proposta antes de alterações extensas.
3. Use branch com escopo curto e PR contra a branch anunciada pelos mantenedores; não faça push diretamente em releases.
4. Escreva testes, atualize documentação e inclua evidências reproduzíveis, sem segredos.
5. Após adoção pública de DCO, use `git commit -s` em cada commit; isso certifica o direito de enviar a contribuição e **não transfere automaticamente titularidade**.
6. Mantenha seu fork sincronizado com o upstream. Faça referência aos issues/PRs relacionados.

Depois da abertura, um exemplo genérico de sincronização será:

```bash
git remote add upstream https://github.com/IAPro-Community/nexus.git
git fetch upstream
git switch sua-branch
git rebase upstream/main # use o alvo anunciado em CONTRIBUTING
```

Nunca force-push em branches de outros mantenedores sem acordo. Prefira integrações modulares e propostas que respeitem Composer, Flow, MissionRunner e independência do Maestro. Os mantenedores explicam rejeições e acolhem abordagens alternativas; merge é uma decisão técnica, não uma obrigação.

## Para quem mantém forks

Divulgue claramente que é uma variante não oficial, documente mudanças e vulnerabilidades específicas, mantenha dependências atualizadas e não reutilize branding oficial de modo que provoque confusão. Respeite as licenças de terceiros e a [política de identidade](../legal/TRADEMARK_POLICY.md).

Para pedidos e dúvidas: [issues](https://github.com/IAPro-Community/nexus/issues), [SUPPORT](../../SUPPORT.md), [GOVERNANCE](../../GOVERNANCE.md).
