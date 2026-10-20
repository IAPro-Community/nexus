# Troca de contas e continuidade de sessões

O Nexus diferencia **trocar o profile de uma mesma CLI** de **transferir contexto para outro provider**. Esse detalhe evita associar incorretamente uma conversa e a quota a outra identidade.

## Cenários

| Cenário | Comportamento esperado |
| --- | --- |
| Codex conta A → Codex conta B | Handoff entre profiles do mesmo provider, quando suportado; exige identidade de destino válida |
| Codex → Claude | Handoff de contexto; **não** reutiliza session ID nativa do Codex como se fosse Claude |
| Mesmo profile faz logout/login com outra conta | Novo `identity_version`; observações anteriores não viram quota LIVE da conta nova |
| Provider sem mecanismo nativo de resume | Estado `UNVERIFIED` ou nova sessão controlada; não prometer continuidade perfeita |

## Invariantes

- Preservar identidade lógica de Agent, Project, workspace e lineage quando a operação for permitida.
- Criar uma nova geração de runtime ao trocar de conta.
- Nunca apresentar sucesso de retomada nativa sem confirmação do provider.
- Proibir transição silenciosa de PIN ou mistura de credenciais.
- `UNKNOWN` para quota/identidade não vira evidência de disponibilidade.

## Diagnóstico

```bash
nexus profiles --json
nexus providers status --json
nexus ps --json
```

A referência CLI inclui `nexus handoff` e `nexus continue`; valide a sintaxe com `nexus --help` na versão instalada. Salve evidência sanitizada de geração de runtime, provider, estado da continuidade e resultado de attach.

Os testes de isolamento de conta e estados de runtime existem no código candidato, mas **o cenário completo de terminal com provider autenticado permanece sem certificado E2E geral**. Consulte [providers e quota](providers-and-usage.md), [continuidade](../architecture/session-continuity.md) e [suporte](../operations/platform-support.md).
