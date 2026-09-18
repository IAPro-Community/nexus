# Guia operacional do Composer

O Composer aprofunda um objetivo até o prompt de desenvolvimento deixar de ser
raso. Em cada rodada ele pergunta só o que ainda falta e ainda não está no
repositório nem nas memórias da instalação/projeto. A saída separa fatos,
premissas, desconhecidos, riscos, critérios de aceite e comandos de verificação.

## Entrada

- **Ideia / Explorar** — intenção ainda vaga.
- **Prompt existente** — revisa lacunas e contradições (`prompt_review`).
- **Anexar arquivo** — grounding com trechos do projeto no prompt final.

O preview do prompt canônico atualiza a cada turno (`brief.preview`).

## Memórias

Hábitos repetidos (verificação, testes, destino preferido) podem ser lembrados
na instalação ou neste projeto. Você vê e apaga as memórias em Settings. Escopo
pontual de uma feature não vira memória global.

## Caminhos de saída

- **Copy Prompt** copia o artefato para uso externo.
- **Send to Agent** envia o prompt para um runtime existente.
- **Turn into Flow** abre o draft no Flow Studio sem iniciar um runtime.
  Se o brief for multi-step (`FLOW_BENEFICIAL` / `FLOW_STRONGLY_RECOMMENDED`),
  o draft nasce já decomposto; caso contrário, é um pacote único.

Nenhum desses caminhos instala skills externas ou exige o Maestro. O usuário
pode editar, aceitar ou rejeitar as inferências antes de continuar.

Para a referência histórica e compatibilidade de links, consulte o [guia
anterior do Composer](../nexus-composer-user-guide.md).
