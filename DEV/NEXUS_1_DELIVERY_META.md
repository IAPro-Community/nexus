# Meta de entrega — Nexus 1.0 (Autopilot)

## Resultado esperado

Entregar um Release Candidate do Nexus que aceite uma tarefa real, execute-a
sem supervisão contínua, sobreviva a falhas, preserve contexto e encerre com
verificação objetiva. Se qualquer evidência crítica faltar, o resultado é
`NO_GO`, com blocker reproduzível.

## Estratégia de execução

O Autopilot conduz as fases `expansion → planning → execution → qa → validation → cleanup`.
O agente líder mantém arquitetura, integração, conflitos, ledger e decisão
final. Subagentes trabalham apenas em lanes com ownership disjunto:

1. **Evidência/release** — audita P0 bloqueados, same-SHA, native CI e packaging.
2. **Resiliência/testes** — fecha fixtures determinísticas de crash, restart,
   provider fallback, artifacts e overnight.
3. **UX/E2E** — valida Resume-first, Needs You, Composer/Flow, teclado, Axe e
   breakpoints.
4. **Revisão** — funcionalidade, segurança e qualidade do diff.

Cada lane retorna paths, comandos, resultado, blocker e risco residual. Nenhum
subagente pode sobrescrever ou reverter trabalho de outro.

## Gates por fase

| Fase | Entrega obrigatória | Gate |
|---|---|---|
| Expansion | spec, não-objetivos, riscos e critérios | `.omx/autopilot/nexus-1-finalization/checkpoint.md` |
| Planning | tarefas com owner, dependências e verify | `tasks.md` completo |
| Execution | implementação/testes por lane | testes focados verdes |
| QA | ciclos de correção | frontend verify, Go test/race/vet, quality/security |
| Validation | aprovações multi-perspectiva | sem findings críticos abertos |
| Release | mesmo SHA em Linux/Windows/macOS/Desktop | matriz imutável + artifacts |
| Dogfood | missão real no próprio Nexus | timeline, agents, artifacts, recovery e DoD |

## Critério de saída

`GO` somente com todos os P0 críticos PASS, dogfooding e overnight PASS, crashes
recuperáveis, restart reconciliado, worktrees seguros, verification anti-falso-DONE,
native Windows/macOS conforme claim público e artifacts do mesmo SHA. Caso
contrário, publicar `CONDITIONAL_GO` ou `NO_GO` — nunca mascarar ausência de
ambiente como sucesso.

## Estado inicial desta meta

Implementação local e gates Linux/Web/Go estão verdes. O ledger atual mantém
`NO_GO` por dogfooding real, provider/crash autenticados, same-SHA CI e execução
nativa Windows/macOS ainda não comprovados. Consulte
[`DEV/NEXUS_1_FINAL_ACCEPTANCE.md`](NEXUS_1_FINAL_ACCEPTANCE.md).

## Blockers externos com owner

| Blocker | Owner | Ambiente necessário | Evidência mínima |
|---|---|---|---|
| Provider/quota/crash recovery | maintainer de credenciais/CI | provider fake ou sandbox autenticado | timeline da geração anterior, checkpoint, nova generation e resultado verificado |
| Windows nativo | maintainer CI Windows | runner Windows com ConPTY, Named Pipes e Desktop | job no SHA candidato, logs e smoke `nexus doctor/version/providers/usage` |
| macOS nativo | maintainer CI macOS | runner macOS com PTY, sockets e Desktop | job no SHA candidato, logs e smoke equivalente |
| Dogfooding | maintainer Nexus | workspace Nexus + provider configurado | mission ID, agents, worktrees, artifacts, falhas, recovery, DoD e diff |

Enquanto esses owners não anexarem a evidência mínima, o Autopilot continua
executando as lanes locais e mantém o release em `NO_GO`.
