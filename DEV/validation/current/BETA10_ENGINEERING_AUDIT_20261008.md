# Nexus v1.0.0 — Auditoria de engenharia da linha beta.10

**Data:** 2026-10-08  
**Baseline revisado:** `release/v1.0.0` em `fbc9703bb59a250bbc13114436a20e93266a5574`; houve o avanço adicional `95c7121d9675002d856fa55fdcfa8d9ce59adf35` durante a revisão.  
**Destino de integração:** `release/v1.0.0` somente via PR; NÃO `main`.  
**Estado:** NO-GO para integração na main, tag estável ou afirmação de certificação.  
**Método:** inventário recursivo dos 1.695 blobs do snapshot beta.9, inspeção direcionada de implementação/configuração e histórico, comparação de refs e evidências de CI. **Não é análise linha a linha de todos os arquivos; os testes locais e de integração não foram executados por este auditor.**

## 1. Bloqueadores comprovados (P0)

### P0-01 — Homologação desacoplada do HEAD

A versão avançou de `1.0.0-beta.9` (`6f7ffbf`) a `1.0.0-beta.10` durante a revisão. O arquivo `docs/release-status.md` confirma que a evidência Windows automatizada reportada pertence à beta.7 (`d840797b`) e que a beta.10 está sem homologação. O workflow CI da beta.10 (`37841875579`) concluiu como `cancelled`; ele não é evidência PASS. PR #45, dedicada à beta.8 Windows, não substitui certificação beta.10. Assinatura, cadeia de confiança, instalador real, testes nativos e smokes de providers/PTY/túnel são gates de distribuição independentes.

**Critério de fechamento:** congelar HEAD final, executar gates do exato SHA em Linux e Windows, controlar jobs cancelados/skipped, anexar evidência imutável por plataforma e exigir assinatura de distribuição para publicação. Não reaproveitar green checks de SHA anterior.

### P0-02 — Fluxo de entrada do terminal continua uma fronteira de risco

Os commits mais recentes corrigiram corrupção CSI/SS3, timers após detach e tratamento de JSON do Codex. `internal/control/web/handler_terminal.go` ainda combina WebSocket, framing, leases, timers, promoção para Mission, trace e leitura de PTY em um único controlador. Não há evidência de regressão nova confirmada por esta auditoria; o risco é a alta superfície de mudança em um caminho que já apresentou defeitos P0.

**Critério de fechamento:** fixtures byte-a-byte em Linux/Windows de CSI, SS3, OSC, UTF-8, JSON comum do provider, fragmentação arbitrária, detach/reconnect e shutdown. Manter `SessionHost` como autoridade final de coalescência; não reinstalar filtros de estado em cada camada.

## 2. Dívida de manutenibilidade (P1)

### P1-01 — Componentes e módulos excessivamente extensos

Contagem aproximada de linhas do código-fonte no baseline inspecionado:

| Arquivo | Linhas | Recomendação |
| --- | ---: | --- |
| `web/src/app/workspace-os.css` | 8.229 | CSS por tema/foundation/layout e módulos SCSS; conservar tokens e snapshots visuais |
| `web/src/i18n/resources.ts` | 5.906 | recursos por namespace/idioma; teste de paridade de chaves |
| `internal/tui/usage_table.go` | 2.262 | separar transformação, estado, rendering Charm e seleção |
| `web/src/types.ts` | 2.090 | tipos por domínio, API pública e dependências acíclicas |
| `internal/app/app.go` | 1.818 | comandos como composição/wiring; serviços por responsabilidade |
| `web/src/workspace/WorkspaceRenderer.tsx` | 1.625 | extrair presenters/renderers por tipo de surface |
| `web/src/nexus/AgentTerminal.tsx` | 1.562 | extrair hooks puros do transporte, keyboard, title, lifecycle |
| `web/src/app/NexusWorkspaceApp.tsx` | 1.467 | separar shell, route controller, sessão e orquestração UI |
| `internal/nexus/runner/runner.go` | 1.421 | split state machine vs checkpointing/side effects |
| `internal/nexus/mission_executor.go` | 1.363 | isolação de seleção de agente, admissão, roteamento e failover |
| `web/src/components/TerminalPane.tsx` | 838 | reutilizar mecanismos já extraídos do AgentTerminal |
| `.github/workflows/ci.yml` | 1.123 | reusable/composite actions e workflows separados por plataforma |

`AGENTS.md` e `docs/engineering/WEB_ENGINEERING_STANDARDS.md` já declaram revisão aos ~200, decomposição aos ~300 e justificativa acima de 400 linhas. Não criar um lint global bloqueador por tamanho sem baseline e exceções: isso impediria manutenção incremental. Adotar orçamento por módulo e regra de não crescimento para os legados, com planos de redução vinculados a testes.

### P1-02 — Duplicação estrutural de terminal

`web/src/nexus/AgentTerminal.tsx` e `web/src/components/TerminalPane.tsx` repetem `attachCustomKeyEventHandler`, zoom/fonte, ciclo de xterm, fit/resize, leitura de títulos, scroll e send WebSocket. A equivalência dessas implementações é uma hipótese forte sustentada pela inspeção, não uma contagem automática de clones. Extrair **somente** primitivas puras e compartilháveis (`useTerminalFontSize`, `terminalKeybindings`, `usePtyTitle`, transport adapters): contratos de reconexão e escolha de controle continuam específicos. Evitar introduzir um terceiro terminal.

### P1-03 — Regras escritas sem enforce automático

- `web/eslint.config.js`: `@typescript-eslint/no-explicit-any: off`, `@typescript-eslint/no-unused-vars: warn`, `react-hooks/exhaustive-deps: warn`; `scripts/**` excluído. Diverge do `AGENTS.md` que proíbe atalhos e `any`.
- `.golangci.yml`: `errcheck` desativado globalmente e exclusões pontuais; revisar erros ignorados em serviços/IO/transporte.
- `web/.stylelintrc.json`: múltiplas validações desativadas, inclusive `no-duplicate-selectors`. Fazer auditoria das violações reais antes de apertar regras.
- `web/tsconfig.json`: `strict: true` já existe — **não** registrar como gap.
- `web/.prettierrc`, Husky, lint-staged, commitlint, Stylelint, ESLint, Go fmt/vet já existem — **não** adicionar ferramentas equivalentes.

Recomendação: tratar regras como erro **primeiro em arquivos novos/modificados** com baseline de dívidas; ampliar a cobertura em waves, sem `--fix` global no pré-merge.

### P1-04 — CI duplicado e pouca rastreabilidade de cobertura

No `web/package.json`, `test:e2e`, `test:a11y` e `test:visual` executam o **mesmo script** `web/scripts/e2e-hardening-verify.mjs`; `.github/workflows/ci.yml` chama os três, recompilando/reexecutando fluxos completos em diferentes steps. Separar suites por selector ou executar uma vez com artefatos tipados de a11y/visual, preservando as três evidências independentes quando necessárias para branch protection. `Makefile coverage` reconhece que frontend statement coverage permanece UNVERIFIED (falta plugin de cobertura). Adotar thresholds incrementais para módulos críticos: auth, tunnel, terminal protocol, scheduler, failover/mission.

Outros pontos: scripts `web/scripts/**` ignorados pelo ESLint; muitos jobs self-hosted/Linux e certificações Windows/macOS dependem de runners e workflow_dispatch. A execução cancelada é blocker factual, não uma aprovação tácita.

### P1-05 — Classificação e testes do protocolo

`web/src/nexus/terminalProtocol.ts` implementa `ProtocolScrubber` com buffer `tail` sem teto explícito. Candidatos iniciados com `{"version":1` podem ficar retidos até próximo LF/flush. Já existe teto de 64 KiB no processamento equivalente em Go (`MaxProtocolPending`), indicando assimetria de limites. **Risco de buffering/latência, não bug demonstrado em produção.** Testar provider JSON longo/fragmentado, frame malformado e parcial antes de alterar filtro, sob pena de reintroduzir vazamento de envelopes internos.

## 3. Release e distribuição

A fonte de verdade é `VERSION`, sincronizada com `web/package.json`; a beta.10 já recebeu um ajuste concorrente na release oficial. O complemento desta PR extrai o algoritmo para funções testáveis, adiciona transação de atualização com rollback de ambos os arquivos e CI explícito de versão. A `main` deve receber **um único conjunto de commits auditado** e tests de instalação/update trust root, e não a soma de PRs em rascunho.

**Mudanças propostas nesta PR de auditoria:**

1. `scripts/release-version.mjs`: pequena função pura `nextBetaVersion` e transação `stageBetaVersionFiles` com rollback byte-a-byte dos dois arquivos.
2. `scripts/release-version.test.mjs`: valida incrementos, rejeição de versão inválida, consistência, metadados e rollback.
3. `scripts/release-beta.mjs`: delega ao helper e distingue build de instalação/publicação.
4. `.github/workflows/ci.yml`: executa regressão do bump e `verify-version-contract.go`.

**Fora do escopo desta PR:** reescrita de terminal, grandes refatorações de estado, endurecimento global do ESLint, nova release/tag. Essas mudanças exigem PRs separados, baseline de testes e gates nativos.

## 4. Gate de conclusão para a main

| Gate | Evidência exigida |
| --- | --- |
| Versionamento | `VERSION` = `web/package.json`; `scripts/verify-version-contract.go`; teste de rollback |
| Qualidade | `format:check`, ESLint, Stylelint, check:styles, `tsc`, Go fmt/vet/golangci |
| Confiabilidade | Vitest + Node tests, Go `-race`, testes de controlo PTY e regressões de auth/tunnel |
| UX | Playwright, Axe, responsividade, contraste e snapshots visuais |
| Compatibilidade | Linux nativo, Windows ConPTY/WebView2/NSIS, macOS quando declarado |
| Segurança | govulncheck, nenhum segredo em logs, isolamento de sessão/tenant, assinatura e trust root |
| Distribuição | Artefatos checksumados e assinados no exato SHA; release e manifesto verificáveis |
| Governança | revisão independente de diffs críticos, CODEOWNERS e nenhum workflow obrigatório cancelado/skipped |

**Veredito:** `NO-GO` para merge à main e distribuição beta.10. A PR de manutenção deve permanecer DRAFT até passar CI no seu próprio SHA. O relatório distingue achados estáticos confirmados de hipóteses de risco que pedem testes; não declara testes executados localmente nem certificação de provedores.
