# Resultados dos Testes de Homologação — Nexus 1.0.0 (Windows)

- **Commit SHA:** `560a83b20c62611734e934d980b5725c3ef24cf1`
- **Branch:** `release/v1.0.0`
- **Data:** 2026-10-18 / 2026-10-19

---

## Matriz Geral de Gates Automatizados

| Gate | Comando | Exit Code | Duração | Status | Evidência / Log |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Go Domain Tests** | `go test ./internal/control/web ./internal/nexus/... ./internal/doctor/...` | 0 | 565s | **PASS** | `logs/02-go-domain-tests.log` |
| **Web Typecheck** | `bun run --cwd web typecheck` | 2 | 29s | **FAIL** | `logs/03-web-typecheck.log` (TS2551 em `NexusHud.tsx:533`) |
| **Web Lint** | `bun run --cwd web lint` | 0 | 12s | **PASS** | `logs/04-web-lint.log` |
| **Web Test** | `bun run --cwd web test` | 0 | 14s | **PASS** | `logs/05-web-test.log` (90 arquivos, 565 testes) |
| **Web Build** | `bun run --cwd web build` | 0 | 12s | **PASS** | `logs/06-web-build.log` (bundle gerado com sucesso) |
| **Tokenbar Certification** | `.\scripts\certify-tokenbar-integration.ps1` | 1 | 5s | **FAIL** | `logs/07-certify-tokenbar.log` (`capacity_lifecycle_test.go` sem gofmt) |
| **Beta Homologation Script** | `.\scripts\homologate-beta.ps1` | 1 | 1s | **BLOCKED** | `logs/08-homologate-beta.log` (exige `go1.26.9`, máquina tem `go1.25.14`) |
| **PowerShell Syntax** | AST Parser em todos os `.ps1` | 0 | 146ms | **PASS** | `logs/09-powershell-syntax.log` |
| **Uninstaller Isolation (PS 5.1)** | `powershell scripts/test-uninstall-windows.ps1` | 1 | 1.2s | **FAIL** | `logs/10-uninstall-windows-smoke.log` (`$IsWindows` indefinido em PS 5.1) |
| **Uninstaller Isolation (pwsh 7)**| `pwsh scripts/test-uninstall-windows.ps1` | 0 | 1.1s | **PASS** | Terminal stdout (PASS com pwsh 7) |
| **Docs Verification** | `bun scripts/docs-verify.mjs` | 1 | 2.4s | **FAIL** | `logs/11-docs-verify.log` (visual manifest stale) |
| **Tracked Go Files gofmt** | `gofmt -l <tracked .go files>` | 1 | 22s | **FAIL** | `logs/12-gofmt-check.log` (`capacity_lifecycle_test.go:121`) |
| **Go Vet** | `go vet ./...` | 0 | 34s | **PASS** | `logs/13-go-vet.log` |
| **Version Contract** | `go run ./scripts/verify-version-contract.go .` | 0 | 1.5s | **PASS** | `logs/14-version-contract.log` (`1.0.0-beta.10` consistente) |
| **Terminal Regression Preflight**| `go test -count=1 ./internal/app ./internal/nexus/promotion ./internal/runtime` | 0 | 32s | **PASS** | `logs/15-regression-terminal.log` |
| **Windows ConPTY Tests** | `go test -v ./internal/control/terminal -run ConPTY` | 0 | 1.4s | **PASS** | Terminal stdout (6 subtestes aprovados) |
| **Terminal Escape Coalescence**| `go test -v ./internal/runtime -run "Escape\|ConPTY"` | 0 | 1.4s | **PASS** | Terminal stdout (5 subtestes aprovados) |
| **SessionHost ConPTY Arrow Input**| `go test -v ./internal/control/host -run "Arrow\|Terminal"` | 0 | 4.4s | **PASS** | Terminal stdout (4 subtestes aprovados) |
| **Windows Desktop & HUD Tests** | `go test -v ./internal/desktop/...` | 0 | 1.3s | **PASS** | Terminal stdout (19 subtestes aprovados) |

---

## Fase 3 — Instalação Limpa em Sandbox Descartável

| Teste | Condição | Resultado | Detalhes |
| :--- | :--- | :--- | :--- |
| **Compilação Binários Candidatos** | `cmd/nexus`, `cmd/nexus-desktop`, `cmd/nexus-hud` | **PASS** | Binários compilados com ldflags de versão e commit `560a83b2`. |
| **Assinatura / Download Remoto** | `.\install.ps1 -Version latest` | **FAIL / 404** | Endpoint de releases remoto `IAPro-Community/nexus-releases` retorna 404 (sem assets publicados). |
| **Instalação com `-InstallDir`** | `.\install.ps1 -BuildFromSource -InstallDir <custom>` | **FAIL / CRASH** | `install.ps1:204`: `Test-Path ""` falha com `ParameterArgumentValidationErrorEmptyStringNotAllowed`. |
| **Instalação Caminho Padrão (Isolada)**| `.\install.ps1 -BuildFromSource` com `$env:LOCALAPPDATA` sandbox | **PASS** | Instala com sucesso `nexus.exe`, `ai.exe`, `.nexus-managed` e executa `nexus doctor`. |
| **Desinstalador em Diretório Gerenciado** | `.\uninstall.ps1 -InstallDir <managed>` | **PASS** | Remove binários Nexus e preserva arquivos de projeto e dados do usuário. |

---

## Fase 4 — Experiência de Primeira Execução

| Cenário | Descrição do Teste | Resultado | Evidência / Log |
| :--- | :--- | :--- | :--- |
| **Cenário A: Nenhum provedor** | Sandbox isolado, PATH limpo (`C:\Windows\system32`), zero provedores instalados. | **PASS** | Permitiu cadastrar projeto sem provedor. `providers status` e `doctor` reportam ausência de ferramentas sem inventar tokens ou cotas. |
| **Cenário B: Codex autenticado** | Reconhecimento de CLI Codex instalada no sistema. | **PASS** | Detectou `codex-cli 0.163.0` em `AppData\Local\Programs\OpenAI\Codex\bin\codex.exe`. Preservou credenciais sem modificá-las. |
| **Cenário C: CLI não autenticada** | `CODEX_HOME` apontado para pasta vazia. | **PASS** | Distinguiu instalação de autenticação: reportou `AUTH_REQUIRED` / `Not authenticated` sem travar e sem apresentar como pronto. |
| **Cenário D: Múltiplas contas Codex** | Criação de perfis `work` e `personal` via `nexus add codex <name> --no-login`. | **PASS** | IDs de conta distintos gerados (`9e6b9371...` e `681177e4...`). Troca de perfil ativo via `nexus use` sem vazamento de cotas. Auditoria de segurança protegeu `.ssh`, `.git-credentials`, `.gnupg`, etc. |
| **Cenário E: Persistência e Upgrade** | Criação de projeto e perfil no SQLite; fechamento e reabertura em segunda execução. | **PASS** | Projeto `Persisted Project [prj_06GN7MSTX07NVRC9XZDX0GGNJ0]` e perfil `persisted_profile` mantidos intactos no banco. |

---

## Fase 5 — Cadastro de Projetos (Windows Edge Cases)

| Caso de Teste | Caminho Testado | Resultado | Detalhes |
| :--- | :--- | :--- | :--- |
| **Caminho com espaços** | `C:\Users\...\Temp\...\Pasta Com Espacos No Nome` | **PASS** | Registrado com sucesso (`prj_06GN7MWQ65MZKX5GK5NN30D2MR`). |
| **Caminho com Unicode e acentos** | `C:\Users\...\Temp\...\Projeto Ação & Código Çã` | **PASS** | Registrado com sucesso (`prj_06GN7MWQC3H70TN3XPGYFQVS2R`). |
| **Diretório Git existente** | `C:\Users\...\Temp\...\Git Repo Project` (com `.git`) | **PASS** | Registrado com sucesso (`prj_06GN7MWQTSCCS2EAREFN4X2VH4`). |
| **Diretório inexistente na CLI** | `C:\Users\...\Temp\...\Nao Existe Mesmoxpto123` | **WARN** | CLI registra o registro no SQLite sem verificar existência prévia no disco (Web UI valida via `nexus.inspectFS`). |
| **Placeholder de Caminho Windows** | `getRepositoryPathPlaceholderKey` em `web/src/...` | **PASS** | Exibe `C:\Projetos\meu-repositorio` (pt-BR) / `C:\Projects\my-repository` (en). Falso template `/projetos/saas-app` eliminado. |
