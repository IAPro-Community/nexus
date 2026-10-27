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

## Failover automático de quota no source beta.10

`ContinuityController` é invocado pelo monitor apenas após uma observação
confiável de **quota esgotada**, associada a um runtime. A ordem de seleção é
outra conta elegível do mesmo provider, depois um provider alternativo.
Perfis que compartilham a mesma conta upstream (`AccountScope` verificado)
**não** representam capacidade independente. A operação limita saltos,
registra a substituição e preserva o lineage do Agent.

Esse caminho é uma tentativa supervisionada de continuidade, não uma garantia
de que todos os providers preservem histórico, ferramentas em andamento ou
estado nativo do terminal. Leituras ausentes/UNKNOWN não autorizam failover;
replay de efeitos colaterais permanece proibido sem prova de segurança.

Leia o [contrato técnico](../architecture/session-continuity.md) para estados,
eventos e testes. A execução nativa real no SHA da release segue pendente.

## Barreira de segurança do handoff (source atual)

O handoff de conta ou contexto exige PID de provider conhecido e parada
comprovada antes de iniciar outro escritor no mesmo workspace. Uma requisição
de parada ou um timeout, por si só, não autorizam continuidade. Se o destino
falha após a parada da origem no handoff de contexto, a origem é registrada
como `STOPPED`, com checkpoint para recuperação explícita; não como
`RUNNING` por uma alteração de registro. Esse hardening não equivale à
homologação E2E com os providers reais.

## Diagnóstico

```bash
nexus profiles --json
nexus providers status --json
nexus ps --json
```

A referência CLI inclui `nexus handoff` e `nexus continue`; valide a sintaxe com `nexus --help` na versão instalada. Salve evidência sanitizada de geração de runtime, provider, estado da continuidade e resultado de attach.

Os testes de isolamento de conta e estados de runtime existem no código candidato, mas **o cenário completo de terminal com provider autenticado permanece sem certificado E2E geral**. Consulte [providers e quota](providers-and-usage.md), [continuidade](../architecture/session-continuity.md) e [suporte](../operations/platform-support.md).
