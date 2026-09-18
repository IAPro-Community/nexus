# Runtime / Recovery Red-Team Review

## Escopo e identidade

- Audited HEAD: `e53f8352e4c3647632ebac7877cf0a6a3bd62647`
- `git rev-parse HEAD` confirmou o SHA solicitado.
- `git diff --quiet HEAD -- internal/nexus internal/control/launcher` confirmou que o código de runtime/recovery inspecionado corresponde ao HEAD.
- A árvore de trabalho contém documentação/evidências não commitadas. Esses arquivos não são tratados como evidência embutida no SHA.

## Veredito

| Dimensão | Veredito |
| --- | --- |
| Recovery | **PARTIAL** |
| Overnight | **NOT_VERIFIED** |
| Resume | **PARTIAL** |
| Recovery/Overnight/Resume para produção | **FAIL** |

Não há base para `CERTIFIED` ou `PASS`: a campanha overnight foi abortada antes de executar qualquer Mission, recuperação, reinício, falha de processo ou soak. Além da ausência de evidência de produto, a revisão estática encontrou caminhos concretos de processo órfão e um watchdog que não aplica seu prazo durante a operação bloqueada.

## Evidência executada nesta revisão

Comando executado localmente no código do SHA auditado:

```text
go test -race -count=1 ./internal/nexus/runner ./internal/nexus ./internal/control/launcher
Go test: 356 passed in 3 packages
```

Isso é evidência de **testes Go locais**, não product E2E. Em particular:

- `internal/nexus/runner/watchdog_test.go` chama `ApplyProgressWatchdog` diretamente com estado sintético.
- `internal/nexus/runner/attention_e2e_test.go`, apesar do nome, usa `MemoryRunRepository` e objetos sintéticos no mesmo processo.
- `internal/nexus/mission_intervention_persistence_test.go` reabre SQLite e usa `restartProbeExecutor`; prova persistência/idempotência no nível de integração Go, não reinício real do binário, SessionHost ou provider.
- Nenhum teste acima executa Mission autenticada por uma superfície pública do produto, mata/reinicia o processo Nexus, injeta falha em provider real ou mede processos órfãos no SO.

## Achados bloqueadores

### RR-01 — HIGH — Watchdog não observa a operação enquanto ela está travada

`ApplyProgressWatchdog` só é chamado no início de `ExecuteNextStep` (`internal/nexus/runner/runner.go:198-204`). Em seguida, alocação, provider, teste ou review roda de forma síncrona. O prazo padrão do watchdog é 15 minutos, mas o timeout padrão do pacote é 60 minutos (`internal/nexus/runner/contract.go:97-104`).

Consequência: uma chamada de provider sem progresso pode ficar bloqueada por até o timeout do pacote antes de o watchdog voltar a ser avaliado. Se uma implementação não respeitar cancelamento, o watchdog pode nunca produzir `FAILED_NO_PROGRESS`. O teste atual valida somente a função pura; não valida supervisão temporal do worker.

### RR-02 — CRITICAL — Falha de handshake após spawn pode deixar SessionHost órfão

O launcher inicia um processo destacado em `internal/control/launcher/launcher.go:194-202`. Se `WaitForEndpoint`, conexão ou `Status` falhar em `launcher.go:204-224`, o caminho apenas marca o registro como `FAILED` e retorna. Não chama `Stop`, não mata o PID recém-criado e não remove o registro.

Consequência: timeout/protocolo quebrado durante startup pode deixar processo destacado vivo sem runtime utilizável. Não há teste de falha pós-spawn que prove cleanup no SO.

### RR-03 — HIGH — “Force kill” no timeout de StopAgent não existe

O comentário em `internal/nexus/nexus.go:623` afirma “force kill”, porém o ramo de timeout (`nexus.go:622-631`) apenas marca o Agent como `FAILED` e retorna erro. Nenhum sinal de kill é enviado nesse ramo. `prodLauncher.Stop` também é apenas uma chamada ao protocolo (`nexus.go:44-55`), sem fallback por PID/process group.

Consequência: um SessionHost ou provider que não responda ao stop pode permanecer vivo depois de o produto declarar falha.

### RR-04 — HIGH — Shutdown cancela workers, fecha SQLite sem join e não encerra runtimes

`Nexus.Shutdown` cancela os mission workers e fecha o store imediatamente (`internal/nexus/nexus.go:254-280`), sem aguardar `worker.done`. Isso permite que goroutines ainda tentem persistir estado depois do fechamento do SQLite. O método também não percorre runtimes registrados para encerrá-los; os hosts são deliberadamente destacados por `Setsid` no Unix e `DETACHED_PROCESS` no Windows.

Consequência: shutdown pode perder a última transição durável e deixar processos de runtime sobrevivendo ao processo principal. Não há teste de shutdown/restart com inspeção real de PID e ausência de órfãos.

## Comportamentos parcialmente sustentados

### Persistência e retomada

Há desenho defensivo útil:

- dispatch `INTENT` é persistido antes do provider e uma retomada com resultado desconhecido bloqueia duplicação;
- resolução de intervenção e pedido de resume possuem chave idempotente persistida;
- leases possuem fencing e renovação;
- `RecoverMissionRuns` reinicia runs duráveis não terminais no startup;
- `FAILED_NO_PROGRESS` é separado de `NEEDS_YOU`;
- `BLOCKED_NEEDS_USER` só retoma por opção tipada e permitida pelo contrato.

Os testes locais dão confiança parcial nessas invariantes. Não promovem o comportamento a PASS de produto porque não há crash/restart real, provider autenticado, falha de processo real, ou observação do SO.

### NEEDS_YOU e FAILED_NO_PROGRESS

A classificação é coerente no código: `BLOCKED_NEEDS_USER` vira `REQUIRE_USER`, enquanto `FAILED_NO_PROGRESS` vira notificação/in-app, nunca `NEEDS_YOU` (`internal/nexus/runner/attention_policy.go:43-66`). O watchdog também não cria intervenção humana.

Limite: essa separação foi verificada apenas em testes de estado sintético. Não existe evidência de uma Mission real atravessando ambos os caminhos pela API/UI e sobrevivendo a restart.

### Resume nativo

`RecoverAgent` classifica resume do provider como `NATIVE_RESUME_UNVERIFIED` (`internal/nexus/nexus.go:730-734,809-819`). Essa classificação é honesta. Não existe evidência de continuidade nativa autenticada para este SHA, portanto não pode ser PASS.

## Overnight: evidência de ausência

`DEV/validation/overnight-production-certification/FINAL_REPORT.md` registra:

- `ABORTED_PRECONDITION_FAILED`;
- campanhas A–F não iniciadas;
- campanha C Recovery `NOT_STARTED`;
- campanha F 8h soak `NOT_STARTED`;
- zero Missions executadas;
- nenhuma injeção de restart/recovery;
- nenhum teste de resource leak/orphan.

`campaign.json` confirma `soak_hours: 0`. `timeline.jsonl` contém somente intake, falha de precondição e abort. Isso é evidência de que a certificação **não aconteceu**, não evidência de PASS.

O relatório posterior de production finalization também declara `NOT_READY_FOR_OVERNIGHT_CERTIFICATION` e ausência de Mission autenticada. Como esse relatório está não commitado, serve apenas como contexto local adicional, não como evidência same-SHA versionada.

## Evidência necessária para sair de FAIL

1. Corrigir os caminhos RR-01 a RR-04 e executar testes de processo reais no mesmo SHA.
2. Product E2E iniciando Mission por API/UI, matando o processo Nexus durante dispatch e validando retomada sem duplicação.
3. Falhas reais de startup/handshake/stop com verificação de PID/process group e zero órfãos.
4. Reinício com `NEEDS_YOU`, resolução idempotente e retomada observada pela superfície pública.
5. Stall real que alcance `FAILED_NO_PROGRESS` dentro do prazo contratado.
6. Campanha overnight elegível de 8 horas no SHA candidato imutável, com timeline, contagem de processos/FDs/memória e resultado de recovery/resume.

## Decisão

**NO-GO para claims de produção em Recovery/Overnight/Resume.** Recovery e Resume têm cobertura unitária/in-process útil, por isso são `PARTIAL`; Overnight é `NOT_VERIFIED`; o conjunto recebe `FAIL` para promoção de produção devido à ausência de product E2E/soak e aos riscos concretos de watchdog e processos órfãos.
