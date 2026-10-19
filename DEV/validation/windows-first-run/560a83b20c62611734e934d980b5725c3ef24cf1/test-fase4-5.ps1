[CmdletBinding()]
param()

$root = (Resolve-Path .).Path
$evidenceDir = Join-Path $root 'DEV\validation\windows-first-run\560a83b20c62611734e934d980b5725c3ef24cf1'
$logsDir = Join-Path $evidenceDir 'logs'
$nexusBin = Join-Path $evidenceDir 'bin\nexus.exe'

$logFase4 = Join-Path $logsDir '17-fase4-scenarios.log'
$logFase5 = Join-Path $logsDir '18-fase5-projects.log'

"" | Set-Content $logFase4
"" | Set-Content $logFase5

function Log4([string]$text) {
    Write-Host $text
    $text | Out-File -FilePath $logFase4 -Append -Encoding utf8
}

function Log5([string]$text) {
    Write-Host $text
    $text | Out-File -FilePath $logFase5 -Append -Encoding utf8
}

# ==============================================================================
# FASE 4 — CENÁRIO A: Nenhum provedor instalado
# ==============================================================================
Log4 "=================================================================="
Log4 "FASE 4 - CENÁRIO A: Nenhum provedor instalado (Clean Room Sandbox)"
Log4 "=================================================================="

$sandboxA = Join-Path $env:TEMP ("nexus-test-scenA-" + [System.Guid]::NewGuid().ToString('N'))
$dataA = Join-Path $sandboxA "data"
$configA = Join-Path $sandboxA "config"
$stateA = Join-Path $sandboxA "state"
$codexA = Join-Path $sandboxA "codex_home"
$projA = Join-Path $sandboxA "test-project"

New-Item -ItemType Directory -Force -Path $dataA, $configA, $stateA, $codexA, $projA | Out-Null

$cleanPath = "C:\Windows\system32;C:\Windows;C:\Windows\System32\Wbem"

# Subshell with stripped PATH and isolated dirs
$cmdA = @"
`$env:NEXUS_DATA_DIR = '$dataA'
`$env:NEXUS_CONFIG_DIR = '$configA'
`$env:NEXUS_STATE_DIR = '$stateA'
`$env:CODEX_HOME = '$codexA'
`$env:PATH = '$cleanPath'

Write-Host "--- Checking providers status --json ---"
& '$nexusBin' providers status --json

Write-Host "--- Checking doctor --json ---"
& '$nexusBin' doctor --json

Write-Host "--- Registering project without any provider ---"
& '$nexusBin' projects add '$projA' --name 'Project Without Provider' --isolation project

Write-Host "--- Listing registered projects ---"
& '$nexusBin' projects list
"@

$resA = powershell -NoProfile -ExecutionPolicy Bypass -Command $cmdA 2>&1
Log4 ($resA -join "`n")

# Assertions for Scenario A
$jsonProvidersA = ($resA | Where-Object { $_ -match '^\[.*\]$' -or $_ -match '^\{.*\}$' })
Log4 "`n[CENÁRIO A ASSERTIONS]"
if ($resA -match 'Project Without Provider' -and $resA -match 'Projeto registrado com sucesso') {
    Log4 "PASS: Project was registered successfully even when NO provider is installed."
} else {
    Log4 "FAIL: Could not register project without provider."
}

# ==============================================================================
# FASE 4 — CENÁRIO B: Codex já instalado e autenticado
# ==============================================================================
Log4 "`n=================================================================="
Log4 "FASE 4 - CENÁRIO B: Codex instalado e autenticado (Real Environment)"
Log4 "=================================================================="

$sandboxB = Join-Path $env:TEMP ("nexus-test-scenB-" + [System.Guid]::NewGuid().ToString('N'))
$dataB = Join-Path $sandboxB "data"
$configB = Join-Path $sandboxB "config"
$stateB = Join-Path $sandboxB "state"
New-Item -ItemType Directory -Force -Path $dataB, $configB, $stateB | Out-Null

$cmdB = @"
`$env:NEXUS_DATA_DIR = '$dataB'
`$env:NEXUS_CONFIG_DIR = '$configB'
`$env:NEXUS_STATE_DIR = '$stateB'

Write-Host "--- providers status --json ---"
& '$nexusBin' providers status --json

Write-Host "--- doctor --json ---"
& '$nexusBin' doctor --json

Write-Host "--- profiles --json ---"
& '$nexusBin' profiles --json
"@

$resB = powershell -NoProfile -ExecutionPolicy Bypass -Command $cmdB 2>&1
Log4 ($resB -join "`n")

# ==============================================================================
# FASE 4 — CENÁRIO C: CLI instalada, mas não autenticada
# ==============================================================================
Log4 "`n=================================================================="
Log4 "FASE 4 - CENÁRIO C: CLI instalada, mas não autenticada"
Log4 "=================================================================="

$sandboxC = Join-Path $env:TEMP ("nexus-test-scenC-" + [System.Guid]::NewGuid().ToString('N'))
$dataC = Join-Path $sandboxC "data"
$configC = Join-Path $sandboxC "config"
$stateC = Join-Path $sandboxC "state"
$codexEmpty = Join-Path $sandboxC "empty_codex_home"
New-Item -ItemType Directory -Force -Path $dataC, $configC, $stateC, $codexEmpty | Out-Null

$cmdC = @"
`$env:NEXUS_DATA_DIR = '$dataC'
`$env:NEXUS_CONFIG_DIR = '$configC'
`$env:NEXUS_STATE_DIR = '$stateC'
`$env:CODEX_HOME = '$codexEmpty'

Write-Host "--- providers status --json (with unauth CODEX_HOME) ---"
& '$nexusBin' providers status --json

Write-Host "--- doctor --json (with unauth CODEX_HOME) ---"
& '$nexusBin' doctor --json
"@

$resC = powershell -NoProfile -ExecutionPolicy Bypass -Command $cmdC 2>&1
Log4 ($resC -join "`n")

# ==============================================================================
# FASE 4 — CENÁRIO D: Múltiplas contas / perfis
# ==============================================================================
Log4 "`n=================================================================="
Log4 "FASE 4 - CENÁRIO D: Múltiplas contas e isolamento de perfis"
Log4 "=================================================================="

$sandboxD = Join-Path $env:TEMP ("nexus-test-scenD-" + [System.Guid]::NewGuid().ToString('N'))
$dataD = Join-Path $sandboxD "data"
$configD = Join-Path $sandboxD "config"
$stateD = Join-Path $sandboxD "state"
New-Item -ItemType Directory -Force -Path $dataD, $configD, $stateD | Out-Null

$cmdD = @"
`$env:NEXUS_DATA_DIR = '$dataD'
`$env:NEXUS_CONFIG_DIR = '$configD'
`$env:NEXUS_STATE_DIR = '$stateD'

Write-Host "--- Adding profile codex work ---"
& '$nexusBin' add codex work --no-login

Write-Host "--- Adding profile codex personal ---"
& '$nexusBin' add codex personal --no-login

Write-Host "--- Listing profiles --json ---"
& '$nexusBin' profiles --json

Write-Host "--- Setting active profile to personal ---"
& '$nexusBin' use codex personal

Write-Host "--- Current active profile ---"
& '$nexusBin' current

Write-Host "--- Security audit on profile isolation ---"
& '$nexusBin' security --json
"@

$resD = powershell -NoProfile -ExecutionPolicy Bypass -Command $cmdD 2>&1
Log4 ($resD -join "`n")

# ==============================================================================
# FASE 4 — CENÁRIO E: Instalação existente, persistência e atualização
# ==============================================================================
Log4 "`n=================================================================="
Log4 "FASE 4 - CENÁRIO E: Persistência SQLite e atualização"
Log4 "=================================================================="

$sandboxE = Join-Path $env:TEMP ("nexus-test-scenE-" + [System.Guid]::NewGuid().ToString('N'))
$dataE = Join-Path $sandboxE "data"
$configE = Join-Path $sandboxE "config"
$stateE = Join-Path $sandboxE "state"
New-Item -ItemType Directory -Force -Path $dataE, $configE, $stateE | Out-Null

$projDirE = Join-Path $sandboxE "persisted-project"
New-Item -ItemType Directory -Force -Path $projDirE | Out-Null

$cmdE1 = @"
`$env:NEXUS_DATA_DIR = '$dataE'
`$env:NEXUS_CONFIG_DIR = '$configE'
`$env:NEXUS_STATE_DIR = '$stateE'

Write-Host "--- Initial creation: add project ---"
& '$nexusBin' projects add '$projDirE' --name 'Persisted Project' --isolation project
& '$nexusBin' add codex persisted_profile --no-login
& '$nexusBin' projects list
"@

$resE1 = powershell -NoProfile -ExecutionPolicy Bypass -Command $cmdE1 2>&1
Log4 "First run:`n$($resE1 -join "`n")"

# Simulate reopening / updating
$cmdE2 = @"
`$env:NEXUS_DATA_DIR = '$dataE'
`$env:NEXUS_CONFIG_DIR = '$configE'
`$env:NEXUS_STATE_DIR = '$stateE'

Write-Host "--- Second run (simulating upgrade / restart) ---"
& '$nexusBin' projects list
& '$nexusBin' profiles --json
"@

$resE2 = powershell -NoProfile -ExecutionPolicy Bypass -Command $cmdE2 2>&1
Log4 "Second run:`n$($resE2 -join "`n")"

if ($resE2 -match 'Persisted Project' -and $resE2 -match 'persisted_profile') {
    Log4 "PASS: SQLite data, projects and profiles preserved across restarts."
} else {
    Log4 "FAIL: Data loss detected across restarts."
}

# ==============================================================================
# FASE 5 — Cadastro de Projetos (Windows Edge Cases)
# ==============================================================================
Log5 "=================================================================="
Log5 "FASE 5: Cadastro de Projetos (Caminhos Reais Windows e Edge Cases)"
Log5 "=================================================================="

$sandbox5 = Join-Path $env:TEMP ("nexus-test-fase5-" + [System.Guid]::NewGuid().ToString('N'))
$data5 = Join-Path $sandbox5 "data"
$config5 = Join-Path $sandbox5 "config"
$state5 = Join-Path $sandbox5 "state"
New-Item -ItemType Directory -Force -Path $data5, $config5, $state5 | Out-Null

# Case 1: Directory with spaces
$dirSpaces = Join-Path $sandbox5 "Pasta Com Espacos No Nome"
New-Item -ItemType Directory -Force -Path $dirSpaces | Out-Null

# Case 2: Directory with Unicode / accents
$dirUnicode = Join-Path $sandbox5 "Projeto Ação & Código Çã"
New-Item -ItemType Directory -Force -Path $dirUnicode | Out-Null

# Case 3: Git initialized directory
$dirGit = Join-Path $sandbox5 "Git Repo Project"
New-Item -ItemType Directory -Force -Path $dirGit | Out-Null
& git -C $dirGit init -q

# Case 4: Nonexistent directory (should fail gracefully)
$dirNonexistent = Join-Path $sandbox5 "Nao Existe Mesmoxpto123"

$cmd5 = @"
`$env:NEXUS_DATA_DIR = '$data5'
`$env:NEXUS_CONFIG_DIR = '$config5'
`$env:NEXUS_STATE_DIR = '$state5'

Write-Host "--- Test 1: Directory with spaces ---"
& '$nexusBin' projects add '$dirSpaces' --name 'Espacos Project'

Write-Host "--- Test 2: Directory with Unicode & accents ---"
& '$nexusBin' projects add '$dirUnicode' --name 'Unicode Project'

Write-Host "--- Test 3: Directory with Git repo ---"
& '$nexusBin' projects add '$dirGit' --name 'Git Project'

Write-Host "--- Test 4: Nonexistent directory ---"
& '$nexusBin' projects add '$dirNonexistent' --name 'Invalid Project'

Write-Host "--- List registered projects ---"
& '$nexusBin' projects list
"@

$res5 = powershell -NoProfile -ExecutionPolicy Bypass -Command $cmd5 2>&1
Log5 ($res5 -join "`n")

# Cleanup sandboxes
Remove-Item -Path $sandboxA, $sandboxB, $sandboxC, $sandboxD, $sandboxE, $sandbox5 -Recurse -Force -ErrorAction SilentlyContinue
Log4 "`nSandboxes cleaned up."
Log5 "`nSandboxes cleaned up."
