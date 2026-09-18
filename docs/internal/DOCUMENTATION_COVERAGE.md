# Cobertura de documentação

Inventário em 2026-09-13. “Documentadas” significa que há uma página pública
orientada a tarefa ou referência com link no mapa de documentação; não é uma
afirmação de certificação operacional em todos os ambientes.

| Área | Features encontradas | Documentadas | Cobertura | Observação |
| --- | ---: | ---: | ---: | --- |
| Core de projetos, agentes e runtimes | 4 | 4 | 100% | cobertura em produto, CLI e arquitetura |
| Providers, perfis e quota | 2 | 2 | 100% | estados de evidência documentados |
| Planejamento e execução | 4 | 4 | 100% | Mission/delegação marcadas como experimentais |
| Web / API local | 2 | 2 | 100% | API classificada como interna |
| CLI e automação | 1 | 1 | 100% | catálogo agrupado, help continua canônico |
| Desktop e plataforma | 1 | 1 | 100% | suporte condicionado à matriz de plataformas |
| Operação, update e segurança | 3 | 3 | 100% | não substitui procedimentos de release em `DEV/` |
| Mídia visual | 1 | 1 | 100% | screenshots e demo sintéticos com manifesto |

## Lacunas honestas

1. Não há evidência nesta árvore de uma API pública/SDK estável; por isso não
   foi criada documentação de integração pública.
2. Não há evidência suficiente para declarar todos os providers autenticados ou
   o Desktop como certificados em todas as plataformas.
3. Capturas documentam Web em fixture sintética. Vídeos longos e GIFs não foram
   adicionados porque já existe uma demonstração WebM reproduzível e arquivos
   maiores não aumentariam a compreensão proporcionalmente.

Atualize este arquivo junto do [inventário](FEATURE_INVENTORY.md) quando uma
capacidade pública entrar, mudar de status ou for removida.
