# Lane 03 — Produto

Objetivo: organizar capacidades pós‑V1 sem colocá-las no caminho crítico da Community.

## P2 — Backlog

- [ ] Secure relay para acesso remoto sem pré-requisito de SSH.
- [ ] Fleet multi-máquina com transporte seguro e localização de runtime.
- [ ] Sincronização e backup em cloud.
- [ ] Jobs agendados sempre ativos.
- [ ] Colaboração de equipe e políticas compartilhadas.
- [ ] Registry privado de skills para organizações.
- [ ] Aprovações remotas.
- [ ] Analytics avançado e histórico de inteligência por modelo/provedor.

## Programa transversal — IAPro SDK (P2)

- [ ] [Nexus #235](https://github.com/IAPro-Community/nexus/issues/235): extrair incrementalmente contratos de providers, quotas, troca de contas, runtime, coordenação entre agentes, AI Control e HUD para biblioteca compartilhada versionada (`IAPro-Community/IAPro-SDK`).
- [ ] [Calango #128](https://github.com/IAPro-Community/calango/issues/128): integrar primeiro slice e validar paridade entre consumidores.

Lane 03 coordena a extração; Lane 01 fornece evidência nativa de runtime e Lane 02 valida regressão e UX. SDK independente dos produtos; sem duplicação permanente, serviço obrigatório ou mudança de defaults. Aceite e sequência S0–S6 na issue #235.

## Guardrails

- Acesso remoto suportado hoje continua sendo por SSH/VPN.
- Não adicionar hospedagem pública, proxy central de tokens, IDE completa ou RBAC empresarial sem uma decisão arquitetural específica.
- Antes de cada item, definir ameaça, isolamento, autorização e estratégia de rollback.

