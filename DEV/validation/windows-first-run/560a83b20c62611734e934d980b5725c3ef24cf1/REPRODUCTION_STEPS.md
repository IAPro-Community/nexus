# Passos de Reprodução (Reproduction Steps) — Nexus 1.0.0 (Windows)

Este documento contém os comandos exatos para reproduzir localmente todos os testes, falhas e validações documentadas na homologação do commit `560a83b20c62611734e934d980b5725c3ef24cf1`.

---

## 1. Verificação do Ambiente e Branch

```powershell
# Verificar branch e commit exato
git branch --show-current
# Esperado: release/v1.0.0

git rev-parse HEAD
# Esperado: 560a83b20c62611734e934d980b5725c3ef24cf1

git status --short
# Esperado: limpo (nenhum arquivo modificado)
```

---

## 2. Reprodução dos Bloqueadores e Falhas Automatizadas

### A. Falha de Typecheck Frontend (ISSUE-01)
```powershell
bun run --cwd web typecheck
# Código de saída esperado: 2
# Erro: TS2551 em src/features/hud/NexusHud.tsx(533,60)
```

### B. Falha de Instalação com -InstallDir (ISSUE-02)
```powershell
$tempDir = Join-Path $env:TEMP "repro-install-dir"
.\install.ps1 -BuildFromSource -InstallDir $tempDir -Yes
# Código de saída esperado: 1
# Erro: Test-Path : Não é possível associar o argumento ao parâmetro 'Path' porque ele é uma cadeia de caracteres vazia (install.ps1:204)
```

### C. Falha de Release Remoto Ausente (ISSUE-03)
```powershell
.\install.ps1 -Version latest
# Código de saída esperado: 1
# Erro: 404 Não Localizado em https://api.github.com/repos/IAPro-Community/nexus-releases/releases/latest
```

### D. Falha de Formatação Go (ISSUE-04)
```powershell
gofmt -l internal/nexus/capacity_lifecycle_test.go
# Esperado: imprime 'internal/nexus/capacity_lifecycle_test.go'

.\scripts\certify-tokenbar-integration.ps1
# Código de saída esperado: 1
# Erro: Go files require gofmt before certifying: internal/nexus/capacity_lifecycle_test.go
```

### E. Bloqueio de Versão de Go no Beta Homologation (ISSUE-05)
```powershell
.\scripts\homologate-beta.ps1
# Código de saída esperado: 1
# Erro: Windows beta homologation requires Go 1.26.9 to match CI; got: go version go1.25.14 windows/amd64
```

### F. Falha em PowerShell 5.1 StrictMode (ISSUE-06)
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-uninstall-windows.ps1
# Código de saída esperado: 1
# Erro: A variável '$IsWindows' não pode ser recuperada porque ainda não foi definida.
```

---

## 3. Reprodução dos Sucessos de Domínio e Terminais ConPTY

### A. Testes Go do Domínio
```powershell
go test ./internal/control/web ./internal/nexus/... ./internal/doctor/...
# Esperado: PASS (exit code 0)
```

### B. Testes Web (Vitest) e Build Frontend
```powershell
bun run --cwd web test
# Esperado: 90 test files passed, 565 passed (exit code 0)

bun run --cwd web build
# Esperado: bundle gerado em web/dist (exit code 0)
```

### C. Testes Nativos ConPTY e Terminais Windows
```powershell
# Alocação ConPTY, CreatePseudoConsole e teclas direcionais
go test -v ./internal/control/terminal -run ConPTY

# Encaminhamento de setas e prevenção de colapso no SessionHost
go test -v ./internal/control/host -run "Arrow|Terminal"

# Coalescência de sequências de escape contra fragmentação ConPTY
go test -v ./internal/runtime -run "Escape|ConPTY"
# Esperado: todos PASS (exit code 0)
```

### D. Testes de Desktop e HUD
```powershell
go test -v ./internal/desktop/...
# Esperado: todos PASS (exit code 0)
```

---

## 4. Reprodução dos Cenários de Primeira Execução (Sandbox)

Para rodar os cenários de primeira execução sem afetar os dados reais:
```powershell
# Executar o script de teste automatizado de cenários
& "DEV\validation\windows-first-run\560a83b20c62611734e934d980b5725c3ef24cf1\test-fase4-5.ps1"
# Esperado: Cenários A, B, C, D e E com PASS
```
