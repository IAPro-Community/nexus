# Providers, contas, quota e custo

O Nexus coordena **CLIs/providers oficiais** e perfis locais. Não cria créditos, não substitui a assinatura de um provider e não considera uma instalação autenticada só porque encontrou o executável.

## TL;DR

1. Execute `nexus doctor` e `nexus providers status --json` para descobrir o que realmente existe no host.
2. Na Web, abra **Resources / Usage** para verificar profiles, identidade, saúde, janelas de uso e fonte da observação.
3. Registre ou autentique a conta **no provider oficial**, conforme o fluxo apresentado pelo Nexus.
4. Use `nexus usage` para leitura humana e `nexus usage --json` para scripts, quando disponível na sua versão.
5. Não tome decisões automáticas com base em valores `UNKNOWN` ou estimativas sem proveniência.

## Modelo de identidade

```text
Provider instalado (ex.: Codex)
  └─ Profile configurado (ex.: pessoal)
      └─ AccountScope verificável (provider/account/identity_version)
          ├─ Observação de quota da conta e ciclo
          ├─ Estado de autenticação/saúde
          └─ Preferência de seleção (PIN / PREFER / AUTO)
```

**Profile é configuração; AccountScope é identidade verificada.** Nome, e-mail e rótulo não autorizam atribuir métricas a outra conta. Uma troca de login pode incrementar `identity_version`. Observações sem vínculo comprovado não devem migrar silenciosamente para a nova conta. Veja [isolamento de contas](../architecture/quota/account-isolation.md) e [identity scoping](../architecture/quota/identity-scoping.md).

## Estados e evidência

| Estado | Interpretação correta |
| --- | --- |
| `LIVE` | Observação recente proveniente da fonte; ainda depende de identidade e janela válidas |
| `CACHED` | Leitura antiga; mostrar timestamp e não fingir que é atual |
| `ESTIMATED` | Cálculo do Nexus, não informação reportada pelo provider |
| `UNKNOWN` | Não existe prova suficiente; **não significa 0% nem 100%** |
| `UNSUPPORTED` | O provider não fornece essa métrica; não inventar porcentagem |
| `RATE_LIMITED` | Consulta bloqueada por limites do provider |
| `UNAVAILABLE` | Fonte inacessível ou problema operacional |

Não confundir **quota**, saúde de autenticação, incidentes do serviço, consumo de tokens, custo estimado e valor faturado. São conceitos distintos.

## Como escolher recursos

`AgentMatcher` considera especialidade e papel; `ResourceScheduler` seleciona um recurso executável conforme autenticação, saúde, políticas, preferências e capacidade observável. O frontend e o HUD não recalculam esse ranking.

- `PIN`: recurso fixado pelo usuário; falhar é melhor que trocar de conta silenciosamente.
- `PREFER`: tenta a escolha preferida, permitindo fallback quando a política o autoriza.
- `AUTO`: considera os recursos elegíveis, com tratamento próprio para quota opaca; não fabrica percentuais.

Veja [resource routing](../architecture/resource-routing.md). `UNKNOWN` deve permanecer distinto de `UNSUPPORTED`, que pode permitir execução quando o recurso está saudável e a política admite capacidade opaca.

## Monitor de capacidade e previsão

O monitor em background deve ser explicitamente habilitado; não existe obrigação de lançar um segundo processo de consulta somente porque o HUD está aberto. A previsão de ritmo usa uma série de amostras `LIVE` verificadas e vinculadas ao mesmo ciclo de reset. Exige pelo menos três observações e dez minutos, e permanece **consultiva**: não altera quota, billing nem as decisões do scheduler.

Veja [Capacity Pace Intelligence](../architecture/quota/pace-intelligence.md) e [HUD](hud.md).

## Cost Intelligence e uso externo

O ledger de custo distingue custo reportado pelo provider, equivalência de API, inferências e desconhecido. **Desconhecido não é custo zero.** Para dados locais do Claude, a importação de transcripts é manual, exige consentimento explícito e não atribui tokens à conta Nexus sem prova. Essas observações ficam fora dos valores efetivos de cobrança e orçamento.

O status público de OpenAI/Claude/GitHub informa incidentes de serviço, **não** a quota pessoal. Consulte [uso externo, fontes e exportação CSV](../architecture/quota/EXTERNAL_USAGE_AND_PUBLIC_STATUS.md).

## Fluxo de diagnóstico

```bash
nexus doctor --json
nexus providers status --json
nexus profiles --json
nexus usage --json
```

Confira a versão instalada com `nexus --help` e o `--help` de cada comando. Para um problema de quota, registre provider, profile (rótulos podem ser anonimizados), estado de evidência, janela, timestamp e erro sanitizado; **não** publique tokens, cookies, identificadores brutos ou JSON de autenticação.

## Limites atuais

O catálogo funcional comprova presença de implementação e arquivo de teste, não valida login real em todas as contas, nem failover autenticado no Windows/Linux/macOS. A matriz de execução e a certificação por SHA estão em [suporte de plataformas](../operations/platform-support.md) e [feature matrix](../reference/feature-matrix.md).
