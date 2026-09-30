# Composer

Composer é uma **refinaria de prompt**: aprofunda um objetivo até o trabalho de desenvolvimento não nascer raso. Ele extrai do usuário (e do contexto já conhecido) resultado observável, restrições, conflitos e critérios de aceite — e produz um `PromptArtifact` estruturado, não apenas prosa.

## Como decide o que perguntar

Composer usa quatro camadas, nesta ordem, e **não pergunta** o que já está coberto:

1. Respostas desta sessão
2. Project Intelligence (fatos observados no repositório)
3. Memória do projeto (hábitos deste repo)
4. Memória da instalação (hábitos locais, estilo ChatGPT)

Quando a Intelligence está disponível, `ExtractBrief` estrutura o texto livre em escopo, restrições, riscos, aceite, testes e ações proibidas (com provenance `INTELLIGENCE`). Heurísticas por keyword continuam como fallback.

Uma pergunta por rodada, com opções quando a inteligência as sugerir. Premissas de baixo impacto aparecem como “Nexus assumiu — editar”, sem bloquear. O checklist por arquétipo cobre feature, bug fix, research, DevOps, architecture e security — incluindo fora de escopo, riscos, ações proibidas, Definition of Done e entregáveis.

## Interview Engine

O Composer mantém um estado de clareza por dimensão (`intent`, `scope`, `context`, `constraints`, `acceptance`, `risk` e `decisions`). Cada pergunta aberta recebe `dimension` e `priority_score`; a próxima pergunta é a de maior ganho esperado de informação, e não apenas o próximo item do checklist.

Fatos inferidos por Project Intelligence ou memória permanecem auditáveis no brief, mas não voltam para a fila de perguntas. Ao iniciar uma elaboração, a UI permite escolher `QUICK`, `STANDARD` ou `DEEP`; `STANDARD` é o padrão adaptativo. Cada perfil define alvo de ambiguidade e orçamento máximo de perguntas, e o contador avança apenas em rodadas reais de resposta/resolução.

Quando o brief já convergiu e existe uma decisão sensível a trade-offs (por exemplo custo, quota, escolha automática ou fallback), o Composer pode abrir uma única **pressure-test question**. Essa passagem desafia a decisão antes da finalização sem criar uma cadeia skill → prompt → skill e sem transformar a resposta em memória durável automaticamente.

## Contexto anexável (`@file`)

Use **Anexar arquivo** na conversa para incluir trechos do repositório no brief. Cada arquivo entra em `referenced_files` com excerpt limitado (budget por arquivo e total) e aparece no preview do prompt sob **Context — Referenced files**. Symlinks que saem do projeto, binários e sessão `FINALIZED` são recusados.

## Preview ao vivo

Antes de finalizar, o Composer mostra o prompt canônico completo (`brief.preview`): Role, Objective, Motivation, Context, Requirements, Out of scope, Decisions, Assumptions, Constraints, Environment, Execution, Risks, Acceptance, Testing, Verification, Definition of Done, Deliverables e gaps aceitos. Prompt existente dispara `prompt_review` (lacunas e contradições).

## Destinos

**Copy Prompt**, **Send to Agent** e **Turn into Flow**. Copiar e enviar ao Agent não exigem Flow. Turn into Flow cria um draft no Flow Studio e não inicia runtime por si só. Quando `flow_suitability` recomenda Flow, a materialização usa a mesma decomposição do Conductor (DAG real); briefs `DIRECT_FIT` viram um pacote enxuto.

Você pode parar, editar, rejeitar inferências ou pedir para lembrar um hábito. Fatos, premissas e desconhecidos permanecem distintos.

Composer é opcional: Direct continua válido sem passar pela refinaria. Flow drafts vazios nascem no Flow Studio; planos multi-step ricos devem vir do Composer.

Veja o [guia operacional do Composer](composer-guide.md).
