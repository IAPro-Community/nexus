# API local

## Escopo e status

A API em `/api/v1` existe para a Web e o Desktop do Nexus. Ela é uma interface
local autenticada do produto, não um contrato público de integração com
compatibilidade externa garantida. Status: **Interna**.

O servidor registra health, bootstrap de sessão, projetos, agents, runtimes,
providers/perfis, recursos, Composer, plans, runs, Mission, atenção, sistema,
filesystem e tunnel em `internal/control/web/routes.go`.

## Autenticação e proteção

- `GET /api/v1/health` é o endpoint público de saúde.
- O bootstrap de navegador usa `POST /api/v1/auth/bootstrap` com um token de
  curta duração entregue na URL local.
- Os demais endpoints passam por middleware de autenticação; operações de
  escrita usam a proteção CSRF da sessão.
- Não exponha a API em uma rede pública nem reutilize tokens de bootstrap.

## Famílias de endpoint

| Família | Exemplos | Finalidade |
| --- | --- | --- |
| sessão | `/session`, `/session/rotate`, `/session/logout` | estabelece e encerra a sessão local |
| projetos | `/projects`, `/projects/{id}` | registra, lista e opera projetos |
| runtime | `/runtimes`, `/runtimes/{id}` | observa e controla sessão supervisionada |
| planejamento | `/composer-sessions`, `/flows/decompose`, `/plans`, `/runs`, `/missions` | Composer, Flow, WorkPlan e Mission |
| recursos | `/providers`, `/profiles`, `/resources` | providers, perfis, disponibilidade e seleção |
| operação | `/system/*`, `/fs/*`, `/tunnel/*`, `/attention` | diagnóstico e operações do host |

Consulte [arquitetura da API](../architecture/api.md) e os handlers em
`internal/control/web/`. Para automação suportada, prefira a
[CLI com `--json`](../cli/README.md) em vez de depender desses endpoints.
