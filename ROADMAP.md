# Roadmap IAPro Nexus

> **Nova trilha de adoção (27/10/2026):** consulte [Plano de adoção e execução](docs/strategy/NEXUS_ADOPTION_EXECUTION_PLAN_2026.md) para marcos, metas, dependências e issues #76–#93. O documento abaixo preserva o histórico de fechamento da v1; homologação e plataformas seguem [release status](docs/release-status.md) e [platform support](docs/operations/platform-support.md). Não tratar a classificação histórica de Linux-only como autorização para publicar Windows/macOS sem certificação ou excluir gates de plataformas hoje declaradas.

Este roadmap descreve a campanha de fechamento do Nexus 1.0. Não é promessa de prazo.
O contrato técnico detalhado está em
[`DEV/SPECS/NEXUS_1_0_CLOSURE.md`](DEV/SPECS/NEXUS_1_0_CLOSURE.md).

## Baseline concluído

O PR #3 foi mergeado na `main` em
`7605f702cce220b4111d51d06a34b37f7e0a9a8d` após CI same-SHA verde no head
`300ccbeee4f145f80ebce13c9cefafcb01acefc3`.

Esse baseline consolida runtime/terminal hardening, Capacity Monitor T0–T9,
routing/memory reliability, Surface Isolation, Flow recovery e os gates Linux/browser.

## Nexus 1.0 — ordem de fechamento

1. **Truth alignment / release baseline**
   - uma fonte atual de release truth;
   - docs/version/changelog coerentes;
   - evidência do baseline preservada.

2. **Production Capacity Intelligence**
   - probes reais por provider;
   - Codex primeiro;
   - UNKNOWN permanece UNKNOWN;
   - recomendações baseadas em evidência real.

3. **Agent Modes + canonical AgentMatcher**
   - presets tipados via AgentSpec;
   - Reviewer/Investigator read-only por padrão;
   - uma única autoridade para escolher WHO;
   - ResourceScheduler continua escolhendo WITH WHAT.

4. **Skills contract v1**
   - discovery por metadata;
   - carga sob demanda;
   - provenance/source/version;
   - integração opcional com Maestro.

5. **Cost Intelligence**
   - token/cost attribution por Project/Branch/Mission/Agent/Runtime/model;
   - origem/qualidade explícita do custo;
   - budgets e anomaly alerts;
   - observabilidade separada de billing.

6. **Unified Work Mode**
   - entrada única orientada a objetivo;
   - Composer → Flow/WorkPlan → Mission reutilizados internamente;
   - team/resource/isolation/budget/gates visíveis antes de iniciar.

7. **Mission 1.0 + Evidence Bundle**
   - cenários live de falha, intervenção, failover, restart e reconnect;
   - DONE somente por contrato de verificação;
   - evidência persistida e navegável.

8. **Memory 1.0**
   - Retain / Recall / Reflect sobre IntelligenceMemory existente;
   - INSTALL e PROJECT;
   - provenance;
   - input explícito atual sempre vence.

9. **Maestro contract v1**
   - capabilities, skills, advice e verification machine-readable;
   - Nexus continua funcional sem Maestro;
   - nenhuma duplicação de método/skills no Core.

10. **Community + v1 release certification**
    - manter Community funcional sem limitação artificial de Agents, Composer, Flow ou Missions;
    - licenciamento open source somente após autorização dos titulares e auditoria de terceiros/segredos;
    - mecanismos de entitlement existentes não devem bloquear o núcleo Community;
    - Linux amd64 same-SHA;
    - install/upgrade/uninstall/update;
    - signed artifacts;
    - dogfooding e soak;
    - BLOCKER=0 / HIGH=0.
    - A certificação nativa de cada plataforma anunciada deve passar no SHA final, conforme a matriz vigente em `docs/operations/platform-support.md`; nunca interpretar esta lista histórica como dispensa de gates.

## Trilha paralela — abertura comunitária e sustentabilidade

A intenção de produto é **Apache-2.0**, não aplicada ainda. O
arquivo `LICENSE` mantém **PolyForm Strict 1.0.0** até confirmação de
titularidade, segurança do histórico, licenças de dependências/assets e
parecer jurídico. Não classificar uma alteração só documental como
relicenciamento ou autorização de forks modificados.

- Preparar governança, documentação de contribuições, política de marca e
  guias de colaboração/upstream (documentação preparada);
- Fechar [matriz legal e security gate](docs/legal/OPEN_SOURCE_TRANSITION.md);
- Atualizar licença, avisos, READMEs e testes no commit de abertura;
- Liberar PRs de código de terceiros sob termos válidos e validação DCO;
- Ativar patrocínios e, depois da adoção, avaliar suporte/managed service.
  Nenhum paywall deve ser requisito do Core Community.

## Fora do Nexus 1.0 salvo blocker

- universal LLM proxy;
- dezenas de adapters novos;
- marketplace público de MCP/skills;
- Jarvis/voz contínua always-on e multimodal avançado (Nexus Voice/STT push-to-talk já está na V1);
- media generation;
- cloud VM agents;
- TEAM shared memory;
- full remote collaboration;
- Stripe/billing;
- novo memory engine;
- novo scheduler;
- novo Mission engine.

## Release rule

Somente declarar `NEXUS_1_0_CANDIDATE = GO` quando o candidato final tiver todos
os gates requeridos verdes no mesmo SHA e evidência nativa das plataformas declaradas.
A lista de plataformas e a política de certificação vigentes são definidas em
`docs/operations/platform-support.md`; o registro histórico deste roadmap não
substitui essa matriz. Nenhuma plataforma pode ser anunciada sem prova exact-SHA.
