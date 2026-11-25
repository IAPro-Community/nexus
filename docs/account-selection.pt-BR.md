# Seleção de contas

[🇬🇧 English](account-selection.md) | **🇧🇷 Português** | [🇪🇸 Español](account-selection.es.md)

O Nexus escolhe uma conta utilizável dentro do provedor solicitado. A decisão considera autenticação, cooldown, quota conhecida, prioridade, vínculo do projeto e estratégia configurada. Seleção automática exige quota recente `LIVE` ou `CACHED` confiável. `UNKNOWN`, `ERROR`, `ESTIMATED` e `UNSUPPORTED` não autorizam escolha automática nem recebem capacidade inventada. Um perfil fixado, vínculo ou padrão explícito segue a política de seleção explícita sem ser ranqueado como quota medida.

Ordem prática: perfil explícito, vínculo do workspace, perfil padrão e seleção automática. Contas desabilitadas, sem autenticação, em cooldown, esgotadas ou sem quota confiável recente são rejeitadas da seleção automática. A troca automática entre contas vem desabilitada por padrão; habilite explicitamente com `nexus config fallback enable` e revogue com `nexus config fallback disable`. O opt-in permite troca antes da execução quando o perfil está ocupado, sujeita à identidade e quota recente. Depois que um processo provider inicia, o Nexus não repete uma operação por quota/rate limit, pois saída e efeitos externos podem ser ambíguos.

Para distribuir novas sessões entre contas com quota recente verificada em ordem round-robin, use `nexus agy:balance` ou `nexus agy:auto` (`nexus run agy:balance` também funciona). A posição é persistida por provedor e protegida contra seleções simultâneas em processos diferentes. Contas sem evidência `LIVE`/`CACHED` confiável são ignoradas. A seleção por rotação não escolhe a maior quota restante e não altera `nexus agy:<perfil>`, que continua fixando exatamente esse perfil. A forma simples `nexus agy` mantém o ranqueamento por capacidade verificada.

Use `nexus explain <provider>` para ver a pontuação, `nexus usage` para capacidade e `nexus bind <provider>:<profile>` para uma preferência por projeto. Os IDs e valores exibidos em `--json` permanecem estáveis.
