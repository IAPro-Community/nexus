## Branch de destino (Nexus 1.0)

- [ ] O campo **base** deste PR é `release/v1.0.0` (não `main`). Se não for, explique o motivo e solicite retarget antes da revisão.
- [ ] Verifiquei o código atual da release para não reintroduzir uma implementação antiga ou duplicada.

## Contexto e problema

<!-- Relate issue/proposta, problema real e por que a alteração é necessária. -->

## Direitos e licenciamento

- [ ] Confirmei que tenho direito de enviar todos os trechos, assets e dependências propostos.
- [ ] Li a licença vigente e o plano de migração: **a PolyForm Strict atual não dá autorização geral para contribuições de patches**. Até o relicenciamento, só enviar código com permissão jurídica específica documentada.
- [ ] Identifiquei partes de terceiros, respectivos autores, termos e avisos quando aplicável.
- [ ] Para patch de terceiro antes do relicenciamento, a autorização específica e documentada para incorporação e relicenciamento futuro foi verificada pelo mantenedor (ou o PR ficará bloqueado para merge).
- [ ] Quando o DCO da fase open source estiver ativado, os commits terão `Signed-off-by` (use `git commit -s`).

## Qualidade e arquitetura

- [ ] Respeitei as responsabilidades Composer / WorkPlan / Flow / MissionRunner e a independência do Maestro.
- [ ] Não criei uma segunda infraestrutura de memória, orquestração, runtime ou editor sem ADR.
- [ ] Adicionei/atualizei testes de regressão e documentação.
- [ ] Se a alteração adiciona recurso visível, atualizei o [catálogo de novidades](../docs/product/feature-discovery.md) (`id`, `revision`, rota, alvo, explicação PT/EN/ES e tour contextual) ou justifiquei sua inaplicabilidade. Não anunciei funcionalidades que ainda não existem.
- [ ] Reutilizei componentes do design system e investiguei bibliotecas open source antes de criar um novo overlay, tour, tooltip ou mecanismo de alertas.
- [ ] Verifiquei compatibilidade, i18n, acessibilidade e Web/Desktop/CLI quando aplicável.
- [ ] Rodei gates relevantes e anexei comandos/resultados do SHA testado.
- [ ] Não incluí segredos, dados pessoais, arquivos de usuário nem material de análise de segurança privada.
- [ ] Não declarei release funcional com base somente em alterações de código.

## Evidências e riscos

<!-- SHA, sistema operacional, comandos, resultados, limitações/skips, screenshots quando úteis. -->

## Documentos relacionados

- [CONTRIBUTING](../CONTRIBUTING.md)
- [GOVERNANCE](../GOVERNANCE.md)
- [Licenciamento e gate](../docs/legal/OPEN_SOURCE_TRANSITION.md)
