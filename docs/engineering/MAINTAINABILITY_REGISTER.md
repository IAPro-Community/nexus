# Dívida de manutenibilidade — linha 1.0

> Registro de engenharia, não autorização para reescrever componentes durante a homologação. A correção deve preservar comportamento, contratos e evidência nativa. Limites abaixo são aproximados na revisão de outubro de 2026; recalcule-os ao iniciar cada refatoração.

| Área | Arquivo | Tamanho observado | Estratégia segura |
| --- | --- | --- | --- |
| CLI | `internal/app/app.go` | ~1.861 linhas | separar registro/dispatch de subcomandos após testes de CLI |
| Workspace | `web/src/workspace/WorkspaceRenderer.tsx` | ~1.688 | extrair seleção de janelas/estado de apresentação mantendo testes de UX |
| Terminal | `web/src/nexus/AgentTerminal.tsx` | ~1.561 | somente após testes PTY/ConPTY de scroll, CSI/SS3, focus e paste |
| UI raiz | `web/src/app/NexusWorkspaceApp.tsx` | ~1.489 | separar ciclo de sessão/rotas sem duplicar autenticação |
| Mission | `internal/nexus/runner/runner.go` | ~1.421 | refatorar etapas por contrato mantendo persistência e fencing |
| Executor | `internal/nexus/mission_executor.go` | ~1.363 | isolar adapters de execução, manter DoD/evidence |
| Traduções | `web/src/i18n/resources.ts` | ~5.934 | modularizar por namespace com verificador de keys PT/EN/ES |

## Regra de entrada

Não usar quantidade de linhas como único sinal de defeito. Primeiro demonstrar acoplamento, responsabilidade duplicada ou custo real de manutenção. Congelar a API; criar testes de caracterização e golden/contract tests para cada extração. Usar commits pequenos, rollback simples e benchmarks quando necessário.

## Gates mínimos

- Terminal: caracteres especiais, seletores, `Ctrl+C`, Unicode, mouse scroll, reconexão e lease de controle.
- Mission: `UNKNOWN` bloqueia replay, retomada durável, intervenção, DoD, isolamento e orçamento de retries.
- UI: regressão Playwright, acessibilidade, Desktop WebView e captura sintética.
- Traduções: invariantes de chaves/parametrização em três idiomas.
- Todos: Go tests/race, frontend quality, `docs-verify`, mesma revisão e smoke nativo relevante.

Nenhum arquivo acima foi refatorado nesta preparação documental. Essa é uma escolha deliberada: **sem ambiente para testar mudanças de runtime, refatoração extensiva acrescentaria risco à release 1.0**.
