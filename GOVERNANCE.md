# Governança do IAPro Nexus Community

**Estado:** maintainer-led, transição preparada para open source; **não** há conselho comunitário, fundação ou pessoa jurídica de manutenção formalmente anunciada. `LICENSE` ainda é PolyForm Strict. O roadmap jurídico está em [OPEN_SOURCE_TRANSITION](docs/legal/OPEN_SOURCE_TRANSITION.md).

## Valores operacionais

1. Código e documentação pública devem refletir o comportamento real, sem prometer certificação sem testes do SHA efetivo.
2. O núcleo Community deve continuar funcional, local, extensível e independente de assinaturas comerciais ou de um único provider/IA.
3. Contribuições e forks serão bem-vindos sob a futura licença open source, após o clearance jurídico; não presumir autorização atual para obras derivadas.
4. Segurança, integridade de credenciais, autorização humana e histórico de Missions têm precedência sobre conveniência e velocidade.
5. Maestro tem governo e memória próprios; Nexus permanece independente e consome seus contratos opcionalmente.

## Quem decide e como

- **Titular dos direitos**: exclusivamente quem puder provar os direitos necessários autoriza o relicenciamento, uso de marca e mudanças de termos. Uma org GitHub não demonstra titularidade por si.
- **Mantenedores do repositório**: fazem triage, revisão, merges, decisões de arquitetura e gates de release; o `.github/CODEOWNERS` identifica o revisor atual, **não** atribui direitos autorais ou autoridade unilateral para relicenciar conteúdos de terceiros.
- **Contribuidores**: participam via issues, documentação, propostas e, após a abertura jurídica, patches/PRs licenciados. Argumentos técnicos são analisados independentemente de patrocínios.
- **Conflitos**: registrar alternativas, risco e justificativa no PR/ADR; divergências sobre licença, autoria, privacidade e segurança devem ser resolvidas fora de discussões públicas quando exigirem dados confidenciais.

## Política de mudanças

- Mudanças simples: issue/PR com escopo claro, teste relevante e revisão de maintainer.
- Mudanças de arquitetura, modelo de dados, interfaces públicas, licenciamento, economia Community/Enterprise e API: RFC/ADR antes da implementação significativa, com período de comentários comunitários proporcional ao impacto.
- Mudanças em ações de agentes, execução remota, permissões, credenciais, isolamento, browser, pagamentos ou supply chain: segunda revisão de segurança quando houver mantenedor capacitado; na ausência, **bloquear release até obter revisão adequada**.
- Regras de proteção recomendadas para `main` e `release/v1.0.0`: sem force-push, aprovação CODEOWNERS, CI obrigatório, histórico visível, builds reproduzíveis e gates de release por plataforma. São **configurações administrativas a habilitar no GitHub**, não controles garantidos por este documento.
- Não conceder acesso direto a branches de release para contribuições externas; usar PRs e revisão.

## Política de publicação

`release/v1.0.0` permanece candidata: estado de código, release assinada e documentação têm gates separados. Somente publicar binário/instalador quando segurança, licenças, artefatos e testes forem comprovados no SHA exato. Não usar doações ou roadmap Enterprise como motivo para contornar gates.

## Revisão comunitária e sustentabilidade

A cada etapa significativa de produto, publicar objetivos, limitações e critérios de aceite. Quando houver capacidade real, triar issues de iniciantes e propostas de integração sem exigir um framework específico. Patrocínios são voluntários e não compram autoridade sobre código ou revisões. Consulte [FUNDING](docs/community/FUNDING.md), [REVIEW_GUIDE](docs/community/REVIEW_GUIDE.md) e [CONTRIBUTING](CONTRIBUTING.md).
