# Composer

Composer é uma **refinaria de prompt**: aprofunda um objetivo até o trabalho de desenvolvimento não nascer raso. Ele extrai do usuário (e do contexto já conhecido) resultado observável, restrições, conflitos e critérios de aceite — e produz um `PromptArtifact` estruturado, não apenas prosa.

## Como decide o que perguntar

Composer usa quatro camadas, nesta ordem, e **não pergunta** o que já está coberto:

1. Respostas desta sessão
2. Project Intelligence (fatos observados no repositório)
3. Memória do projeto (hábitos deste repo)
4. Memória da instalação (hábitos locais, estilo ChatGPT)

Uma pergunta por rodada, com opções quando a inteligência as sugerir. Premissas de baixo impacto aparecem como “Nexus assumiu — editar”, sem bloquear.

## Destinos

**Copy Prompt**, **Send to Agent** e **Turn into Flow**. Copiar e enviar ao Agent não exigem Flow. Turn into Flow cria um draft no Flow Studio e não inicia runtime por si só.

Você pode parar, editar, rejeitar inferências ou pedir para lembrar um hábito. Fatos, premissas e desconhecidos permanecem distintos.

Composer é opcional: Direct e Flow com objetivo próprio continuam válidos.

Veja o [guia operacional do Composer](composer-guide.md).
