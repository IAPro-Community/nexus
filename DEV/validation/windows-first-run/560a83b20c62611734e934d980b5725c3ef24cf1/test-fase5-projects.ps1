[CmdletBinding()]
param()

$root = (Resolve-Path .).Path
$evidenceDir = Join-Path $root 'DEV\validation\windows-first-run\560a83b20c62611734e934d980b5725c3ef24cf1'
$logsDir = Join-Path $evidenceDir 'logs'
$nexusBin = Join-Path $evidenceDir 'bin\nexus.exe'
$logFase5 = Join-Path $logsDir '18-fase5-projects.log'

"" | Set-Content $logFase5 -Encoding utf8

function Log5([string]$text) {
    Write-Host $text
    $text | Out-File -FilePath $logFase5 -Append -Encoding utf8
}

Log5 "=================================================================="
Log5 "FASE 5: Cadastro de Projetos (Caminhos Reais Windows e Edge Cases)"
Log5 "=================================================================="

$sandbox5 = Join-Path $env:TEMP ("nexus-test-fase5-" + [System.Guid]::NewGuid().ToString('N'))
$data5 = Join-Path $sandbox5 "data"
$config5 = Join-Path $sandbox5 "config"
$state5 = Join-Path $sandbox5 "state"
New-Item -ItemType Directory -Force -Path $data5, $config5, $state5 | Out-Null

$env:NEXUS_DATA_DIR = $data5
$env:NEXUS_CONFIG_DIR = $config5
$env:NEXUS_STATE_DIR = $state5

try {
    # Case 1: Directory with spaces
    $dirSpaces = Join-Path $sandbox5 "Pasta Com Espacos No Nome"
    New-Item -ItemType Directory -Force -Path $dirSpaces | Out-Null
    Log5 "`n--- Test 1: Directory with spaces ($dirSpaces) ---"
    $r1 = & $nexusBin projects add $dirSpaces --name 'Espacos Project' 2>&1
    Log5 ($r1 -join "`n")

    # Case 2: Directory with Unicode / accents
    $dirUnicode = Join-Path $sandbox5 "Projeto Ação & Código Çã"
    New-Item -ItemType Directory -Force -Path $dirUnicode | Out-Null
    Log5 "`n--- Test 2: Directory with Unicode and accents ($dirUnicode) ---"
    $r2 = & $nexusBin projects add $dirUnicode --name 'Unicode Project' 2>&1
    Log5 ($r2 -join "`n")

    # Case 3: Git initialized directory
    $dirGit = Join-Path $sandbox5 "Git Repo Project"
    New-Item -ItemType Directory -Force -Path $dirGit | Out-Null
    & git -C $dirGit init -q
    Log5 "`n--- Test 3: Directory with Git repo ($dirGit) ---"
    $r3 = & $nexusBin projects add $dirGit --name 'Git Project' 2>&1
    Log5 ($r3 -join "`n")

    # Case 4: Nonexistent directory (should fail gracefully)
    $dirNonexistent = Join-Path $sandbox5 "Nao Existe Mesmoxpto123"
    Log5 "`n--- Test 4: Nonexistent directory ($dirNonexistent) ---"
    $r4 = & $nexusBin projects add $dirNonexistent --name 'Invalid Project' 2>&1
    Log5 ($r4 -join "`n")

    # Case 5: List registered projects
    Log5 "`n--- List registered projects ---"
    $rList = & $nexusBin projects list 2>&1
    Log5 ($rList -join "`n")

    # Assertions
    Log5 "`n[FASE 5 ASSERTIONS]"
    $hasSpaces = ($rList -match 'Espacos Project')
    $hasUnicode = ($rList -match 'Unicode Project')
    $hasGit = ($rList -match 'Git Project')
    $rejectedNonexistent = ($r4 -match 'não existe' -or $r4 -match 'no such file' -or $r4 -match 'erro' -or $r4 -match 'invalid' -or $LASTEXITCODE -ne 0)

    Log5 "Project with spaces registered: $hasSpaces"
    Log5 "Project with Unicode registered: $hasUnicode"
    Log5 "Project with Git registered: $hasGit"
    Log5 "Nonexistent path properly handled: $rejectedNonexistent"

    if ($hasSpaces -and $hasUnicode -and $hasGit) {
        Log5 "`nFASE 5 PROJECT REGISTRATION: PASS"
    } else {
        Log5 "`nFASE 5 PROJECT REGISTRATION: FAIL"
    }
} finally {
    Remove-Item Env:NEXUS_DATA_DIR -ErrorAction SilentlyContinue
    Remove-Item Env:NEXUS_CONFIG_DIR -ErrorAction SilentlyContinue
    Remove-Item Env:NEXUS_STATE_DIR -ErrorAction SilentlyContinue
    Remove-Item -Path $sandbox5 -Recurse -Force -ErrorAction SilentlyContinue
    Log5 "`nSandbox cleaned up."
}
