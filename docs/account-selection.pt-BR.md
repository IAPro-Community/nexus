# Seleção de contas

[🇬🇧 English](account-selection.md) | **🇧🇷 Português** | [🇪🇸 Español](account-selection.es.md)

O Nexus escolhe uma conta utilizável dentro do provedor solicitado. A decisão considera autenticação, cooldown, quota conhecida, prioridade, vínculo do projeto e estratégia configurada. `UNKNOWN` continua sem evidência suficiente e não autoriza seleção automática. Quando o provedor declara quota `UNSUPPORTED`, a conta autenticada e saudável continua candidata como capacidade opaca, sem percentual inventado e abaixo de recursos com capacidade medida.

Ordem prática: perfil explícito, vínculo do workspace, perfil padrão e seleção automática. Contas desabilitadas, sem autenticação ou em rate limit são rejeitadas. Se a conta escolhida falhar por limite, o fallback pode tentar outra conta elegível do mesmo provedor.

Para distribuir novas sessões entre as contas em ordem round-robin, use `nexus agy:balance` ou `nexus agy:auto` (`nexus run agy:balance` também funciona). A posição é persistida por provedor e protegida contra seleções simultâneas em processos diferentes. A cada chamada, o Nexus avança para a próxima conta elegível; contas desabilitadas, não autenticadas, em cooldown, esgotadas ou com quota `UNKNOWN` são ignoradas. A seleção por rotação não escolhe a maior quota restante e não altera `nexus agy:<perfil>`, que continua fixando exatamente esse perfil. A forma simples `nexus agy` mantém o ranqueamento atual por capacidade.

Use `nexus explain <provider>` para ver a pontuação, `nexus usage` para capacidade e `nexus bind <provider>:<profile>` para uma preferência por projeto. Os IDs e valores exibidos em `--json` permanecem estáveis.
