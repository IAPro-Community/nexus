# Evolução Estratégica, Competitiva, Arquitetural e Operacional do IAPro Nexus

> **Branch ativa:** `release/v1.0.0` (base `4c381001` -> implementação `11e5cfba`)
>
> **Data:** `2026-11-27`
>
> **Missão:** Consolidar o Nexus como ambiente central diário de engenharia orientada a agentes, priorizando **performance e estabilidade**, **baixo atrito operacional**, **gestão inteligente multi-conta conforme com ToS**, **execução paralela em worktrees** e **automações determinísticas**.

---

## Índice dos Documentos

1. [01 — Diagnóstico Atual e Análise Competitiva](./01-DIAGNOSIS-AND-COMPETITIVE-ANALYSIS.md) — inventário real do código, funcionalidades subdescobertas, pesquisa de mercado (`2026-11-27`) e matriz comparativa de 26 critérios.
2. [02 — Auditoria de Performance, Benchmarks Reproduzíveis e UX Diária](./02-PERFORMANCE-AND-USABILITY-AUDIT.md) — medições ao vivo **ANTES vs DEPOIS**, análise honesta da meta de 250 MB em WebKitGTK, auditoria de polling/armazenamento e melhorias de experiência diária.
3. [03 — Arquitetura Alvo e Propostas de Evolução](./03-TARGET-ARCHITECTURE-AND-PROPOSALS.md) — propostas para Multi-Conta + Pools ToS-compliant, Capacity Monitor, Execução Paralela + Worktrees + Diff Comments, Automação `Event -> Condition -> Action -> Result`, e Atenção/NEX HUD.
4. [04 — Plano Executável, Registro de Melhorias, Cenários e Code Review](./04-EXECUTABLE-PLAN-AND-VERIFICATION.md) — plano executável completo, registro das entregas P0/P1 realizadas, matriz de validação A–I e code review final.

---

## Avaliação das Fronteiras do Ecossistema (`Nexus` vs `IAPro-SDK` vs `Orquestrador Maestro`)

Consistente com [ADR-iapro-sdk-consumer.md](../architecture/ADR-iapro-sdk-consumer.md) e [MAESTRO_EXTERNAL_AUTHORITY_ADR.md](../architecture/MAESTRO_EXTERNAL_AUTHORITY_ADR.md):

| Responsabilidade | Dono Correto | O que hoje está duplicado / deve migrar | O que permanece estritamente fora do Nexus |
| --- | --- | --- | --- |
| **Contratos puros de provedor (`Spec`, capacidades, modelos), parsing JSONL, motor puro de `quota`/`eligibility`/`routing`/`capacity`/`cost`, componentes UI headless base** | **IAPro-SDK** (`../IAPro-SDK`) | Tipos e cálculos duplicados entre `internal/core/{provider,quota,scheduler}` e o SDK compartilhado (`quota`, `routing`, `capacity`, `providers`) devem convergir gradualmente para o SDK após testes de paridade | Lógica específica de banco local, systemd/daemon ou janela Wails |
| **Ambiente diário do desenvolvedor:** gerenciamento de projetos/worktrees, terminais PTY (`__control-host`), isolamento de diretórios/credenciais por perfil, daemon HTTP/WS, Wails Desktop + NEX HUD, central de atenção e automações locais | **IAPro Nexus** (este repositório) | Consolidar um único contrato de `versionprobe` e `Detect` entre `internal/core/provider/adapters/*` e `internal/control/driver/*` (**progresso real em P0-1**) | Nunca reimplementar runtime autônomo próprio, nem proxy OAuth |
| **Governança autônoma, memória durável, ledger de missões longas e orquestração estratégica multi-agente** | **Orquestrador Maestro** (autoridade externa) | Nexus exibe estado observado e despacha/supervisiona sessões autorizadas sem duplicar memória nem ledger de runtime | Motor de planejamento/memória profundo do Maestro |

---

## Roadmap Priorizado

### `P0` — Performance, Estabilidade e Higiene Crítica *(Implementado e Validado ao Vivo)*
- **P0-1:** Cache por fingerprint de binário + coalescência `singleflight` em `versionprobe`, consolidação dos 6 drivers/adapters duplicados e cache 60s de fallback em `cursor.DetectHostAuth` (**`/api/v1/providers` caiu de `2,299 ms` para `4.23 ms`**; **`/api/v1/resources` caiu de `1,795 ms` para `7.67 ms`**).
- **P0-2:** `QuotaMonitorService` ignora provedores `QuotaUnsupported` e reutiliza snapshots frescos (`< 5m`) para contas ociosas sem runtime ativo.
- **P0-3:** Fast path zero-subprocesso `.git/HEAD` (`?summary=1`) para taskbar de branch + intervalo 30s (`38.93 ms` -> **`0.49 ms`**, 0 processos `git`).
- **P0-4:** Desacoplamento de `/api/v1/providers` da cadência de 30s para 5 minutos em `useNexusData.ts`.
- **P0-5:** Script reproduzível `scripts/benchmark-nexus-evolution.py` + benchmarks Go (`testing.B`).

### `P1` — Produtividade Diária, Redução Real de Memória Desktop/HUD e Capacidade *(Parcialmente Entregue + Próximo Ciclo)*
- **P1-1 (Entregue):** Atalhos de Command Palette para troca rápida de branch/worktree (`nexus:open-branch-switcher`) e refresh global sincronizado (PT-BR/EN/ES).
- **P1-2:** Redução estrutural de RAM Desktop+HUD (`~1.04–1.35 GB` -> meta intermediária `< 450 MB`): tornar `nexus-hud` WebView **sob demanda** (tray/indicador GTK nativo leve `< 25 MB` quando recolhido, abrindo WebView só em alerta ou interação), além de suspender renderização de terminais xterm em abas não visíveis.
- **P1-3:** Fluxo **1-clique "Novo Agente em Worktree Isolado"** (Command Palette + botão Overview) com seleção automática de conta saudável e suporte a `.nexus/setup.sh`.
- **P1-4:** **Revisão de Diff Inline com Comentário -> Agente** (enviar trechos/linhas comentadas direto ao PTY ou Composer) sem construir editor completo.
- **P1-5:** **Start-of-Day Briefing** no Overview (fila de atenção bloqueada + último projeto/branch/sessões recuperáveis + saúde das contas).
- **P1-6:** Limpeza segura/retenção em `~/.local/share/ai-manager/profiles` (hoje `48 GB`) e pruning de `web-auth/listen-*.json` obsoletos (hoje 28 arquivos mortos observados).

### `P2` — Pools Inteligentes, Automação Determinística e Consolidação SDK
- **P2-1:** Account Pools ToS-compliant (`CONSUME_FIRST`, `PRESERVE_PRIMARY`, `COST_CAP`) com UI de explicabilidade + `burn_rate` no Capacity Monitor.
- **P2-2:** Motor declarativo local `Event -> Condition -> Action -> Result` com dry-run e guardas de orçamento (auto-verify pós-agente, retomada controlada no reset de quota).
- **P2-3:** Consolidação de `internal/core/quota` + `internal/core/provider` dentro do `IAPro-SDK` removendo código espelhado.
- **P2-4:** Unificação eventual de `nexus-desktop` + `nexus-hud` em janela auxiliar ou WebView compartilhada / avaliação de Tauri/nativo para atingir `<= 250 MB` no processo completo.

### `P3` — Diferenciação Avançada e Ecossistema
- **P3-1:** Conclusão dos 11 gates humanos do mascote NEX (`NEX-ID-v1` + Retro HD) e liberação conjunta HUD + Site.
- **P3-2:** Conectores externos opcionais (webhooks GitHub PR/CI, Telegram/mobile companion seguro sob autenticação explícita).
- **P3-3:** Certificação E2E nativa em Windows e macOS (`NATIVE_WINDOWS` / `NATIVE_MACOS`).

---

## Backlog das Oportunidades Não Implementadas Neste Ciclo

Itens investigados e priorizados que permaneceram em backlog com escopo preservado:

1. **Redução de WebKitGTK Dual Stack (`~1.04–1.35 GB` -> `< 450 MB` P1 / `< 250 MB` P2)** via HUD tray nativo / WebView sob demanda e lazy-mount/skip-render de terminais `xterm.js` ocultos.
2. **Diff Viewer inline com comentários enviados ao agente** (gap competitivo direto frente a Orca, Superset e Vibe Kanban).
3. **Launcher 1-clique em Worktree Isolado + `.nexus/setup.sh`** e **Start-of-Day Continuity View**.
4. **Motor de Automação `Event -> Condition -> Action -> Result`** fora de Composer/Flow.
5. **Account Pools & Depletion Rate (`burn_rate`) no Capacity Monitor**.
6. **Política de retenção/limpeza dos 48 GB de caches de perfil** em `~/.local/share/ai-manager/profiles` + remoção de arquivos mortos `web-auth/listen-*.json` de PIDs encerrados.
7. **Preview/Handoff DOM** (conceito Chromium Design Mode estilo Orca reutilizando o servidor Browser MCP já existente no Nexus).
8. **Consolidação restante de adapters/quota no `IAPro-SDK`** e **portabilidade validada Windows/macOS**.
