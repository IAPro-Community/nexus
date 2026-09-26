# Nexus 1.0.0-beta.3 — convergência e auditoria de homologação Linux + Windows

Atualizado em **2026-09-26**.

## Veredito atual

**SOURCE_REMEDIATION_CONVERGED / NATIVE_RETEST_REQUIRED**

A branch de convergência reúne os defeitos encontrados nas homologações Linux e
Windows e os gaps adicionais descobertos durante a reauditoria. Isso ainda não
é uma certificação de plataforma: não existe evidência de uma execução nativa
beta.3 no mesmo SHA final para Linux e Windows.

Não criar tag/release estável a partir deste estado.

## Baseline

- base beta.3: `release/v1.0.0-beta.3@61b87409ba93f307ce2f95f7ca76348a310e3d95`;
- branch de convergência: `fix/beta3-homologation-convergence-20260925`;
- evidência Linux beta.2 informada na campanha local:
  `fix/beta2-linux-autopilot-20260925@e4574e4cfd4676032e2284f8a1b2a3c5885e287e`;
- remediação Windows beta.2:
  `fix/beta2-windows-autopilot-20260925-1039@a9f6dda3ff063b7b46d2260680a75d09cfa661c3`.

A beta.2 é usada somente como fonte de achados. Nenhum PASS da beta.2 é
promovido automaticamente para a beta.3.

## Auditoria dos relatórios existentes

### Windows — `DEV/validation/BETA2_HOMOLOGATION.md`

O relatório é internamente coerente e **não declara PASS**. Ele registra:

- execução inicial FAIL;
- correções de source aplicadas;
- estado final `FAIL / RETEST REQUIRED`;
- browser/E2E, Go tests, Wails/HUD e packaging ainda dependentes de novo reteste;
- screenshots/documentação exigindo recaptura real;
- NSIS/AuthentiCode ainda exigindo evidência nativa.

**Conclusão da auditoria:** relatório válido como registro de defeitos e
remediação, mas inválido como prova de aprovação Windows.

### Linux — `DEV/FINAL_VALIDATION_REPORT.md`

O arquivo encontrado na branch Linux é datado de **2026-09-01**, referencia
`feat/nexus-canonical-review-rebuild` e checkpoint `1e6404ac...`. Portanto,
ele **não é o relatório da homologação beta.2 de 25/09** e não pode certificar
nem beta.2 nem beta.3.

A campanha Linux beta.2 em `e4574e4...` permanece evidência histórica/local
informada durante a homologação, útil para reproduzir os achados, mas insuficiente
para certificar o HEAD beta.3.

**Conclusão da auditoria:** não existe no repositório um relatório Linux beta.3
same-SHA que permita marcar a plataforma como PASS.

## Correções convergidas da campanha Linux/Windows

- `.gitattributes` para line endings determinísticos;
- comandos Bun corrigidos para execução cross-platform;
- version contract alinhado a `1.0.0-beta.3` em VERSION, Web, Desktop e HUD;
- schema test deriva a maior migration embutida, sem constante 26/27/28;
- launcher standalone protegido contra sinalizar o próprio processo;
- OpenCode golden e normalização de versão alinhados;
- assertions NSIS e política de release alinhadas à beta.3;
- SCSS visually-hidden sem `clip: rect(...)`;
- HOME/USERPROFILE/XDG/Nexus data isolados na homologação Windows;
- E2E/browser e docs capture sem pressupostos GNU Make/binário extensionless;
- `nexus.exe` resolvido corretamente no Windows;
- homologador Windows com SHA esperado, toolchain pinada, lint, security,
  version contract, browser matrix e Wails Desktop/HUD;
- recaptura documental exact-SHA Windows;
- homologador Linux com exact-SHA e arquivos Go rastreados.

## Gaps adicionais descobertos nesta reauditoria e corrigidos

### 1. Testes do launcher dependiam de `sleep` Unix

Os testes de launcher/orphan usavam `exec.Command("sleep", "30")`, o que não é
hermético no Windows.

Correção:

- processo auxiliar escrito em Go, executável em Linux/Windows;
- testes migrados para esse helper;
- regressão explícita para PID/HostPID iguais ao PID do Nexus atual.

### 2. Proteção self-kill estava incompleta

A primeira correção protegia `HostPID == os.Getpid()`, mas `sess.PID` ainda
podia apontar para o processo atual.

Correção:

- `forceKillSession` ignora PID e HostPID iguais ao processo atual;
- `sessionProcessAlive` aplica a mesma regra.

### 3. Maestro sync reconhecia PowerShell, mas só executava Bash

`SyncPreview()` descobria `sync-skills.ps1`, porém `ApplySyncPreview()`
aceitava somente `.sh` e invocava `bash`.

Correção:

- `.sh` usa Bash;
- `.ps1` usa `pwsh` / `powershell.exe` / `powershell`;
- Windows PowerShell recebe `-ExecutionPolicy Bypass` apenas para o script
  oficial allowlisted;
- argumentos são enviados como argv, sem shell interpolation.

### 4. Fakes de Maestro não testavam o contrato real no Windows

`WriteFakeBinaryWithScript` ignora o corpo shell no Windows. Testes de
`advise`, `verify` e `capabilities` dependiam desse corpo.

Correção:

- novo fake CLI determinístico compilado em Go;
- matching exato de argv;
- stdout/stderr/exit code configuráveis;
- marker de invocação para testar fail-before-exec;
- testes de contrato Maestro migrados para o helper cross-platform.

O helper foi exercitado isoladamente e comprovou execução/exit-code
determinísticos; isso não substitui a suíte completa do repositório.

### 5. PRs da release não disparavam o workflow

`ci.yml` aceitava PR apenas para `main/master`, enquanto o PR de correção
tem base `release/v1.0.0-beta.3`.

Correção:

- trigger inclui `release/**`.

Foi criado o PR temporário #26 contra `main`, cujo base SHA é idêntico ao da
release beta.3, para tentar exercitar exatamente o mesmo diff. Nenhum workflow
run foi criado. A ausência de jobs é classificada como **infraestrutura/Actions
indisponível**, não PASS nem FAIL funcional.

### 6. Homologador Linux era mais fraco que o CI

O script Linux não cobria todos os gates exigidos pelo workflow.

Correção:

- Linux amd64 obrigatório;
- Go 1.25.14 e Bun 1.3.9 obrigatórios;
- trust-root canônico;
- golangci-lint v2.12.2;
- govulncheck v1.7.0;
- version contract;
- HOME/XDG/Nexus data isolados;
- full race suite;
- Agent Interop/OpenCode regressions;
- browser E2E completo: hardening, Composer, surface, capacity, HUD,
  Composer spine/vague/themes, Axe e visual;
- Desktop + HUD Wails;
- GoReleaser v2.18.0 snapshot;
- verificação da matriz Linux-only;
- DEB inspect/extract/execute smoke.

O install/uninstall real do DEB continua pertencendo ao job containerizado do
CI, evitando alterar o host de homologação.

### 7. Recaptura documental Linux faltava

Foi adicionado `scripts/refresh-beta-docs-linux.sh` com a mesma disciplina
exact-SHA da versão Windows.

## Estado das evidências beta.3

| Gate | Source | Execução beta.3 same-SHA | Estado |
| --- | --- | --- | --- |
| Version contract | corrigido | não executado no runner final | RETEST REQUIRED |
| Frontend quality/unit/build | harness pronto | não executado no SHA final | RETEST REQUIRED |
| Go vet/tests/race | harness pronto | não executado no SHA final | RETEST REQUIRED |
| golangci-lint | pinado | não executado no SHA final | RETEST REQUIRED |
| govulncheck | pinado | não executado no SHA final | RETEST REQUIRED |
| Browser/Axe/visual | portado | não executado no SHA final | RETEST REQUIRED |
| Codex terminal arrows/native TUI | correções presentes | não executado nativamente | MANUAL RETEST |
| Mission/Direct behavior | regressões presentes | não executado nativamente | MANUAL RETEST |
| OpenCode projection/enforcement | beta.3 preservada | sem provider live same-SHA | MANUAL RETEST |
| Maestro Windows parity | corrigida | sem Windows full suite | RETEST REQUIRED |
| Voice | source integrado | sem mic/STT/fallback live same-SHA | MANUAL RETEST |
| Linux Desktop/HUD | harness pronto | sem execução final | RETEST REQUIRED |
| Windows Desktop/HUD | harness pronto | sem execução final | RETEST REQUIRED |
| Linux GoReleaser/DEB | harness pronto | sem execução final | RETEST REQUIRED |
| Windows NSIS/AuthentiCode | workflow pronto | sem runner/signing final | RETEST REQUIRED |
| Docs screenshots | helpers prontos | manifesto atual ainda exige recaptura | BLOCKING |
| GitHub Actions | workflow corrigido | nenhum run criado para PR #26 | INFRA BLOCKED |

## Sequência obrigatória para fechamento

### Linux

1. checkout limpo da branch de convergência;
2. definir `NEXUS_EXPECTED_SHA=$(git rev-parse HEAD)`;
3. rodar `./scripts/refresh-beta-docs-linux.sh`;
4. revisar e commitar somente evidência visual gerada;
5. atualizar para o novo exact SHA;
6. rodar `./scripts/homologate-beta.sh`;
7. executar os checks manuais emitidos em `MANUAL_CHECKS.md`.

### Windows AMD64

1. checkout limpo do mesmo SHA final;
2. definir `NEXUS_EXPECTED_SHA` para esse SHA;
3. rodar `.\scripts\refresh-beta-docs-windows.ps1`;
4. revisar e commitar somente evidência visual gerada;
5. atualizar ambos os ambientes para o novo SHA;
6. rodar `.\scripts\homologate-beta.ps1 -ExpectedSha <SHA>`;
7. executar os checks manuais;
8. executar o job native Windows/signing do CI quando Actions estiver disponível.

## Critério de fechamento

Só alterar o veredito para PASS quando Linux e Windows reportarem o **mesmo SHA
final**, sem dirty tree, e todos os gates obrigatórios daquela plataforma
estiverem verdes.

Até lá:

- Linux beta.2 PASS histórico != beta.3 PASS;
- Windows beta.2 remediation != Windows PASS;
- cross-compile != native certification;
- source review != runtime evidence;
- ausência de CI != PASS.
