# Caminho Direct

```text
Project → + New → AI Session → Provider → Terminal
```

Use-o para uma alteração localizada, exploração ou debugging. Ele não exige Composer, Flow, Mission ou Maestro.

O provider continua responsável pelo comportamento do seu CLI. Verifique os binários detectados com `nexus providers` e mantenha as políticas locais de comandos, workspace e credenciais.


## Promoção automática para Mission

No terminal supervisionado, o Nexus pode identificar que uma solicitação se
beneficia de Mission. Essa promoção é **Lead-first e fail-open para o Lead**:

- o Enter fica retido enquanto o Nexus tenta a admissão da Mission;
- o texto original permanece no provider até a admissão realmente passar;
- se a promoção foi apenas heurística e a admissão falhar, o mesmo prompt é
  submetido ao Lead sem ser perdido;
- se o usuário pediu Mission explicitamente (ou confirmou em modo ASK), a
  solicitação não degrada silenciosamente para Direct: o Nexus mostra os
  blockers exatos e preserva o plano rejeitado para diagnóstico.

Uma Mission só é anunciada como iniciada depois de existir um `run_id`.

O reconhecimento de pedido explícito usa palavras/tokens de Mission, não
substrings. Termos comuns de desenvolvimento como `permission`, `submission`
ou `transmission` não transformam uma solicitação em pedido explícito de
Mission.

Falhas comprovadamente anteriores ao `MissionRunner` também preservam a
semântica Lead-first: em promoção heurística o prompt pode voltar ao Lead com
segurança; em pedido explícito o Nexus mostra a etapa de preparação e não faz
downgrade silencioso.
