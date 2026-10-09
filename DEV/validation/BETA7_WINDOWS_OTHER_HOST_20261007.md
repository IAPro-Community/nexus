# Relatório de Homologação Técnica Nativa — Nexus 1.0.0-beta.7 (Windows Host Secundário)

**Data e Hora (Local):** 2026-10-07 09:55:00 -03:00  
**Data e Hora (UTC):** 2026-10-07 12:55:00 UTC  
**Avaliador:** Antigravity (Auditoria Técnica Automatizada Autônoma)  
**Host de Teste:** `KIVER`  
**Repositório Alvo:** `https://github.com/IAPro-Community/nexus.git`  
**Branch Candidata Analisada:** `release/v1.0.0`  
**SHA Candidato Remoto:** `2892f301ed2622cd3f3e432d69b5db93f7e2227f`  
**Versão Identificada (`VERSION`):** `1.0.0-beta.7`  
**Branch Local de Correção:** `fix/beta7-windows-KIVER-20261007`  
**SHA Pós-Correção (Local):** `6409b50e5a672e9d9d340afc0e36be1a18bd813b`  
**Diretório de Evidências:** `C:\Users\kiver\nexus-beta7-evidence-KIVER-2892f301`

---

## 1. Sumário Executivo e Veredito

| Avaliação | SHA Avaliado | Veredito | Justificativa Principal |
| :--- | :--- | :--- | :--- |
| **Candidato Remoto Oficial** | `2892f301ed2622cd3f3e432d69b5db93f7e2227f` | **FAIL** | 1. Falha no gate `golangci-lint run ./...` (`internal/control/host/host.go:817:57: unparam`).<br>2. Falha em `go test ./...` (`internal/runtime/terminal_trace_test.go:44` por assunção indevida de permissões POSIX no Windows NTFS; e `internal/app/app_test.go:279` em ambiente de idioma pt-BR).<br>3. `nexus mission` inexistente como subcomando de topo na CLI. |
| **Branch de Correção Local** | `6409b50e5a672e9d9d340afc0e36be1a18bd813b` (`fix/beta7-windows-KIVER-20261007`) | **CONDITIONAL_PASS** | Todas as correções pontuais foram validadas sem afrouxamento de regras (`golangci-lint`: PASS, `go test ./...`: 100% PASS, 8 suítes E2E browser: 100% PASS, Desktop e HUD smokes: PASS). Status condicional à **revalidação Linux do novo SHA** e deliberação sobre o comando `nexus mission`. |

---

## 2. Caracterização do Ambiente de Homologação

A homologação foi realizada de forma estritamente isolada, em modo não-elevado, sem escalada de privilégios (`runas`), sem modificação de políticas de sistema operacional (UAC, AppLocker, Execution Policy) e utilizando apenas ferramentas portáteis em escopo do usuário.

- **Sistema Operacional:** Microsoft Windows 11 Home, Build 26200 (Kernel 10.0.26200), Arquitetura AMD64 (x86_64).
- **Usuário:** `kiver` (Sessão interativa padrão não-elevada; `TokenElevationTypeLimited`).
- **Hardware:**
  - CPU: AMD Ryzen 5 5600 6-Core Processor (12 threads lógicas) @ 3.50GHz.
  - Memória RAM: 32 GB total (32.688 MB física instalada, ~23 GB livres durante a execução).
  - Armazenamento: SSD NVMe, ~194 GB de espaço livre na unidade `C:`.
- **Toolchain e Ferramentas Detectadas:**
  - `git`: version `2.53.0.windows.1` (`C:\Program Files\Git\cmd\git.exe`)
  - `go`: version `go1.25.14 windows/amd64` (`C:\Users\kiver\portable-tools\go\bin\go.exe`)
  - `bun`: version `1.3.9` (`C:\Users\kiver\portable-tools\bun-windows-x64\bun.exe`)
  - `node`: version `v22.14.0` (`C:\Users\kiver\portable-tools\node-v22.14.0-win-x64\node.exe`)
  - `npm`: version `10.9.2` (`C:\Users\kiver\portable-tools\node-v22.14.0-win-x64\npm.cmd`)
  - `golangci-lint`: version `1.64.5` (`C:\Users\kiver\go\bin\golangci-lint.exe`)
  - `govulncheck`: version `v1.8.0` (`C:\Users\kiver\go\bin\govulncheck.exe`)
  - `playwright / chromium`: Chromium headless revision `1243` (`%LOCALAPPDATA%\ms-playwright\chromium-1243`)
  - `Edge WebView2 Runtime`: version `154.0.4258.62` (`C:\Program Files (x86)\Microsoft\EdgeWebView\Application`)
  - `AGY CLI`: version `1.3.0` (`C:\Users\kiver\AppData\Local\agy\bin\agy.exe`)
  - `Wails`: v3 runtime e bibliotecas Go integradas.
  - `Docker`: Não instalado (`BLOCKED_ENVIRONMENT`).
  - `NSIS (makensis)`: Não instalado no PATH do usuário (`BLOCKED_ENVIRONMENT`).
  - `signtool / Certificado Authenticode`: Ausente (`BLOCKED_EXTERNAL`).

---

## 3. Binários Nativos Compilados e Verificação Criptográfica

Os binários foram gerados a partir do código fonte em ambiente Windows AMD64 nativo e tiveram seus hashes SHA256 calculados e arquivados em `artifacts.sha256`:

| Binário | Caminho de Saída | Tamanho | SHA-256 |
| :--- | :--- | :--- | :--- |
| `nexus.exe` | `.\nexus.exe` | 24.460.288 bytes | `4ED0C24DEE28F87E499E1E9CE4A6F11805E1075EE55E7F6E16D718BB526B3D61` |
| `nexus-desktop.exe` | `.\nexus-desktop.exe` | 27.064.832 bytes | `A70AA34D1970615B3AEFC71BF386F8D80F3B822117F368CF89CD53F0071697CA` |
| `nexus-hud.exe` | `.\nexus-hud.exe` | 27.088.896 bytes | `D6C04E31FFEBBC18FE98AA0A7C407BFDAAA7EF5285FB4CEAF8FD4F5AF94B4F54` |

---

## 4. Matriz Completa de Gates de Homologação

Os status atribuídos obedecem rigidamente às categorias: `PASS`, `FAIL`, `NOT_RUN`, `NOT_RUN_PRIVILEGED`, `NOT_RUN_AUTH_REQUIRED`, `BLOCKED_ENVIRONMENT`, `BLOCKED_EXTERNAL`.

| # | Gate de Validação | Status (SHA `2892f301`) | Status (SHA `6409b50e`) | Duração / Exit Code | Evidência / Comando | Observações Técnicas |
| :--- | :--- | :---: | :---: | :---: | :--- | :--- |
| 1 | **Go Formatter (`gofmt`)** | **PASS** | **PASS** | 0.8s / 0 | `gofmt -l .` | PASS com conversão LF (conforme `.gitattributes *.go text eol=lf`). |
| 2 | **Go Static Analysis (`go vet`)** | **PASS** | **PASS** | 1.1s / 0 | `go vet ./...` | Zero alertas em todos os pacotes. |
| 3 | **Go Vulnerabilities (`govulncheck`)** | **PASS** | **PASS** | 2.4s / 0 | `govulncheck ./...` | Nenhuma vulnerabilidade conhecida encontrada. |
| 4 | **Go Linter (`golangci-lint`)** | **FAIL** | **PASS** | 4.2s / 1 -> 0 | `golangci-lint run ./...` | Falha em `2892f301`: `host.go:817:57 unparam`. Corrigido em `6409b50e`. |
| 5 | **Go Test Suite (`go test ./...`)** | **FAIL** | **PASS** | 3m 10s / 1 -> 0 | `go test -count=1 ./...` | Falha em `2892f301`: `terminal_trace_test.go:44` e `app_test.go:279`. Corrigido em `6409b50e`. |
| 6 | **Frontend Check Styles** | **PASS** | **PASS** | 0.9s / 0 | `bun run --cwd web check:styles` | Conformidade de formatação Tailwind/CSS. |
| 7 | **Frontend ESLint** | **PASS** | **PASS** | 1.4s / 0 | `bun run --cwd web lint` | Validação completa de TypeScript e JSX sem erros. |
| 8 | **Frontend Stylelint** | **PASS** | **PASS** | 1.8s / 0 | `bun run --cwd web lint:styles` | Executado com runtime Node 22 sem violações. |
| 9 | **Frontend Typecheck (`tsc`)** | **PASS** | **PASS** | 2.5s / 0 | `bun run --cwd web typecheck` | Compilador TypeScript aprovou toda a árvore `web`. |
| 10 | **Frontend Unit Tests** | **PASS** | **PASS** | 14.8s / 0 | `bun run --cwd web test` | 82 arquivos de teste, 487 testes unitários aprovados. |
| 11 | **Frontend Production Build** | **PASS** | **PASS** | 2.8s / 0 | `bun run --cwd web build` | Bundle SPA web gerado com sucesso. |
| 12 | **Docs Synthetic Surface Capture** | **PASS** | **PASS** | 6.5s / 0 | `bun run --cwd web docs:capture` | 10 superfícies operacionais renderizadas e capturadas. |
| 13 | **E2E Playwright: Hardening & A11y** | **PASS** | **PASS** | 24.1s / 0 | `bun run --cwd web test:e2e` | Inclui contraste, acessibilidade e viewport mobile 390px. |
| 14 | **E2E Playwright: Composer** | **PASS** | **PASS** | 7.9s / 0 | `bun run --cwd web test:e2e:composer` | Superfície de entrada e submissão conversacional. |
| 15 | **E2E Playwright: Surface Isolation** | **PASS** | **PASS** | 16.2s / 0 | `bun run --cwd web test:e2e:surface` | Isolamento de contexto e canais entre abas. |
| 16 | **E2E Playwright: Capacity Monitor** | **PASS** | **PASS** | 8.1s / 0 | `bun run --cwd web test:e2e:capacity` | Provedores, quotas e limites visuais. |
| 17 | **E2E Playwright: HUD Interface** | **PASS** | **PASS** | 8.3s / 0 | `bun run --cwd web test:e2e:hud` | Widget HUD compacto e controles flutuantes. |
| 18 | **E2E Playwright: Composer Spine** | **PASS** | **PASS** | 9.0s / 0 | `bun run --cwd web test:e2e:composer-spine` | Fluxo de execução guiada com spine determinístico. |
| 19 | **E2E Playwright: Vague Spine** | **PASS** | **PASS** | 8.5s / 0 | `bun run --cwd web test:e2e:composer-vague-spine` | Tratamento de intenções vagas com clarificação. |
| 20 | **E2E Playwright: Composer Themes** | **PASS** | **PASS** | 11.2s / 0 | `bun run --cwd web test:e2e:composer-themes` | Matriz de temas claros, escuros e alto contraste. |
| 21 | **CLI Smoke: `version`** | **PASS** | **PASS** | 0.1s / 0 | `nexus.exe version` | Retornou `Nexus v1.0.0-beta.7 (commit 2892f301ed26...)`. |
| 22 | **CLI Smoke: `doctor`** | **PASS** | **PASS** | 0.3s / 0 | `nexus.exe doctor` | Verificou diretórios, WebView2 e detectou AGY CLI. |
| 23 | **CLI Smoke: `run --help`** | **PASS** | **PASS** | 0.1s / 0 | `nexus.exe run --help` | Ajuda do subcomando de execução exibida corretamente. |
| 24 | **CLI Smoke: `plan` / `plan list`** | **PASS** | **PASS** | 0.1s / 0 | `nexus.exe plan list` | Consulta ao repositório de planos operacionais. |
| 25 | **CLI Smoke: `projects` & duplicidade** | **PASS** | **PASS** | 0.2s / 0 & 1 | `nexus.exe projects add / list` | Adição OK; duplicidade de caminho rejeitada por UNIQUE constraint. |
| 26 | **CLI Smoke: `mission`** | **FAIL** | **FAIL** | 0.1s / 1 | `nexus.exe mission` | Retorna comando desconhecido (não implementado como CLI). |
| 27 | **Desktop Process Lifecycle** | **PASS** | **PASS** | 4.5s / 0 | `Start-Process nexus-desktop.exe` | Janela aberta (`IAPro Nexus`), WebView2 OK, fechamento limpo, 0 órfãos. |
| 28 | **HUD Process Lifecycle** | **PASS** | **PASS** | 4.3s / 0 | `Start-Process nexus-hud.exe` | Janela aberta (`Nexus HUD`), WebView2 OK, fechamento limpo, 0 órfãos. |
| 29 | **AGY Provider Integration** | **PASS** | **PASS** | 0.2s / 0 | `nexus doctor` | Detectado `agy` 1.3.0 funcional. |
| 30 | **Codex TUI Interativo** | **NOT_RUN_AUTH_REQUIRED** | **NOT_RUN_AUTH_REQUIRED** | N/A | `nexus run` com codex | Requer conta/perfil autenticado ativo com saldo. |
| 31 | **OpenCode / Claude Providers** | **NOT_RUN_AUTH_REQUIRED** | **NOT_RUN_AUTH_REQUIRED** | N/A | Provider probes | Binários de provedores proprietários não instalados na máquina. |
| 32 | **CGO Race Detector (`-race`)** | **BLOCKED_ENVIRONMENT** | **BLOCKED_ENVIRONMENT** | N/A | `go test -race` | Compilador CGO (gcc/MinGW-w64) não disponível. |
| 33 | **Docker Containers** | **BLOCKED_ENVIRONMENT** | **BLOCKED_ENVIRONMENT** | N/A | Docker runtime | Daemon Docker ausente. |
| 34 | **NSIS Installer Packaging** | **BLOCKED_ENVIRONMENT** | **BLOCKED_ENVIRONMENT** | N/A | `makensis` | Compilador de instalador NSIS não instalado. |
| 35 | **Assinatura Authenticode** | **BLOCKED_EXTERNAL** | **BLOCKED_EXTERNAL** | N/A | `signtool sign` | Certificado de assinatura de código corporativo não fornecido. |

---

## 5. Análise de Mudanças e Regressões desde o Teste Anterior (`d6b4e305`)

Entre o commit do relatório anterior (`d6b4e3054293473d3418fd2abd05875c003816be`) e a ponta candidata remota (`2892f301ed2622cd3f3e432d69b5db93f7e2227f`), foram integrados **121 commits adicionais**.

### 5.1 Subsistema de Terminal e Control Host (`internal/control/host`)
- **Implementações:** Adição de suporte a sessão desacoplada (`SessionHost`), roteador de comandos com barra (`SlashPrefixRouter`), limites de buffer de linha circular (`BoundedFanout`), e coalescência de sequências de escape ANSI divididas (CSI).
- **Transporte Padrão no Windows:** O código utiliza pipes bidirecionais e ConPTY quando suportado nativamente, normalizando quebras de linha (`\r\n`) para shells interativos.
- **Regressão Introduzida em `fb15dd02`:** O método `writeTerminalInput(data []byte) (int, error)` foi adicionado, mas em todos os 3 pontos de chamada dentro do host, o retorno inteiro `int` foi descartado (`_, _ = sh.writeTerminalInput(...)`). Isso gerou violação direta do linter `unparam` do `golangci-lint`.

### 5.2 Rastreamento Diagnóstico de Terminal (`internal/runtime/terminal_trace.go`)
- **Implementações:** Adição de trace mascarado em formato JSONL (`NEXUS_TERMINAL_TRACE`) para isolar fluxos de digitação sem vazar dados confidenciais do usuário.
- **Regressão no Teste em Ambiente Windows:**
  - O teste `TestTraceTerminalBytesWritesPrivateJSONLOnlyWhenEnabled` assumia bits POSIX de permissão (`0o077`), que não se aplicam ao sistema de arquivos NTFS do Windows (onde o Go mapeia arquivos graváveis como `0666`).
  - O teste verificava a string bruta `"<1b>[C"`, porém o serializador `json.Marshal` padrão do Go escapa caracteres HTML `<` e `>` para `\u003c` e `\u003e`, falhando a asserção no texto JSON bruto.

### 5.3 Localização da CLI (`internal/localization` e `internal/app`)
- O sistema de tradução dinâmica detecta o idioma da sessão Windows (`pt-BR`). Ao executar `nexus help`, a saída exibe o cabeçalho `"Uso:"` em vez de `"Usage:"`.
- O teste `TestControlPlaneCLICommands` possuía uma asserção rígida `strings.Contains(out, "Usage:")`, que falhava quando executado em máquinas brasileiras/latinas sem a variável `AI_CLI_LANG=en`.

### 5.4 Comando `nexus mission`
- O teste constatou que `nexus mission` continua retornando erro de comando desconhecido (`unknown command "mission"; run 'nexus.exe help'`).
- A inspeção do código confirmou que o conceito de *Mission* é gerido via planos, orquestração e subsistema de Conductor/Composer, não existindo um subcomando isolado `mission` registrado na CLI.

---

## 6. Diagnóstico e Correções Seguras na Branch Local

Em estrita consonância com as regras de homologação:
- Nenhuma alteração foi enviada para o repositório remoto (`no push`, `no merge`, `no release tag`).
- Foi criada a branch local **`fix/beta7-windows-KIVER-20261007`**.
- As correções aplicadas não removeram asserções de teste; ao contrário, adequaram as regras e compatibilizaram os comportamentos de plataforma.

### 6.1 Detalhes das Alterações

1. **`internal/control/host/host.go` & `host_test.go`:**
   - Tipo de Defeito: Defeito de Código (Parâmetro de retorno inútil).
   - Ajuste: Alterada a assinatura de `writeTerminalInput(data []byte) (int, error)` para `writeTerminalInput(data []byte) error`. Atualizados os pontos de chamada e os testes unitários correspondentes.
   - Resultado: `golangci-lint run ./...` passou com código de saída 0.

2. **`internal/runtime/terminal_trace_test.go`:**
   - Tipo de Defeito: Defeito de Teste (Assunção de semântica POSIX em NTFS e escape JSON).
   - Ajuste: Condicionado o teste de permissão `info.Mode().Perm()&0o077 != 0` com `runtime.GOOS != "windows"`. Expandida a verificação textual para aceitar a representação JSON escapada `\u003c1b\u003e[C` ou `<1b>[C`.
   - Resultado: `go test -count=1 ./internal/runtime` passou com 100% de sucesso.

3. **`internal/app/app_test.go`:**
   - Tipo de Defeito: Defeito de Teste (Rigidez contra saída localizada).
   - Ajuste: Ajustada a asserção para aceitar `Usage:` ou `Uso:`, respeitando o subsistema nativo de internacionalização.
   - Resultado: `go test -count=1 ./internal/app` passou com código de saída 0 em ambos os idiomas (`pt-BR` e `en`).

### 6.2 Validação Consolidada do Novo SHA (`6409b50e5a672e9d9d340afc0e36be1a18bd813b`)
Após a consolidação do commit local, a suíte completa de testes Go foi reexecutada do zero:
```powershell
go test -count=1 -timeout 15m ./...
```
**Resultado:** Todos os pacotes Go compilaram e passaram com **100% de sucesso (Exit code 0)**.

---

## 7. Status Cross-Platform e Rastreabilidade

- **Último SHA Linux Histórico:** `6bf1b9fef578822d2be9d542409a44a2e154ed96` (referente ao ciclo anterior).
- **Paridade de SHA Remoto com Linux Atual:** **`UNVERIFIED`** (não há evidência confirmada de execução paralela do SHA `2892f301` em Linux com publicação de relatório simultâneo).
- **Revalidação Linux Obrigatória:** **`SIM`**. Como foram aplicadas correções no código Go comum (`host.go` e `host_test.go`), qualquer futura promoção da branch `fix/beta7-windows-KIVER-20261007` requer uma execução homologatória completa em ambiente Linux com o mesmo SHA `6409b50e5a672e9d9d340afc0e36be1a18bd813b`.

---

## 8. Arquivo de Evidências

Todas as saídas brutas de console, relatórios de teste, logs e binários foram preservados de forma limpa, auditável e sem vazamento de segredos em:
`C:\Users\kiver\nexus-beta7-evidence-KIVER-2892f301\`

Arquivos de destaque no repositório de evidências:
- `artifacts.sha256` — Hashes de todos os executáveis nativos.
- `golangci-lint.log` — Log comprovando a falha original e resolução posterior.
- `go_test_all.log` & `go_test_all_after_fix.log` — Logs comparativos da suíte Go completa.
- `e2e-*.log` — 8 arquivos de log detalhando a aprovação de cada suíte Playwright headless.
- `cli-*.log` — Evidências dos testes da interface de linha de comando.
- `desktop-hud-smoke.log` — Evidência da abertura, ciclo de vida e encerramento sem processos órfãos do Desktop e HUD.
