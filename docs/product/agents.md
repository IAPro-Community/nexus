# Agents

Um Agent é uma identidade persistente; uma Runtime generation é o processo que está trabalhando agora. Essa distinção permite observar falhas e recuperação sem chamar um novo processo de “mesmo runtime”.

Use Agents para tarefas recorrentes ou papéis especializados. Use AI Session para uma interação concreta. Estados de continuidade devem ser interpretados literalmente: `UNKNOWN` e `UNVERIFIED` não significam sucesso.
