[CmdletBinding()]
param()

$root = (Resolve-Path .).Path
$evidenceDir = Join-Path $root 'DEV\validation\windows-first-run\560a83b20c62611734e934d980b5725c3ef24cf1'
$logsDir = Join-Path $evidenceDir 'logs'
if (-not (Test-Path $logsDir)) {
    New-Item -ItemType Directory -Force -Path $logsDir | Out-Null
}

function Run-Step([string]$Name, [string]$LogFile, [scriptblock]$Action) {
    Write-Host "==> Running: $Name"
    $fullLogPath = Join-Path $logsDir $LogFile
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    $exitCode = 0
    try {
        & $Action 2>&1 | Tee-Object -FilePath $fullLogPath
        if ($LASTEXITCODE -ne $null) {
            $exitCode = $LASTEXITCODE
        }
    } catch {
        $_ | Out-File -FilePath $fullLogPath -Append
        $exitCode = 1
    }
    $sw.Stop()
    $status = if ($exitCode -eq 0) { "PASS" } else { "FAIL (exit $exitCode)" }
    Write-Host "[$status] $Name (${sw.ElapsedMilliseconds} ms)"
    return [PSCustomObject]@{
        Name = $Name
        ExitCode = $exitCode
        DurationMs = $sw.ElapsedMilliseconds
        LogFile = $LogFile
    }
}

# 1. PowerShell Syntax
Run-Step "PowerShell Syntax Validation" "09-powershell-syntax.log" {
    $scripts = @('install.ps1', 'uninstall.ps1', 'scripts/test-uninstall-windows.ps1', 'scripts/homologate-beta.ps1', 'scripts/refresh-beta-docs-windows.ps1')
    foreach ($item in $scripts) {
        $tokens = $null; $errors = $null
        [System.Management.Automation.Language.Parser]::ParseFile((Resolve-Path (Join-Path $root $item)).Path, [ref]$tokens, [ref]$errors) | Out-Null
        if ($errors.Count -gt 0) {
            throw "Syntax error in $item : $($errors -join '; ')"
        }
        Write-Host "PASS: $item"
    }
}

# 2. Windows uninstaller isolation test
Run-Step "Windows Uninstaller Isolation Unit Test" "10-uninstall-windows-smoke.log" {
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $root 'scripts/test-uninstall-windows.ps1')
}

# 3. Documentation verification
Run-Step "Documentation Verification (docs-verify.mjs)" "11-docs-verify.log" {
    bun scripts/docs-verify.mjs
}

# 4. Tracked Go files gofmt check
Run-Step "Tracked Go files gofmt check" "12-gofmt-check.log" {
    $trackedGo = @(git ls-files -- '*.go')
    $unformatted = @()
    foreach ($gf in $trackedGo) {
        $res = @(gofmt -l (Join-Path $root $gf))
        if ($res.Count -gt 0) {
            $unformatted += $res
        }
    }
    if ($unformatted.Count -gt 0) {
        Write-Host "Unformatted files found:"
        $unformatted | ForEach-Object { Write-Host "  $_" }
        throw "gofmt required on $($unformatted.Count) file(s)"
    } else {
        Write-Host "All $($trackedGo.Count) tracked Go files are properly formatted."
    }
}

# 5. Go vet
Run-Step "Go vet ./..." "13-go-vet.log" {
    go vet ./...
}

# 6. Version contract check
Run-Step "Version contract check" "14-version-contract.log" {
    go run ./scripts/verify-version-contract.go .
}

# 7. Interactive terminal regression preflight
Run-Step "Provider-native Mission/terminal regression preflight" "15-regression-terminal.log" {
    go test -count=1 ./internal/app ./internal/nexus/promotion ./internal/runtime
}
