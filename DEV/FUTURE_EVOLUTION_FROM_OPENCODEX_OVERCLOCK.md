# Evolução futura a partir de OpenCodex / Overclock (NÃO implementar agora)

Oportunidades registradas durante a rodada incremental de 2026-09-19. Cada item
traz ideia, inspiração, seam existente, valor, risco e fase recomendada.

1. **Universal LLM proxy / Responses + Messages gateways** — insp. OpenCodex —
   seam: `internal/core/provider/adapters/*`, `internal/control/web` — valor:
   compatibilidade de API; risco: superfície de segurança e custo altos — fase: pós-1.0.
2. **40+ provider adapters / novo provider framework** — insp. OpenCodex —
   seam: `internal/core/provider/spec` (freeze) — valor: cobertura; risco: quebra
   do contract freeze — fase: incremental por provider, um por vez.
3. **SpendLedger / billing engine / price catalog** — seam: `store`, telemetry —
   valor: custo por execução; risco: dinheiro real exige auditoria — fase: dedicada.
4. **Marketplace / remote skill registry** — seam: SkillCatalog — valor:
   distribuição; risco: supply-chain — fase: dedicada com threat model.
5. **Recipe entity / Harness entity persistentes** — insp. Overclock — seam:
   `PackageRun/WorkPlan/MissionRunner` — valor: reuso de playbooks; risco:
   segundo orquestrador — fase: só como projeção, nunca como engine novo.
6. **Novo memory engine (OverMemory)** — REJECTED como engine; referência usada
   para fortalecer `IntelligenceMemory` — fase: nenhuma.
7. **MCP platform** — seam: `internal/control` — valor: ferramentas externas;
   risco: execução remota — fase: dedicada com sandbox.
8. **Voice / Whisper / Jarvis / media generation / screenshot engine** — seam:
   nenhuma (fora do produto) — fase: nenhuma antes de 1.0.
9. **Tray application / menu bar** — seam: Wails (desbloqueado) — valor:
   visibilidade de capacity; risco: app segunda superfície — fase: spec própria.
10. **Novos daemons / battery-activity detector** — seam: lifecycle Nexus —
    valor: cadência adaptativa real; risco: processo fantasma — fase: com T8 real.
11. **Probes de produção por provider** (contrato `quota.Probe` existe; registro
    de produção é `emptyProbeRegistry`) — seam: `capacitymonitor.ProbeRegistry`
    — valor: monitor com dados reais — fase: próxima, um provider por vez.
12. **Engine ≠ Backend ≠ Transport refactor** — fase: nenhuma nesta linha; só se
    o executor real exigir.
