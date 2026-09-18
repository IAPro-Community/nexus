# Revisão independente Windows / release

## Veredito da plataforma

**NOT_VERIFIED**

**Gate de certificação de produção: FAIL.**

O candidato `e53f8352e4c3647632ebac7877cf0a6a3bd62647` não pode ser declarado
“Windows certified”. Não existe evidência de execução nativa Windows para esse
SHA. O host desta revisão é Linux e nenhum resultado local foi promovido a
evidência Windows.

## Identidade e publicação do candidato

- SHA auditado: `e53f8352e4c3647632ebac7877cf0a6a3bd62647`
  (`fix(release): close production certification blockers`).
- O checkout já havia avançado para `37f970b82605ca5c528d39c5dd4d6d0167fa4be5`;
  os arquivos Windows/release relevantes não mudaram entre os dois commits.
- O branch remoto `origin/feat/nexus-maximum-delivery` aponta para
  `d15aa712dd4da433e5d0bad01c169205197865c6`.
- `git branch -r --contains e53f835...` não retornou branch remoto.
- `git ls-remote --heads origin` não contém o SHA auditado.
- `gh run list --commit e53f835...` retornou `[]`.
- A API `actions/runs?head_sha=e53f835...` retornou
  `{"total_count":0,"runs":[]}`.
- A API de check-runs respondeu `422 No commit found for SHA`, coerente com o
  commit ainda não publicado no GitHub.

## Achados

### BLOCKER — Ausência de execução Windows same-SHA

Não há run, job, log, artifact ou check do GitHub Actions para o SHA auditado.
Logo, não existe prova nativa para Go/vet/test, ConPTY, Named Pipe, PowerShell,
Wails, NSIS, smoke ou updater. A presença de jobs no YAML é evidência estática,
não certificação de plataforma.

### HIGH — O gate ConPTY não prova que ConPTY foi usado

O backend Windows pode cair deliberadamente para `standard pipes (ConPTY
unavailable)`. Os testes de host exercitam PowerShell e Named Pipe, mas não
afirmam `Mechanism() == "ConPTY (CreatePseudoConsole)"` nem
`SupportsResize() == true`. O teste direto de execução do backend em
`internal/control/terminal/terminal_test.go` está excluído de Windows por
`//go:build !windows`. Portanto, até um futuro run verde do job chamado
“ConPTY & Named Pipe E2E” poderá passar pelo fallback sem provar ConPTY real,
resize real ou raw mode.

### HIGH — Wails é compilado, mas não iniciado no Windows

O job `desktop-windows` compila e empacota `nexus-desktop.exe`. O smoke do NSIS
confirma que o arquivo foi instalado, porém executa somente `nexus.exe version`;
não inicia o desktop, não verifica criação da janela/WebView2 e não captura
falha de startup. Não há smoke runtime do Wails Windows.

### HIGH — `install.ps1` não possui smoke funcional nativo

O job Windows apenas analisa a sintaxe do script. Ele não executa o fluxo de
download, assinatura, checksum, extração, PATH, atalho ou `doctor`.
Além disso, `Extract-ShaFromManifest` chama `python3` sem verificar sua
disponibilidade, apesar do caminho ser descrito como “Zero-Toolchain Release”.
Falta prova em uma imagem Windows limpa, sem Python/OpenSSL pré-instalado.

### HIGH — Updater não foi provado contra um executável Windows em uso

O updater substitui o binário com `os.Rename`. Os testes usam arquivos
temporários inertes; não exercitam atualização do `nexus.exe` atualmente em
execução, bloqueio de arquivo do Windows, processo auxiliar, reinício ou
rollback pós-falha real. Não há teste Windows específico para esse contrato.

### MEDIUM — NSIS e artefatos de release não possuem evidência executada

O workflow define construção NSIS, instalação silenciosa, verificação dos três
executáveis, execução do CLI e desinstalação. Isso é um desenho de gate
relevante, mas não foi executado para o candidato. Também não há Authenticode
do instalador ou dos executáveis; attestation/checksum de release não substitui
assinatura de código Windows.

### MEDIUM — Cross-compile Linux participa da cadeia Windows

O snapshot GoReleaser roda em `ubuntu-latest` e produz
`nexus_Windows_x86_64.zip` e `nexus_Windows_arm64.zip`. O NSIS consome o CLI
Windows desse snapshot cross-compilado. O workflow também define build/test
nativo em `windows-latest` e o NSIS executaria o CLI instalado, portanto o
cross-compile não é, por desenho, o único gate. Porém, sem o run same-SHA, na
prática só há definição estática e eventual build Linux local; isso não pode
substituir execução nativa.

### MEDIUM — Windows ARM64 é apenas cross-build

O snapshot exige o ZIP Windows ARM64, mas não há job Windows ARM64 nativo,
smoke, Wails ARM64 ou NSIS ARM64. A matriz nativa usa apenas
`windows-latest` e o instalador NSIS é exclusivamente `x86_64`.

## Cobertura estática observada

O CI define jobs nativos `windows-latest` para:

- Go vet/test, pacotes de terminal/protocolo e runtime/web;
- build e smoke básico do `nexus.exe`;
- build Wails Windows;
- construção, instalação e desinstalação do NSIS.

O release workflow contém um gate de CI same-SHA e exige os jobs Windows,
Wails e NSIS antes de promover artifacts. Esse desenho é positivo, mas nunca
foi executado para o SHA auditado e, portanto, não altera o veredito.

## Evidência obrigatória ausente

Para reavaliar a certificação Windows, é necessário no mínimo:

1. publicar exatamente `e53f8352e4c3647632ebac7877cf0a6a3bd62647`;
2. obter um run completo e verde de `ci.yml` com `headSha` exatamente igual;
3. preservar logs/artifacts dos jobs Windows E2E, Wails e NSIS desse run;
4. afirmar em teste que o backend efetivo é ConPTY, com I/O e resize, sem
   aceitar o fallback como sucesso do gate ConPTY;
5. iniciar o Wails instalado e provar startup/WebView2 em Windows;
6. executar `install.ps1` ponta a ponta em Windows limpo;
7. exercitar update e rollback reais do executável Windows em uso;
8. definir e verificar a política de Authenticode;
9. se Windows ARM64 fizer parte da promessa de produção, executar smoke nativo
   ARM64 do CLI/desktop/installer.

Até essas evidências existirem para o mesmo SHA, o resultado permanece
**NOT_VERIFIED** e a certificação de produção permanece **FAIL**.
