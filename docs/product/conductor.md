# Conductor

Conductor é a camada de orquestração que conecta seus objetivos a execução autônoma. Quando você descreve o que quer fazer, Conductor decide se precisa apenas de uma sessão direta ou se o trabalho justifica uma Mission completa.

## Como funciona

```text
Você: "Implementar autenticação JWT"
         ↓
    Conductor analisa o objetivo
         ↓
    Decomposição → Plano → Mission
         ↓
    Agent executa (com revisão)
         ↓
    Resultado sintetizado para você
```

Conductor preserva seus agentes existentes e usa o mesmo pipeline de execução que o Direct — não cria uma nova camada ou engine duplicada.

## Quando usar

| Situação | Caminho |
|---|---|
| Tarefa simples (corrigir typo, ajustar cor) | **Direct** — Conductor direciona direto |
| Trabalho composto (implementar feature, refatorar módulo) | **Conductor → Mission** — cria plano, executa, valida |
| Objetivo claro mas complexo | **Conductor** — você descreve, ele orquestra |

## O que muda para você

- **Menos decisões manuais**: descreva o objetivo e Conductor decide o melhor caminho.
- **Execução verificada**: cada execução passa por revisão antes de ser considerada concluída.
- **Visibilidade**: cada etapa da execução é registrada para diagnóstico.
- **Segurança**: o agente nunca recebe permissão implícita para commit, push ou deploy.

## Configuração

Conductor mantém uma política global que você pode ajustar:

| Opção | Padrão | Descrição |
|---|---|---|
| `review_mode` | automático | Modo de revisão read-only |
| `retention_days` | 90 | Dias de retenção de eventos |
| `integration_opt_in` | false | Habilitar integração fast-forward |

Para ver a política atual:

```bash
nexus conductor policy get
```

Para alterar:

```bash
nexus conductor policy set '{"retention_days": 30}'
```

## Exemplo

```bash
# Despachar um objetivo
nexus conductor dispatch "Implementar autenticação JWT para a API"

# Em formato JSON
nexus conductor dispatch --json "Refatorar o módulo de pagamentos"
```

## Segurança

- Worker efêmero é descartado automaticamente após a execução.
- Nenhuma credencial ou prompt sensível aparece em logs ou API.
- Git merge (integração) é desabilitado por padrão.
- Provider nunca recebe permissão para operações Git destrutivas.
