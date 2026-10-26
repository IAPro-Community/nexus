# Nexus Composer — elaboração inteligente

O Composer transforma uma ideia, inclusive de alguém sem familiaridade com tecnologia ou com o produto, em um planejamento fundamentado, revisável e preparado para execução. Ele **não** é uma sequência obrigatória de formulários. O usuário apresenta o problema que conhece; a inteligência propõe caminhos, identifica lacunas relevantes e sugere como produzir um resultado verificável.

## Pré-requisito: Nexus Intelligence

Antes de iniciar ou editar uma elaboração pelo Composer, configure **Configurações → Nexus Intelligence** com um provedor de IA. São aceitos os modos de CLI com perfil e capacidades headless validadas, ou API OpenAI-compatible com credenciais e modelo adequados.

A interface bloqueia o Composer sem provider disponível e oferece acesso às configurações. A API valida a configuração e, ao criar uma elaboração, executa um round-trip real e limitado por timeout antes de persistir a sessão. Um CLI instalado, mas não autenticado, não é prova suficiente de funcionamento. Erros retornam instruções para reconfigurar.

Essa exigência vale para Composer; **Direct, terminais, Agent e Maestro não dependem de configuração de inteligência para funcionar**.

## Experiência de descoberta

1. Descreva a ideia ou forneça um prompt existente. Não é necessário conhecer arquitetura, stack, roadmap ou Definition of Done.
2. O Nexus recupera contexto de projeto disponível, memórias e evidências de repositório. Informações não observadas **não** são apresentadas como verificadas.
3. O provider configurado extrai itens fundamentados e até três abordagens sugeridas. Recomendações são registradas separadamente como fatos `PROPOSED`, origem `INTELLIGENCE`, e nunca são tomadas como decisões aprovadas.
4. O Composer mostra logo na abertura um resumo de objetivo, opções propostas e, somente quando material, uma decisão contextual. O painel lateral permanece editável e inspecionável.
5. Continue conversando em linguagem natural, corrigindo hipóteses ou pedindo comparação de alternativas. O contexto e o plano são atualizados incrementalmente.
6. Revise o artefato final antes de enviar para Agent ou gerar um draft Flow.

### Política de perguntas

O motor **não** injeta mais os mesmos campos ausentes como perguntas automáticas de fora de escopo, testes, riscos, stack, ações proibidas, Definition of Done e entregáveis. O checklist continua servindo internamente para avaliar a qualidade do planejamento.

Só aparecem perguntas efetivamente produzidas a partir de uma decisão material, com justificativa e, quando possível, opções compreensíveis. Uma decisão de alto impacto, irreversível ou relacionada à segurança pode ser `BLOCKING`; detalhes com padrão seguro são premissas revisáveis.

A entrevista `STANDARD` não executa um pressure-test ritual ao final. O teste adicional de trade-offs fica restrito ao modo `DEEP`, quando houver gatilho. `QUICK` também permanece disponível.

Sessões legadas seguem legíveis: questões antigas não respondidas de `ARCHETYPE`/`REFINERY` não voltam à fila de perguntas após atualização do brief. Respostas já confirmadas ou dispensadas são preservadas.

## Artefato e execução

A representação canônica continua sendo o `LivingBrief` + `PromptArtifact`, com contexto, motivações, hipóteses, recomendações, evidências e critérios de aceite. **WorkPlan** é a fonte de verdade para o planejamento executável; **Flow** é a representação visual e editável do mesmo plano, sem criar scheduler paralelo.

Destinos:
- **Copy Prompt**: compartilha o artefato para outro contexto;
- **Send to Agent**: inicia uma execução assistida sem Flow obrigatório;
- **Turn into Flow**: cria um draft WorkPlan/Flow para revisar dependências e etapas, sem iniciar uma Mission automaticamente.

GO, verificações e conclusão com evidência continuam governados pelos sistemas de execução existentes; Composer não possui uma infraestrutura concorrente de execução ou memória. Maestro permanece opcional e independente.

## Segurança e limites

- Uma recomendação da IA **não** autoriza deploy, push ou edição destrutiva.
- A ausência de acesso ao repositório não vira “análise técnica realizada”.
- Falhas de IA devem ser apresentadas ao usuário e jamais mascaradas como planejamento confirmado.
- As versões finais precisam de homologação real por SHA e plataforma. Alteração de código na `release/v1.0.0` não significa distribuição estável certificada.

Consulte também [guia operacional](composer-guide.md), [Flow](flow.md) e [status da release](../release-status.md).
