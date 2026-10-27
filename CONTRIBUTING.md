# Contribuindo para o IAPro Nexus

O Nexus quer se tornar uma **plataforma open source comunitária** para planejamento e operação com agentes de IA. **Neste momento, o texto jurídico vigente é [LICENSE](LICENSE) — PolyForm Strict 1.0.0, não OSI open source, sem autorização geral para publicar versões modificadas.** A [migração proposta para Apache-2.0](docs/legal/OPEN_SOURCE_TRANSITION.md) depende de validação da titularidade, segurança e licenças de terceiros.

Enquanto essa migração não for efetivada, ajude por **issues, sugestões de produto, reprodução de bugs, propostas de arquitetura e feedback de documentação**. Não tratamos a abertura de PR com código de terceiros como autorização presumida para incorporar esse código. Uma contribuição de patch precisa de autorização específica/documentada até a migração.

## Onde contribuir desde já

- Bugs: [bug report](https://github.com/IAPro-Community/nexus/issues/new/choose), sempre com versão/SHA, SO e passos de reprodução. Não publicar tokens nem logs privados.
- Melhorias: descreva o problema de usuário, alternativas, proposta e critério de aceite. Converse em issue antes de mudanças extensas.
- Documentação: aponte arquivos, trechos e correções propostas em issues; mantenedores com direitos confirmados podem aplicá-las.
- Segurança: siga [SECURITY](SECURITY.md); vulnerabilidades ainda não corrigidas não pertencem a issues públicas.
- Veja a [governança](GOVERNANCE.md), [guia de revisão](docs/community/REVIEW_GUIDE.md), [roadmap](ROADMAP.md) e [diretrizes de fork/upstream](docs/community/FORKS_AND_UPSTREAM.md).

## Fluxo de PR depois do relicenciamento

Após o anúncio verificável da licença open source e da branch-alvo de contribuição:

1. Confira o [CODE_OF_CONDUCT](CODE_OF_CONDUCT.md), issue/escopo e documentação técnica.
2. Crie branch de escopo único e PR pequeno; não envie conteúdo sem direitos claros nem arquivos gerados de outra aplicação sem licença compatível.
3. Acrescente testes, documentação, changelog quando necessário, notas de acessibilidade/i18n e evidências de execução.
4. Assine seus commits usando `git commit -s` (**DCO 1.1**) quando o mecanismo for ativado. O sign-off certifica sua autorização para contribuir sob a licença do projeto; **não transfere seus direitos patrimoniais automaticamente**. Consulte https://developercertificate.org/.
5. Declare dependências, modelos, assets, licenças e condições de redistribuição. Não copie código fechado nem documentos de terceiros sem autorização.
6. Respeite o modelo de autoridades do Nexus: Composer elabora; WorkPlan estrutura; Flow projeta; MissionRunner executa; Maestro permanece opcional e independente.
7. Aguarde revisão dos mantenedores. CI verde é necessária mas não garante merge ou release; decisão e motivos devem ser documentados.

**PRs não concedem prioridade comercial nem direito automático a merge.** Contribuições pertencem a seus autores, sujeitas à licença aplicável e instrumentos válidos que autorizarem seu uso.

## Verificação técnica sugerida

```bash
git status --short
go test ./...
cd web
bun install --frozen-lockfile
bun run quality:full
```

Testes e builds não substituem a homologação nativa de release por SHA, especialmente para terminais, agentes, desktop e atualizações.

## Sustentabilidade

O objetivo é manter a Community funcional sem assinaturas obrigatórias. [Financiamento](docs/community/FUNDING.md) é voluntário; patrocínio não influencia triagem de segurança nem decisões de revisão. A marca e os assets visuais têm direitos próprios: [Política de identidade](docs/legal/TRADEMARK_POLICY.md).
