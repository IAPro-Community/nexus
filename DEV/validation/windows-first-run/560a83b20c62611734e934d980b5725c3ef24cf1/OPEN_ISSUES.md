# Registro de Problemas Identificados (Open Issues) — Nexus 1.0.0 (Windows)

- **Commit SHA:** `560a83b20c62611734e934d980b5725c3ef24cf1`
- **Data da Homologação:** 2026-10-18 / 2026-10-19

---

## Tabela de Severidade e Impacto

| ID | Título | Severidade | Impacto no Lançamento | Status |
| :--- | :--- | :--- | :--- | :--- |
| **ISSUE-01** | Erro de tipagem TypeScript no componente `NexusHud.tsx` | **BLOCKER** | Bloqueia gate de qualidade `typecheck` e script oficial de certificação | Aberto |
| **ISSUE-02** | Crash no `install.ps1` ao fornecer `-InstallDir` no Windows | **BLOCKER** | Impede instalação em diretórios customizados ou sandboxes | Aberto |
| **ISSUE-03** | Inexistência de assets pré-compilados no repositório de releases | **BLOCKER** | `install.ps1 -Version latest` falha com HTTP 404 para usuários comuns | Aberto |
| **ISSUE-04** | Arquivo Go não formatado com `gofmt` (`capacity_lifecycle_test.go`) | **HIGH** | Bloqueia script `certify-tokenbar-integration.ps1` e CI estrito | Aberto |
| **ISSUE-05** | Restrição de versão rígida `go1.26.9` em `homologate-beta.ps1` | **HIGH** | Impede homologação em máquinas Windows com Go 1.25.x | Aberto |
| **ISSUE-06** | Incompatibilidade de `$IsWindows` no PowerShell 5.1 em `test-uninstall-windows.ps1` | **MEDIUM** | Falha de teste no PowerShell padrão do Windows sem `pwsh` | Aberto |
| **ISSUE-07** | Manifesto visual de documentação desatualizado (`docs-verify.mjs`) | **MEDIUM** | Gate de documentação falha após alterações recentes de interface | Aberto |
| **ISSUE-08** | CLI `nexus projects add` não valida existência da pasta no disco | **LOW** | Permite registrar caminhos inexistentes via linha de comando | Aberto |

---

## Detalhamento Técnico dos Problemas

### ISSUE-01 — Erro de tipagem TypeScript em `NexusHud.tsx` (BLOCKER)
- **Comando:** `bun run --cwd web typecheck`
- **Código de Saída:** `2`
- **Trecho do Log:**
  ```text
  src/features/hud/NexusHud.tsx(533,60): error TS2551: Property 'known-partial' does not exist on type '{ readonly reported: "Informado pelo provedor"; readonly estimated: "Estimativa"; readonly 'known-part': "Parcial"; readonly unknown: "Desconhecido"; }'. Did you mean 'known-part'?
  ```
- **Causa Raiz:** O tipo do mapa de rótulos define a chave como `'known-part'`, mas a linha 533 do componente `NexusHud.tsx` tenta acessar `'known-partial'`.
- **Correção Necessária:** Alinhar o identificador em `NexusHud.tsx` para `'known-part'` (ou atualizar o dicionário de tipos correspondente).

---

### ISSUE-02 — Crash no `install.ps1` ao utilizar `-InstallDir` no Windows (BLOCKER)
- **Comando:** `.\install.ps1 -BuildFromSource -InstallDir "C:\Caminho\Qualquer" -Yes`
- **Código de Saída:** `1`
- **Trecho do Log:**
  ```text
  Test-Path : Não é possível associar o argumento ao parâmetro 'Path' porque ele é uma cadeia de caracteres vazia.
  No D:\Dev\nexus-remediation\nexus-homologation\install.ps1:204 caractere:34
  + if ($IsWindowsOS -and (Test-Path $LegacyTargetDir)) {
  +                                  ~~~~~~~~~~~~~~~~
      + CategoryInfo          : InvalidData: (:) [Test-Path], ParentContainsErrorRecordException
      + FullyQualifiedErrorId : ParameterArgumentValidationErrorEmptyStringNotAllowed,Microsoft.PowerShell.Commands.TestPathCommand
  ```
- **Causa Raiz:** Na linha 189 de `install.ps1`, quando `$InstallDir` é passado, `$LegacyTargetDir` é definido como `""`. Na linha 204, o instalador executa `Test-Path $LegacyTargetDir`. Como `$ErrorActionPreference = 'Stop'`, passar uma string vazia para `Test-Path` dispara erro fatal imediato no PowerShell do Windows.
- **Correção Necessária:** Alterar a linha 204 para verificar se a variável não está vazia:
  ```powershell
  if ($IsWindowsOS -and (-not [string]::IsNullOrWhiteSpace($LegacyTargetDir)) -and (Test-Path $LegacyTargetDir)) {
  ```

---

### ISSUE-03 — Inexistência de assets pré-compilados em `IAPro-Community/nexus-releases` (BLOCKER)
- **Comando:** `.\install.ps1 -Version latest`
- **Código de Saída:** `1`
- **Trecho do Log:**
  ```text
  Invoke-RestMethod : O servidor remoto retornou um erro: (404) Não Localizado.
  (endpoint: https://api.github.com/repos/IAPro-Community/nexus-releases/releases/latest)
  ```
- **Causa Raiz:** Não há release publicada no repositório de distribuição binária pública. O usuário comum do Windows não consegue executar a instalação simplificada de um clique sem compilar do zero via código-fonte.
- **Correção Necessária:** Publicar o release oficial com os binários assinados, os pacotes zip (`nexus_Windows_x86_64.zip`, `nexus-desktop_Windows_x86_64.zip`), o instalador NSIS e os arquivos `update-manifest.json` com `update-manifest.sig`.

---

### ISSUE-04 — Arquivo Go com violação de formatação `gofmt` (HIGH)
- **Comando:** `gofmt -l internal/nexus/capacity_lifecycle_test.go`
- **Código de Saída:** `0` (imprime arquivo não formatado)
- **Trecho do Diff:**
  ```diff
  --- internal/nexus/capacity_lifecycle_test.go.orig
  +++ internal/nexus/capacity_lifecycle_test.go
  @@ -121,1 +121,1 @@
  -			ObservedAt: now.Add(-time.Duration(20-i*10) * time.Minute),
  +			ObservedAt:  now.Add(-time.Duration(20-i*10) * time.Minute),
  ```
- **Causa Raiz:** Falta de alinhamento de espaço simples no struct literal.
- **Correção Necessária:** Executar `gofmt -w internal/nexus/capacity_lifecycle_test.go`.

---

### ISSUE-05 — Restrição rígida de Go `go1.26.9` em `homologate-beta.ps1` (HIGH)
- **Comando:** `.\scripts\homologate-beta.ps1`
- **Código de Saída:** `1`
- **Trecho do Log:**
  ```text
  Windows beta homologation requires Go 1.26.9 to match CI; got: go version go1.25.14 windows/amd64
  ```
- **Causa Raiz:** O script `scripts/homologate-beta.ps1` exige estritamente a versão `go1.26.9` via regex na linha 45, enquanto `certify-tokenbar-integration.ps1` aceita `go >= 1.25`. Como a versão oficial atual instalada é `go1.25.14`, o gate beta é bloqueado localmente.
- **Correção Necessária:** Permitir Go >= 1.25 na validação local ou documentar o requisito explícito de upgrade de Go no ambiente de build.

---

### ISSUE-06 — Incompatibilidade com Windows PowerShell 5.1 em `test-uninstall-windows.ps1` (MEDIUM)
- **Comando:** `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-uninstall-windows.ps1`
- **Código de Saída:** `1`
- **Trecho do Log:**
  ```text
  A variável '$IsWindows' não pode ser recuperada porque ainda não foi definida.
  + CategoryInfo: InvalidOperation: (IsWindows:String) [test-uninstall-windows.ps1], RuntimeException
  ```
- **Causa Raiz:** O PowerShell 5.1 (padrão de fábrica do Windows) não possui a variável automática `$IsWindows` (introduzida no PowerShell Core 6+). Sob `Set-StrictMode -Version Latest`, tentar ler `$IsWindows` lança uma exceção terminating.
- **Correção Necessária:** Em `scripts/test-uninstall-windows.ps1`, inicializar ou verificar `$IsWindows`:
  ```powershell
  $isWin = if (Get-Variable -Name IsWindows -ErrorAction SilentlyContinue) { $IsWindows } else { ($env:OS -like "*Windows*") }
  ```

---

### ISSUE-07 — Manifesto Visual de Documentação Desatualizado (MEDIUM)
- **Comando:** `bun scripts/docs-verify.mjs`
- **Código de Saída:** `1`
- **Trecho do Log:**
  ```text
  ✗ visual manifest is stale after visual changes since 93becc78e6b6c6f801ffc390abdd89687ef868b5: cmd/nexus-desktop/main.go, internal/desktop/hud_window.go, web/src/features/projects/AddProjectModal.tsx, ...
  ```
- **Causa Raiz:** Foram adicionadas melhorias nas telas do Nexus sem a devida atualização do manifesto visual com `bun scripts/refresh-beta-docs-windows.ps1`.
- **Correção Necessária:** Atualizar o manifesto visual da documentação antes de fechar a tag de release.
